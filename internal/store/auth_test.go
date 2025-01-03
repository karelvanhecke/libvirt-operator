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
	"os"
	"path/filepath"
	"testing"

	"github.com/karelvanhecke/libvirt-operator/internal/store"
	"k8s.io/apimachinery/pkg/types"
)

func TestAuthStore(t *testing.T) {
	store.TmpDir = t.TempDir()
	s := store.NewAuthStore()

	ctx := context.Background()
	uid := types.UID("e9454b3d-fb8f-4aa9-b9da-c13d858656e9")
	fileName := "fake-credential"

	var entry *store.AuthEntry
	for i := range 2 {
		gen := int64(i)

		fileContent := "fake-content"
		files := []*store.File{store.NewFile(fileName, []byte(fileContent))}

		oldPath := ""
		oldDir := ""
		if entry != nil {
			oldPath = entry.GetPath()
			p, err := filepath.EvalSymlinks(oldPath)
			if err != nil {
				t.Fatal(err)
			}
			oldDir = p
		}

		if err := s.Register(ctx, uid, gen, files); err != nil {
			t.Fail()
		}

		e, ok := s.Lookup(uid)
		if !ok {
			t.Fail()
		}
		if e.Generation() != gen {
			t.Fail()
		}
		entry = e

		path := entry.GetPath()
		data, err := os.ReadFile(filepath.Join(path, fileName))
		if err != nil {
			t.Fail()
		}
		if string(data) != string(fileContent) {
			t.Fail()
		}

		if oldPath != "" {
			if path != oldPath {
				t.Fail()
			}
		}

		if oldDir != "" {
			_, err = os.Stat(oldDir)
			if err == nil {
				t.Fail()
			}
			if !os.IsNotExist(err) {
				t.Fail()
			}
		}
	}

	link := entry.GetPath()
	target, err := filepath.EvalSymlinks(link)
	if err != nil {
		t.Fatal()
	}

	s.Deregister(ctx, uid)

	for _, path := range []string{link, target} {
		_, err := os.Stat(path)
		if err == nil {
			t.Fail()
		}
		if !os.IsNotExist(err) {
			t.Fail()
		}
	}
}
