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
	ConditionMessageNetworkDataRetrievalInProgress = "Network data retrieval in progress"
	ConditionMessageNetworkDataRetrievalSucceeded  = "Network data retrieval succeeded"
)

type NetworkReconciler struct {
	client.Client
	HostStore *store.HostStore
}

func (r *NetworkReconciler) Reconcile(ctx context.Context, req ctrl.Request) (res ctrl.Result, err error) {
	network := &v1alpha1.Network{}

	err = r.Get(ctx, req.NamespacedName, network)
	if err != nil {
		return
	}

	if !meta.IsStatusConditionPresentAndEqual(network.Status.Conditions, ConditionTypeDataRetrieved, metav1.ConditionTrue) {
		meta.SetStatusCondition(&network.Status.Conditions, metav1.Condition{
			Type:               ConditionTypeDataRetrieved,
			Status:             metav1.ConditionFalse,
			Message:            ConditionMessageNetworkDataRetrievalInProgress,
			Reason:             ConditionReasonInProgress,
			LastTransitionTime: metav1.Now(),
		})
		err = r.Status().Update(ctx, network)
		if err != nil {
			return
		}
	}

	if network.DeletionTimestamp.IsZero() {
		if controllerutil.AddFinalizer(network, Finalizer) {
			err = r.Update(ctx, network)
			if err != nil {
				return
			}
		}
	} else {
		if controllerutil.ContainsFinalizer(network, Finalizer) {
			// TODO
			ctrl.LoggerFrom(ctx).V(1).Info("check if network is referred by domains")
		}
		controllerutil.RemoveFinalizer(network, Finalizer)
		err = r.Update(ctx, network)
		if err != nil {
			return
		}
	}

	hostRef := &v1alpha1.Host{}
	if err := r.Get(ctx, types.NamespacedName{Name: network.Spec.HostRef.Name, Namespace: network.Namespace}, hostRef); err != nil {
		if !meta.IsStatusConditionTrue(network.Status.Conditions, ConditionTypeCreated) {
			meta.SetStatusCondition(&network.Status.Conditions, metav1.Condition{
				Type:               ConditionTypeDataRetrieved,
				Status:             metav1.ConditionFalse,
				Message:            ConditionMessageHostNotFound,
				Reason:             ConditionReasonFailed,
				LastTransitionTime: metav1.Now(),
			})
			if err := r.Status().Update(ctx, network); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, err
	}

	hostEntry, found := r.HostStore.Lookup(hostRef.UID)
	if !found {
		if !meta.IsStatusConditionTrue(network.Status.Conditions, ConditionTypeCreated) {
			meta.SetStatusCondition(&network.Status.Conditions, metav1.Condition{
				Type:               ConditionTypeDataRetrieved,
				Status:             metav1.ConditionFalse,
				Message:            ConditionMessageWaitingForHost,
				Reason:             ConditionReasonFailed,
				LastTransitionTime: metav1.Now(),
			})
			if err := r.Status().Update(ctx, network); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{Requeue: true}, nil
	}
	hClient, end := hostEntry.Session()
	defer end()

	if ts := network.Status.LastUpdate; ts != nil {
		if d := time.Since(ts.Time); d < DataRefreshInterval {
			return ctrl.Result{RequeueAfter: DataRefreshInterval - d}, nil
		}
	}

	networkID := libvirt.Network{}
	switch {
	case network.Spec.Name != nil:
		networkID, err = hClient.NetworkLookupByName(*network.Spec.Name)
		if err != nil {
			return
		}
	case network.Spec.UUID != nil:
		var u uuid.UUID
		u, err = uuid.Parse(*network.Spec.UUID)
		if err != nil {
			return
		}
		networkID, err = hClient.NetworkLookupByUUID(libvirt.UUID(u))
		if err != nil {
			return
		}
	}

	var networkActive int32
	networkActive, err = hClient.NetworkIsActive(networkID)
	if err != nil {
		return
	}

	active := networkActive == 1

	var u uuid.UUID
	u, err = uuid.FromBytes(networkID.UUID[:])
	if err != nil {
		return
	}

	network.Status.Active = &active
	network.Status.Identifier = &v1alpha1.LibvirtIdentifierWithUUID{
		LibvirtIdentifier: v1alpha1.LibvirtIdentifier{
			Name: networkID.Name,
		},
		UUID: u.String(),
	}
	network.Status.LastUpdate = &metav1.Time{Time: time.Now()}

	meta.SetStatusCondition(&network.Status.Conditions, metav1.Condition{
		Type:               ConditionTypeDataRetrieved,
		Status:             metav1.ConditionTrue,
		Message:            ConditionMessageNetworkDataRetrievalSucceeded,
		Reason:             ConditionReasonSucceeded,
		LastTransitionTime: metav1.Now(),
	})
	err = r.Status().Update(ctx, network)
	if err != nil {
		return
	}

	if network.Labels == nil {
		network.Labels = map[string]string{LabelKeyHost: network.Spec.HostRef.Name}
	} else {
		network.Labels[LabelKeyHost] = network.Spec.HostRef.Name
	}
	err = r.Update(ctx, network)
	if err != nil {
		return
	}

	return ctrl.Result{RequeueAfter: DataRefreshInterval}, nil
}

func (r *NetworkReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).For(&v1alpha1.Network{}).Complete(r)
}
