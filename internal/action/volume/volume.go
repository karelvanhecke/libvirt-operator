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

package volume

import (
	"crypto"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/digitalocean/go-libvirt"
	"github.com/karelvanhecke/libvirt-operator/internal/action"
	"github.com/karelvanhecke/libvirt-operator/internal/utils"
	"libvirt.org/go/libvirtxml"
)

type Action struct {
	*libvirt.Libvirt
	name  string
	pool  libvirt.StoragePool
	id    *libvirt.StorageVol
	state *libvirtxml.StorageVolume

	size         *libvirtxml.StorageVolumeSize
	format       *libvirtxml.StorageVolumeTargetFormat
	backingStore *libvirtxml.StorageVolumeBackingStore
	source       *os.File
}

func NewAction(client *libvirt.Libvirt, name string, pool string, size *libvirtxml.StorageVolumeSize, format *libvirtxml.StorageVolumeTargetFormat) (action.VolumeAction, error) {
	p, err := client.StoragePoolLookupByName(pool)
	if err != nil {
		return nil, err
	}

	a := &Action{
		Libvirt: client,
		name:    name,
		pool:    p,
		size:    size,
		format:  format,
	}

	v, err := a.StorageVolLookupByName(a.pool, a.name)
	if err != nil {
		if e, ok := err.(libvirt.Error); ok {
			if e.Code == uint32(libvirt.ErrNoStorageVol) {
				return a, nil
			}
		}
		return nil, err
	}

	xml, err := a.StorageVolGetXMLDesc(v, 0)
	if err != nil {
		return nil, err
	}

	state := &libvirtxml.StorageVolume{}
	if err := state.Unmarshal(xml); err != nil {
		return nil, err
	}
	a.id = &v
	a.state = state
	return a, nil
}

func (a *Action) State() bool {
	return a.id != nil
}

func (a *Action) WithBackingStore(name string) error {
	bs, err := a.StorageVolLookupByName(a.pool, name)
	if err != nil {
		return err
	}

	xml, err := a.StorageVolGetXMLDesc(bs, 0)
	if err != nil {
		return err
	}

	info := &libvirtxml.StorageVolume{}
	if err := info.Unmarshal(xml); err != nil {
		return err
	}

	a.backingStore = &libvirtxml.StorageVolumeBackingStore{
		Path:   info.Target.Path,
		Format: info.Target.Format,
	}

	if a.size == nil {
		a.size = info.Capacity
	}

	return nil
}

func (a *Action) WithSource(url string, checksum *string) error {
	// #nosec G107
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	a.source, err = os.CreateTemp("", "volume-source")
	if err != nil {
		return err
	}

	if _, err := io.Copy(a.source, resp.Body); err != nil {
		return err
	}

	if checksum != nil {
		hashes := map[string]crypto.Hash{
			"sha256": crypto.SHA256,
			"sha512": crypto.SHA512,
		}
		c := strings.Split(strings.ToLower(*checksum), ":")
		h, supported := hashes[c[0]]
		if !supported {
			return fmt.Errorf("unsupported hash: %s", *checksum)
		}
		hash := h.New()

		if _, err := a.source.Seek(0, 0); err != nil {
			return err
		}
		if _, err := io.Copy(hash, a.source); err != nil {
			return err
		}
		if hashString := hex.EncodeToString(hash.Sum(nil)); hashString != c[1] {
			return fmt.Errorf("download failed checksum verification, hash: %s", hashString)
		}
	}

	if _, err := a.source.Seek(0, 0); err != nil {
		return err
	}

	return nil
}

func (a *Action) Create() error {
	def := &libvirtxml.StorageVolume{
		Name:     a.name,
		Capacity: a.size,
		Target:   &libvirtxml.StorageVolumeTarget{Format: a.format},
	}

	if a.backingStore != nil {
		def.BackingStore = a.backingStore
	}

	xml, err := def.Marshal()
	if err != nil {
		return err
	}

	v, err := a.StorageVolCreateXML(a.pool, xml, 0)
	if err != nil {
		return err
	}

	if a.source != nil {
		return a.StorageVolUpload(v, a.source, 0, 0, libvirt.StorageVolUploadSparseStream)
	}

	return nil
}

func (a *Action) Update() error {
	if a.id == nil {
		return errors.New("can not update non-existing volume")
	}

	if a.size == nil {
		return nil
	}

	s := utils.ConvertToBytes(a.state.Capacity.Value, a.state.Capacity.Unit)
	r := utils.ConvertToBytes(a.size.Value, a.size.Unit)

	if r > s {
		return a.StorageVolResize(*a.id, r, 0)
	}

	return nil
}

func (a *Action) Delete() error {
	if a.id == nil {
		return nil
	}
	return a.StorageVolDelete(*a.id, libvirt.StorageVolDeleteNormal)
}
