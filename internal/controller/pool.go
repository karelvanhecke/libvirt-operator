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
	"github.com/google/uuid"
	"github.com/karelvanhecke/libvirt-operator/api/v1alpha1"
	"github.com/karelvanhecke/libvirt-operator/internal/store"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	ConditionMessagePoolDataRetrievalInProgress = "Pool data retrieval in progress"
	ConditionMessagePoolDataRetrievalSucceeded  = "Pool data retrieval succeeded"
	ConditionMessagePoolInUseByVolume           = "Pool in use by volume"
)

type PoolReconciler struct {
	client.Client
	HostStore *store.HostStore
}

func (r *PoolReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	pool := &v1alpha1.Pool{}

	if err := r.Get(ctx, req.NamespacedName, pool); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if pool.Status.Capacity == nil {
		meta.SetStatusCondition(&pool.Status.Conditions, metav1.Condition{
			Type:               ConditionTypeDataRetrieved,
			Status:             metav1.ConditionFalse,
			Message:            ConditionMessagePoolDataRetrievalInProgress,
			Reason:             ConditionReasonInProgress,
			LastTransitionTime: metav1.Now(),
		})
		if err := r.Status().Update(ctx, pool); err != nil {
			return ctrl.Result{}, err
		}
	}

	if pool.DeletionTimestamp.IsZero() {
		if !controllerutil.ContainsFinalizer(pool, Finalizer) {
			controllerutil.AddFinalizer(pool, Finalizer)
			if err := r.Update(ctx, pool); err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		if controllerutil.ContainsFinalizer(pool, Finalizer) {
			volumes := &v1alpha1.VolumeList{}
			labelReq, err := labels.NewRequirement(LabelKeyPool, selection.Equals, []string{pool.Name})
			if err != nil {
				return ctrl.Result{}, err
			}
			if err := r.List(ctx, volumes, &client.ListOptions{LabelSelector: labels.NewSelector().Add(*labelReq)}); err != nil {
				return ctrl.Result{}, err
			}

			if len(volumes.Items) > 0 {
				meta.SetStatusCondition(&pool.Status.Conditions, metav1.Condition{
					Type:               ConditionTypeDeletionProbihibited,
					Status:             metav1.ConditionTrue,
					Message:            ConditionMessagePoolInUseByVolume,
					Reason:             ConditionReasonInUse,
					LastTransitionTime: metav1.Now(),
				})
				if err := r.Status().Update(ctx, pool); err != nil {
					return ctrl.Result{}, err
				}
				return ctrl.Result{Requeue: true}, nil
			}
			controllerutil.RemoveFinalizer(pool, Finalizer)
			err = r.Update(ctx, pool)
			return ctrl.Result{}, err

		}
	}

	if pool.Status.Capacity != nil {
		if d := time.Since(pool.Status.Capacity.LastUpdate.Time); d < DataRefreshInterval {
			return ctrl.Result{RequeueAfter: DataRefreshInterval - d}, nil
		}
	}

	hostRef := &v1alpha1.Host{}
	if err := r.Get(ctx, types.NamespacedName{Name: pool.Spec.HostRef.Name, Namespace: pool.Namespace}, hostRef); err != nil {
		if !meta.IsStatusConditionTrue(pool.Status.Conditions, ConditionTypeCreated) {
			meta.SetStatusCondition(&pool.Status.Conditions, metav1.Condition{
				Type:               ConditionTypeDataRetrieved,
				Status:             metav1.ConditionFalse,
				Message:            ConditionMessageHostNotFound,
				Reason:             ConditionReasonFailed,
				LastTransitionTime: metav1.Now(),
			})
			if err := r.Status().Update(ctx, pool); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, err
	}

	hostEntry, found := r.HostStore.Lookup(hostRef.UID)
	if !found {
		if !meta.IsStatusConditionTrue(pool.Status.Conditions, ConditionTypeCreated) {
			meta.SetStatusCondition(&pool.Status.Conditions, metav1.Condition{
				Type:               ConditionTypeDataRetrieved,
				Status:             metav1.ConditionFalse,
				Message:            ConditionMessageWaitingForHost,
				Reason:             ConditionReasonFailed,
				LastTransitionTime: metav1.Now(),
			})
			if err := r.Status().Update(ctx, pool); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{Requeue: true}, nil
	}
	hClient, end := hostEntry.Session()
	defer end()

	var poolID libvirt.StoragePool
	switch {
	case pool.Spec.Name != nil:
		p, err := hClient.StoragePoolLookupByName(*pool.Spec.Name)
		if err != nil {
			return ctrl.Result{}, err
		}
		poolID = p
	case pool.Spec.UUID != nil:
		u, err := uuid.Parse(*pool.Spec.UUID)
		if err != nil {
			return ctrl.Result{}, err
		}
		p, err := hClient.StoragePoolLookupByUUID(libvirt.UUID(u))
		if err != nil {
			return ctrl.Result{}, err
		}
		poolID = p
	}

	u, err := uuid.FromBytes(poolID.UUID[:])
	if err != nil {
		return ctrl.Result{}, err
	}
	pool.Status.Identifier = &v1alpha1.LibvirtIdentifierWithUUID{
		LibvirtIdentifier: v1alpha1.LibvirtIdentifier{Name: poolID.Name},
		UUID:              u.String(),
	}

	state, cap, alloc, avail, err := hClient.StoragePoolGetInfo(poolID)
	if err != nil {
		return ctrl.Result{}, err
	}

	poolRunning := (state == uint8(libvirt.StoragePoolRunning))
	pool.Status.Active = &poolRunning
	pool.Status.Capacity = &v1alpha1.PoolCapacity{
		Capacity:   int64(cap),   // #nosec #G115
		Allocation: int64(alloc), // #nosec #G115
		Available:  int64(avail), // #nosec #G115
		LastUpdate: metav1.Now(),
	}

	meta.SetStatusCondition(&pool.Status.Conditions, metav1.Condition{
		Type:               ConditionTypeDataRetrieved,
		Status:             metav1.ConditionTrue,
		Message:            ConditionMessagePoolDataRetrievalSucceeded,
		Reason:             ConditionReasonSucceeded,
		LastTransitionTime: metav1.Now(),
	})
	if err := r.Status().Update(ctx, pool); err != nil {
		return ctrl.Result{}, err
	}

	if pool.Labels == nil {
		pool.Labels = map[string]string{LabelKeyHost: pool.Spec.HostRef.Name}
	} else {
		pool.Labels[LabelKeyHost] = pool.Spec.HostRef.Name
	}
	if err := r.Update(ctx, pool); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{RequeueAfter: DataRefreshInterval}, nil
}

func (r *PoolReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).For(&v1alpha1.Pool{}).Complete(r)
}
