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
	f.WithPool(&libvirtxml.StoragePool{
		Name: name,
	}, int32(libvirt.StoragePoolRunning), nil)

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
	f.WithPool(&libvirtxml.StoragePool{
		Name: pool,
	}, int32(libvirt.StoragePoolRunning), []*libvirtxml.StorageVolume{
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
	f.WithPool(&libvirtxml.StoragePool{
		Name: pool,
	}, int32(libvirt.StoragePoolRunning), nil)

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

	f.WithPool(&libvirtxml.StoragePool{
		Name: pool,
	}, int32(libvirt.StoragePoolRunning), []*libvirtxml.StorageVolume{{Name: name}})

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

	f.WithPool(&libvirtxml.StoragePool{Name: pool}, int32(libvirt.StoragePoolRunning), nil)

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

	f.WithPool(&libvirtxml.StoragePool{Name: pool}, int32(libvirt.StoragePoolRunning), []*libvirtxml.StorageVolume{v})

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
	f.WithPool(&libvirtxml.StoragePool{Name: pool}, int32(libvirt.StoragePoolRunning), []*libvirtxml.StorageVolume{v})

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
	f.WithPool(&libvirtxml.StoragePool{Name: pool}, int32(libvirt.StoragePoolRunning), []*libvirtxml.StorageVolume{v})

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
	f.WithPool(&libvirtxml.StoragePool{Name: pool}, int32(libvirt.StoragePoolRunning), []*libvirtxml.StorageVolume{{Name: name}})

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

func TestDomainDefineXML(t *testing.T) {
	f := fake.New()

	x := &libvirtxml.Domain{
		Name: "fake-domain",
	}

	s, err := x.Marshal()
	if err != nil {
		t.Fatal(err)
	}

	d, err := f.DomainDefineXML(s)
	if err != nil {
		t.Fail()
	}
	if d.Name != x.Name {
		t.Fail()
	}
}

func TestDomainCreate(t *testing.T) {
	f := fake.New()

	name := "fake-domain"
	d := libvirt.Domain{Name: name}

	f.WithDomain(&libvirtxml.Domain{Name: name}, int32(libvirt.DomainShutoff))

	if err := f.DomainCreate(d); err != nil {
		t.Fail()
	}

	state, reason, err := f.DomainGetState(d, 0)
	if err != nil {
		t.Fatal(err)
	}

	if state != int32(libvirt.DomainRunning) || reason != int32(libvirt.DomainRunningBooted) {
		t.Fail()
	}
}

func TestDomainShutdown(t *testing.T) {
	f := fake.New()

	name := "fake-domain"
	d := libvirt.Domain{Name: name}

	f.WithDomain(&libvirtxml.Domain{Name: name}, int32(libvirt.DomainRunning))

	if err := f.DomainShutdown(d); err != nil {
		t.Fail()
	}

	state, reason, err := f.DomainGetState(d, 0)
	if err != nil {
		t.Fatal(err)
	}

	if state != int32(libvirt.DomainShutoff) || reason != int32(libvirt.DomainShutoffShutdown) {
		t.Fail()
	}
}

func TestDomainDefineXMLAlreadyExists(t *testing.T) {
	f := fake.New()

	d := &libvirtxml.Domain{
		Name: "fake-domain",
	}

	f.WithDomain(d, 0)

	s, err := d.Marshal()
	if err != nil {
		t.Fatal(err)
	}

	_, err = f.DomainDefineXML(s)
	if err == nil || err.Error() != fake.ErrDomainAlreadyExist {
		t.Fail()
	}
}

func TestDomainCreateAlreadyRunning(t *testing.T) {
	f := fake.New()

	name := "fake-domain"
	d := libvirt.Domain{Name: name}

	f.WithDomain(&libvirtxml.Domain{Name: name}, int32(libvirt.DomainRunning))

	err := f.DomainCreate(d)
	if err == nil || err.Error() != fake.ErrDomainAlreadyRunning {
		t.Fail()
	}
}

func TestDomainCreateAlreadyShutoff(t *testing.T) {
	f := fake.New()

	name := "fake-domain"
	d := libvirt.Domain{Name: name}

	f.WithDomain(&libvirtxml.Domain{Name: name}, int32(libvirt.DomainShutoff))

	err := f.DomainShutdown(d)
	if err == nil || err.Error() != fake.ErrDomainAlreadyShutoff {
		t.Fail()
	}
}

func TestDomainLookupByName(t *testing.T) {
	f := fake.New()

	name := "fake-domain"
	d := libvirt.Domain{Name: name}

	f.WithDomain(&libvirtxml.Domain{Name: name}, int32(libvirt.DomainShutoff))

	l, err := f.DomainLookupByName(name)
	if err != nil {
		t.Fail()
	}
	if l.Name != d.Name {
		t.Fail()
	}
}

func TestDomainLookupByNameNotFound(t *testing.T) {
	f := fake.New()

	name := "fake-domain"

	_, err := f.DomainLookupByName(name)
	if err == nil {
		t.Fail()
	}
	if err.Error() != fake.ErrDomainNotExist {
		t.Fail()
	}
	e, ok := err.(libvirt.Error)
	if !ok {
		t.Fail()
	}
	if e.Code != uint32(libvirt.ErrNoDomain) {
		t.Fail()
	}
}

func TestDomainGetXMLDesc(t *testing.T) {
	f := fake.New()

	name := "fake-domain"

	d := &libvirtxml.Domain{Name: name, VCPU: &libvirtxml.DomainVCPU{Value: 1}}
	f.WithDomain(d, int32(libvirt.DomainShutoff))

	s, err := f.DomainGetXMLDesc(libvirt.Domain{Name: name}, 0)
	if err != nil {
		t.Fail()
	}
	x := &libvirtxml.Domain{}
	if err := x.Unmarshal(s); err != nil {
		t.Fatal(err)
	}
	if x.Name != d.Name || x.VCPU.Value != d.VCPU.Value {
		t.Fail()
	}
}

func TestNodeDeviceGetXMLDesc(t *testing.T) {
	f := fake.New()

	name := "fake-nodedev"

	f.WithNodeDev(&libvirtxml.NodeDevice{Name: name}, 1)

	xml, err := f.NodeDeviceGetXMLDesc(name, 0)
	if err != nil {
		t.Fail()
	}

	dev := &libvirtxml.NodeDevice{}
	if err := dev.Unmarshal(xml); err != nil {
		t.Fail()
	}
	if dev.Name != name {
		t.Fail()
	}
}

func TestNetworkIsActive(t *testing.T) {
	f := fake.New()

	name := "fake-network"

	f.WithNetwork(&libvirtxml.Network{Name: name}, 1)
	active, err := f.NetworkIsActive(libvirt.Network{Name: name})
	if err != nil {
		t.Fail()
	}
	if active != 1 {
		t.Fail()
	}
}

func TestNodeDeviceIsActive(t *testing.T) {
	f := fake.New()

	name := "fake-noddev"

	f.WithNodeDev(&libvirtxml.NodeDevice{Name: name}, 1)
	active, err := f.NodeDeviceIsActive(name)
	if err != nil {
		t.Fail()
	}
	if active != 1 {
		t.Fail()
	}
}

func TestStoragePoolIsActive(t *testing.T) {
	f := fake.New()

	name := "fake-noddev"

	f.WithPool(&libvirtxml.StoragePool{Name: name}, 1, nil)
	active, err := f.StoragePoolIsActive(libvirt.StoragePool{Name: name})
	if err != nil {
		t.Fail()
	}
	if active != 1 {
		t.Fail()
	}
}

func TestStoragePoolGetInfo(t *testing.T) {
	f := fake.New()

	name := "fake-pool"
	f.WithPool(&libvirtxml.StoragePool{Name: name,
		Capacity:   &libvirtxml.StoragePoolSize{Unit: "bytes", Value: 2},
		Allocation: &libvirtxml.StoragePoolSize{Unit: "bytes", Value: 1},
		Available:  &libvirtxml.StoragePoolSize{Unit: "bytes", Value: 1}},
		int32(libvirt.StoragePoolRunning), nil)

	state, cap, alloc, avail, err := f.StoragePoolGetInfo(libvirt.StoragePool{Name: name})
	if err != nil {
		t.Fail()
	}
	if state != uint8(libvirt.StoragePoolRunning) {
		t.Fail()
	}
	if cap-alloc != avail {
		t.Fail()
	}
}

func TestNetworkLookupByName(t *testing.T) {
	f := fake.New()

	name := "fake-network"
	f.WithNetwork(&libvirtxml.Network{Name: name}, 0)

	n, err := f.NetworkLookupByName(name)
	if err != nil {
		t.Fail()
	}
	if n.Name != name {
		t.Fail()
	}
}

func TestNodeDeviceLookupByName(t *testing.T) {
	f := fake.New()

	name := "fake-nodedev"
	f.WithNodeDev(&libvirtxml.NodeDevice{Name: name}, 0)

	n, err := f.NodeDeviceLookupByName(name)
	if err != nil {
		t.Fail()
	}
	if n.Name != name {
		t.Fail()
	}
}
