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

package action

import (
	"github.com/digitalocean/go-libvirt"
	"github.com/karelvanhecke/libvirt-operator/internal/host"
	"libvirt.org/go/libvirtxml"
)

type DomainAction struct {
	host.Client
	name         string
	id           *libvirt.Domain
	definedState *libvirtxml.Domain
	state        int32
}

func NewDomainAction(client host.Client, name string) (*DomainAction, error) {
	d, err := client.DomainLookupByName(name)

	a := &DomainAction{
		Client: client,
		name:   name,
	}

	if err != nil {
		if e, ok := err.(libvirt.Error); ok {
			if e.Code == uint32(libvirt.ErrNoDomain) {
				return a, err
			}
		}
		return nil, err
	}

	a.id = &d

	xml, err := a.DomainGetXMLDesc(d, 0)
	if err != nil {
		return nil, err
	}
	a.definedState = &libvirtxml.Domain{}
	if err := a.definedState.Unmarshal(xml); err != nil {
		return nil, err
	}

	state, _, err := a.DomainGetState(d, 0)
	if err != nil {
		return nil, err
	}
	a.state = state

	return a, nil
}

func (a *DomainAction) State() (exists bool, state int32) {
	return a.id != nil, a.state
}

func (a *DomainAction) Create() error {
	return nil
}

func (a *DomainAction) Update() error {
	return nil
}

func (a *DomainAction) Delete() error {
	return nil
}
