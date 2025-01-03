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

	"github.com/karelvanhecke/libvirt-operator/api/v1alpha1"
	"github.com/karelvanhecke/libvirt-operator/internal/action"
	"github.com/karelvanhecke/libvirt-operator/internal/store"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/types"
	"libvirt.org/go/libvirtxml"
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
	ConditionMessageBackingStoreNotExist     = "Backing store volume does not exist"
	ConditionMessageBackingStoreNotCreated   = "Backing store volume has not yet been created"
	ConditionMessageIsBackingStore           = "Volume is currently in use as a backingstore"
	ConditionMessageBackingStoreNotSamePool  = "Backing store volume does not exist on the same pool"
	ConditionMessageVolumeCreationInProgress = "Volume creation in progress"
	ConditionMessageVolumeCreationFailed     = "Volume creation failed"
	ConditionMessageVolumeCreationSucceeded  = "Volume creation succeeded"
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

	if volume.Status.Conditions == nil {
		meta.SetStatusCondition(&volume.Status.Conditions, metav1.Condition{
			Type:               ConditionTypeCreated,
			Status:             metav1.ConditionFalse,
			Message:            ConditionMessageVolumeCreationInProgress,
			Reason:             ConditionReasonInProgress,
			LastTransitionTime: metav1.Now(),
		})
		if err := r.Status().Update(ctx, volume); err != nil {
			return ctrl.Result{}, err
		}
	}

	poolRef := &v1alpha1.Pool{}
	if err := r.Get(ctx, types.NamespacedName{Name: volume.Spec.PoolRef.Name, Namespace: volume.Namespace}, poolRef); err != nil {
		if !meta.IsStatusConditionTrue(volume.Status.Conditions, ConditionTypeCreated) {
			meta.SetStatusCondition(&volume.Status.Conditions, metav1.Condition{
				Type:               ConditionTypeCreated,
				Status:             metav1.ConditionFalse,
				Message:            ConditionMessagePoolNotFound,
				Reason:             ConditionReasonFailed,
				LastTransitionTime: metav1.Now(),
			})
			if err := r.Status().Update(ctx, volume); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, err
	}

	if poolRef.Status.Identifier == nil || poolRef.Status.Active == nil || !*poolRef.Status.Active {
		if !meta.IsStatusConditionTrue(volume.Status.Conditions, ConditionTypeCreated) {
			meta.SetStatusCondition(&volume.Status.Conditions, metav1.Condition{
				Type:               ConditionTypeCreated,
				Status:             metav1.ConditionFalse,
				Message:            ConditionMessageWaitingForPool,
				Reason:             ConditionReasonFailed,
				LastTransitionTime: metav1.Now(),
			})
			if err := r.Status().Update(ctx, volume); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{Requeue: true}, nil
	}

	pool, err := resolvePoolIdentifier(poolRef.Status.Identifier)
	if err != nil {
		return ctrl.Result{}, err
	}

	hostRef := &v1alpha1.Host{}
	if err := r.Get(ctx, types.NamespacedName{Name: poolRef.Spec.HostRef.Name, Namespace: volume.Namespace}, hostRef); err != nil {
		if !meta.IsStatusConditionTrue(volume.Status.Conditions, ConditionTypeCreated) {
			meta.SetStatusCondition(&volume.Status.Conditions, metav1.Condition{
				Type:               ConditionTypeCreated,
				Status:             metav1.ConditionFalse,
				Message:            ConditionMessageHostNotFound,
				Reason:             ConditionReasonFailed,
				LastTransitionTime: metav1.Now(),
			})
			if err := r.Status().Update(ctx, volume); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, err
	}

	hostEntry, found := r.HostStore.Lookup(hostRef.UID)
	if !found {
		if !meta.IsStatusConditionTrue(volume.Status.Conditions, ConditionTypeCreated) {
			meta.SetStatusCondition(&volume.Status.Conditions, metav1.Condition{
				Type:               ConditionTypeCreated,
				Status:             metav1.ConditionFalse,
				Message:            ConditionMessageWaitingForHost,
				Reason:             ConditionReasonFailed,
				LastTransitionTime: metav1.Now(),
			})
			if err := r.Status().Update(ctx, volume); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{Requeue: true}, nil
	}
	hClient, end := hostEntry.Session()
	defer end()

	// #nosec #G115
	size := &libvirtxml.StorageVolumeSize{Unit: volume.Spec.Size.Unit, Value: uint64(volume.Spec.Size.Value)}

	action, err := action.NewVolumeAction(hClient, volume.Namespace+":"+volume.Name, pool, size, &libvirtxml.StorageVolumeTargetFormat{Type: volume.Spec.Format})
	if err != nil {
		return ctrl.Result{}, err
	}
	exists := action.State()

	if volume.DeletionTimestamp.IsZero() {
		if !controllerutil.ContainsFinalizer(volume, Finalizer) {
			controllerutil.AddFinalizer(volume, Finalizer)
			if err := r.Update(ctx, volume); err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		if controllerutil.ContainsFinalizer(volume, Finalizer) {
			if exists {
				if err := r.delete(ctx, volume, action); err != nil {
					if err.Error() == ErrVolumeIsBackingStore {
						ctrl.LoggerFrom(ctx).V(1).Info(ErrVolumeIsBackingStore, "volume", volume.Name)
						return ctrl.Result{Requeue: true}, nil
					}
					return ctrl.Result{}, err
				}
			}
			controllerutil.RemoveFinalizer(volume, Finalizer)
			err := r.Update(ctx, volume)
			return ctrl.Result{}, err
		}
	}

	if exists {
		err := action.Update()
		return ctrl.Result{}, err
	}

	if err := r.create(ctx, volume, action); err != nil {
		if err.Error() == ErrBackingStoreNotCreated {
			ctrl.LoggerFrom(ctx).V(1).Info(ErrBackingStoreNotCreated, "volume", volume.Name)
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
		meta.SetStatusCondition(&volume.Status.Conditions, metav1.Condition{
			Type:               ConditionTypeDeletionProbihibited,
			Status:             metav1.ConditionTrue,
			Message:            ConditionMessageIsBackingStore,
			Reason:             ConditionReasonInUse,
			LastTransitionTime: metav1.Now(),
		})
		if err := r.Status().Update(ctx, volume); err != nil {
			return err
		}
		return errors.New(ErrVolumeIsBackingStore)
	}

	return action.Delete()
}

func (r *VolumeReconciler) create(ctx context.Context, volume *v1alpha1.Volume, action *action.VolumeAction) error {
	switch {
	case volume.Spec.Source != nil:
		if err := action.WithRemoteSource(volume.Spec.Source.URL, volume.Spec.Source.Checksum); err != nil {
			return err
		}
	case volume.Spec.BackingStoreRef != nil:
		backingStore := &v1alpha1.Volume{}
		if err := r.Get(ctx, types.NamespacedName{Name: volume.Spec.BackingStoreRef.Name, Namespace: volume.Namespace}, backingStore); err != nil {
			meta.SetStatusCondition(&volume.Status.Conditions, metav1.Condition{
				Type:               ConditionTypeCreated,
				Status:             metav1.ConditionFalse,
				Message:            ConditionMessageBackingStoreNotExist,
				Reason:             ConditionReasonFailed,
				LastTransitionTime: metav1.Now(),
			})
			if err := r.Status().Update(ctx, volume); err != nil {
				return err
			}
			return err
		}

		if backingStore.Spec.PoolRef != volume.Spec.PoolRef {
			meta.SetStatusCondition(&volume.Status.Conditions, metav1.Condition{
				Type:               ConditionTypeCreated,
				Status:             metav1.ConditionFalse,
				Message:            ConditionMessageBackingStoreNotSamePool,
				Reason:             ConditionReasonFailed,
				LastTransitionTime: metav1.Now(),
			})
			if err := r.Status().Update(ctx, volume); err != nil {
				return err
			}
			return errors.New(ErrBackingStoreNotSamePool)
		}

		if !meta.IsStatusConditionTrue(backingStore.Status.Conditions, ConditionTypeCreated) {
			meta.SetStatusCondition(&volume.Status.Conditions, metav1.Condition{
				Type:               ConditionTypeCreated,
				Status:             metav1.ConditionFalse,
				Message:            ConditionMessageBackingStoreNotCreated,
				Reason:             ConditionReasonFailed,
				LastTransitionTime: metav1.Now(),
			})
			if err := r.Status().Update(ctx, volume); err != nil {
				return err
			}
			return errors.New(ErrBackingStoreNotCreated)
		}

		err := action.WithBackingStore(backingStore.Namespace + ":" + backingStore.Name)
		if err != nil {
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
		meta.SetStatusCondition(&volume.Status.Conditions, metav1.Condition{
			Type:               ConditionTypeCreated,
			Status:             metav1.ConditionFalse,
			Message:            ConditionMessageVolumeCreationFailed,
			Reason:             ConditionReasonFailed,
			LastTransitionTime: metav1.Now(),
		})
		if err := r.Status().Update(ctx, volume); err != nil {
			return err
		}
		return err
	}
	meta.SetStatusCondition(&volume.Status.Conditions, metav1.Condition{
		Type:               ConditionTypeCreated,
		Status:             metav1.ConditionTrue,
		Message:            ConditionMessageVolumeCreationSucceeded,
		Reason:             ConditionReasonSucceeded,
		LastTransitionTime: metav1.Now(),
	})
	return r.Status().Update(ctx, volume)
}
