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

	"k8s.io/apimachinery/pkg/types"
)

type AuthFile interface {
	Name() string
	Data() []byte
}

type AuthEntry interface {
	SetPath(string)
	GetPath() string
	Version() string
}

type AuthStore interface {
	Entry(version string) AuthEntry
	File(name string, data []byte) AuthFile
	Register(ctx context.Context, uid types.UID, entry AuthEntry, files []AuthFile) error
	Deregister(ctx context.Context, uid types.UID)
	Lookup(uid types.UID) (auth AuthEntry, found bool)
}
