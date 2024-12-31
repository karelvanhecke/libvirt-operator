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
	"bytes"
	"encoding/json"
	"fmt"
	"runtime/debug"

	"gopkg.in/yaml.v3"
)

var (
	version = ""
)

type Info struct {
	Version       string `yaml:"version" json:"version"`
	GitCommit     string `yaml:"gitCommit" json:"gitCommit"`
	GitCommitDate string `yaml:"gitCommitDate" json:"gitCommitDate"`
	GitTreeState  string `yaml:"gitTreeState" json:"gitTreeState"`
	GoVersion     string `yaml:"goVersion" json:"goVersion"`
	Compiler      string `yaml:"compiler" json:"compiler"`
	Platform      string `yaml:"platform" json:"platform"`
}

func NewInfo() *Info {
	i := &Info{}

	var os, arch string
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return i
	}

	for _, v := range bi.Settings {
		switch v.Key {
		case "vcs.revision":
			i.GitCommit = v.Value
		case "vcs.time":
			i.GitCommitDate = v.Value
		case "vcs.modified":
			if v.Value == "true" {
				i.GitTreeState = "dirty"
			} else {
				i.GitTreeState = "clean"
			}
		case "GOOS":
			os = v.Value
		case "GOARCH":
			arch = v.Value
		case "-compiler":
			i.Compiler = v.Value
		}
	}
	if version == "" {
		i.Version = "sha-" + i.GitCommit
	} else {
		i.Version = version
	}
	i.GoVersion = bi.GoVersion
	i.Platform = os + "/" + arch

	return i
}

func (i *Info) String() string {
	return fmt.Sprintf("version: %s", i.Version)
}

func (i *Info) JSON() (string, error) {
	s, err := json.MarshalIndent(i, "", "")
	if err != nil {
		return "", err
	}
	return string(s), nil
}

func (i *Info) YAML() (string, error) {
	buf := bytes.NewBuffer([]byte{})
	enc := yaml.NewEncoder(buf)
	enc.SetIndent(2)
	err := enc.Encode(i)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}
