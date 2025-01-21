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

type CloudInitDefaulter struct{}

func (CloudInitDefaulter) Default(ctx context.Context, obj runtime.Object) error {
	if err := generateName(obj); err != nil {
		return err
	}

	ci, ok := obj.(*v1alpha1.CloudInit)
	if !ok {
		return errors.New("object is not of CloudInit type")
	}

	if ci.Spec.Metadata != nil {
		return nil
	}

	ci.Spec.Metadata = &v1alpha1.CloudInitMetadata{
		LocalHostname: ci.ResourceName(),
	}

	return nil
}

func SetupCloudInitWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).For(&v1alpha1.CloudInit{}).
		WithDefaulter(CloudInitDefaulter{}).
		WithValidator(&CloudInitValidator{Client: mgr.GetClient()}).
		Complete()
}

type CloudInitValidator struct {
	client.Client
}

func (civ *CloudInitValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	cloudInit, err := isCloudInitObject(obj)
	if err != nil {
		return nil, err
	}
	cloudInits := &v1alpha1.CloudInitList{}
	if err := civ.List(ctx, cloudInits); err != nil {
		return nil, err
	}
	for _, ci := range cloudInits.Items {
		if ci.Spec.Name == cloudInit.Spec.Name && ci.Spec.PoolRef.Name == cloudInit.Spec.PoolRef.Name {
			return nil, errors.New("name already in use in pool")
		}
	}
	return nil, nil
}

func (civ *CloudInitValidator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	_, err := isCloudInitObject(newObj)
	if err != nil {
		return nil, err
	}
	return nil, nil
}

func (civ *CloudInitValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	_, err := isCloudInitObject(obj)
	if err != nil {
		return nil, err
	}
	return nil, nil
}

func isCloudInitObject(obj runtime.Object) (*v1alpha1.CloudInit, error) {
	ci, ok := obj.(*v1alpha1.CloudInit)
	if !ok {
		return nil, errors.New("not a cloud-init object")
	}
	return ci, nil
}
