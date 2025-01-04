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

	"github.com/diskfs/go-diskfs/backend/file"
	"github.com/diskfs/go-diskfs/filesystem/iso9660"
	"github.com/karelvanhecke/libvirt-operator/api/v1alpha1"
	"github.com/karelvanhecke/libvirt-operator/internal/action"
	"github.com/karelvanhecke/libvirt-operator/internal/store"
	"gopkg.in/yaml.v3"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"libvirt.org/go/libvirtxml"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	ConditionMessageCloudInitCreationInProgress = "cloud-init creation in progress"
	ConditionMessageCloudInitCreationFailed     = "cloud-init creation failed"
	ConditionMessageCloudInitCreationSucceeded  = "cloud-init creation succeeded"
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

	metadata := ci.Spec.Metadata
	if metadata == nil {
		metadata = &v1alpha1.CloudInitMetadata{
			InstanceID:    string(ci.UID),
			LocalHostname: ci.Name,
		}
	}

	sourceFiles, size, err := cloudInitSourceFiles(metadata, ci.Spec.UserData, ci.Spec.NetworkConfig, ci.Spec.VendorData)
	if err != nil {
		return ctrl.Result{}, err
	}

	if size < 2048 {
		size = 2048
	}

	poolRef := &v1alpha1.Pool{}
	if err := r.Get(ctx, types.NamespacedName{Name: ci.Spec.PoolRef.Name, Namespace: ci.Namespace}, poolRef); err != nil {
		if !meta.IsStatusConditionTrue(ci.Status.Conditions, ConditionTypeCreated) {
			meta.SetStatusCondition(&ci.Status.Conditions, metav1.Condition{
				Type:               ConditionTypeCreated,
				Status:             metav1.ConditionFalse,
				Message:            ConditionMessagePoolNotFound,
				Reason:             ConditionReasonFailed,
				LastTransitionTime: metav1.Now(),
			})
			if err := r.Status().Update(ctx, ci); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, err
	}

	if poolRef.Status.Identifier == nil || poolRef.Status.Active == nil || !*poolRef.Status.Active {
		if !meta.IsStatusConditionTrue(ci.Status.Conditions, ConditionTypeCreated) {
			meta.SetStatusCondition(&ci.Status.Conditions, metav1.Condition{
				Type:               ConditionTypeCreated,
				Status:             metav1.ConditionFalse,
				Message:            ConditionMessageWaitingForPool,
				Reason:             ConditionReasonFailed,
				LastTransitionTime: metav1.Now(),
			})
			if err := r.Status().Update(ctx, ci); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{Requeue: true}, nil
	}

	pool, err := resolvePoolIdentifier(poolRef.Status.Identifier)
	if err != nil {
		return ctrl.Result{}, err
	}

	hostRef := &v1alpha1.Host{}
	if err := r.Get(ctx, types.NamespacedName{Name: poolRef.Spec.HostRef.Name, Namespace: ci.Namespace}, hostRef); err != nil {
		if !meta.IsStatusConditionTrue(ci.Status.Conditions, ConditionTypeCreated) {
			meta.SetStatusCondition(&ci.Status.Conditions, metav1.Condition{
				Type:               ConditionTypeCreated,
				Status:             metav1.ConditionFalse,
				Message:            ConditionMessageHostNotFound,
				Reason:             ConditionReasonFailed,
				LastTransitionTime: metav1.Now(),
			})
			if err := r.Status().Update(ctx, ci); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, err
	}

	hostEntry, found := r.HostStore.Lookup(hostRef.UID)
	if !found {
		if !meta.IsStatusConditionTrue(ci.Status.Conditions, ConditionTypeCreated) {
			meta.SetStatusCondition(&ci.Status.Conditions, metav1.Condition{
				Type:               ConditionTypeCreated,
				Status:             metav1.ConditionFalse,
				Message:            ConditionMessageWaitingForHost,
				Reason:             ConditionReasonFailed,
				LastTransitionTime: metav1.Now(),
			})
			if err := r.Status().Update(ctx, ci); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{Requeue: true}, nil
	}
	hClient, end := hostEntry.Session()
	defer end()

	// #nosec #G115
	action, err := action.NewVolumeAction(hClient, ci.Namespace+":"+"cidata:"+ci.Name, pool,
		&libvirtxml.StorageVolumeSize{Unit: "bytes", Value: uint64(size)},
		&libvirtxml.StorageVolumeTargetFormat{Type: "raw"})
	if err != nil {
		return ctrl.Result{}, err
	}

	if ci.DeletionTimestamp.IsZero() {
		if !controllerutil.ContainsFinalizer(ci, Finalizer) {
			controllerutil.AddFinalizer(ci, Finalizer)
			if err := r.Update(ctx, ci); err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		if controllerutil.ContainsFinalizer(ci, Finalizer) {
			if action.State() {
				if err := action.Delete(); err != nil {
					return ctrl.Result{}, err
				}
			}
			controllerutil.RemoveFinalizer(ci, Finalizer)
			err := r.Update(ctx, ci)
			return ctrl.Result{}, err
		}
	}

	if action.State() {
		return ctrl.Result{}, nil
	}

	meta.SetStatusCondition(&ci.Status.Conditions, metav1.Condition{
		Type:               ConditionTypeCreated,
		Status:             metav1.ConditionFalse,
		Message:            ConditionMessageCloudInitCreationInProgress,
		Reason:             ConditionReasonInProgress,
		LastTransitionTime: metav1.Now(),
	})
	if err := r.Status().Update(ctx, ci); err != nil {
		return ctrl.Result{}, err
	}

	iso, err := cloudInitCreateISO(sourceFiles, size)
	if err != nil {
		return ctrl.Result{}, err
	}

	action.WithLocalSource(iso)

	if ci.Labels == nil {
		ci.Labels = map[string]string{LabelKeyPool: ci.Spec.PoolRef.Name}
	} else {
		ci.Labels[LabelKeyPool] = ci.Spec.PoolRef.Name
	}
	if err := r.Update(ctx, ci); err != nil {
		return ctrl.Result{}, err
	}

	if err := action.Create(); err != nil {
		meta.SetStatusCondition(&ci.Status.Conditions, metav1.Condition{
			Type:               ConditionTypeCreated,
			Status:             metav1.ConditionFalse,
			Message:            ConditionMessageCloudInitCreationFailed,
			Reason:             ConditionReasonFailed,
			LastTransitionTime: metav1.Now(),
		})
		if err := r.Status().Update(ctx, ci); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, err
	}

	meta.SetStatusCondition(&ci.Status.Conditions, metav1.Condition{
		Type:               ConditionTypeCreated,
		Status:             metav1.ConditionTrue,
		Message:            ConditionMessageCloudInitCreationSucceeded,
		Reason:             ConditionReasonSucceeded,
		LastTransitionTime: metav1.Now(),
	})
	if err := r.Status().Update(ctx, ci); err != nil {
		return ctrl.Result{}, err
	}

	if err := os.Remove(iso.Name()); err != nil {
		ctrl.LoggerFrom(ctx).Error(err, "could not cleanup cached cloud-init iso")
	}

	return ctrl.Result{}, nil
}

func (r *CloudInitReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).For(&v1alpha1.CloudInit{}).Complete(r)
}

func cloudInitCloudConfig(config *v1alpha1.CloudInitCloudConfig) ([]byte, error) {
	return yamlMarshallWithEncoder(bytes.NewBuffer([]byte("#cloud-config\n")), config)
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

	return yamlMarshallWithEncoder(bytes.NewBuffer([]byte{}), cfg)
}

func yamlMarshallWithEncoder(buffer *bytes.Buffer, data any) ([]byte, error) {
	enc := yaml.NewEncoder(buffer)
	enc.SetIndent(2)
	if err := enc.Encode(data); err != nil {
		return nil, err
	}
	if err := enc.Close(); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func cloudInitSourceFiles(metadata *v1alpha1.CloudInitMetadata,
	userData *v1alpha1.CloudInitCloudConfig,
	networkConfig *v1alpha1.CloudInitNetworkConfig,
	vendorData *v1alpha1.CloudInitCloudConfig) (sourceFiles map[string][]byte, size int64, err error) {

	sourceFiles = map[string][]byte{}

	addToSize := func(s []byte) {
		size = size + int64(len(s))
	}

	metadataBytes, err := yamlMarshallWithEncoder(bytes.NewBuffer([]byte{}), metadata)
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

func cloudInitCreateISO(sourceFiles map[string][]byte, size int64) (iso *os.File, err error) {
	iso, err = os.CreateTemp("", "iso.")
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

	return
}
