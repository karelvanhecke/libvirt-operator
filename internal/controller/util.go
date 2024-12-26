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

package controller

import (
	"slices"

	"github.com/karelvanhecke/libvirt-operator/api/v1alpha1"
)

func poolAvailable(pools []v1alpha1.Pool, preferredPool *string) (string, bool) {
	var pool string
	if p := preferredPool; p != nil {
		if i := slices.IndexFunc(pools, func(hp v1alpha1.Pool) bool { return hp.Name == *p }); i != -1 {
			pool = *p
		} else {
			return "", false
		}
	} else {
		if i := slices.IndexFunc(pools, func(hp v1alpha1.Pool) bool {
			if hp.Default != nil {
				return *hp.Default
			}
			return false
		}); i != -1 {
			pool = pools[i].Name
		} else {
			pool = pools[0].Name
		}
	}
	return pool, true
}
