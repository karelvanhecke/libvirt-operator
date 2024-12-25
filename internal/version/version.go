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

package version

import (
	"fmt"
	"runtime"
)

var (
	version   = "dev"
	commit    = "N/A"
	buildDate = "N/A"
)

func Info() {
	fmt.Printf("version: %s\ncommit: %s\nbuild date: %s\ngo version: %s\nplatform: %s\n",
		version,
		commit,
		buildDate,
		runtime.Version(),
		runtime.GOOS+"/"+runtime.GOARCH)
}
