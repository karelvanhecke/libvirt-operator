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

	"github.com/digitalocean/go-libvirt/socket"
	"github.com/digitalocean/go-libvirt/socket/dialers"
	"github.com/karelvanhecke/libvirt-operator/api/v1alpha1"
	"github.com/karelvanhecke/libvirt-operator/internal/store"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type HostReconciler struct {
	client.Client
	AuthStore store.AuthStore
	HostStore store.HostStore
}

func (r *HostReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	host := &v1alpha1.Host{}
	if err := r.Get(ctx, req.NamespacedName, host); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if host.DeletionTimestamp.IsZero() {
		if !controllerutil.ContainsFinalizer(host, Finalizer) {
			controllerutil.AddFinalizer(host, Finalizer)
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
			volumes := &v1alpha1.VolumeList{}
			if err := r.List(ctx, volumes, &client.ListOptions{LabelSelector: labels.NewSelector().Add(*labelSelector)}); err != nil {
				return ctrl.Result{}, err
			}

			if len(volumes.Items) > 0 {
				return ctrl.Result{}, fmt.Errorf("can not delete host %s while in use by volumes", host.Name)
			}

			r.HostStore.Deregister(ctx, host.UID)

			controllerutil.RemoveFinalizer(host, Finalizer)
			if err := r.Update(ctx, host); err != nil {
				return ctrl.Result{}, err
			}
		}

		return ctrl.Result{}, nil
	}

	auth := &v1alpha1.Auth{}

	if err := r.Get(ctx, types.NamespacedName{Name: host.Spec.AuthRef.Name, Namespace: host.Namespace}, auth); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	authEntry, found := r.AuthStore.Lookup(auth.UID)
	if !found {
		ctrl.LoggerFrom(ctx).V(1).Info("auth not yet available", "auth", auth.Name)
		return ctrl.Result{Requeue: true}, nil
	}

	combinedVersion := host.ResourceVersion + "-" + authEntry.Version()

	hostEntry, found := r.HostStore.Lookup(host.UID)
	if found {
		if combinedVersion == hostEntry.Version() {
			return ctrl.Result{}, nil
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

	r.HostStore.Register(ctx, host.UID, r.HostStore.Entry(combinedVersion, dialer))

	if host.Labels == nil {
		host.Labels = make(map[string]string)
	}
	if host.Labels[LabelKeyAuth] != auth.Name {
		host.Labels[LabelKeyAuth] = auth.Name
		if err := r.Update(ctx, host); err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func (r *HostReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).For(&v1alpha1.Host{}).Watches(&v1alpha1.Auth{},
		handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, o client.Object) []reconcile.Request {
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
