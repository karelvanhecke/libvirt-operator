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
	"libvirt.org/go/libvirtxml"
)

type HostProbe struct {
	arch        string
	cpus        uint
	memoryTotal uint64
	memoryFree  uint64
	numa        bool
}

func NewHostProbe(client host.Client) (*HostProbe, error) {
	p := &HostProbe{}

	capsData, err := client.Capabilities()
	if err != nil {
		return nil, err
	}
	caps := &libvirtxml.Caps{}
	if err := caps.Unmarshal(string(capsData)); err != nil {
		return nil, err
	}
	if cpu := caps.Host.CPU; cpu != nil {
		p.arch = cpu.Arch
	} else {
		p.arch = "unknown"
	}
	if topo := caps.Host.NUMA; topo != nil {
		if cells := topo.Cells; cells != nil {
			p.numa = cells.Num > 1
			for _, c := range cells.Cells {
				if cpus := c.CPUS; cpus != nil {
					p.cpus += cpus.Num
				}
			}
		}
	}

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

func (p *HostProbe) CPUS() uint {
	return p.cpus
}

func (p *HostProbe) MemoryTotal() uint64 {
	return p.memoryTotal
}

func (p *HostProbe) MemoryFree() uint64 {
	return p.memoryFree
}

func (p *HostProbe) Arch() string {
	return p.arch
}

func (p *HostProbe) NUMA() bool {
	return p.numa
}
