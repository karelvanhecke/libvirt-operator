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

	"github.com/digitalocean/go-libvirt"
	"github.com/digitalocean/go-libvirt/socket"
	"k8s.io/apimachinery/pkg/types"
)

type HostEntry interface {
	Session() (client *libvirt.Libvirt, end func())
	Version() string
	StartMon(ctx context.Context, uid types.UID)
	EndMon(ctx context.Context, uid types.UID)
}

type HostStore interface {
	Entry(version string, dialer socket.Dialer) HostEntry
	Register(ctx context.Context, uid types.UID, entry HostEntry)
	Deregister(ctx context.Context, uid types.UID)
	Lookup(uid types.UID) (entry HostEntry, found bool)
}
