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
	"context"
	"crypto"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/ARM-software/golang-utils/utils/safecast"
	"github.com/digitalocean/go-libvirt"
	"github.com/karelvanhecke/libvirt-operator/internal/host"
	"github.com/karelvanhecke/libvirt-operator/internal/util"
	"libvirt.org/go/libvirtxml"
)

const (
	ErrUnsupportedHash   = "unsupported hash"
	ErrChecksumFail      = "download failed checksum verification"
	ErrNoExistingVolume  = "volume does not exist"
	ErrVolumeShrinking   = "shrinking a volume is not supported"
	ErrLiveResizeNoMatch = "could not match volume to a disk during live resize"
)

type VolumeAction struct {
	host.Client
	name     string
	poolName string
	pool     libvirt.StoragePool

	id  *libvirt.StorageVol
	def *libvirtxml.StorageVolume

	source *os.File
}

func NewVolumeAction(client host.Client, name string, pool string) (*VolumeAction, error) {
	a := &VolumeAction{
		Client:   client,
		name:     name,
		poolName: pool,
	}

	p, err := a.StoragePoolLookupByName(pool)
	if err != nil {
		return nil, err
	}
	a.pool = p

	id, err := a.StorageVolLookupByName(a.pool, a.name)
	if err != nil {
		if e, ok := err.(libvirt.Error); ok {
			if e.Code == uint32(libvirt.ErrNoStorageVol) {
				a.def = &libvirtxml.StorageVolume{
					Name: a.name,
				}
				return a, nil
			}
		}
		return nil, err
	}
	a.id = &id

	xml, err := a.StorageVolGetXMLDesc(*a.id, 0)
	if err != nil {
		return nil, err
	}

	a.def = &libvirtxml.StorageVolume{}
	if err := a.def.Unmarshal(xml); err != nil {
		return nil, err
	}

	return a, nil
}

func (a *VolumeAction) State() (exists bool) {
	return a.id != nil
}

func (a *VolumeAction) Size(unit string, value uint64) {
	a.def.Capacity = &libvirtxml.StorageVolumeSize{
		Unit:  unit,
		Value: value,
	}
}

func (a *VolumeAction) Format(format string) {
	a.def.Target = &libvirtxml.StorageVolumeTarget{
		Format: &libvirtxml.StorageVolumeTargetFormat{
			Type: format,
		},
	}
}

func (a *VolumeAction) BackingStore(name string, pool string) error {
	p, err := a.StoragePoolLookupByName(pool)
	if err != nil {
		return err
	}

	bs, err := a.StorageVolLookupByName(p, name)
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

	a.def.BackingStore = &libvirtxml.StorageVolumeBackingStore{
		Path:   info.Target.Path,
		Format: info.Target.Format,
	}

	if a.def.Capacity == nil {
		a.Size(info.Capacity.Unit, info.Capacity.Value)
	}

	return nil
}

func (a *VolumeAction) LocalSource(file *os.File) {
	a.source = file
}

func (a *VolumeAction) RemoteSource(ctx context.Context, url string, checksum *string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

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
			return errors.New(ErrUnsupportedHash)
		}
		hash := h.New()

		if _, err := a.source.Seek(0, 0); err != nil {
			return err
		}
		if _, err := io.Copy(hash, a.source); err != nil {
			return err
		}
		if hashString := hex.EncodeToString(hash.Sum(nil)); hashString != c[1] {
			return errors.New(ErrChecksumFail)
		}
	}

	if _, err := a.source.Seek(0, 0); err != nil {
		return err
	}

	if a.def.Capacity == nil {
		a.Size("", safecast.ToUint64(resp.ContentLength))
	}

	return nil
}

func (a *VolumeAction) Create() error {
	xml, err := a.def.Marshal()
	if err != nil {
		return err
	}

	v, err := a.StorageVolCreateXML(a.pool, xml, 0)
	if err != nil {
		return err
	}
	a.id = &v

	if a.source != nil {
		return a.StorageVolUpload(v, a.source, 0, 0, libvirt.StorageVolUploadSparseStream)
	}
	return nil
}

func (a *VolumeAction) ResizeRequired(unit string, value uint64) (bool, error) {
	current := util.ConvertToBytes(a.def.Capacity.Value, a.def.Capacity.Unit)
	desired := util.ConvertToBytes(value, unit)

	if desired < current {
		return false, errors.New(ErrVolumeShrinking)
	}

	if desired > current {
		return true, nil
	}

	return false, nil
}

func (a *VolumeAction) Resize(unit string, value uint64) error {
	b := util.ConvertToBytes(value, unit)
	return a.StorageVolResize(*a.id, b, 0)
}

func (a *VolumeAction) LiveResize(domain string, unit string, value uint64) error {
	dom, err := a.DomainLookupByName(domain)
	if err != nil {
		if err, ok := err.(libvirt.Error); ok {
			if err.Code == uint32(libvirt.ErrNoDomain) {
				return a.Resize(unit, value)
			}
		}
		return err
	}

	state, _, err := a.DomainGetState(dom, 0)
	if err != nil {
		return err
	}

	if state != int32(libvirt.DomainRunning) {
		return a.Resize(unit, value)
	}

	xml, err := a.DomainGetXMLDesc(dom, 0)
	if err != nil {
		return err
	}

	desc := &libvirtxml.Domain{}
	if err := desc.Unmarshal(xml); err != nil {
		return err
	}

	for _, disk := range desc.Devices.Disks {
		ok := false
		if f := disk.Source.File; f != nil {
			if f.File == a.def.Target.Path {
				ok = true
			}
		}

		if b := disk.Source.Block; b != nil {
			if b.Dev == a.def.Target.Path {
				ok = true
			}
		}

		if v := disk.Source.Volume; v != nil {
			if v.Volume == a.name && v.Pool == a.poolName {
				ok = true
			}
		}

		if ok {
			return a.DomainBlockResize(dom, disk.Target.Dev, util.ConvertToBytes(value, unit), libvirt.DomainBlockResizeBytes)
		}
	}

	return errors.New(ErrLiveResizeNoMatch)
}

func (a *VolumeAction) Delete() error {
	return a.StorageVolDelete(*a.id, libvirt.StorageVolDeleteNormal)
}

func (a *VolumeAction) CleanupSource() error {
	if err := a.source.Close(); err != nil {
		return err
	}
	return os.Remove(a.source.Name())
}
