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

package auth

import (
	"context"
	"os"
	"path/filepath"
	"sync"

	"github.com/karelvanhecke/libvirt-operator/internal/store"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

type File struct {
	name string
	data []byte
}

type Entry struct {
	path    string
	version string
}

type Store struct {
	mu      sync.Mutex
	entries map[types.UID]store.AuthEntry
}

func NewStore() *Store {
	return &Store{entries: make(map[types.UID]store.AuthEntry)}
}

func (s *Store) Entry(version string) store.AuthEntry {
	return &Entry{
		version: version,
	}
}

func (s *Store) File(name string, data []byte) store.AuthFile {
	return &File{
		name: name,
		data: data,
	}
}

func (s *Store) Register(ctx context.Context, uid types.UID, entry store.AuthEntry, files []store.AuthFile) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	oldEntry, found := s.entries[uid]

	dir, err := os.MkdirTemp("", string(uid)+".")
	if err != nil {
		return err
	}

	for _, file := range files {
		path := filepath.Join(dir, file.Name())
		f, err := os.Create(path)
		if err != nil {
			return err
		}
		if _, err := f.Write(file.Data()); err != nil {
			return err
		}
		if err := f.Close(); err != nil {
			ctrl.LoggerFrom(ctx).Error(err, "could not close file", "file", path)
		}
	}

	if !found {
		entry.SetPath(filepath.Join(os.TempDir(), string(uid)))

		if err := os.Symlink(dir, entry.GetPath()); err != nil {
			return err
		}

	} else {
		entry.SetPath(oldEntry.GetPath())
		oldDir, err := filepath.EvalSymlinks(entry.GetPath())
		if err != nil {
			return err
		}
		if err := os.Symlink(dir, entry.GetPath()+".tmp"); err != nil {
			return err
		}
		if err := os.Rename(entry.GetPath()+".tmp", entry.GetPath()); err != nil {
			return err
		}
		if err := os.RemoveAll(oldDir); err != nil {
			ctrl.LoggerFrom(ctx).Error(err, "failed to cleanup old auth dir", "auth", uid)
		}
	}

	s.entries[uid] = entry
	return nil
}

func (s *Store) Deregister(ctx context.Context, uid types.UID) {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry, found := s.entries[uid]
	if found {
		delete(s.entries, uid)
		if err := os.RemoveAll(entry.GetPath()); err != nil {
			ctrl.LoggerFrom(ctx).Error(err, "failed to cleanup auth dir", "auth", uid)
		}
	}
}

func (s *Store) Lookup(uid types.UID) (auth store.AuthEntry, found bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	auth, found = s.entries[uid]
	return
}

func (e *Entry) SetPath(path string) {
	e.path = path
}

func (e *Entry) GetPath() string {
	return e.path
}

func (e *Entry) Version() string {
	return e.version
}

func (f *File) Name() string {
	return f.name
}

func (f *File) Data() []byte {
	return f.data
}
