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
	CondMsgPoolDataRetrievalInProgress = "Pool data retrieval in progress"
	CondMsgPoolDataRetrievalSucceeded  = "Pool data retrieval succeeded"
	CondMsgPoolInUseByVolume           = "Pool in use by volume"
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

	if !meta.IsStatusConditionPresentAndEqual(pool.Status.Conditions, CondTypeDataRetrieved, metav1.ConditionTrue) {
		if err := r.setStatusCondition(ctx, pool, CondTypeDataRetrieved, metav1.ConditionFalse, CondMsgPoolDataRetrievalInProgress, CondReasonInProgress); err != nil {
			return ctrl.Result{}, err
		}
	}

	if pool.DeletionTimestamp.IsZero() {
		if controllerutil.AddFinalizer(pool, Finalizer) {
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
				if err := r.setStatusCondition(ctx, pool, CondTypeDeletionProbihibited, metav1.ConditionTrue, CondMsgPoolInUseByVolume, CondReasonInUse); err != nil {
					return ctrl.Result{}, err
				}
				return ctrl.Result{Requeue: true}, nil
			}
			controllerutil.RemoveFinalizer(pool, Finalizer)
			err = r.Update(ctx, pool)
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	if pool.Status.Capacity != nil {
		if d := time.Since(pool.Status.Capacity.LastUpdate.Time); d < DataRefreshInterval {
			return ctrl.Result{RequeueAfter: DataRefreshInterval - d}, nil
		}
	}

	host := &v1alpha1.Host{}
	if err := r.Get(ctx, types.NamespacedName{Name: pool.Spec.HostRef.Name, Namespace: pool.Namespace}, host); err != nil {
		if !meta.IsStatusConditionTrue(pool.Status.Conditions, CondTypeCreated) {
			if err := r.setStatusCondition(ctx, pool, CondTypeDataRetrieved, metav1.ConditionFalse, CondMsgHostNotFound, CondReasonFailed); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, err
	}

	hostEntry, found := r.HostStore.Lookup(host.UID)
	if !found {
		if !meta.IsStatusConditionTrue(pool.Status.Conditions, CondTypeCreated) {
			if err := r.setStatusCondition(ctx, pool, CondTypeDataRetrieved, metav1.ConditionFalse, CondMsgWaitingForHost, CondReasonInProgress); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{Requeue: true}, nil
	}
	hostClient, end := hostEntry.Session()
	defer end()

	var poolID libvirt.StoragePool
	switch {
	case pool.Spec.Name != nil:
		p, err := hostClient.StoragePoolLookupByName(*pool.Spec.Name)
		if err != nil {
			return ctrl.Result{}, err
		}
		poolID = p
	case pool.Spec.UUID != nil:
		u, err := uuid.Parse(*pool.Spec.UUID)
		if err != nil {
			return ctrl.Result{}, err
		}
		p, err := hostClient.StoragePoolLookupByUUID(libvirt.UUID(u))
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

	state, cap, alloc, avail, err := hostClient.StoragePoolGetInfo(poolID)
	if err != nil {
		return ctrl.Result{}, err
	}

	poolRunning := (state == uint8(libvirt.StoragePoolRunning))
	pool.Status.Active = &poolRunning
	pool.Status.Capacity = &v1alpha1.PoolCapacity{
		Capacity:   safecast.ToInt64(cap),
		Allocation: safecast.ToInt64(alloc),
		Available:  safecast.ToInt64(avail),
		LastUpdate: metav1.Now(),
	}

	if err := r.setStatusCondition(ctx, pool, CondTypeDataRetrieved, metav1.ConditionTrue, CondMsgPoolDataRetrievalSucceeded, CondReasonSucceeded); err != nil {
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

func (r *PoolReconciler) setStatusCondition(ctx context.Context, pool *v1alpha1.Pool, cType string, status metav1.ConditionStatus, msg string, reason string) error {
	meta.SetStatusCondition(&pool.Status.Conditions, metav1.Condition{
		Type:               cType,
		Status:             status,
		Message:            msg,
		Reason:             reason,
		LastTransitionTime: metav1.Now(),
	})
	return r.Status().Update(ctx, pool)
}
