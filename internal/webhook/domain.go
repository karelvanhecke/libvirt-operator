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

type DomainDefaulter struct{}

func (DomainDefaulter) Default(ctx context.Context, obj runtime.Object) error {
	return generateName(obj)
}

func SetupDomainWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).For(&v1alpha1.Domain{}).
		WithDefaulter(DomainDefaulter{}).
		WithValidator(&DomainValidator{Client: mgr.GetClient()}).
		Complete()
}

type DomainValidator struct {
	client.Client
}

func (dv *DomainValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	domain, err := isDomainObject(obj)
	if err != nil {
		return nil, err
	}
	domains := &v1alpha1.DomainList{}
	if err := dv.List(ctx, domains); err != nil {
		return nil, err
	}
	for _, d := range domains.Items {
		if d.Spec.Name == domain.Spec.Name && d.Spec.HostRef.Name == domain.Spec.HostRef.Name {
			return nil, errors.New("name already in use on host")
		}
	}
	return nil, nil
}

func (dv *DomainValidator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	_, err := isDomainObject(newObj)
	if err != nil {
		return nil, err
	}
	return nil, nil
}

func (dv *DomainValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	_, err := isDomainObject(obj)
	if err != nil {
		return nil, err
	}
	return nil, nil
}

func isDomainObject(obj runtime.Object) (*v1alpha1.Domain, error) {
	d, ok := obj.(*v1alpha1.Domain)
	if !ok {
		return nil, errors.New("not a domain object")
	}
	return d, nil
}
