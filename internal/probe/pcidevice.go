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

package probe

import (
	"errors"

	"github.com/digitalocean/go-libvirt"
	"github.com/karelvanhecke/libvirt-operator/internal/host"
	"libvirt.org/go/libvirtxml"
)

type PCIDeviceProbe struct {
	exists   bool
	active   bool
	domain   uint
	bus      uint
	slot     uint
	function uint
}

func NewPCIDeviceProbe(client host.Client, name string) (*PCIDeviceProbe, error) {
	p := &PCIDeviceProbe{}

	_, err := client.NodeDeviceLookupByName(name)
	if err != nil {
		if err, ok := err.(libvirt.Error); ok {
			if err.Code == uint32(libvirt.ErrNoNodeDevice) {
				return p, nil
			}
		}
		return nil, err
	}
	p.exists = true

	active, err := client.NodeDeviceIsActive(name)
	if err != nil {
		return nil, err
	}
	p.active = active == 1

	xml, err := client.NodeDeviceGetXMLDesc(name, 0)
	if err != nil {
		return nil, err
	}
	dev := &libvirtxml.NodeDevice{}
	if err := dev.Unmarshal(xml); err != nil {
		return nil, err
	}

	pci := dev.Capability.PCI
	if pci == nil {
		return nil, errors.New("is not a pci device")
	}

	if pci.Domain == nil || pci.Bus == nil || pci.Slot == nil || pci.Function == nil {
		return nil, errors.New("address could not be read")
	}
	p.domain = *pci.Domain
	p.bus = *pci.Bus
	p.slot = *pci.Slot
	p.function = *pci.Function

	return p, nil
}

func (p *PCIDeviceProbe) Exists() bool {
	return p.exists
}

func (p *PCIDeviceProbe) Active() bool {
	return p.active
}

func (p *PCIDeviceProbe) Domain() uint {
	return p.domain
}

func (p *PCIDeviceProbe) Bus() uint {
	return p.bus
}

func (p *PCIDeviceProbe) Slot() uint {
	return p.slot
}

func (p *PCIDeviceProbe) Function() uint {
	return p.function
}
