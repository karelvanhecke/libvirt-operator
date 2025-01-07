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

package store

import (
	"context"
	"sync"
	"time"

	"github.com/digitalocean/go-libvirt/socket"
	"github.com/karelvanhecke/libvirt-operator/internal/host"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/uuid"
	ctrl "sigs.k8s.io/controller-runtime"
)

var (
	WaitForSessionRetry = 1 * time.Minute
)

type HostEntry struct {
	mu         sync.Mutex
	generation int64
	client     host.Client
	sessions   map[types.UID]struct{}
	mon        chan struct{}
}

type HostStore struct {
	mu      sync.Mutex
	entries map[types.UID]*HostEntry
}

func NewHostStore() *HostStore {
	return &HostStore{entries: make(map[types.UID]*HostEntry)}
}

func (s *HostStore) Register(ctx context.Context, uid types.UID, generation int64, dialer socket.Dialer) {
	s.mu.Lock()
	defer s.mu.Unlock()

	oldEntry, found := s.entries[uid]

	entry := &HostEntry{
		generation: generation,
		client:     host.New(dialer),
	}

	if err := entry.client.Connect(); err != nil {
		ctrl.LoggerFrom(ctx).Error(err, "failed to connect host host")
	}

	go entry.startMon(ctx, uid)

	s.entries[uid] = entry

	if found {
		go oldEntry.endMon(ctx, uid)
	}
}

func (s *HostStore) Deregister(ctx context.Context, uid types.UID) {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry, found := s.entries[uid]
	if found {
		delete(s.entries, uid)
		go entry.endMon(ctx, uid)
	}
}

func (s *HostStore) Lookup(uid types.UID) (entry *HostEntry, found bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry, found = s.entries[uid]
	return
}

func (e *HostEntry) Session() (client host.Client, end func()) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.sessions == nil {
		e.sessions = make(map[types.UID]struct{})
	}

	sessionID := uuid.NewUUID()
	end = func() {
		e.mu.Lock()
		defer e.mu.Unlock()
		delete(e.sessions, sessionID)
	}
	e.sessions[sessionID] = struct{}{}
	return e.client, end
}

func (e *HostEntry) Generation() int64 {
	return e.generation
}

func (e *HostEntry) startMon(ctx context.Context, uid types.UID) {
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

func (e *HostEntry) endMon(ctx context.Context, uid types.UID) {
	e.mon <- struct{}{}
	for {
		if e.client != nil && e.client.IsConnected() {
			if len(e.sessions) > 0 {
				time.Sleep(WaitForSessionRetry)
				continue
			}
			if err := e.client.Disconnect(); err != nil {
				ctrl.LoggerFrom(ctx).Error(err, "failed to disconnect client", "host", uid)
			}
		}
		return
	}
}
