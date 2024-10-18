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

package host

import (
	"context"
	"sync"
	"time"

	"github.com/digitalocean/go-libvirt"
	"github.com/digitalocean/go-libvirt/socket"
	"github.com/karelvanhecke/libvirt-operator/internal/store"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/uuid"
	ctrl "sigs.k8s.io/controller-runtime"
)

type Entry struct {
	version  string
	client   *libvirt.Libvirt
	sessions map[types.UID]struct{}
	mon      chan struct{}
}

type Store struct {
	mu      sync.Mutex
	entries map[types.UID]store.HostEntry
}

func NewStore() *Store {
	return &Store{entries: make(map[types.UID]store.HostEntry)}
}

func (s *Store) Entry(version string, dialer socket.Dialer) store.HostEntry {
	return &Entry{
		version: version,
		client:  libvirt.NewWithDialer(dialer),
	}
}

func (s *Store) Register(ctx context.Context, uid types.UID, entry store.HostEntry) {
	s.mu.Lock()
	defer s.mu.Unlock()

	oldEntry, found := s.entries[uid]

	go entry.StartMon(ctx, uid)

	s.entries[uid] = entry

	if found {
		go oldEntry.EndMon(ctx, uid)
	}
}

func (s *Store) Deregister(ctx context.Context, uid types.UID) {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry, found := s.entries[uid]
	if found {
		delete(s.entries, uid)
		go entry.EndMon(ctx, uid)
	}
}

func (s *Store) Lookup(uid types.UID) (entry store.HostEntry, found bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry, found = s.entries[uid]
	return
}

func (e *Entry) Session() (client *libvirt.Libvirt, end func()) {
	if e.sessions == nil {
		e.sessions = make(map[types.UID]struct{})
	}

	sessionID := uuid.NewUUID()
	end = func() {
		delete(e.sessions, sessionID)
	}
	e.sessions[sessionID] = struct{}{}
	return e.client, end
}

func (e *Entry) Version() string {
	return e.version
}

func (e *Entry) StartMon(ctx context.Context, uid types.UID) {
	e.mon = make(chan struct{})
	for {
		if e.client != nil {
			select {
			case <-e.client.Disconnected():
				if err := e.client.Connect(); err != nil {
					ctrl.LoggerFrom(ctx).Error(err, "could not reconnect client", "host", uid)
				}
			case <-ctx.Done():
				return
			case <-e.mon:
				return
			}
		}
	}
}

func (e *Entry) EndMon(ctx context.Context, uid types.UID) {
	e.mon <- struct{}{}
	for {
		if e.client != nil && e.client.IsConnected() {
			if len(e.sessions) > 0 {
				time.Sleep(1 * time.Minute)
				continue
			}
			if err := e.client.Disconnect(); err != nil {
				ctrl.LoggerFrom(ctx).Error(err, "failed to disconnect client", "host", uid)
			}
		}
		return
	}
}
