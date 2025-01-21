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

package webhook

import (
	"context"
	"errors"

	"github.com/karelvanhecke/libvirt-operator/api/v1alpha1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

type VolumeDefaulter struct{}

func (VolumeDefaulter) Default(ctx context.Context, obj runtime.Object) error {
	return generateName(obj)
}

func SetupVolumeWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).For(&v1alpha1.Volume{}).
		WithDefaulter(VolumeDefaulter{}).
		WithValidator(&VolumeValidator{Client: mgr.GetClient()}).
		Complete()
}

type VolumeValidator struct {
	client.Client
}

func (vv *VolumeValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	volume, err := isVolumeObject(obj)
	if err != nil {
		return nil, err
	}
	volumes := &v1alpha1.VolumeList{}
	if err := vv.List(ctx, volumes); err != nil {
		return nil, err
	}
	for _, v := range volumes.Items {
		if v.Spec.Name == volume.Spec.Name && v.Spec.PoolRef.Name == volume.Spec.PoolRef.Name {
			return nil, errors.New("name already in use in pool")
		}
	}

	return nil, nil
}

func (vv *VolumeValidator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	_, err := isVolumeObject(newObj)
	if err != nil {
		return nil, err
	}
	return nil, nil
}

func (vv *VolumeValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	_, err := isVolumeObject(obj)
	if err != nil {
		return nil, err
	}
	return nil, nil
}

func isVolumeObject(obj runtime.Object) (*v1alpha1.Volume, error) {
	v, ok := obj.(*v1alpha1.Volume)
	if !ok {
		return nil, errors.New("not a volume object")
	}
	return v, nil
}
