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

	"github.com/karelvanhecke/libvirt-operator/api/v1alpha1"
	"github.com/karelvanhecke/libvirt-operator/internal/action"
	"github.com/karelvanhecke/libvirt-operator/internal/store"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
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

	hostRef := &v1alpha1.Host{}
	if err := r.Get(ctx, types.NamespacedName{Name: domain.Spec.HostRef.Name, Namespace: domain.Namespace}, hostRef); err != nil {
		if !meta.IsStatusConditionTrue(domain.Status.Conditions, ConditionTypeCreated) {
			meta.SetStatusCondition(&domain.Status.Conditions, metav1.Condition{
				Type:               ConditionTypeCreated,
				Status:             metav1.ConditionFalse,
				Message:            ConditionMessageHostNotFound,
				Reason:             ConditionReasonFailed,
				LastTransitionTime: metav1.Time{Time: time.Now()},
			})
			if err := r.Status().Update(ctx, domain); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, err
	}

	hostEntry, found := r.HostStore.Lookup(hostRef.UID)
	if !found {
		if !meta.IsStatusConditionTrue(domain.Status.Conditions, ConditionTypeCreated) {
			meta.SetStatusCondition(&domain.Status.Conditions, metav1.Condition{
				Type:               ConditionTypeCreated,
				Status:             metav1.ConditionFalse,
				Message:            ConditionMessageWaitingForHost,
				Reason:             ConditionReasonFailed,
				LastTransitionTime: metav1.Time{Time: time.Now()},
			})
			if err := r.Status().Update(ctx, domain); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{Requeue: true}, nil
	}
	hClient, end := hostEntry.Session()
	defer end()

	_, err := action.NewDomainAction(hClient, domain.Name)
	if err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *DomainReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).For(&v1alpha1.Domain{}).Complete(r)
}
