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

package cmd

import (
	"fmt"

	"github.com/karelvanhecke/libvirt-operator/internal/version"
	"github.com/spf13/cobra"
)

func newVersionCmd() *cobra.Command {
	var output string

	cmd := &cobra.Command{
		Use:   "version",
		Short: "Show version info",
		Run: func(_ *cobra.Command, _ []string) {
			runVersion(output)
		},
	}

	cmd.Flags().StringVarP(&output, "output", "o", "", "set the version output format")

	return cmd
}

func runVersion(output string) {
	i := version.NewInfo()

	var o string

	switch output {
	case "json":
		info, err := i.JSON()
		if err != nil {
			o = err.Error()
			break
		}
		o = info
	case "yaml":
		info, err := i.YAML()
		if err != nil {
			o = err.Error()
			break
		}
		o = info
	default:
		o = i.String()
	}

	fmt.Println(o)
}
