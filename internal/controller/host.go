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
	"github.com/digitalocean/go-libvirt/socket"
	"github.com/digitalocean/go-libvirt/socket/dialers"
	"github.com/karelvanhecke/libvirt-operator/api/v1alpha1"
	"github.com/karelvanhecke/libvirt-operator/internal/probe"
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
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
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

	if host.DeletionTimestamp.IsZero() {
		if controllerutil.AddFinalizer(host, v1alpha1.Finalizer) {
			if err := r.Update(ctx, host); err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		if controllerutil.ContainsFinalizer(host, v1alpha1.Finalizer) {
			labelSelector, err := labels.NewRequirement(v1alpha1.HostLabel, selection.Equals, []string{host.Name})
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
			domains := &v1alpha1.DomainList{}
			if err := r.List(ctx, domains, &client.ListOptions{LabelSelector: labels.NewSelector().Add(*labelSelector)}); err != nil {
				return ctrl.Result{}, err
			}

			if len(pools.Items)+len(networks.Items)+len(pciDevices.Items)+len(domains.Items) > 0 {
				if err := r.setStatusCondition(ctx, host, v1alpha1.ConditionDeletionPrevented, metav1.ConditionTrue, "host is in use by another resource", v1alpha1.ConditionInUse); err != nil {
					return ctrl.Result{}, err
				}
				return ctrl.Result{Requeue: true}, nil
			}

			r.HostStore.Deregister(ctx, host.UID)

			controllerutil.RemoveFinalizer(host, v1alpha1.Finalizer)
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
			if probed := meta.FindStatusCondition(host.Status.Conditions, v1alpha1.ConditionProbed); probed != nil && probed.Status == metav1.ConditionTrue {
				if d := time.Since(probed.LastTransitionTime.Time); d < dataRefreshInterval {
					return ctrl.Result{RequeueAfter: dataRefreshInterval - d}, nil
				}
			}
			return r.probe(ctx, host, hostEntry)
		}
	}

	var dialer socket.Dialer

	switch auth.Spec.Type {
	case v1alpha1.SSHAuth:
		opts := []dialers.SSHOption{dialers.WithSSHAuthMethods((&dialers.SSHAuthMethods{}).PrivKey()), dialers.UseKeyFile(filepath.Join(authEntry.GetPath(), privateKey)), dialers.UseSSHUsername(*auth.Spec.Username)}

		if auth.Spec.Verify != nil && !*auth.Spec.Verify {
			opts = append(opts, dialers.WithAcceptUnknownHostKey())
		} else {
			opts = append(opts, dialers.UseKnownHostsFile(filepath.Join(authEntry.GetPath(), knownHosts)))
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

	if util.SetLabel(&host.ObjectMeta, v1alpha1.AuthLabel, auth.Name) {
		if err := r.Update(ctx, host); err != nil {
			return ctrl.Result{}, err
		}
	}

	hostEntry, _ = r.HostStore.Lookup(host.UID)

	return r.probe(ctx, host, hostEntry)
}

func (r *HostReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).For(&v1alpha1.Host{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).Watches(&v1alpha1.Auth{},
		handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, o client.Object) []reconcile.Request {
			labelSelector, err := labels.NewRequirement(v1alpha1.AuthLabel, selection.Equals, []string{o.GetName()})
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
		}), builder.WithPredicates(watchPredicate)).Complete(r)
}

func (r *HostReconciler) probe(ctx context.Context, host *v1alpha1.Host, hostEntry *store.HostEntry) (ctrl.Result, error) {
	if err := r.setStatusCondition(ctx, host, v1alpha1.ConditionProbed, metav1.ConditionFalse, "New probe required", v1alpha1.ConditionRequired); err != nil {
		return ctrl.Result{}, err
	}

	hostClient, end, err := hostEntry.Session()
	if err != nil {
		if err := r.setStatusCondition(ctx, host, v1alpha1.ConditionProbed, metav1.ConditionFalse, conditionHostClientNotReady, v1alpha1.ConditionUnmetRequirements); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}
	defer end()

	probe, err := probe.NewHostProbe(hostClient)
	if err != nil {
		if err := r.setStatusCondition(ctx, host, v1alpha1.ConditionProbed, metav1.ConditionFalse, "Probe could not be completed", v1alpha1.ConditionError); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, err
	}

	host.Status.Capacity = &v1alpha1.HostCapability{
		Arch: probe.Arch(),
		CPUS: safecast.ToInt32(probe.CPUS()),
		Memory: v1alpha1.HostMemory{
			Total: safecast.ToInt64(probe.MemoryTotal()),
			Free:  safecast.ToInt64(probe.MemoryFree()),
		},
		NUMA: probe.NUMA(),
	}

	return ctrl.Result{RequeueAfter: dataRefreshInterval}, r.setStatusCondition(ctx, host, v1alpha1.ConditionProbed, metav1.ConditionTrue, conditionProbeCompleted, v1alpha1.ConditionCompleted)
}

func (r *HostReconciler) setStatusCondition(ctx context.Context, host *v1alpha1.Host, cType string, status metav1.ConditionStatus, msg string, reason string) error {
	c := metav1.Condition{
		Type:    cType,
		Status:  status,
		Message: msg,
		Reason:  reason,
	}

	if meta.SetStatusCondition(&host.Status.Conditions, c) {
		return r.Status().Update(ctx, host)
	}
	return nil
}
