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

package action

import (
	"errors"
	"slices"
	"strings"
	"time"

	"github.com/digitalocean/go-libvirt"
	"github.com/karelvanhecke/libvirt-operator/internal/host"
	"libvirt.org/go/libvirtxml"
)

const (
	ErrUnsupportedDiskDevice     = "disk device is not supported"
	ErrUnsupportedDiskSourceType = "disk source type is not supported"
	ErrShutdownTimedOut          = "shutdown timed out"
)

type DomainAction struct {
	host.Client
	name string

	id    *libvirt.Domain
	def   *libvirtxml.Domain
	state int32
}

func NewDomainAction(client host.Client, name string) (*DomainAction, error) {
	a := &DomainAction{
		Client: client,
		name:   name,
	}

	id, err := a.DomainLookupByName(a.name)
	if err != nil {
		if e, ok := err.(libvirt.Error); ok {
			if e.Code == uint32(libvirt.ErrNoDomain) {
				a.def = &libvirtxml.Domain{
					Type: "kvm",
					OS: &libvirtxml.DomainOS{
						Type:        &libvirtxml.DomainOSType{Type: "hvm", Machine: "q35"},
						BootDevices: []libvirtxml.DomainBootDevice{{Dev: "hd"}},
					},
					Features: &libvirtxml.DomainFeatureList{
						ACPI: &libvirtxml.DomainFeature{},
						APIC: &libvirtxml.DomainFeatureAPIC{},
					},
					Name:   a.name,
					VCPU:   &libvirtxml.DomainVCPU{},
					Memory: &libvirtxml.DomainMemory{},
					Devices: &libvirtxml.DomainDeviceList{
						Controllers: []libvirtxml.DomainController{
							{
								Type:   "scsi",
								Model:  "virtio-scsi",
								Driver: &libvirtxml.DomainControllerDriver{},
							},
						},
						RNGs: []libvirtxml.DomainRNG{{
							Model: "virtio",
							Backend: &libvirtxml.DomainRNGBackend{
								Random: &libvirtxml.DomainRNGBackendRandom{
									Device: "/dev/urandom",
								},
							},
						}},
						Serials: []libvirtxml.DomainSerial{
							{
								Target: &libvirtxml.DomainSerialTarget{},
							},
						},
						Consoles: []libvirtxml.DomainConsole{
							{
								TTY: "pty",
								Target: &libvirtxml.DomainConsoleTarget{
									Type: "virtio",
								},
							},
						},
						Channels: []libvirtxml.DomainChannel{
							{
								Source: &libvirtxml.DomainChardevSource{
									UNIX: &libvirtxml.DomainChardevSourceUNIX{},
								},
								Target: &libvirtxml.DomainChannelTarget{
									VirtIO: &libvirtxml.DomainChannelTargetVirtIO{
										Name: "org.qemu.guest_agent.0",
									},
								},
							},
						},
					},
					SecLabel: []libvirtxml.DomainSecLabel{
						{
							Type: "dynamic",
						},
					},
				}
				return a, nil
			}
		}
		return nil, err
	}

	a.id = &id

	state, _, err := a.DomainGetState(*a.id, 0)
	if err != nil {
		return nil, err
	}
	a.state = state

	xml, err := a.DomainGetXMLDesc(*a.id, 0)
	if err != nil {
		return nil, err
	}

	a.def = &libvirtxml.Domain{}
	err = a.def.Unmarshal(xml)
	if err != nil {
		return nil, err
	}

	return a, nil
}

func (a *DomainAction) State() (exists bool, shutoff bool) {
	return a.id != nil, a.state == int32(libvirt.DomainShutoff)
}

func (a *DomainAction) VCPU(value uint) {
	a.def.VCPU.Value = value
}

func (a *DomainAction) VCPUPlacement(placement string) {
	a.def.VCPU.Placement = placement
}

func (a *DomainAction) Memory(value uint, unit string) {
	a.def.Memory.Value = value
	a.def.Memory.Unit = unit
}

func (a *DomainAction) CPUMode(mode string) {
	if a.def.CPU == nil {
		a.def.CPU = &libvirtxml.DomainCPU{}
	}
	a.def.CPU.Mode = mode
}

func (a *DomainAction) CPUCacheMode(mode string) {
	if a.def.CPU == nil {
		a.def.CPU = &libvirtxml.DomainCPU{}
	}
	a.def.CPU.Cache = &libvirtxml.DomainCPUCache{Mode: mode}
}

func (a *DomainAction) NumatuneMode(mode string) {
	a.def.NUMATune = &libvirtxml.DomainNUMATune{
		Memory: &libvirtxml.DomainNUMATuneMemory{
			Mode: mode,
		},
	}
}

func (a *DomainAction) SCSIMultiqueue(queues uint) {
	for _, c := range a.def.Devices.Controllers {
		if c.Type == "scsi" && c.Model == "virtio-scsi" {
			c.Driver.Queues = &queues
		}
	}
}

func (a *DomainAction) UEFI() {
	a.def.OS.Firmware = "efi"
}

func (a *DomainAction) NoSecureBoot() {
	a.def.OS.Loader = &libvirtxml.DomainLoader{
		Secure: "no",
	}
}

func (a *DomainAction) Disk(volume string, pool string, wwn *string, readonly *bool) error {
	d := libvirtxml.DomainDisk{
		Driver: &libvirtxml.DomainDiskDriver{
			Name:        "qemu",
			Discard:     "unmap",
			DetectZeros: "unmap",
		},
		Device: "disk",
		Source: &libvirtxml.DomainDiskSource{},
		Target: &libvirtxml.DomainDiskTarget{
			Dev: a.getNextAvailableTargetDev(),
			Bus: "scsi",
		},
	}

	if readonly != nil && *readonly {
		d.ReadOnly = &libvirtxml.DomainDiskReadOnly{}
		d.Driver.Discard = ""
		d.Driver.DetectZeros = ""
	}

	p, err := a.StoragePoolLookupByName(pool)
	if err != nil {
		return err
	}
	v, err := a.StorageVolLookupByName(p, volume)
	if err != nil {
		return err
	}
	desc, err := a.getVolumeSourceDescription(v)
	if err != nil {
		return err
	}

	if format := desc.Target.Format.Type; format == "iso" {
		d.Driver.Type = "raw"
	} else {
		d.Driver.Type = format
	}

	if desc.Type != "file" && desc.Type != "block" {
		d.Source.Volume = &libvirtxml.DomainDiskSourceVolume{
			Pool:   pool,
			Volume: volume,
		}
		return nil
	}

	if err := setDomainDiskSource(d.Source, desc.Type, desc.Target.Path); err != nil {
		return err
	}

	if chain := desc.BackingStore; chain != nil {
		d.BackingStore = &libvirtxml.DomainDiskBackingStore{Source: &libvirtxml.DomainDiskSource{}}
		current := d.BackingStore
		for {
			v, err := a.StorageVolLookupByKey(chain.Path)
			if err != nil {
				return err
			}
			desc, err := a.getVolumeSourceDescription(v)
			if err != nil {
				return err
			}
			current.Format = &libvirtxml.DomainDiskFormat{Type: desc.Target.Format.Type}
			if err := setDomainDiskSource(current.Source, desc.Type, desc.Target.Path); err != nil {
				return err
			}
			chain = desc.BackingStore
			if chain != nil {
				current.BackingStore = &libvirtxml.DomainDiskBackingStore{Source: &libvirtxml.DomainDiskSource{}}
				current = current.BackingStore
				continue
			}
			break
		}
	}

	if wwn != nil {
		d.WWN = *wwn
	}

	a.def.Devices.Disks = append(a.def.Devices.Disks, d)
	return nil
}

func (a *DomainAction) getVolumeSourceDescription(volume libvirt.StorageVol) (*libvirtxml.StorageVolume, error) {
	xml, err := a.StorageVolGetXMLDesc(volume, 0)
	if err != nil {
		return nil, err
	}
	desc := &libvirtxml.StorageVolume{}
	return desc, desc.Unmarshal(xml)
}

func setDomainDiskSource(source *libvirtxml.DomainDiskSource, sourceType string, path string) error {
	switch sourceType {
	case "file":
		source.File = &libvirtxml.DomainDiskSourceFile{
			File: path,
		}
	case "block":
		source.Block = &libvirtxml.DomainDiskSourceBlock{
			Dev: path,
		}
	default:
		return errors.New(ErrUnsupportedDiskSourceType)
	}
	return nil
}

func (a *DomainAction) Interface(network string, queues *uint) {
	i := libvirtxml.DomainInterface{
		Driver: &libvirtxml.DomainInterfaceDriver{
			Name: "vhost",
		},
		Model: &libvirtxml.DomainInterfaceModel{
			Type: "virtio",
		},
		Source: &libvirtxml.DomainInterfaceSource{
			Network: &libvirtxml.DomainInterfaceSourceNetwork{
				Network: network,
			},
		},
	}

	if queues != nil {
		i.Driver.Queues = *queues
	}

	a.def.Devices.Interfaces = append(a.def.Devices.Interfaces, i)
}

func (a *DomainAction) PCIPassthrough(domain uint, bus uint, slot uint, function uint) {
	a.def.Devices.Hostdevs = append(a.def.Devices.Hostdevs, libvirtxml.DomainHostdev{
		Managed: "yes",
		SubsysPCI: &libvirtxml.DomainHostdevSubsysPCI{
			Driver: &libvirtxml.DomainHostdevSubsysPCIDriver{Name: "vfio"},
			Source: &libvirtxml.DomainHostdevSubsysPCISource{
				Address: &libvirtxml.DomainAddressPCI{
					Domain:   &domain,
					Bus:      &bus,
					Slot:     &slot,
					Function: &function,
				},
			},
		},
	})
}

func (a *DomainAction) Create() error {
	xml, err := a.def.Marshal()
	if err != nil {
		return err
	}
	id, err := a.DomainDefineXML(xml)
	a.id = &id
	return err
}

func (a *DomainAction) Start() error {
	return a.DomainCreate(*a.id)
}

func (a *DomainAction) Shutdown() error {
	if err := a.DomainShutdown(*a.id); err != nil {
		return err
	}
	retry := 0
	state, _, err := a.DomainGetState(*a.id, 0)
	for state != int32(libvirt.DomainShutoff) {
		if err != nil {
			return err
		}
		if retry == 30 {
			return errors.New(ErrShutdownTimedOut)
		}
		time.Sleep(1 * time.Second)
		retry += 1
		state, _, err = a.DomainGetState(*a.id, 0)
	}
	return nil
}

func (a *DomainAction) Delete() error {
	return a.DomainUndefineFlags(*a.id, libvirt.DomainUndefineNvram)
}

func (a *DomainAction) getNextAvailableTargetDev() string {
	letters := []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k", "l", "m", "n", "o", "p", "q", "r", "s", "t", "u", "v", "w", "x", "y", "z"}
	id := []string{letters[0]}
	var dev string
	for {
		dev = "sd" + strings.Join(id, "")
		if slices.IndexFunc(a.def.Devices.Disks, func(d libvirtxml.DomainDisk) bool { return d.Target.Dev == dev }) == -1 {
			break
		}
		last := len(id) - 1
		if index := slices.Index(letters, id[last]); index != 25 {
			id[last] = letters[index+1]
		} else {
			id = append(id, letters[0])
		}
	}
	return dev
}
