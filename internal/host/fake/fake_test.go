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

package fake_test

import (
	"strings"
	"testing"
	"time"

	"github.com/digitalocean/go-libvirt"
	"github.com/karelvanhecke/libvirt-operator/internal/host/fake"
	"libvirt.org/go/libvirtxml"
)

func TestConnectSuccess(t *testing.T) {
	f := fake.New()

	if err := f.Connect(); err != nil {
		t.Fail()
	}
}

func TestConnectFail(t *testing.T) {
	f := fake.New()

	f.WithConnectFail()
	if err := f.Connect(); err == nil {
		t.Fail()
	}
}

func TestDisconnectSuccess(t *testing.T) {
	f := fake.New()

	if err := f.Connect(); err != nil {
		t.Fatal(err.Error())
	}

	if err := f.Disconnect(); err != nil {
		t.Fail()
	}
}

func TestDisconnectFail(t *testing.T) {
	f := fake.New()

	if err := f.Connect(); err != nil {
		t.Fatal(err.Error())
	}

	f.WithDisconnectFail()

	if err := f.Disconnect(); err == nil {
		t.Fail()
	}
}

func TestDisconnected(t *testing.T) {
	f := fake.New()

	if err := f.Connect(); err != nil {
		t.Fatal(err.Error())
	}

	c := make(chan struct{})

	go func() {
		time.Sleep(5 * time.Second)
		c <- struct{}{}
	}()

	go func() {
		time.Sleep(3 * time.Second)
		if err := f.Disconnect(); err != nil {
			t.Log(err.Error())
		}
	}()

	select {
	case <-f.Disconnected():
	case <-c:
		t.Fail()
	}
}

func TestIsConnected(t *testing.T) {
	f := fake.New()

	if err := f.Connect(); err != nil {
		t.Fatal(err.Error())
	}

	if !f.IsConnected() {
		t.Fail()
	}
}

func TestIsNotConnected(t *testing.T) {
	f := fake.New()

	if f.IsConnected() {
		t.Fail()
	}
}

func TestStoragePoolLookupByName(t *testing.T) {
	f := fake.New()

	name := "fake-pool"
	f.WithPool(libvirtxml.StoragePool{
		Name: name,
	}, nil)

	p, err := f.StoragePoolLookupByName(name)
	if err != nil {
		t.Fail()
	}

	if p.Name != name {
		t.Fail()
	}
}

func TestStoragePoolNotFound(t *testing.T) {
	f := fake.New()

	if _, err := f.StoragePoolLookupByName("fake-pool"); err != nil {
		if err.Error() != fake.ErrPoolNotExist {
			t.Fail()
		}
	}
}

func TestStorageVolLookupByName(t *testing.T) {
	f := fake.New()

	pool := "fake-pool"
	name := "fake-volume"
	f.WithPool(libvirtxml.StoragePool{
		Name: pool,
	}, []*libvirtxml.StorageVolume{
		{
			Name: name,
		},
	})

	v, err := f.StorageVolLookupByName(libvirt.StoragePool{Name: pool}, name)
	if err != nil {
		t.Fail()
	}

	if v.Name != name || v.Pool != pool {
		t.Fail()
	}
}

func TestStorageVolNotFound(t *testing.T) {
	f := fake.New()

	pool := "fake-pool"
	f.WithPool(libvirtxml.StoragePool{
		Name: pool,
	}, nil)

	_, err := f.StorageVolLookupByName(libvirt.StoragePool{Name: pool}, "fake-volume")
	if err == nil {
		t.Fail()
	}

	if err.Error() != fake.ErrVolumeNotExist {
		t.Fail()
	}
}

func TestStorageVolGetXMLDesc(t *testing.T) {
	f := fake.New()

	pool := "fake-pool"
	name := "fake-volume"

	f.WithPool(libvirtxml.StoragePool{
		Name: pool,
	}, []*libvirtxml.StorageVolume{{Name: name}})

	s, err := f.StorageVolGetXMLDesc(libvirt.StorageVol{Name: name, Pool: pool}, 0)
	if err != nil {
		t.Fail()
	}

	v := libvirtxml.StorageVolume{}
	if err := v.Unmarshal(s); err != nil {
		t.Fail()
	}

	if v.Name != name {
		t.Fail()
	}
}

func TestStorageVolCreateXML(t *testing.T) {
	f := fake.New()

	pool := "fake-pool"
	name := "fake-volume"

	f.WithPool(libvirtxml.StoragePool{Name: pool}, nil)

	v := libvirtxml.StorageVolume{
		Name: name,
	}
	s, err := v.Marshal()
	if err != nil {
		t.Fatal(err)
	}

	vol, err := f.StorageVolCreateXML(libvirt.StoragePool{Name: pool}, s, 0)
	if err != nil {
		t.Fail()
	}

	if vol.Name != name || vol.Pool != pool {
		t.Fail()
	}
}

func TestStorageVolAlreadyExists(t *testing.T) {
	f := fake.New()

	pool := "fake-pool"
	name := "fake-volume"

	v := &libvirtxml.StorageVolume{
		Name: name,
	}

	f.WithPool(libvirtxml.StoragePool{Name: pool}, []*libvirtxml.StorageVolume{v})

	s, err := v.Marshal()
	if err != nil {
		t.Fatal(err)
	}

	_, err = f.StorageVolCreateXML(libvirt.StoragePool{Name: pool}, s, 0)
	if err == nil {
		t.Fail()
	}
	if err.Error() != fake.ErrVolumeExists {
		t.Fail()
	}
}

func TestStorageVolUpload(t *testing.T) {
	f := fake.New()

	pool := "fake-pool"
	name := "fake-volume"
	content := "fake-content"

	v := &libvirtxml.StorageVolume{Name: name, Capacity: &libvirtxml.StorageVolumeSize{}}
	f.WithPool(libvirtxml.StoragePool{Name: pool}, []*libvirtxml.StorageVolume{v})

	if err := f.StorageVolUpload(libvirt.StorageVol{Name: name, Pool: pool}, strings.NewReader(content), 0, 0, 0); err != nil {
		t.Fail()
	}

	if uint64(len(content)) != v.Capacity.Value {
		t.Fail()
	}
}

func TestStorageVolResize(t *testing.T) {
	f := fake.New()

	pool := "fake-pool"
	name := "fake-volume"

	newCap := uint64(2)

	v := &libvirtxml.StorageVolume{Name: name, Capacity: &libvirtxml.StorageVolumeSize{Value: 1}}
	f.WithPool(libvirtxml.StoragePool{Name: pool}, []*libvirtxml.StorageVolume{v})

	if err := f.StorageVolResize(libvirt.StorageVol{Name: name, Pool: pool}, newCap, 0); err != nil {
		t.Fail()
	}

	if v.Capacity.Value != newCap {
		t.Fail()
	}
}

func TestStorageVolDelete(t *testing.T) {
	f := fake.New()

	pool := "fake-pool"
	name := "fake-volume"
	f.WithPool(libvirtxml.StoragePool{Name: pool}, []*libvirtxml.StorageVolume{{Name: name}})

	if err := f.StorageVolDelete(libvirt.StorageVol{Name: name, Pool: pool}, 0); err != nil {
		t.Fail()
	}

	_, err := f.StorageVolLookupByName(libvirt.StoragePool{Name: pool}, name)
	if err == nil {
		t.Fail()
	}

	if err.Error() != fake.ErrVolumeNotExist {
		t.Fail()
	}
}
