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
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// Labels
const (
	LabelKeyBackingStore = "libvirt.karelvanhecke.com/backingstore"
)

// Errors
const (
	ErrVolumeIsBackingStore        = "volume can not be deleted while used as backing store"
	ErrVolumeSourceAndBackingStore = "volume can not have a source and backing store at the same time"
	ErrBackingStoreNotCreated      = "backing store volume has not yet been created"
	ErrBackingStoreNotSamePool     = "backing store does not exist on the same pool"
)

// Condition messages
const (
	CondMsgBackingStoreNotExist     = "Backing store volume does not exist"
	CondMsgBackingStoreNotCreated   = "Backing store volume has not yet been created"
	CondMsgIsBackingStore           = "Volume is currently in use as a backingstore"
	CondMsgBackingStoreNotSamePool  = "Backing store volume does not exist on the same pool"
	CondMsgVolumeCreationInProgress = "Volume creation in progress"
	CondMsgVolumeCreationFailed     = "Volume creation failed"
	CondMsgVolumeCreationSucceeded  = "Volume creation succeeded"
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

	if !meta.IsStatusConditionTrue(volume.Status.Conditions, CondTypeCreated) {
		if err := r.setStatusCondition(ctx, volume, CondTypeCreated, metav1.ConditionFalse, CondMsgVolumeCreationInProgress, CondReasonInProgress); err != nil {
			return ctrl.Result{}, err
		}
	}

	pool, host, err := r.resolveRefs(ctx, volume)
	if err != nil {
		return ctrl.Result{}, err
	}

	poolID, err := resolvePoolIdentifier(pool.Status.Identifier)
	if err != nil {
		if err.Error() == ErrIDNotSet {
			if !meta.IsStatusConditionTrue(volume.Status.Conditions, CondTypeCreated) {
				if err := r.setStatusCondition(ctx, volume, CondTypeCreated, metav1.ConditionFalse, CondMsgWaitingForPool, CondReasonInProgress); err != nil {
					return ctrl.Result{}, err
				}
			}
			return ctrl.Result{Requeue: true}, nil
		}
		return ctrl.Result{}, err
	}

	hostEntry, found := r.HostStore.Lookup(host.UID)
	if !found {
		if !meta.IsStatusConditionTrue(volume.Status.Conditions, CondTypeCreated) {
			if err := r.setStatusCondition(ctx, volume, CondTypeCreated, metav1.ConditionFalse, CondMsgWaitingForHost, CondReasonInProgress); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{Requeue: true}, nil
	}
	hostClient, end := hostEntry.Session()
	defer end()

	action, err := action.NewVolumeAction(hostClient, util.LibvirtNamespacedName(volume.Namespace, volume.Name), poolID)
	if err != nil {
		return ctrl.Result{}, err
	}

	if volume.DeletionTimestamp.IsZero() {
		if controllerutil.AddFinalizer(volume, Finalizer) {
			if err := r.Update(ctx, volume); err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		if controllerutil.ContainsFinalizer(volume, Finalizer) {
			if action.State() {
				if err := r.delete(ctx, volume, action); err != nil {
					if msg := err.Error(); msg == ErrVolumeIsBackingStore {
						ctrl.LoggerFrom(ctx).V(1).Info(msg)
						return ctrl.Result{Requeue: true}, nil
					}
					return ctrl.Result{}, err
				}
			}
			controllerutil.RemoveFinalizer(volume, Finalizer)
			err := r.Update(ctx, volume)
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	if action.State() {
		if volume.Spec.Size == nil {
			return ctrl.Result{}, nil
		}
		err := action.Update(volume.Spec.Size.Unit, safecast.ToUint64(volume.Spec.Size.Value))
		return ctrl.Result{}, err
	}

	if err := r.create(ctx, volume, action); err != nil {
		if msg := err.Error(); msg == ErrBackingStoreNotCreated {
			ctrl.LoggerFrom(ctx).V(1).Info(msg)
			return ctrl.Result{Requeue: true}, nil
		}
	}
	return ctrl.Result{}, nil
}

func (r *VolumeReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).For(&v1alpha1.Volume{}).Complete(r)
}

func (r *VolumeReconciler) delete(ctx context.Context, volume *v1alpha1.Volume, action *action.VolumeAction) error {
	volumes := &v1alpha1.VolumeList{}
	labelReq, err := labels.NewRequirement(LabelKeyBackingStore, selection.Equals, []string{volume.Name})
	if err != nil {
		return err
	}
	if err := r.List(ctx, volumes, &client.ListOptions{LabelSelector: labels.NewSelector().Add(*labelReq)}); err != nil {
		return err
	}

	if len(volumes.Items) > 0 {
		if err := r.setStatusCondition(ctx, volume, CondTypeDeletionProbihibited, metav1.ConditionTrue, CondMsgIsBackingStore, CondReasonInUse); err != nil {
			return err
		}
		return errors.New(ErrVolumeIsBackingStore)
	}

	return action.Delete()
}

func (r *VolumeReconciler) create(ctx context.Context, volume *v1alpha1.Volume, action *action.VolumeAction) error {
	if size := volume.Spec.Size; size != nil {
		action.Size(volume.Spec.Size.Unit, safecast.ToUint64(volume.Spec.Size.Value))
	}
	action.Format(volume.Spec.Format)

	hasSource := volume.Spec.Source != nil

	switch {
	case hasSource:
		if err := action.RemoteSource(ctx, volume.Spec.Source.URL, volume.Spec.Source.Checksum); err != nil {
			return err
		}
	case volume.Spec.BackingStoreRef != nil:
		backingStore := &v1alpha1.Volume{}
		if err := r.Get(ctx, types.NamespacedName{Name: volume.Spec.BackingStoreRef.Name, Namespace: volume.Namespace}, backingStore); err != nil {
			if err := r.setStatusCondition(ctx, volume, CondTypeCreated, metav1.ConditionFalse, CondMsgBackingStoreNotExist, CondReasonFailed); err != nil {
				return err
			}
			return err
		}

		if backingStore.Spec.PoolRef != volume.Spec.PoolRef {
			if err := r.setStatusCondition(ctx, volume, CondTypeCreated, metav1.ConditionFalse, CondMsgBackingStoreNotSamePool, CondReasonFailed); err != nil {
				return err
			}
			return errors.New(ErrBackingStoreNotSamePool)
		}

		if !meta.IsStatusConditionTrue(backingStore.Status.Conditions, CondTypeCreated) {
			if err := r.setStatusCondition(ctx, volume, CondTypeCreated, metav1.ConditionFalse, CondMsgBackingStoreNotCreated, CondReasonInProgress); err != nil {
				return err
			}
			return errors.New(ErrBackingStoreNotCreated)
		}

		if err := action.BackingStore(backingStore.Namespace + ":" + backingStore.Name); err != nil {
			return err
		}
		if volume.Labels == nil {
			volume.Labels = map[string]string{LabelKeyBackingStore: backingStore.Name}
		} else {
			volume.Labels[LabelKeyBackingStore] = backingStore.Name
		}
		if err := r.Update(ctx, volume); err != nil {
			return err
		}
	}
	if volume.Labels == nil {
		volume.Labels = map[string]string{LabelKeyPool: volume.Spec.PoolRef.Name}
	} else {
		volume.Labels[LabelKeyPool] = volume.Spec.PoolRef.Name
	}
	if err := r.Update(ctx, volume); err != nil {
		return err
	}
	if err := action.Create(); err != nil {
		if err := r.setStatusCondition(ctx, volume, CondTypeCreated, metav1.ConditionFalse, CondMsgVolumeCreationFailed, CondReasonFailed); err != nil {
			return err
		}
		return err
	}
	if hasSource {
		if err := action.CleanupSource(); err != nil {
			ctrl.LoggerFrom(ctx).Error(err, "could not cleanup volume source")
		}
	}

	return r.setStatusCondition(ctx, volume, CondTypeCreated, metav1.ConditionTrue, CondMsgVolumeCreationSucceeded, CondReasonSucceeded)
}

func (r *VolumeReconciler) resolveRefs(ctx context.Context, volume *v1alpha1.Volume) (*v1alpha1.Pool, *v1alpha1.Host, error) {
	pool := &v1alpha1.Pool{}
	if err := r.Get(ctx, types.NamespacedName{Name: volume.Spec.PoolRef.Name, Namespace: volume.Namespace}, pool); err != nil {
		if !meta.IsStatusConditionTrue(volume.Status.Conditions, CondTypeCreated) {
			if err := r.setStatusCondition(ctx, volume, CondTypeCreated, metav1.ConditionFalse, CondMsgPoolNotFound, CondReasonFailed); err != nil {
				return nil, nil, err
			}
		}
		return nil, nil, err
	}

	host := &v1alpha1.Host{}
	if err := r.Get(ctx, types.NamespacedName{Name: pool.Spec.HostRef.Name, Namespace: volume.Namespace}, host); err != nil {
		if !meta.IsStatusConditionTrue(volume.Status.Conditions, CondTypeCreated) {
			if err := r.setStatusCondition(ctx, volume, CondTypeCreated, metav1.ConditionFalse, CondMsgHostNotFound, CondReasonFailed); err != nil {
				return nil, nil, err
			}
		}
		return nil, nil, err
	}

	return pool, host, nil
}

func (r *VolumeReconciler) setStatusCondition(ctx context.Context, volume *v1alpha1.Volume, cType string, status metav1.ConditionStatus, msg string, reason string) error {
	meta.SetStatusCondition(&volume.Status.Conditions, metav1.Condition{
		Type:               cType,
		Status:             status,
		Message:            msg,
		Reason:             reason,
		LastTransitionTime: metav1.Now(),
	})
	return r.Status().Update(ctx, volume)
}
