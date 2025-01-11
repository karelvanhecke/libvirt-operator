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
	"errors"

	"github.com/ARM-software/golang-utils/utils/safecast"
	"github.com/karelvanhecke/libvirt-operator/api/v1alpha1"
	"github.com/karelvanhecke/libvirt-operator/internal/action"
	"github.com/karelvanhecke/libvirt-operator/internal/store"
	"github.com/karelvanhecke/libvirt-operator/internal/util"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

const (
	ErrVolumeNotSameHost    = "all volumes must exist in pools on the same host as the domain"
	ErrNetworkNotSameHost   = "all networks must be on the same host as the domain"
	ErrPCIDeviceNotSameHost = "all pci devices must be on the same host as the domain"
	ErrVolumeNotReady       = "volume is not ready"
	ErrNetworkNotReady      = "network is not ready"
	ErrPCIDeviceNotReady    = "pci device is not ready"
)

type DomainReconciler struct {
	client.Client
	HostStore *store.HostStore
}

func (r *DomainReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	domain := &v1alpha1.Domain{}
	if err := r.Get(ctx, req.NamespacedName, domain); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	host := &v1alpha1.Host{}
	if err := r.Get(ctx, types.NamespacedName{Name: domain.Spec.HostRef.Name, Namespace: domain.Namespace}, host); err != nil {
		if !meta.IsStatusConditionTrue(domain.Status.Conditions, v1alpha1.ConditionReady) {
			if err := r.setStatusCondition(ctx, domain, v1alpha1.ConditionReady, metav1.ConditionFalse, err.Error(), v1alpha1.ConditionUnmetRequirements); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, err
	}

	hostEntry, found := r.HostStore.Lookup(host.UID)
	if !found {
		if !meta.IsStatusConditionTrue(domain.Status.Conditions, v1alpha1.ConditionReady) {
			if err := r.setStatusCondition(ctx, domain, v1alpha1.ConditionReady, metav1.ConditionFalse, conditionHostClientNotReady, v1alpha1.ConditionUnmetRequirements); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{Requeue: true}, nil
	}
	hostClient, end := hostEntry.Session()
	defer end()

	action, err := action.NewDomainAction(hostClient, util.LibvirtNamespacedName(domain.Namespace, domain.Name))
	if err != nil {
		return ctrl.Result{}, err
	}

	exists, shutoff := action.State()

	if domain.DeletionTimestamp.IsZero() {
		if controllerutil.AddFinalizer(domain, v1alpha1.Finalizer) {
			if err := r.Update(ctx, domain); err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		if controllerutil.ContainsFinalizer(domain, v1alpha1.Finalizer) {
			if exists {
				if !shutoff {
					if err := action.Shutdown(); err != nil {
						return ctrl.Result{}, err
					}
				}
				if err := action.Delete(); err != nil {
					return ctrl.Result{}, err
				}
			}
			controllerutil.RemoveFinalizer(domain, v1alpha1.Finalizer)
			err := r.Update(ctx, domain)
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	if exists {
		if !meta.IsStatusConditionTrue(domain.Status.Conditions, v1alpha1.ConditionReady) {
			if err := r.setStatusCondition(ctx, domain, v1alpha1.ConditionReady, metav1.ConditionTrue, conditionCreationSucceeded, v1alpha1.ConditionCreated); err != nil {
				return ctrl.Result{}, err
			}
		}
		if s := domain.Spec.Shutoff; s != nil && *s {
			if !shutoff {
				if err := action.Shutdown(); err != nil {
					return ctrl.Result{}, err
				}
			}
		} else {
			if shutoff {
				if err := action.Start(); err != nil {
					return ctrl.Result{}, err
				}
			}
		}
		return ctrl.Result{}, nil
	}

	action.VCPU(safecast.ToUint(domain.Spec.VCPU.Value))
	action.Memory(safecast.ToUint(domain.Spec.Memory.Value), domain.Spec.Memory.Unit)
	for _, disk := range domain.Spec.Disks {
		volume := &v1alpha1.Volume{}
		if err := r.Get(ctx, types.NamespacedName{Name: disk.VolumeRef.Name, Namespace: domain.Namespace}, volume); err != nil {
			return ctrl.Result{}, err
		}
		if !meta.IsStatusConditionTrue(volume.Status.Conditions, v1alpha1.ConditionReady) {
			if err := r.setStatusCondition(ctx, domain, v1alpha1.ConditionReady, metav1.ConditionFalse, ErrVolumeNotReady, v1alpha1.ConditionUnmetRequirements); err != nil {
				return ctrl.Result{}, err
			}
			return ctrl.Result{Requeue: true}, nil
		}
		if volume.Status.Host != domain.Spec.HostRef.Name {
			return ctrl.Result{}, errors.New(ErrVolumeNotSameHost)
		}
		if err := action.Disk(util.LibvirtNamespacedName(volume.Namespace, volume.Name), volume.Status.Pool, disk.WWN, nil); err != nil {
			return ctrl.Result{}, err
		}
		if util.SetLabel(&domain.ObjectMeta, v1alpha1.DiskLabelPrefix+"/"+volume.Name, "") {
			if err := r.Update(ctx, domain); err != nil {
				return ctrl.Result{}, err
			}
		}
	}

	for _, in := range domain.Spec.Interfaces {
		network := &v1alpha1.Network{}
		if err := r.Get(ctx, types.NamespacedName{Name: in.NetworkRef.Name, Namespace: domain.Namespace}, network); err != nil {
			return ctrl.Result{}, err
		}
		if network.Spec.HostRef.Name != domain.Spec.HostRef.Name {
			return ctrl.Result{}, errors.New(ErrNetworkNotSameHost)
		}
		if !meta.IsStatusConditionTrue(network.Status.Conditions, v1alpha1.ConditionReady) {
			if err := r.setStatusCondition(ctx, domain, v1alpha1.ConditionReady, metav1.ConditionFalse, ErrNetworkNotReady, v1alpha1.ConditionUnmetRequirements); err != nil {
				return ctrl.Result{}, err
			}
			return ctrl.Result{Requeue: true}, nil
		}
		var queues *uint
		if q := in.Queues; q != nil {
			q := safecast.ToUint(*q)
			queues = &q
		}
		action.Interface(network.Spec.Name, queues)
		if util.SetLabel(&domain.ObjectMeta, v1alpha1.InterfaceLabelPrefix+"/"+network.Name, "") {
			if err := r.Update(ctx, domain); err != nil {
				return ctrl.Result{}, err
			}
		}
	}

	for _, pci := range domain.Spec.PCIPassthrough {
		pciDevice := &v1alpha1.PCIDevice{}
		if err := r.Get(ctx, types.NamespacedName{Name: pci.PCIDeviceRef.Name, Namespace: domain.Namespace}, pciDevice); err != nil {
			return ctrl.Result{}, err
		}
		if pciDevice.Spec.HostRef.Name != domain.Spec.HostRef.Name {
			return ctrl.Result{}, errors.New(ErrPCIDeviceNotSameHost)
		}
		if !meta.IsStatusConditionTrue(pciDevice.Status.Conditions, v1alpha1.ConditionReady) {
			if err := r.setStatusCondition(ctx, domain, v1alpha1.ConditionReady, metav1.ConditionFalse, ErrPCIDeviceNotReady, v1alpha1.ConditionUnmetRequirements); err != nil {
				return ctrl.Result{}, err
			}
			return ctrl.Result{Requeue: true}, nil
		}
		action.PCIPassthrough(safecast.ToUint(pciDevice.Status.Address.Domain),
			safecast.ToUint(pciDevice.Status.Address.Bus),
			safecast.ToUint(pciDevice.Status.Address.Slot),
			safecast.ToUint(pciDevice.Status.Address.Function))
		if util.SetLabel(&domain.ObjectMeta, v1alpha1.PCIPassthroughLabelPrefix+"/"+pciDevice.Name, "") {
			if err := r.Update(ctx, domain); err != nil {
				return ctrl.Result{}, err
			}
		}
	}

	if queues := domain.Spec.SCSIQueues; queues != nil {
		queues := safecast.ToUint(*queues)
		action.SCSIMultiqueue(queues)
	}

	if numa := domain.Spec.NumatuneMode; numa != nil {
		action.NumatuneMode(*numa)
	}

	if uefi := domain.Spec.UEFI; uefi != nil && *uefi {
		action.UEFI()
	}

	if secboot := domain.Spec.SecureBoot; secboot != nil && !*secboot {
		action.NoSecureBoot()
	}

	if placement := domain.Spec.VCPU.Placement; placement != nil {
		action.VCPUPlacement(*placement)
	}

	if cpu := domain.Spec.CPU; cpu != nil {
		if mode := cpu.Mode; mode != nil {
			action.CPUMode(*mode)
		}
		if mode := cpu.CacheMode; mode != nil {
			action.CPUCacheMode(*mode)
		}
	}

	if cloudinit := domain.Spec.CloudInit; cloudinit != nil {
		ci := &v1alpha1.CloudInit{}
		if err := r.Get(ctx, types.NamespacedName{Name: cloudinit.CloudInitRef.Name, Namespace: domain.Namespace}, ci); err != nil {
			return ctrl.Result{}, err
		}
		if !meta.IsStatusConditionTrue(ci.Status.Conditions, v1alpha1.ConditionReady) {
			if err := r.setStatusCondition(ctx, domain, v1alpha1.ConditionReady, metav1.ConditionFalse, ErrVolumeNotReady, v1alpha1.ConditionUnmetRequirements); err != nil {
				return ctrl.Result{}, err
			}
			return ctrl.Result{Requeue: true}, nil
		}
		if ci.Status.Host != domain.Spec.HostRef.Name {
			return ctrl.Result{}, errors.New(ErrVolumeNotSameHost)
		}
		readonly := true
		if err := action.Disk(util.LibvirtNamespacedName(ci.Namespace, CIPrefix+ci.Name), ci.Status.Pool, nil, &readonly); err != nil {
			return ctrl.Result{}, err
		}
		if util.SetLabel(&domain.ObjectMeta, v1alpha1.CloudInitLabel, ci.Name) {
			if err := r.Update(ctx, domain); err != nil {
				return ctrl.Result{}, err
			}
		}
	}

	if err := action.Create(); err != nil {
		if err := r.setStatusCondition(ctx, domain, v1alpha1.ConditionReady, metav1.ConditionFalse, err.Error(), v1alpha1.ConditionError); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, err
	}

	if util.SetLabel(&domain.ObjectMeta, v1alpha1.HostLabel, host.Name) {
		if err := r.Update(ctx, domain); err != nil {
			return ctrl.Result{}, err
		}
	}

	start := true
	if s := domain.Spec.Shutoff; s != nil && *s {
		start = false
	}

	if start {
		if err := action.Start(); err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, r.setStatusCondition(ctx, domain, v1alpha1.ConditionReady, metav1.ConditionTrue, conditionCreationSucceeded, v1alpha1.ConditionCreated)
}

func (r *DomainReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).For(&v1alpha1.Domain{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).Complete(r)
}

func (r *DomainReconciler) setStatusCondition(ctx context.Context, domain *v1alpha1.Domain, cType string, status metav1.ConditionStatus, msg string, reason string) error {
	c := metav1.Condition{
		Type:    cType,
		Status:  status,
		Message: msg,
		Reason:  reason,
	}

	if meta.SetStatusCondition(&domain.Status.Conditions, c) {
		return r.Status().Update(ctx, domain)
	}
	return nil
}
