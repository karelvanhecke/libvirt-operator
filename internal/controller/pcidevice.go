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

	"github.com/ARM-software/golang-utils/utils/safecast"
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
	CondMsgPCIDeviceDataRetrievalInProgress = "PCIDevice data retrieval in progress"
	CondMsgPCIDeviceDataRetrievalSucceeded  = "PCIDevice data retrieval succeeded"
)

type PCIDeviceReconciler struct {
	client.Client
	HostStore *store.HostStore
}

func (r *PCIDeviceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	pciDevice := &v1alpha1.PCIDevice{}

	err := r.Get(ctx, req.NamespacedName, pciDevice)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if !meta.IsStatusConditionPresentAndEqual(pciDevice.Status.Conditions, CondTypeDataRetrieved, metav1.ConditionTrue) {
		if err := r.setStatusCondition(ctx, pciDevice, CondTypeDataRetrieved, metav1.ConditionFalse, CondMsgPCIDeviceDataRetrievalInProgress, CondReasonInProgress); err != nil {
			return ctrl.Result{}, err
		}
	}

	if pciDevice.DeletionTimestamp.IsZero() {
		if controllerutil.AddFinalizer(pciDevice, Finalizer) {
			if err := r.Update(ctx, pciDevice); err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		if controllerutil.ContainsFinalizer(pciDevice, Finalizer) {
			// TODO
			ctrl.LoggerFrom(ctx).V(1).Info("check if pciDevice is referred by domains")
			controllerutil.RemoveFinalizer(pciDevice, Finalizer)
			err := r.Update(ctx, pciDevice)
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	host := &v1alpha1.Host{}
	if err := r.Get(ctx, types.NamespacedName{Name: pciDevice.Spec.HostRef.Name, Namespace: pciDevice.Namespace}, host); err != nil {
		if !meta.IsStatusConditionTrue(pciDevice.Status.Conditions, CondTypeCreated) {
			if err := r.setStatusCondition(ctx, pciDevice, CondTypeDataRetrieved, metav1.ConditionFalse, CondMsgHostNotFound, CondReasonFailed); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, err
	}

	hostEntry, found := r.HostStore.Lookup(host.UID)
	if !found {
		if !meta.IsStatusConditionTrue(pciDevice.Status.Conditions, CondTypeCreated) {
			if err := r.setStatusCondition(ctx, pciDevice, CondTypeDataRetrieved, metav1.ConditionFalse, CondMsgWaitingForHost, CondReasonInProgress); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{Requeue: true}, nil
	}
	hostClient, end := hostEntry.Session()
	defer end()

	if ts := pciDevice.Status.LastUpdate; ts != nil {
		if d := time.Since(ts.Time); d < DataRefreshInterval {
			return ctrl.Result{RequeueAfter: DataRefreshInterval - d}, nil
		}
	}

	pciDeviceID, err := hostClient.NodeDeviceLookupByName(pciDevice.Spec.Name)
	if err != nil {
		return ctrl.Result{}, err
	}

	pciDeviceActive, err := hostClient.NodeDeviceIsActive(pciDevice.Spec.Name)
	if err != nil {
		return ctrl.Result{}, err
	}

	pciDeviceXML, err := hostClient.NodeDeviceGetXMLDesc(pciDeviceID.Name, 0)
	if err != nil {
		return ctrl.Result{}, err
	}
	pciDeviceInfo := &libvirtxml.NodeDevice{}
	if err := pciDeviceInfo.Unmarshal(pciDeviceXML); err != nil {
		return ctrl.Result{}, err
	}

	active := pciDeviceActive == 1

	pciDevice.Status.Active = &active
	pciDevice.Status.Identifier = &v1alpha1.LibvirtIdentifier{
		Name: pciDeviceID.Name,
	}

	pciDevice.Status.Address = &v1alpha1.PCIDeviceAddress{
		Domain:   safecast.ToInt32(*pciDeviceInfo.Capability.PCI.Domain),
		Bus:      safecast.ToInt32(*pciDeviceInfo.Capability.PCI.Bus),
		Slot:     safecast.ToInt32(*pciDeviceInfo.Capability.PCI.Slot),
		Function: safecast.ToInt32(*pciDeviceInfo.Capability.PCI.Function),
	}

	pciDevice.Status.LastUpdate = &metav1.Time{Time: time.Now()}

	if err := r.setStatusCondition(ctx, pciDevice, CondTypeDataRetrieved, metav1.ConditionTrue, CondMsgPCIDeviceDataRetrievalSucceeded, CondReasonSucceeded); err != nil {
		return ctrl.Result{}, err
	}

	if pciDevice.Labels == nil {
		pciDevice.Labels = map[string]string{LabelKeyHost: pciDevice.Spec.HostRef.Name}
	} else {
		pciDevice.Labels[LabelKeyHost] = pciDevice.Spec.HostRef.Name
	}
	if err := r.Update(ctx, pciDevice); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{RequeueAfter: DataRefreshInterval}, nil
}

func (r *PCIDeviceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).For(&v1alpha1.PCIDevice{}).Complete(r)
}

func (r *PCIDeviceReconciler) setStatusCondition(ctx context.Context, pciDevice *v1alpha1.PCIDevice, cType string, status metav1.ConditionStatus, msg string, reason string) error {
	meta.SetStatusCondition(&pciDevice.Status.Conditions, metav1.Condition{
		Type:               cType,
		Status:             status,
		Message:            msg,
		Reason:             reason,
		LastTransitionTime: metav1.Now(),
	})
	return r.Status().Update(ctx, pciDevice)
}
