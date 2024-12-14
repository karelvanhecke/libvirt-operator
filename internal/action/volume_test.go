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

package action_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/digitalocean/go-libvirt"
	"github.com/karelvanhecke/libvirt-operator/internal/action"
	"github.com/karelvanhecke/libvirt-operator/internal/host/fake"
	"libvirt.org/go/libvirtxml"
)

func TestVolumeExists(t *testing.T) {
	f := fake.New()

	pool := "fake-pool"
	volume := "fake-volume"
	f.WithPool(libvirtxml.StoragePool{Name: pool}, []*libvirtxml.StorageVolume{{Name: volume}})

	a, err := action.NewVolumeAction(f, volume, pool, &libvirtxml.StorageVolumeSize{}, &libvirtxml.StorageVolumeTargetFormat{})
	if err != nil {
		t.Fatal()
	}
	if !a.State() {
		t.Fail()
	}
}

func TestVolumeNotExists(t *testing.T) {
	f := fake.New()

	pool := "fake-pool"
	f.WithPool(libvirtxml.StoragePool{Name: pool}, nil)

	a, err := action.NewVolumeAction(f, "fake-volume", pool, &libvirtxml.StorageVolumeSize{}, &libvirtxml.StorageVolumeTargetFormat{})
	if err != nil {
		t.Fatal()
	}
	if a.State() {
		t.Fail()
	}
}

func TestVolumeCreate(t *testing.T) {
	f := fake.New()

	pool := "fake-pool"
	volume := "fake-volume"
	format := "qcow2"
	unit := "bytes"
	value := uint64(1000)
	f.WithPool(libvirtxml.StoragePool{Name: pool}, nil)

	a, err := action.NewVolumeAction(f, volume, pool, &libvirtxml.StorageVolumeSize{Unit: unit, Value: value}, &libvirtxml.StorageVolumeTargetFormat{Type: format})
	if err != nil {
		t.Fatal()
	}

	if err := a.Create(); err != nil {
		t.Fail()
	}

	x, err := f.StorageVolGetXMLDesc(libvirt.StorageVol{Name: volume, Pool: pool}, 0)
	if err != nil {
		t.Fail()
	}
	v := libvirtxml.StorageVolume{}
	if err := v.Unmarshal(x); err != nil {
		t.Fatal()
	}

	if v.Name != volume ||
		v.Target.Format.Type != format ||
		v.Capacity.Unit != unit ||
		v.Capacity.Value != value {
		t.Fail()
	}
}

func TestVolumeCreateWithBackingStore(t *testing.T) {
	f := fake.New()

	pool := "fake-pool"
	volume := "fake-volume"
	format := "qcow2"
	unit := "bytes"
	value := uint64(1000)
	path := "fake-path"
	backingStore := "fake-backingstore"
	f.WithPool(libvirtxml.StoragePool{Name: pool}, []*libvirtxml.StorageVolume{
		{
			Name: backingStore,
			Capacity: &libvirtxml.StorageVolumeSize{
				Unit:  unit,
				Value: value,
			},
			Target: &libvirtxml.StorageVolumeTarget{
				Format: &libvirtxml.StorageVolumeTargetFormat{
					Type: format,
				},
				Path: path,
			},
		},
	})

	a, err := action.NewVolumeAction(f, volume, pool, nil, &libvirtxml.StorageVolumeTargetFormat{Type: format})
	if err != nil {
		t.Fatal()
	}

	if err := a.WithBackingStore(backingStore); err != nil {
		t.Fail()
	}

	if err := a.Create(); err != nil {
		t.Fail()
	}

	x, err := f.StorageVolGetXMLDesc(libvirt.StorageVol{Name: volume, Pool: pool}, 0)
	if err != nil {
		t.Fail()
	}
	v := libvirtxml.StorageVolume{}
	if err := v.Unmarshal(x); err != nil {
		t.Fatal()
	}

	if v.Name != volume ||
		v.Target.Format.Type != format ||
		v.Capacity.Unit != unit ||
		v.Capacity.Value != value ||
		v.BackingStore.Path != path {
		t.Fail()
	}
}

func TestVolumeCreateWithSource(t *testing.T) {
	f := fake.New()

	pool := "fake-pool"
	volume := "fake-volume"
	format := "qcow2"
	f.WithPool(libvirtxml.StoragePool{Name: pool}, nil)

	source := []byte("fake-source")
	checksum := "sha256:bb92dcbbdf410e3bd2e139fc2cb7c9ff4e490cfe3aa968779615324669e44152"
	server := httptest.NewServer(
		http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				_, err := w.Write(source)
				if err != nil {
					t.Fatal()
				}
			},
		),
	)
	defer server.Close()

	a, err := action.NewVolumeAction(f, volume, pool, &libvirtxml.StorageVolumeSize{}, &libvirtxml.StorageVolumeTargetFormat{Type: format})
	if err != nil {
		t.Fatal()
	}

	if err := a.WithSource(server.URL, &checksum); err != nil {
		t.Fail()
	}

	if err := a.Create(); err != nil {
		t.Fail()
	}

	x, err := f.StorageVolGetXMLDesc(libvirt.StorageVol{Name: volume, Pool: pool}, 0)
	if err != nil {
		t.Fail()
	}
	v := libvirtxml.StorageVolume{}
	if err := v.Unmarshal(x); err != nil {
		t.Fatal()
	}

	if v.Name != volume ||
		v.Target.Format.Type != format ||
		v.Capacity.Unit != "bytes" ||
		v.Capacity.Value != uint64(len(source)) {
		t.Fail()
	}
}

func TestVolumeUpdate(t *testing.T) {
	f := fake.New()

	pool := "fake-pool"
	volume := "fake-volume"
	unit := "bytes"
	value := uint64(1000)
	newValue := uint64(2000)
	f.WithPool(libvirtxml.StoragePool{Name: pool}, []*libvirtxml.StorageVolume{
		{
			Name:     volume,
			Capacity: &libvirtxml.StorageVolumeSize{Unit: unit, Value: value},
		},
	})

	a, err := action.NewVolumeAction(f, volume, pool, &libvirtxml.StorageVolumeSize{Unit: unit, Value: newValue}, nil)
	if err != nil {
		t.Fatal()
	}

	if err := a.Update(); err != nil {
		t.Fail()
	}

	x, err := f.StorageVolGetXMLDesc(libvirt.StorageVol{Name: volume, Pool: pool}, 0)
	if err != nil {
		t.Fail()
	}
	v := libvirtxml.StorageVolume{}
	if err := v.Unmarshal(x); err != nil {
		t.Fatal()
	}

	if v.Capacity.Unit != unit ||
		v.Capacity.Value != newValue {
		t.Fail()
	}
}

func TestVolumeDelete(t *testing.T) {
	f := fake.New()

	pool := "fake-pool"
	volume := "fake-volume"

	f.WithPool(libvirtxml.StoragePool{Name: pool}, []*libvirtxml.StorageVolume{{Name: volume}})

	a, err := action.NewVolumeAction(f, volume, pool, nil, nil)
	if err != nil {
		t.Fatal()
	}

	if err := a.Delete(); err != nil {
		t.Fail()
	}
}
