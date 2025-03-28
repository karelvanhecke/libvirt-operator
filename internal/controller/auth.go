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

	"github.com/karelvanhecke/libvirt-operator/api/v1alpha1"
	"github.com/karelvanhecke/libvirt-operator/internal/store"
	"github.com/karelvanhecke/libvirt-operator/internal/util"
	corev1 "k8s.io/api/core/v1"
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

type AuthReconciler struct {
	client.Client
	AuthStore *store.AuthStore
}

func (r *AuthReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	auth := &v1alpha1.Auth{}

	if err := r.Get(ctx, req.NamespacedName, auth); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	entry, found := r.AuthStore.Lookup(auth.UID)

	if auth.DeletionTimestamp.IsZero() {
		if controllerutil.AddFinalizer(auth, v1alpha1.Finalizer) {
			if err := r.Update(ctx, auth); err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		if controllerutil.ContainsFinalizer(auth, v1alpha1.Finalizer) {
			labelSelector, err := labels.NewRequirement(v1alpha1.AuthLabel, selection.Equals, []string{auth.Name})
			if err != nil {
				return ctrl.Result{}, err
			}
			hosts := &v1alpha1.HostList{}
			if err := r.List(ctx, hosts, &client.ListOptions{LabelSelector: labels.NewSelector().Add(*labelSelector)}); err != nil {
				return ctrl.Result{}, err
			}

			if len(hosts.Items) > 0 {
				if err := r.setStatusCondition(ctx, auth, v1alpha1.ConditionDeletionPrevented, metav1.ConditionTrue, "auth is currently in use by host", v1alpha1.ConditionInUse); err != nil {
					return ctrl.Result{}, err
				}
				return ctrl.Result{Requeue: true}, nil
			}

			r.AuthStore.Deregister(ctx, auth.UID)

			controllerutil.RemoveFinalizer(auth, v1alpha1.Finalizer)
			err = r.Update(ctx, auth)
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}
	secret := &corev1.Secret{}

	if err := r.Get(ctx, auth.SecretRef(), secret); err != nil {
		return ctrl.Result{}, err
	}

	combinedGeneration := auth.Generation + secret.Generation

	if found {
		if combinedGeneration == entry.Generation() {
			return ctrl.Result{}, nil
		}
	}

	files := []*store.File{}
	switch auth.Spec.Type {
	case v1alpha1.SSHAuth:
		if secret.Type != corev1.SecretTypeSSHAuth {
			return ctrl.Result{}, fmt.Errorf("ssh auth requires a secret of type: %s", corev1.SecretTypeSSHAuth)
		}
		files = append(files, store.NewFile(privateKey, secret.Data["ssh-privatekey"]))

		if auth.Spec.Verify == nil || *auth.Spec.Verify {
			files = append(files, store.NewFile(knownHosts, []byte(*auth.Spec.KnownHosts)))
		}
	case v1alpha1.TLSAuth:
		if secret.Type != corev1.SecretTypeTLS {
			return ctrl.Result{}, fmt.Errorf("tls auth requires a secret of type: %s", corev1.SecretTypeTLS)
		}

		files = append(files,
			store.NewFile(clientCert, secret.Data["tls.crt"]),
			store.NewFile(clientKey, secret.Data["tls.key"]))

		if auth.Spec.Verify == nil || *auth.Spec.Verify {
			files = append(files, store.NewFile(caCert, []byte(*auth.Spec.Ca)))
		}
	default:
		return ctrl.Result{}, fmt.Errorf("unsupported auth type: %s", auth.Spec.Type)
	}

	if err := r.AuthStore.Register(ctx, auth.UID, combinedGeneration, files); err != nil {
		return ctrl.Result{}, err
	}

	if util.SetLabel(&auth.ObjectMeta, v1alpha1.SecretLabel, secret.Name) {
		return ctrl.Result{}, r.Update(ctx, auth)
	}

	return ctrl.Result{}, nil
}

func (r *AuthReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).For(&v1alpha1.Auth{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).Watches(&corev1.Secret{},
		handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, o client.Object) []reconcile.Request {
			labelSelector, err := labels.NewRequirement(v1alpha1.SecretLabel, selection.Equals, []string{o.GetName()})
			if err != nil {
				return []reconcile.Request{}
			}
			authList := &v1alpha1.AuthList{}
			if err := r.List(ctx, authList, &client.ListOptions{LabelSelector: labels.NewSelector().Add(*labelSelector)}); err != nil {
				ctrl.LoggerFrom(ctx).Error(err, "could not get list of Auth objects")
				return []reconcile.Request{}
			}

			queued := []reconcile.Request{}
			for _, auth := range authList.Items {
				queued = append(queued, reconcile.Request{NamespacedName: types.NamespacedName{
					Name:      auth.Name,
					Namespace: auth.Namespace,
				}})
			}
			return queued
		}), builder.WithPredicates(watchPredicate)).Complete(r)
}

func (r *AuthReconciler) setStatusCondition(ctx context.Context, auth *v1alpha1.Auth, cType string, status metav1.ConditionStatus, msg string, reason string) error {
	c := metav1.Condition{
		Type:    cType,
		Status:  status,
		Message: msg,
		Reason:  reason,
	}

	if meta.SetStatusCondition(&auth.Status.Conditions, c) {
		return r.Status().Update(ctx, auth)
	}
	return nil
}
