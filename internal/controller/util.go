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
	"errors"

	"github.com/digitalocean/go-libvirt"
	"github.com/google/uuid"
	"github.com/karelvanhecke/libvirt-operator/api/v1alpha1"
)

const (
	ErrIDNotSet = "id is not set"
)

func resolvePoolIdentifier(id *v1alpha1.LibvirtIdentifierWithUUID) (libvirt.StoragePool, error) {
	if id == nil {
		return libvirt.StoragePool{}, errors.New(ErrIDNotSet)
	}

	u, err := uuid.Parse(id.UUID)
	if err != nil {
		return libvirt.StoragePool{}, err
	}

	return libvirt.StoragePool{Name: id.Name, UUID: libvirt.UUID(u)}, nil
}
