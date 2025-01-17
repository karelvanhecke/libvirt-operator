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
	"github.com/karelvanhecke/libvirt-operator/internal/probe"
	"github.com/karelvanhecke/libvirt-operator/internal/store"
	"github.com/karelvanhecke/libvirt-operator/internal/util"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
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

	if pciDevice.DeletionTimestamp.IsZero() {
		if controllerutil.AddFinalizer(pciDevice, v1alpha1.Finalizer) {
			if err := r.Update(ctx, pciDevice); err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		if controllerutil.ContainsFinalizer(pciDevice, v1alpha1.Finalizer) {
			labelSelector, err := labels.NewRequirement(v1alpha1.PCIPassthroughLabelPrefix+"/"+pciDevice.Name, selection.Equals, []string{""})
			if err != nil {
				return ctrl.Result{}, err
			}
			domains := &v1alpha1.DomainList{}
			if err := r.List(ctx, domains, &client.ListOptions{LabelSelector: labels.NewSelector().Add(*labelSelector)}); err != nil {
				return ctrl.Result{}, err
			}
			if len(domains.Items) > 0 {
				if err := r.setStatusCondition(ctx, pciDevice, v1alpha1.ConditionDeletionPrevented, metav1.ConditionTrue, "PCI device is currently in use by domain", v1alpha1.ConditionInUse); err != nil {
					return ctrl.Result{}, err
				}
				return ctrl.Result{Requeue: true}, nil
			}
			controllerutil.RemoveFinalizer(pciDevice, v1alpha1.Finalizer)
			err = r.Update(ctx, pciDevice)
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	if probed := meta.FindStatusCondition(pciDevice.Status.Conditions, v1alpha1.ConditionProbed); probed != nil && probed.Status == metav1.ConditionTrue {
		if d := time.Since(probed.LastTransitionTime.Time); d < dataRefreshInterval {
			return ctrl.Result{RequeueAfter: dataRefreshInterval - d}, nil
		}
	}

	if err := r.setStatusCondition(ctx, pciDevice, v1alpha1.ConditionProbed, metav1.ConditionFalse, "New probe required", v1alpha1.ConditionRequired); err != nil {
		return ctrl.Result{}, err
	}

	host := &v1alpha1.Host{}
	if err := r.Get(ctx, pciDevice.HostRef(), host); err != nil {
		if err := r.setStatusCondition(ctx, pciDevice, v1alpha1.ConditionProbed, metav1.ConditionFalse, err.Error(), v1alpha1.ConditionUnmetRequirements); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, err
	}

	hostEntry, found := r.HostStore.Lookup(host.UID)
	if !found {
		if err := r.setStatusCondition(ctx, pciDevice, v1alpha1.ConditionProbed, metav1.ConditionFalse, conditionHostClientNotReady, v1alpha1.ConditionUnmetRequirements); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}
	hostClient, end, err := hostEntry.Session()
	if err != nil {
		if err := r.setStatusCondition(ctx, pciDevice, v1alpha1.ConditionProbed, metav1.ConditionFalse, conditionHostClientNotReady, v1alpha1.ConditionUnmetRequirements); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}
	defer end()

	probe, err := probe.NewPCIDeviceProbe(hostClient, pciDevice.ResourceName())
	if err != nil {
		if err := r.setStatusCondition(ctx, pciDevice, v1alpha1.ConditionProbed, metav1.ConditionFalse, "Probe could not be completed: "+err.Error(), v1alpha1.ConditionError); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, err
	}

	if !probe.Exists() {
		if err := r.setStatusCondition(ctx, pciDevice, v1alpha1.ConditionReady, metav1.ConditionFalse, "PCI device does not exist", v1alpha1.ConditionNotExist); err != nil {
			return ctrl.Result{}, err
		}
	} else {
		if probe.Active() {
			if err := r.setStatusCondition(ctx, pciDevice, v1alpha1.ConditionReady, metav1.ConditionTrue, "PCI device is active", v1alpha1.ConditionActive); err != nil {
				return ctrl.Result{}, err
			}
		} else {
			if err := r.setStatusCondition(ctx, pciDevice, v1alpha1.ConditionReady, metav1.ConditionFalse, "PCI device is not active", v1alpha1.ConditionNotActive); err != nil {
				return ctrl.Result{}, err
			}
		}
	}

	pciDevice.Status.Address = &v1alpha1.PCIDeviceAddress{
		Domain:   safecast.ToInt32(probe.Domain()),
		Bus:      safecast.ToInt32(probe.Bus()),
		Slot:     safecast.ToInt32(probe.Slot()),
		Function: safecast.ToInt32(probe.Function()),
	}

	if err := r.setStatusCondition(ctx, pciDevice, v1alpha1.ConditionProbed, metav1.ConditionTrue, conditionProbeCompleted, v1alpha1.ConditionCompleted); err != nil {
		return ctrl.Result{}, err
	}

	if util.SetLabel(&pciDevice.ObjectMeta, v1alpha1.HostLabel, pciDevice.Spec.HostRef.Name) {
		if err := r.Update(ctx, pciDevice); err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{RequeueAfter: dataRefreshInterval}, nil
}

func (r *PCIDeviceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).For(&v1alpha1.PCIDevice{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).Complete(r)
}

func (r *PCIDeviceReconciler) setStatusCondition(ctx context.Context, pciDevice *v1alpha1.PCIDevice, cType string, status metav1.ConditionStatus, msg string, reason string) error {
	c := metav1.Condition{
		Type:    cType,
		Status:  status,
		Message: msg,
		Reason:  reason,
	}
	if meta.SetStatusCondition(&pciDevice.Status.Conditions, c) {
		return r.Status().Update(ctx, pciDevice)
	}
	return nil
}
