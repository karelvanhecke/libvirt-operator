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

package fake

import (
	"errors"
	"slices"

	"github.com/digitalocean/go-libvirt"
	"libvirt.org/go/libvirtxml"
)

const (
	ErrVolumeExists   = "volume already exists"
	ErrVolumeNotExist = "volume does not exist"
)

type Pool struct {
	state   int32
	xml     *libvirtxml.StoragePool
	volumes []*libvirtxml.StorageVolume
}

func (p *Pool) getVolumeByName(name string) (*libvirtxml.StorageVolume, error) {
	i := slices.IndexFunc(p.volumes, func(volume *libvirtxml.StorageVolume) bool { return volume.Name == name })
	if i == -1 {
		return nil, libvirt.Error{
			Code:    uint32(libvirt.ErrNoStorageVol),
			Message: ErrVolumeNotExist,
		}
	}

	return p.volumes[i], nil
}

func (p *Pool) createVol(xml string) (*libvirtxml.StorageVolume, error) {
	vol := &libvirtxml.StorageVolume{}
	err := vol.Unmarshal(xml)
	if err != nil {
		return nil, err
	}
	_, err = p.getVolumeByName(vol.Name)
	if err == nil {
		return nil, errors.New(ErrVolumeExists)
	}

	if err.Error() != ErrVolumeNotExist {
		return nil, err
	}

	p.volumes = append(p.volumes, vol)
	return vol, nil
}

func (p *Pool) deleteVol(name string) error {
	if _, err := p.getVolumeByName(name); err != nil {
		return err
	}
	p.volumes = slices.DeleteFunc(p.volumes, func(v *libvirtxml.StorageVolume) bool { return v.Name == name })
	return nil
}
