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
	"fmt"
	"path/filepath"
	"time"

	"github.com/ARM-software/golang-utils/utils/safecast"
	"github.com/digitalocean/go-libvirt"
	"github.com/digitalocean/go-libvirt/socket"
	"github.com/digitalocean/go-libvirt/socket/dialers"
	"github.com/karelvanhecke/libvirt-operator/api/v1alpha1"
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
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	CondMsgHostDataRetrievalInProgress = "host data retrieval in progress"
	CondMsgHostInUseByResource         = "host is in use by another resource"
	CondMsgHostDataRetrievalSucceeded  = "Host data retrieval succeeded"
)

type HostReconciler struct {
	client.Client
	AuthStore *store.AuthStore
	HostStore *store.HostStore
}

func (r *HostReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	host := &v1alpha1.Host{}
	if err := r.Get(ctx, req.NamespacedName, host); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if !meta.IsStatusConditionPresentAndEqual(host.Status.Conditions, CondTypeDataRetrieved, metav1.ConditionTrue) {
		if err := r.setStatusCondition(ctx, host, CondTypeDataRetrieved, metav1.ConditionFalse, CondMsgHostDataRetrievalInProgress, CondReasonInProgress); err != nil {
			return ctrl.Result{}, err
		}
	}

	if host.DeletionTimestamp.IsZero() {
		if controllerutil.AddFinalizer(host, Finalizer) {
			if err := r.Update(ctx, host); err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		if controllerutil.ContainsFinalizer(host, Finalizer) {
			labelSelector, err := labels.NewRequirement(LabelKeyHost, selection.Equals, []string{host.Name})
			if err != nil {
				return ctrl.Result{}, err
			}
			pools := &v1alpha1.PoolList{}
			if err := r.List(ctx, pools, &client.ListOptions{LabelSelector: labels.NewSelector().Add(*labelSelector)}); err != nil {
				return ctrl.Result{}, err
			}
			networks := &v1alpha1.NetworkList{}
			if err := r.List(ctx, networks, &client.ListOptions{LabelSelector: labels.NewSelector().Add(*labelSelector)}); err != nil {
				return ctrl.Result{}, err
			}
			pciDevices := &v1alpha1.PCIDeviceList{}
			if err := r.List(ctx, pciDevices, &client.ListOptions{LabelSelector: labels.NewSelector().Add(*labelSelector)}); err != nil {
				return ctrl.Result{}, err
			}

			if len(pools.Items)+len(networks.Items)+len(pciDevices.Items) > 0 {
				if err := r.setStatusCondition(ctx, host, CondTypeDeletionProbihibited, metav1.ConditionTrue, CondMsgHostInUseByResource, CondReasonInUse); err != nil {
					return ctrl.Result{}, err
				}
				return ctrl.Result{Requeue: true}, nil
			}

			r.HostStore.Deregister(ctx, host.UID)

			controllerutil.RemoveFinalizer(host, Finalizer)
			err = r.Update(ctx, host)
			return ctrl.Result{}, err

		}
		return ctrl.Result{}, nil
	}

	auth := &v1alpha1.Auth{}

	if err := r.Get(ctx, types.NamespacedName{Name: host.Spec.AuthRef.Name, Namespace: host.Namespace}, auth); err != nil {
		return ctrl.Result{}, err
	}

	authEntry, found := r.AuthStore.Lookup(auth.UID)
	if !found {
		ctrl.LoggerFrom(ctx).V(1).Info("auth not yet available", "auth", auth.Name)
		return ctrl.Result{Requeue: true}, nil
	}

	combinedGeneration := host.Generation + authEntry.Generation()

	hostEntry, found := r.HostStore.Lookup(host.UID)
	if found {
		if combinedGeneration == hostEntry.Generation() {
			if host.Status.Capacity != nil {
				if d := time.Since(host.Status.Capacity.LastUpdate.Time); d < DataRefreshInterval {
					return ctrl.Result{RequeueAfter: DataRefreshInterval - d}, nil
				}
			}
			if err := r.updateHostCapacity(ctx, host, hostEntry); err != nil {
				return ctrl.Result{}, err
			}
			return ctrl.Result{RequeueAfter: DataRefreshInterval}, nil
		}
	}

	var dialer socket.Dialer

	switch auth.Spec.Type {
	case v1alpha1.SSHAuth:
		opts := []dialers.SSHOption{dialers.WithSSHAuthMethods((&dialers.SSHAuthMethods{}).PrivKey()), dialers.UseKeyFile(filepath.Join(authEntry.GetPath(), PrivateKey)), dialers.UseSSHUsername(*auth.Spec.Username)}

		if auth.Spec.Verify != nil && !*auth.Spec.Verify {
			opts = append(opts, dialers.WithAcceptUnknownHostKey())
		} else {
			opts = append(opts, dialers.UseKnownHostsFile(filepath.Join(authEntry.GetPath(), KnownHosts)))
		}

		if port := host.Spec.Port; port != nil {
			opts = append(opts, dialers.UseSSHPort(string(*port)))
		}

		dialer = dialers.NewSSH(host.Spec.Address, opts...)
	case v1alpha1.TLSAuth:
		opts := []dialers.TLSOption{dialers.UsePKIPath(authEntry.GetPath())}

		if auth.Spec.Verify != nil && !*auth.Spec.Verify {
			opts = append(opts, dialers.WithInsecureNoVerify())
		}

		if port := host.Spec.Port; port != nil {
			opts = append(opts, dialers.UseTLSPort(string(*port)))
		}

		dialer = dialers.NewTLS(host.Spec.Address, opts...)
	default:
		return ctrl.Result{}, fmt.Errorf("unsupported auth type: %s", auth.Spec.Type)
	}

	r.HostStore.Register(ctx, host.UID, combinedGeneration, dialer)
	hostEntry, _ = r.HostStore.Lookup(host.UID)

	if host.Labels == nil {
		host.Labels = make(map[string]string)
	}
	if host.Labels[LabelKeyAuth] != auth.Name {
		host.Labels[LabelKeyAuth] = auth.Name
		if err := r.Update(ctx, host); err != nil {
			return ctrl.Result{}, err
		}
	}

	if err := r.updateHostCapacity(ctx, host, hostEntry); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{RequeueAfter: DataRefreshInterval}, nil
}

func (r *HostReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).For(&v1alpha1.Host{}).Watches(&v1alpha1.Auth{},
		handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, o client.Object) []reconcile.Request {
			if !o.GetDeletionTimestamp().IsZero() {
				return nil
			}
			labelSelector, err := labels.NewRequirement(LabelKeyAuth, selection.Equals, []string{o.GetName()})
			if err != nil {
				return []reconcile.Request{}
			}
			hostList := &v1alpha1.HostList{}
			if err := r.List(ctx, hostList, &client.ListOptions{LabelSelector: labels.NewSelector().Add(*labelSelector)}); err != nil {
				ctrl.LoggerFrom(ctx).Error(err, "could not get list of Host objects")
				return []reconcile.Request{}
			}

			queued := []reconcile.Request{}
			for _, host := range hostList.Items {
				queued = append(queued, reconcile.Request{NamespacedName: types.NamespacedName{
					Name:      host.Name,
					Namespace: host.Namespace,
				}})
			}
			return queued
		})).Complete(r)
}

func (r *HostReconciler) updateHostCapacity(ctx context.Context, host *v1alpha1.Host, hostEntry *store.HostEntry) error {
	hClient, end := hostEntry.Session()
	defer end()

	_, _, cpu, _, _, _, _, _, err := hClient.NodeGetInfo()
	if err != nil {
		return err
	}

	memory := v1alpha1.HostMemory{}

	memStats, _, err := hClient.NodeGetMemoryStats(4, int32(libvirt.NodeMemoryStatsAllCells), 0)
	if err != nil {
		return err
	}

	for _, memStat := range memStats {
		switch memStat.Field {
		case libvirt.NodeMemoryStatsTotal:
			memory.Total = safecast.ToInt64(util.ConvertToBytes(memStat.Value, "KB"))
		case libvirt.NodeMemoryStatsFree:
			memory.Free = safecast.ToInt64(util.ConvertToBytes(memStat.Value, "KB"))
		case libvirt.NodeMemoryStatsBuffers:
			memory.Free += safecast.ToInt64(util.ConvertToBytes(memStat.Value, "KB"))
		case libvirt.NodeMemoryStatsCached:
			memory.Free += safecast.ToInt64(util.ConvertToBytes(memStat.Value, "KB"))
		}
	}

	cap := &v1alpha1.HostCapacity{
		CPU:        cpu,
		Memory:     memory,
		LastUpdate: metav1.Now(),
	}
	host.Status.Capacity = cap

	return r.setStatusCondition(ctx, host, CondTypeDataRetrieved, metav1.ConditionTrue, CondMsgHostDataRetrievalSucceeded, CondReasonSucceeded)
}

func (r *HostReconciler) setStatusCondition(ctx context.Context, host *v1alpha1.Host, cType string, status metav1.ConditionStatus, msg string, reason string) error {
	meta.SetStatusCondition(&host.Status.Conditions, metav1.Condition{
		Type:               cType,
		Status:             status,
		Message:            msg,
		Reason:             reason,
		LastTransitionTime: metav1.Now(),
	})
	return r.Status().Update(ctx, host)
}
