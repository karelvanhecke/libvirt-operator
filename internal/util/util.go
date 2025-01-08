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

package util

import (
	"bytes"

	"gopkg.in/yaml.v3"
)

func ConvertToBytes(value uint64, unit string) uint64 {
	const B = 1
	const KB = 1000
	const KiB = 1024
	const MB = KB * 1000
	const MiB = KiB * 1024
	const GB = MB * 1000
	const GiB = MiB * 1024
	const TB = GB * 1000
	const TiB = GiB * 1024
	const PB = TB * 1000
	const PiB = TiB * 1024
	const EB = PB * 1000
	const EiB = PiB * 1024

	unitConversion := map[string]uint64{
		"":      B,
		"B":     B,
		"bytes": B,
		"KB":    KB,
		"K":     KiB,
		"KiB":   KiB,
		"MB":    MB,
		"M":     MiB,
		"MiB":   MiB,
		"GB":    GB,
		"G":     GiB,
		"GiB":   GiB,
		"TB":    TB,
		"T":     TiB,
		"TiB":   TiB,
		"PB":    PB,
		"P":     PiB,
		"PiB":   PiB,
		"EB":    EB,
		"E":     EiB,
		"EiB":   EiB,
	}

	return value * unitConversion[unit]
}

func LibvirtNamespacedName(namespace string, name string) string {
	return namespace + ":" + name
}

func Marshal(buffer *bytes.Buffer, data any) ([]byte, error) {
	enc := yaml.NewEncoder(buffer)
	enc.SetIndent(2)
	if err := enc.Encode(data); err != nil {
		return nil, err
	}
	if err := enc.Close(); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}
