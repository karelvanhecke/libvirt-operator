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
	"github.com/digitalocean/go-libvirt"
	"github.com/karelvanhecke/libvirt-operator/internal/host"
)

type PoolProbe struct {
	exists     bool
	active     bool
	capacity   uint64
	allocation uint64
	available  uint64
}

func NewPoolProbe(client host.Client, name string) (*PoolProbe, error) {
	p := &PoolProbe{}

	pool, err := client.StoragePoolLookupByName(name)
	if err != nil {
		if err, ok := err.(libvirt.Error); ok {
			if err.Code == uint32(libvirt.ErrNoStoragePool) {
				return p, nil
			}
		}
		return nil, err
	}
	p.exists = true

	state, cap, alloc, avail, err := client.StoragePoolGetInfo(pool)
	if err != nil {
		return nil, err
	}

	p.active = state == uint8(libvirt.StoragePoolRunning)
	p.capacity = cap
	p.allocation = alloc
	p.available = avail

	return p, nil
}

func (p *PoolProbe) Exists() bool {
	return p.exists
}

func (p *PoolProbe) Active() bool {
	return p.active
}

func (p *PoolProbe) Capacity() uint64 {
	return p.capacity
}

func (p *PoolProbe) Allocation() uint64 {
	return p.allocation
}

func (p *PoolProbe) Available() uint64 {
	return p.available
}
