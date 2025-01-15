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
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// Errors
const (
	ErrVolumeInUse              = "volume is in use by another resource"
	ErrBackingStoreNotCreated   = "backing store volume has not yet been created"
	ErrBackingStoreNotSameHost  = "backing store does not exist on the same host"
	ErrCannotResizeBackingStore = "can not resize volume that is used as a backing store"
	ErrVolumeMultipleDomains    = "volume linked to multiple domains"
)

type VolumeReconciler struct {
	client.Client
	HostStore *store.HostStore
}

func (r *VolumeReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	volume := &v1alpha1.Volume{}

	if err := r.Get(ctx, req.NamespacedName, volume); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	pool, host, err := r.resolveRefs(ctx, volume)
	if err != nil {
		if !meta.IsStatusConditionTrue(volume.Status.Conditions, v1alpha1.ConditionReady) {
			if err := r.setStatusCondition(ctx, volume, v1alpha1.ConditionReady, metav1.ConditionFalse, err.Error(), v1alpha1.ConditionUnmetRequirements); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, err
	}

	if !meta.IsStatusConditionTrue(pool.Status.Conditions, v1alpha1.ConditionReady) {
		if !meta.IsStatusConditionTrue(volume.Status.Conditions, v1alpha1.ConditionReady) {
			if err := r.setStatusCondition(ctx, volume, v1alpha1.ConditionReady, metav1.ConditionFalse, conditionPoolNotReady, v1alpha1.ConditionUnmetRequirements); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{Requeue: true}, nil
	}

	hostEntry, found := r.HostStore.Lookup(host.UID)
	if !found {
		if !meta.IsStatusConditionTrue(volume.Status.Conditions, v1alpha1.ConditionReady) {
			if err := r.setStatusCondition(ctx, volume, v1alpha1.ConditionReady, metav1.ConditionFalse, conditionHostClientNotReady, v1alpha1.ConditionUnmetRequirements); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{Requeue: true}, nil
	}
	hostClient, end, err := hostEntry.Session()
	if err != nil {
		if !meta.IsStatusConditionTrue(volume.Status.Conditions, v1alpha1.ConditionReady) {
			if err := r.setStatusCondition(ctx, volume, v1alpha1.ConditionReady, metav1.ConditionFalse, conditionHostClientNotReady, v1alpha1.ConditionUnmetRequirements); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{Requeue: true}, nil
	}
	defer end()

	action, err := action.NewVolumeAction(hostClient, volume.Name, pool.Spec.Name)
	if err != nil {
		return ctrl.Result{}, err
	}

	if volume.DeletionTimestamp.IsZero() {
		if controllerutil.AddFinalizer(volume, v1alpha1.Finalizer) {
			if err := r.Update(ctx, volume); err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		if controllerutil.ContainsFinalizer(volume, v1alpha1.Finalizer) {
			if action.State() {
				if err := r.delete(ctx, volume, action); err != nil {
					if msg := err.Error(); msg == ErrVolumeInUse {
						ctrl.LoggerFrom(ctx).V(1).Info(msg)
						return ctrl.Result{Requeue: true}, nil
					}
					return ctrl.Result{}, err
				}
			}
			controllerutil.RemoveFinalizer(volume, v1alpha1.Finalizer)
			err := r.Update(ctx, volume)
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	if action.State() {
		if !meta.IsStatusConditionTrue(volume.Status.Conditions, v1alpha1.ConditionReady) {
			if err := r.setStatusCondition(ctx, volume, v1alpha1.ConditionReady, metav1.ConditionTrue, conditionCreationSucceeded, v1alpha1.ConditionCreated); err != nil {
				return ctrl.Result{}, err
			}
		}
		if volume.Spec.Size == nil {
			return ctrl.Result{}, nil
		}
		u := volume.Spec.Size.Unit
		v := safecast.ToUint64(volume.Spec.Size.Value)
		if ok, err := action.ResizeRequired(u, v); err != nil {
			return ctrl.Result{}, err
		} else {
			if !ok {
				return ctrl.Result{}, nil
			}
		}

		if ok, err := r.isNotBackingStore(ctx, volume.Name); err != nil {
			return ctrl.Result{}, err
		} else {
			if !ok {
				return ctrl.Result{}, errors.New(ErrCannotResizeBackingStore)
			}
		}

		if d, ok, err := r.usedByDomain(ctx, volume.Name); err != nil {
			return ctrl.Result{}, err
		} else {
			if ok {
				return ctrl.Result{}, action.LiveResize(d.Name, u, v)
			}
		}
		return ctrl.Result{}, action.Resize(u, v)
	}

	if err := r.create(ctx, volume, pool, action); err != nil {
		if msg := err.Error(); msg == ErrBackingStoreNotCreated {
			ctrl.LoggerFrom(ctx).V(1).Info(msg)
			return ctrl.Result{Requeue: true}, nil
		}
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

func (r *VolumeReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).For(&v1alpha1.Volume{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).Complete(r)
}

func (r *VolumeReconciler) isNotBackingStore(ctx context.Context, name string) (bool, error) {
	labelReq, err := labels.NewRequirement(v1alpha1.VolumeLabel, selection.Equals, []string{name})
	if err != nil {
		return false, err
	}
	volumes := &v1alpha1.VolumeList{}
	if err := r.List(ctx, volumes, &client.ListOptions{LabelSelector: labels.NewSelector().Add(*labelReq)}); err != nil {
		return false, err
	}

	if len(volumes.Items) > 0 {
		return false, nil
	}
	return true, nil
}

func (r *VolumeReconciler) usedByDomain(ctx context.Context, name string) (domain v1alpha1.Domain, ok bool, err error) {
	labelSelector, err := labels.NewRequirement(v1alpha1.DiskLabelPrefix+"/"+name, selection.Equals, []string{""})
	if err != nil {
		return v1alpha1.Domain{}, false, err
	}
	domains := &v1alpha1.DomainList{}
	if err := r.List(ctx, domains, &client.ListOptions{LabelSelector: labels.NewSelector().Add(*labelSelector)}); err != nil {
		return v1alpha1.Domain{}, false, err
	}

	if len(domains.Items) > 1 {
		return v1alpha1.Domain{}, false, errors.New(ErrVolumeMultipleDomains)
	}

	if len(domains.Items) == 0 {
		return v1alpha1.Domain{}, false, nil
	}

	return domains.Items[0], true, nil
}

func (r *VolumeReconciler) delete(ctx context.Context, volume *v1alpha1.Volume, action *action.VolumeAction) error {
	volumes := &v1alpha1.VolumeList{}
	labelReq, err := labels.NewRequirement(v1alpha1.VolumeLabel, selection.Equals, []string{volume.Name})
	if err != nil {
		return err
	}
	if err := r.List(ctx, volumes, &client.ListOptions{LabelSelector: labels.NewSelector().Add(*labelReq)}); err != nil {
		return err
	}

	labelSelector, err := labels.NewRequirement(v1alpha1.DiskLabelPrefix+"/"+volume.Name, selection.Equals, []string{""})
	if err != nil {
		return err
	}
	domains := &v1alpha1.DomainList{}
	if err := r.List(ctx, domains, &client.ListOptions{LabelSelector: labels.NewSelector().Add(*labelSelector)}); err != nil {
		return err
	}

	if len(volumes.Items)+len(domains.Items) > 0 {
		if err := r.setStatusCondition(ctx, volume, v1alpha1.ConditionDeletionPrevented, metav1.ConditionTrue, ErrVolumeInUse, v1alpha1.ConditionInUse); err != nil {
			return err
		}
		return errors.New(ErrVolumeInUse)
	}

	return action.Delete()
}

func (r *VolumeReconciler) create(ctx context.Context, volume *v1alpha1.Volume, pool *v1alpha1.Pool, action *action.VolumeAction) error {
	if size := volume.Spec.Size; size != nil {
		action.Size(volume.Spec.Size.Unit, safecast.ToUint64(volume.Spec.Size.Value))
	}
	action.Format(volume.Spec.Format)

	hasSource := volume.Spec.Source != nil

	switch {
	case hasSource:
		if err := action.RemoteSource(ctx, volume.Spec.Source.URL, volume.Spec.Source.Checksum); err != nil {
			if err := r.setStatusCondition(ctx, volume, v1alpha1.ConditionReady, metav1.ConditionFalse, err.Error(), v1alpha1.ConditionError); err != nil {
				return err
			}
			return err
		}
	case volume.Spec.BackingStoreRef != nil:
		backingStore := &v1alpha1.Volume{}
		if err := r.Get(ctx, types.NamespacedName{Name: volume.Spec.BackingStoreRef.Name, Namespace: volume.Namespace}, backingStore); err != nil {
			if err := r.setStatusCondition(ctx, volume, v1alpha1.ConditionReady, metav1.ConditionFalse, err.Error(), v1alpha1.ConditionUnmetRequirements); err != nil {
				return err
			}
			return err
		}

		if !meta.IsStatusConditionTrue(backingStore.Status.Conditions, v1alpha1.ConditionReady) {
			if err := r.setStatusCondition(ctx, volume, v1alpha1.ConditionReady, metav1.ConditionFalse, ErrBackingStoreNotCreated, v1alpha1.ConditionUnmetRequirements); err != nil {
				return err
			}
			return errors.New(ErrBackingStoreNotCreated)
		}

		if backingStore.Status.Host != pool.Spec.HostRef.Name {
			if err := r.setStatusCondition(ctx, volume, v1alpha1.ConditionReady, metav1.ConditionFalse, ErrBackingStoreNotSameHost, v1alpha1.ConditionUnmetRequirements); err != nil {
				return err
			}
			return errors.New(ErrBackingStoreNotSameHost)
		}

		if err := action.BackingStore(backingStore.Name, backingStore.Status.Pool); err != nil {
			if err := r.setStatusCondition(ctx, volume, v1alpha1.ConditionReady, metav1.ConditionFalse, err.Error(), v1alpha1.ConditionError); err != nil {
				return err
			}
			return err
		}

		if util.SetLabel(&volume.ObjectMeta, v1alpha1.VolumeLabel, backingStore.Name) {
			if err := r.Update(ctx, volume); err != nil {
				return err
			}
		}
	}
	if util.SetLabel(&volume.ObjectMeta, v1alpha1.PoolLabel, volume.Spec.PoolRef.Name) {
		if err := r.Update(ctx, volume); err != nil {
			return err
		}
	}

	if err := action.Create(); err != nil {
		if err := r.setStatusCondition(ctx, volume, v1alpha1.ConditionReady, metav1.ConditionFalse, err.Error(), v1alpha1.ConditionError); err != nil {
			return err
		}
		return err
	}
	if hasSource {
		if err := action.CleanupSource(); err != nil {
			ctrl.LoggerFrom(ctx).Error(err, "could not cleanup volume source")
		}
	}

	volume.Status.Pool = pool.Spec.Name
	volume.Status.Host = pool.Spec.HostRef.Name

	return r.setStatusCondition(ctx, volume, v1alpha1.ConditionReady, metav1.ConditionTrue, conditionCreationSucceeded, v1alpha1.ConditionCreated)
}

func (r *VolumeReconciler) resolveRefs(ctx context.Context, volume *v1alpha1.Volume) (*v1alpha1.Pool, *v1alpha1.Host, error) {
	pool := &v1alpha1.Pool{}
	if err := r.Get(ctx, types.NamespacedName{Name: volume.Spec.PoolRef.Name, Namespace: volume.Namespace}, pool); err != nil {
		return nil, nil, err
	}

	host := &v1alpha1.Host{}
	if err := r.Get(ctx, types.NamespacedName{Name: pool.Spec.HostRef.Name, Namespace: volume.Namespace}, host); err != nil {
		return nil, nil, err
	}

	return pool, host, nil
}

func (r *VolumeReconciler) setStatusCondition(ctx context.Context, volume *v1alpha1.Volume, cType string, status metav1.ConditionStatus, msg string, reason string) error {
	c := metav1.Condition{
		Type:    cType,
		Status:  status,
		Message: msg,
		Reason:  reason,
	}

	if meta.SetStatusCondition(&volume.Status.Conditions, c) {
		return r.Status().Update(ctx, volume)
	}
	return nil
}
