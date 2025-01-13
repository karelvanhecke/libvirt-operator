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

package store_test

import (
	"context"
	"testing"
	"time"

	"github.com/digitalocean/go-libvirt/socket"
	"github.com/digitalocean/go-libvirt/socket/dialers"
	"github.com/karelvanhecke/libvirt-operator/internal/host"
	"github.com/karelvanhecke/libvirt-operator/internal/host/fake"
	"github.com/karelvanhecke/libvirt-operator/internal/store"
	"k8s.io/apimachinery/pkg/types"
)

func init() {
	host.New = func(dialer socket.Dialer) host.Client { return fake.New() }
	store.WaitForSessionRetry = 100 * time.Millisecond
}

func TestHostStore(t *testing.T) {
	s := store.NewHostStore()

	ctx := context.Background()
	uid := types.UID("e9454b3d-fb8f-4aa9-b9da-c13d858656e9")

	var entry *store.HostEntry
	for i := range 2 {
		gen := int64(i)
		s.Register(ctx, uid, gen, dialers.NewLocal())

		e, ok := s.Lookup(uid)
		if !ok {
			t.Fail()
		}
		if e.Generation() != gen {
			t.Fail()
		}
		entry = e
	}

	client, end, err := entry.Session()
	if err != nil {
		t.Fail()
	}

	time.Sleep(1 * time.Second)
	if !client.IsConnected() {
		t.Fail()
	}

	s.Deregister(ctx, uid)

	_, ok := s.Lookup(uid)
	if ok {
		t.Fail()
	}

	time.Sleep(1 * time.Second)
	if !client.IsConnected() {
		t.Fail()
	}

	end()

	time.Sleep(1 * time.Second)
	if client.IsConnected() {
		t.Fail()
	}
}
