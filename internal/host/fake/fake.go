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
	"io"
	"slices"

	"github.com/digitalocean/go-libvirt"
	"libvirt.org/go/libvirtxml"
)

const (
	ErrConnectFailed     = "connect to host failed"
	ErrDisconnectFailed  = "disconnect from host failed"
	ErrPoolNotExist      = "pool does not exist"
	ErrUnsupportedDialer = "unsupported dialer"
)

type Fake struct {
	connectFail    bool
	disconnectFail bool
	disconnected   chan struct{}
	pools          []*Pool
}

func New() *Fake {
	f := &Fake{disconnected: make(chan struct{})}
	close(f.disconnected)
	return f
}

func (f *Fake) WithConnectFail() {
	f.connectFail = true
}

func (f *Fake) WithDisconnectFail() {
	f.disconnectFail = true
}

func (f *Fake) WithPool(pool libvirtxml.StoragePool, volumes []*libvirtxml.StorageVolume) {
	p := &Pool{
		xml:     pool,
		volumes: volumes,
	}
	f.pools = append(f.pools, p)
}

func (f *Fake) Connect() error {
	if f.connectFail {
		return errors.New(ErrConnectFailed)
	}

	f.disconnected = make(chan struct{})

	return nil
}

func (f *Fake) Disconnected() <-chan struct{} {
	return f.disconnected
}

func (f *Fake) Disconnect() error {
	if f.disconnectFail {
		return errors.New(ErrDisconnectFailed)
	}

	close(f.disconnected)
	return nil
}

func (f *Fake) IsConnected() bool {
	select {
	case <-f.disconnected:
		return false
	default:
		return true
	}
}

func (f *Fake) StoragePoolLookupByName(name string) (rPool libvirt.StoragePool, err error) {
	pool, err := f.getPoolByName(name)
	if err != nil {
		return libvirt.StoragePool{}, err
	}

	return libvirt.StoragePool{Name: pool.xml.Name}, nil
}

func (f *Fake) StorageVolLookupByName(pool libvirt.StoragePool, name string) (rVol libvirt.StorageVol, err error) {
	p, err := f.getPoolByName(pool.Name)
	if err != nil {
		return libvirt.StorageVol{}, err
	}

	v, err := p.getVolumeByName(name)
	if err != nil {
		return libvirt.StorageVol{}, err
	}

	return libvirt.StorageVol{
		Pool: pool.Name,
		Name: v.Name,
		Key:  v.Key,
	}, nil
}

func (f *Fake) StorageVolGetXMLDesc(vol libvirt.StorageVol, flags uint32) (rXML string, err error) {
	p, err := f.getPoolByName(vol.Pool)
	if err != nil {
		return "", err
	}
	v, err := p.getVolumeByName(vol.Name)
	if err != nil {
		return "", err
	}

	return v.Marshal()
}

func (f *Fake) StorageVolCreateXML(pool libvirt.StoragePool, xml string, flags libvirt.StorageVolCreateFlags) (rVol libvirt.StorageVol, err error) {
	p, err := f.getPoolByName(pool.Name)
	if err != nil {
		return libvirt.StorageVol{}, err
	}

	v, err := p.createVol(xml)
	if err != nil {
		return libvirt.StorageVol{}, err
	}

	return libvirt.StorageVol{Pool: p.xml.Name, Name: v.Name, Key: v.Key}, nil
}

func (f *Fake) StorageVolUpload(vol libvirt.StorageVol, outStream io.Reader, offset uint64, length uint64, flags libvirt.StorageVolUploadFlags) (err error) {
	p, err := f.getPoolByName(vol.Pool)
	if err != nil {
		return err
	}
	v, err := p.getVolumeByName(vol.Name)
	if err != nil {
		return err
	}

	bytes, err := io.ReadAll(outStream)
	if err != nil {
		return err
	}

	v.Capacity.Unit = "bytes"
	v.Capacity.Value = uint64(len(bytes))

	return nil
}

func (f *Fake) StorageVolResize(vol libvirt.StorageVol, capacity uint64, flags libvirt.StorageVolResizeFlags) (err error) {
	p, err := f.getPoolByName(vol.Pool)
	if err != nil {
		return err
	}
	v, err := p.getVolumeByName(vol.Name)
	if err != nil {
		return err
	}

	v.Capacity.Unit = "bytes"
	v.Capacity.Value = uint64(capacity)

	return nil
}

func (f *Fake) StorageVolDelete(vol libvirt.StorageVol, flags libvirt.StorageVolDeleteFlags) (err error) {
	p, err := f.getPoolByName(vol.Pool)
	if err != nil {
		return err
	}

	return p.deleteVol(vol.Name)
}

func (f *Fake) getPoolByName(name string) (*Pool, error) {
	i := slices.IndexFunc(f.pools, func(pool *Pool) bool { return pool.xml.Name == name })
	if i == -1 {
		return nil, errors.New(ErrPoolNotExist)
	}
	return f.pools[i], nil
}
