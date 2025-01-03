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
	"os"
	"path/filepath"
	"sync"

	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

var (
	TmpDir = os.TempDir()
)

type File struct {
	name string
	data []byte
}

type AuthEntry struct {
	path       string
	generation int64
}

type AuthStore struct {
	mu      sync.Mutex
	entries map[types.UID]*AuthEntry
}

func NewAuthStore() *AuthStore {
	return &AuthStore{entries: make(map[types.UID]*AuthEntry)}
}

func NewFile(name string, data []byte) *File {
	return &File{
		name: name,
		data: data,
	}
}

func (s *AuthStore) Register(ctx context.Context, uid types.UID, generation int64, files []*File) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	oldEntry, found := s.entries[uid]

	dir, err := os.MkdirTemp(TmpDir, string(uid)+".")
	if err != nil {
		return err
	}

	for _, file := range files {
		path := filepath.Join(dir, file.name)
		f, err := os.Create(path)
		if err != nil {
			return err
		}
		if _, err := f.Write(file.data); err != nil {
			return err
		}
		if err := f.Close(); err != nil {
			ctrl.LoggerFrom(ctx).Error(err, "could not close file", "file", path)
		}
	}

	entry := &AuthEntry{
		generation: generation,
	}

	if !found {
		entry.path = filepath.Join(TmpDir, string(uid))
		if err := os.Symlink(dir, entry.GetPath()); err != nil {
			return err
		}
	} else {
		entry.path = oldEntry.GetPath()
		path := entry.GetPath()
		tmpPath := path + ".tmp"

		oldDir, err := filepath.EvalSymlinks(path)
		if err != nil {
			return err
		}

		if err := os.Symlink(dir, tmpPath); err != nil {
			return err
		}
		if err := os.Rename(tmpPath, path); err != nil {
			return err
		}
		if err := os.RemoveAll(oldDir); err != nil {
			ctrl.LoggerFrom(ctx).Error(err, "failed to cleanup old auth dir", "auth", uid)
		}
	}

	s.entries[uid] = entry
	return nil
}

func (s *AuthStore) Deregister(ctx context.Context, uid types.UID) {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry, found := s.entries[uid]
	link := entry.GetPath()
	target, err := filepath.EvalSymlinks(link)
	if err != nil {
		ctrl.LoggerFrom(ctx).Error(err, "failed to resolve link", "auth", uid)
	}
	if found {
		delete(s.entries, uid)
		if err := os.Remove(link); err != nil {
			ctrl.LoggerFrom(ctx).Error(err, "failed to cleanup auth link", "auth", uid)
		}
		if err := os.RemoveAll(target); err != nil {
			ctrl.LoggerFrom(ctx).Error(err, "failed to cleanup auth dir", "auth", uid)
		}
	}
}

func (s *AuthStore) Lookup(uid types.UID) (auth *AuthEntry, found bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	auth, found = s.entries[uid]
	return
}

func (e *AuthEntry) GetPath() string {
	return e.path
}

func (e *AuthEntry) Generation() int64 {
	return e.generation
}
