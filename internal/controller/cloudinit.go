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
	"bytes"
	"context"
	"os"
	"path/filepath"

	"github.com/ARM-software/golang-utils/utils/safecast"
	"github.com/diskfs/go-diskfs/backend/file"
	"github.com/diskfs/go-diskfs/filesystem/iso9660"
	"github.com/karelvanhecke/libvirt-operator/api/v1alpha1"
	"github.com/karelvanhecke/libvirt-operator/internal/action"
	"github.com/karelvanhecke/libvirt-operator/internal/store"
	"github.com/karelvanhecke/libvirt-operator/internal/util"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

const (
	CIPrefix = "cidata:"
)

type CloudInitReconciler struct {
	client.Client
	HostStore *store.HostStore
}

func (r *CloudInitReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	ci := &v1alpha1.CloudInit{}
	if err := r.Get(ctx, req.NamespacedName, ci); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	pool, host, err := r.resolveRefs(ctx, ci)
	if err != nil {
		if !meta.IsStatusConditionTrue(ci.Status.Conditions, v1alpha1.ConditionReady) {
			if err := r.setStatusCondition(ctx, ci, v1alpha1.ConditionReady, metav1.ConditionFalse, err.Error(), v1alpha1.ConditionUnmetRequirements); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, err
	}

	if !meta.IsStatusConditionTrue(pool.Status.Conditions, v1alpha1.ConditionReady) {
		if !meta.IsStatusConditionTrue(ci.Status.Conditions, v1alpha1.ConditionReady) {
			if err := r.setStatusCondition(ctx, ci, v1alpha1.ConditionReady, metav1.ConditionFalse, conditionPoolNotReady, v1alpha1.ConditionUnmetRequirements); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{Requeue: true}, nil
	}

	hostEntry, found := r.HostStore.Lookup(host.UID)
	if !found {
		if !meta.IsStatusConditionTrue(ci.Status.Conditions, v1alpha1.ConditionReady) {
			if err := r.setStatusCondition(ctx, ci, v1alpha1.ConditionReady, metav1.ConditionFalse, conditionHostClientNotReady, v1alpha1.ConditionUnmetRequirements); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{Requeue: true}, nil
	}
	hostClient, end, err := hostEntry.Session()
	if err != nil {
		if !meta.IsStatusConditionTrue(ci.Status.Conditions, v1alpha1.ConditionReady) {
			if err := r.setStatusCondition(ctx, ci, v1alpha1.ConditionReady, metav1.ConditionFalse, conditionHostClientNotReady, v1alpha1.ConditionUnmetRequirements); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{Requeue: true}, nil
	}
	defer end()

	action, err := action.NewVolumeAction(hostClient, ci.ResourceName(), pool.ResourceName())
	if err != nil {
		return ctrl.Result{}, err
	}

	exists := action.State()

	if ci.DeletionTimestamp.IsZero() {
		if controllerutil.AddFinalizer(ci, v1alpha1.Finalizer) {
			if err := r.Update(ctx, ci); err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		if controllerutil.ContainsFinalizer(ci, v1alpha1.Finalizer) {
			if exists {
				labelSelector, err := labels.NewRequirement(v1alpha1.CloudInitLabel, selection.Equals, []string{ci.Name})
				if err != nil {
					return ctrl.Result{}, err
				}
				domains := &v1alpha1.DomainList{}
				if err := r.List(ctx, domains, &client.ListOptions{LabelSelector: labels.NewSelector().Add(*labelSelector)}); err != nil {
					return ctrl.Result{}, err
				}
				if len(domains.Items) > 0 {
					if err := r.setStatusCondition(ctx, ci, v1alpha1.ConditionDeletionPrevented, metav1.ConditionTrue, "cloud-init is currently in use by domain", v1alpha1.ConditionInUse); err != nil {
						return ctrl.Result{}, err
					}
					return ctrl.Result{Requeue: true}, nil
				}
				if err := action.Delete(); err != nil {
					return ctrl.Result{}, err
				}
			}
			controllerutil.RemoveFinalizer(ci, v1alpha1.Finalizer)
			err := r.Update(ctx, ci)
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	if exists {
		if !meta.IsStatusConditionTrue(ci.Status.Conditions, v1alpha1.ConditionReady) {
			ci.Status.Pool = pool.ResourceName()
			ci.Status.Host = pool.Spec.HostRef.Name
			if err := r.setStatusCondition(ctx, ci, v1alpha1.ConditionReady, metav1.ConditionTrue, conditionCreationSucceeded, v1alpha1.ConditionCreated); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	sourceFiles, size, err := cloudInitSourceFiles(ci.Spec.Metadata, ci.Spec.UserData, ci.Spec.NetworkConfig, ci.Spec.VendorData)
	if err != nil {
		if err := r.setStatusCondition(ctx, ci, v1alpha1.ConditionReady, metav1.ConditionFalse, err.Error(), v1alpha1.ConditionError); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, err
	}

	if size < 2048 {
		size = 2048
	}

	iso, err := cloudInitCreateISO(sourceFiles, size)
	if err != nil {
		if err := r.setStatusCondition(ctx, ci, v1alpha1.ConditionReady, metav1.ConditionFalse, err.Error(), v1alpha1.ConditionError); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, err
	}

	action.Size("bytes", safecast.ToUint64(size))
	action.Format("raw")
	action.LocalSource(iso)

	if err = action.Create(); err != nil {
		if err := r.setStatusCondition(ctx, ci, v1alpha1.ConditionReady, metav1.ConditionFalse, err.Error(), v1alpha1.ConditionError); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, err
	}

	if util.SetLabel(&ci.ObjectMeta, v1alpha1.PoolLabel, ci.Spec.PoolRef.Name) {
		if err := r.Update(ctx, ci); err != nil {
			return ctrl.Result{}, err
		}
	}

	ci.Status.Pool = pool.ResourceName()
	ci.Status.Host = pool.Spec.HostRef.Name

	if err := r.setStatusCondition(ctx, ci, v1alpha1.ConditionReady, metav1.ConditionTrue, conditionCreationSucceeded, v1alpha1.ConditionCreated); err != nil {
		return ctrl.Result{}, err
	}

	if err := action.CleanupSource(); err != nil {
		ctrl.LoggerFrom(ctx).Error(err, "could not cleanup cached cloud-init iso")
	}

	return ctrl.Result{}, nil
}

func (r *CloudInitReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).For(&v1alpha1.CloudInit{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).Complete(r)
}

func (r *CloudInitReconciler) resolveRefs(ctx context.Context, ci *v1alpha1.CloudInit) (*v1alpha1.Pool, *v1alpha1.Host, error) {
	pool := &v1alpha1.Pool{}
	if err := r.Get(ctx, ci.PoolRef(), pool); err != nil {
		return nil, nil, err
	}

	host := &v1alpha1.Host{}
	if err := r.Get(ctx, pool.HostRef(), host); err != nil {
		return nil, nil, err
	}

	return pool, host, nil
}

func (r *CloudInitReconciler) setStatusCondition(ctx context.Context, ci *v1alpha1.CloudInit, cType string, status metav1.ConditionStatus, msg string, reason string) error {
	c := metav1.Condition{
		Type:    cType,
		Status:  status,
		Message: msg,
		Reason:  reason,
	}
	if meta.SetStatusCondition(&ci.Status.Conditions, c) {
		return r.Status().Update(ctx, ci)
	}
	return nil
}

func cloudInitCloudConfig(config *v1alpha1.CloudInitCloudConfig) ([]byte, error) {
	return util.Marshal(bytes.NewBuffer([]byte("#cloud-config\n")), config)
}

func cloudInitNetworkConfig(config *v1alpha1.CloudInitNetworkConfig) ([]byte, error) {
	type networkVersionedConfig struct {
		Version int32                            `yaml:"version"`
		Config  *v1alpha1.CloudInitNetworkConfig `yaml:",inline"`
	}

	type networkConfig struct {
		Network networkVersionedConfig `yaml:"network"`
	}

	cfg := &networkConfig{
		Network: networkVersionedConfig{
			Version: 2,
			Config:  config,
		},
	}

	return util.Marshal(bytes.NewBuffer([]byte{}), cfg)
}

func cloudInitSourceFiles(metadata *v1alpha1.CloudInitMetadata,
	userData *v1alpha1.CloudInitCloudConfig,
	networkConfig *v1alpha1.CloudInitNetworkConfig,
	vendorData *v1alpha1.CloudInitCloudConfig) (sourceFiles map[string][]byte, size int64, err error) {

	sourceFiles = map[string][]byte{}

	addToSize := func(s []byte) {
		size = size + int64(len(s))
	}

	metadataBytes, err := util.Marshal(bytes.NewBuffer([]byte{}), metadata)
	if err != nil {
		return nil, 0, err
	}
	sourceFiles["meta-data"] = metadataBytes
	addToSize(metadataBytes)

	if userData != nil {
		userDataBytes, err := cloudInitCloudConfig(userData)
		if err != nil {
			return nil, 0, err
		}
		sourceFiles["user-data"] = userDataBytes
		addToSize(userDataBytes)
	}

	if vendorData != nil {
		vendorDataBytes, err := cloudInitCloudConfig(vendorData)
		if err != nil {
			return nil, 0, err
		}
		sourceFiles["vendor-data"] = vendorDataBytes
		addToSize(vendorDataBytes)
	}
	if networkConfig != nil {
		networkConfigBytes, err := cloudInitNetworkConfig(networkConfig)
		if err != nil {
			return nil, 0, err
		}
		sourceFiles["network-config"] = networkConfigBytes
		addToSize(networkConfigBytes)
	}
	return
}

func cloudInitCreateISO(sourceFiles map[string][]byte, size int64) (*os.File, error) {
	iso, err := os.CreateTemp("", "iso.")
	if err != nil {
		return nil, err
	}

	isoFS, err := iso9660.Create(file.New(iso, false), 36864+size, 0, 0, "")
	if err != nil {
		return nil, err
	}

	for file, data := range sourceFiles {
		file, err := os.Create(filepath.Join(isoFS.Workspace(), file))
		if err != nil {
			return nil, err
		}
		_, err = file.Write(data)
		if err != nil {
			return nil, err
		}
		err = file.Close()
		if err != nil {
			return nil, err
		}
	}

	if err := isoFS.Finalize(iso9660.FinalizeOptions{
		RockRidge:        true,
		VolumeIdentifier: "cidata",
	}); err != nil {
		return nil, err
	}

	return iso, nil
}
