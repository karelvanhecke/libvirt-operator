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
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	CondMsgNetworkDataRetrievalInProgress = "Network data retrieval in progress"
	CondMsgNetworkDataRetrievalSucceeded  = "Network data retrieval succeeded"
)

type NetworkReconciler struct {
	client.Client
	HostStore *store.HostStore
}

func (r *NetworkReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	network := &v1alpha1.Network{}

	if err := r.Get(ctx, req.NamespacedName, network); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if !meta.IsStatusConditionPresentAndEqual(network.Status.Conditions, CondTypeDataRetrieved, metav1.ConditionTrue) {
		if err := r.setStatusCondition(ctx, network, CondTypeDataRetrieved, metav1.ConditionFalse, CondMsgNetworkDataRetrievalInProgress, CondReasonInProgress); err != nil {
			return ctrl.Result{}, err
		}
	}

	if network.DeletionTimestamp.IsZero() {
		if controllerutil.AddFinalizer(network, Finalizer) {
			if err := r.Update(ctx, network); err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		if controllerutil.ContainsFinalizer(network, Finalizer) {
			// TODO
			ctrl.LoggerFrom(ctx).V(1).Info("check if network is referred by domains")
			controllerutil.RemoveFinalizer(network, Finalizer)
			err := r.Update(ctx, network)
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	host := &v1alpha1.Host{}
	if err := r.Get(ctx, types.NamespacedName{Name: network.Spec.HostRef.Name, Namespace: network.Namespace}, host); err != nil {
		if !meta.IsStatusConditionTrue(network.Status.Conditions, CondTypeCreated) {
			if err := r.setStatusCondition(ctx, network, CondTypeDataRetrieved, metav1.ConditionFalse, CondMsgHostNotFound, CondReasonFailed); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, err
	}

	hostEntry, found := r.HostStore.Lookup(host.UID)
	if !found {
		if !meta.IsStatusConditionTrue(network.Status.Conditions, CondTypeCreated) {
			if err := r.setStatusCondition(ctx, network, CondTypeDataRetrieved, metav1.ConditionFalse, CondMsgWaitingForHost, CondReasonInProgress); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{Requeue: true}, nil
	}
	hostClient, end := hostEntry.Session()
	defer end()

	if ts := network.Status.LastUpdate; ts != nil {
		if d := time.Since(ts.Time); d < DataRefreshInterval {
			return ctrl.Result{RequeueAfter: DataRefreshInterval - d}, nil
		}
	}

	networkID := libvirt.Network{}
	switch {
	case network.Spec.Name != nil:
		id, err := hostClient.NetworkLookupByName(*network.Spec.Name)
		if err != nil {
			return ctrl.Result{}, err
		}
		networkID = id
	case network.Spec.UUID != nil:
		u, err := uuid.Parse(*network.Spec.UUID)
		if err != nil {
			return ctrl.Result{}, err
		}
		id, err := hostClient.NetworkLookupByUUID(libvirt.UUID(u))
		if err != nil {
			return ctrl.Result{}, err
		}
		networkID = id
	}

	networkActive, err := hostClient.NetworkIsActive(networkID)
	if err != nil {
		return ctrl.Result{}, err
	}

	active := networkActive == 1

	u, err := uuid.FromBytes(networkID.UUID[:])
	if err != nil {
		return ctrl.Result{}, err
	}

	network.Status.Active = &active
	network.Status.Identifier = &v1alpha1.LibvirtIdentifierWithUUID{
		LibvirtIdentifier: v1alpha1.LibvirtIdentifier{
			Name: networkID.Name,
		},
		UUID: u.String(),
	}
	network.Status.LastUpdate = &metav1.Time{Time: time.Now()}

	if err := r.setStatusCondition(ctx, network, CondTypeDataRetrieved, metav1.ConditionTrue, CondMsgNetworkDataRetrievalSucceeded, CondReasonSucceeded); err != nil {
		return ctrl.Result{}, err
	}

	if network.Labels == nil {
		network.Labels = map[string]string{LabelKeyHost: network.Spec.HostRef.Name}
	} else {
		network.Labels[LabelKeyHost] = network.Spec.HostRef.Name
	}
	if err = r.Update(ctx, network); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{RequeueAfter: DataRefreshInterval}, nil
}

func (r *NetworkReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).For(&v1alpha1.Network{}).Complete(r)
}

func (r *NetworkReconciler) setStatusCondition(ctx context.Context, network *v1alpha1.Network, cType string, status metav1.ConditionStatus, msg string, reason string) error {
	meta.SetStatusCondition(&network.Status.Conditions, metav1.Condition{
		Type:               cType,
		Status:             status,
		Message:            msg,
		Reason:             reason,
		LastTransitionTime: metav1.Now(),
	})
	return r.Status().Update(ctx, network)
}
