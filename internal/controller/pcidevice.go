/*
Copyright 2024 Karel Van Hecke

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"time"

	"github.com/digitalocean/go-libvirt"
	"github.com/karelvanhecke/libvirt-operator/api/v1alpha1"
	"github.com/karelvanhecke/libvirt-operator/internal/store"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"libvirt.org/go/libvirtxml"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	ConditionMessagePCIDeviceDataRetrievalInProgress = "PCIDevice data retrieval in progress"
	ConditionMessagePCIDeviceDataRetrievalSucceeded  = "PCIDevice data retrieval succeeded"
)

type PCIDeviceReconciler struct {
	client.Client
	HostStore *store.HostStore
}

func (r *PCIDeviceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (res ctrl.Result, err error) {
	pciDevice := &v1alpha1.PCIDevice{}

	err = r.Get(ctx, req.NamespacedName, pciDevice)
	if err != nil {
		return
	}

	if !meta.IsStatusConditionPresentAndEqual(pciDevice.Status.Conditions, ConditionTypeDataRetrieved, metav1.ConditionTrue) {
		meta.SetStatusCondition(&pciDevice.Status.Conditions, metav1.Condition{
			Type:               ConditionTypeDataRetrieved,
			Status:             metav1.ConditionFalse,
			Message:            ConditionMessagePCIDeviceDataRetrievalInProgress,
			Reason:             ConditionReasonInProgress,
			LastTransitionTime: metav1.Now(),
		})
		err = r.Status().Update(ctx, pciDevice)
		if err != nil {
			return
		}
	}

	if pciDevice.DeletionTimestamp.IsZero() {
		if controllerutil.AddFinalizer(pciDevice, Finalizer) {
			err = r.Update(ctx, pciDevice)
			if err != nil {
				return
			}
		}
	} else {
		if controllerutil.ContainsFinalizer(pciDevice, Finalizer) {
			// TODO
			ctrl.LoggerFrom(ctx).V(1).Info("check if pciDevice is referred by domains")
		}
		controllerutil.RemoveFinalizer(pciDevice, Finalizer)
		err = r.Update(ctx, pciDevice)
		if err != nil {
			return
		}
	}

	hostRef := &v1alpha1.Host{}
	if err := r.Get(ctx, types.NamespacedName{Name: pciDevice.Spec.HostRef.Name, Namespace: pciDevice.Namespace}, hostRef); err != nil {
		if !meta.IsStatusConditionTrue(pciDevice.Status.Conditions, ConditionTypeCreated) {
			meta.SetStatusCondition(&pciDevice.Status.Conditions, metav1.Condition{
				Type:               ConditionTypeDataRetrieved,
				Status:             metav1.ConditionFalse,
				Message:            ConditionMessageHostNotFound,
				Reason:             ConditionReasonFailed,
				LastTransitionTime: metav1.Now(),
			})
			if err := r.Status().Update(ctx, pciDevice); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, err
	}

	hostEntry, found := r.HostStore.Lookup(hostRef.UID)
	if !found {
		if !meta.IsStatusConditionTrue(pciDevice.Status.Conditions, ConditionTypeCreated) {
			meta.SetStatusCondition(&pciDevice.Status.Conditions, metav1.Condition{
				Type:               ConditionTypeDataRetrieved,
				Status:             metav1.ConditionFalse,
				Message:            ConditionMessageWaitingForHost,
				Reason:             ConditionReasonFailed,
				LastTransitionTime: metav1.Now(),
			})
			if err := r.Status().Update(ctx, pciDevice); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{Requeue: true}, nil
	}
	hClient, end := hostEntry.Session()
	defer end()

	if ts := pciDevice.Status.LastUpdate; ts != nil {
		if d := time.Since(ts.Time); d < DataRefreshInterval {
			return ctrl.Result{RequeueAfter: DataRefreshInterval - d}, nil
		}
	}

	var pciDeviceID libvirt.NodeDevice
	pciDeviceID, err = hClient.NodeDeviceLookupByName(pciDevice.Spec.Name)

	var pciDeviceActive int32
	pciDeviceActive, err = hClient.NodeDeviceIsActive(pciDevice.Spec.Name)
	if err != nil {
		return
	}

	var pciDeviceXML string
	pciDeviceXML, err = hClient.NodeDeviceGetXMLDesc(pciDeviceID.Name, 0)
	if err != nil {
		return
	}
	pciDeviceInfo := &libvirtxml.NodeDevice{}
	err = pciDeviceInfo.Unmarshal(pciDeviceXML)
	if err != nil {
		return
	}

	active := pciDeviceActive == 1

	pciDevice.Status.Active = &active
	pciDevice.Status.Identifier = &v1alpha1.LibvirtIdentifier{
		Name: pciDeviceID.Name,
	}

	pciDevice.Status.Address = &v1alpha1.PCIDeviceAddress{
		Domain:   int32(*pciDeviceInfo.Capability.PCI.Domain),   // #nosec #G115
		Bus:      int32(*pciDeviceInfo.Capability.PCI.Bus),      // #nosec #G115
		Slot:     int32(*pciDeviceInfo.Capability.PCI.Slot),     // #nosec #G115
		Function: int32(*pciDeviceInfo.Capability.PCI.Function), // #nosec #G115
	}

	pciDevice.Status.LastUpdate = &metav1.Time{Time: time.Now()}

	meta.SetStatusCondition(&pciDevice.Status.Conditions, metav1.Condition{
		Type:               ConditionTypeDataRetrieved,
		Status:             metav1.ConditionTrue,
		Message:            ConditionMessagePCIDeviceDataRetrievalSucceeded,
		Reason:             ConditionReasonSucceeded,
		LastTransitionTime: metav1.Now(),
	})
	err = r.Status().Update(ctx, pciDevice)
	if err != nil {
		return
	}

	if pciDevice.Labels == nil {
		pciDevice.Labels = map[string]string{LabelKeyHost: pciDevice.Spec.HostRef.Name}
	} else {
		pciDevice.Labels[LabelKeyHost] = pciDevice.Spec.HostRef.Name
	}
	err = r.Update(ctx, pciDevice)
	if err != nil {
		return
	}

	return ctrl.Result{RequeueAfter: DataRefreshInterval}, nil
}

func (r *PCIDeviceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).For(&v1alpha1.PCIDevice{}).Complete(r)
}
