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

package probe

import (
	"github.com/digitalocean/go-libvirt"
	"github.com/karelvanhecke/libvirt-operator/internal/host"
	"github.com/karelvanhecke/libvirt-operator/internal/util"
)

type NodeProbe struct {
	cpu         int32
	memoryTotal uint64
	memoryFree  uint64
}

func NewNodeProbe(client host.Client) (*NodeProbe, error) {
	p := &NodeProbe{}

	_, _, cpu, _, _, _, _, _, err := client.NodeGetInfo()
	if err != nil {
		return nil, err
	}
	p.cpu = cpu

	memStats, _, err := client.NodeGetMemoryStats(4, int32(libvirt.NodeMemoryStatsAllCells), 0)
	if err != nil {
		return nil, err
	}

	for _, memStat := range memStats {
		switch memStat.Field {
		case libvirt.NodeMemoryStatsTotal:
			p.memoryTotal = util.ConvertToBytes(memStat.Value, "KB")
		case libvirt.NodeMemoryStatsFree:
			p.memoryFree = util.ConvertToBytes(memStat.Value, "KB")
		case libvirt.NodeMemoryStatsBuffers:
			p.memoryFree += util.ConvertToBytes(memStat.Value, "KB")
		case libvirt.NodeMemoryStatsCached:
			p.memoryFree += util.ConvertToBytes(memStat.Value, "KB")
		}
	}

	return p, nil
}

func (p *NodeProbe) CPU() int32 {
	return p.cpu
}

func (p *NodeProbe) MemoryTotal() uint64 {
	return p.memoryTotal
}

func (p *NodeProbe) MemoryFree() uint64 {
	return p.memoryFree
}
