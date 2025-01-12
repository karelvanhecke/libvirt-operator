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

package host

import (
	"io"

	"github.com/digitalocean/go-libvirt"
	"github.com/digitalocean/go-libvirt/socket"
)

var (
	New = func(dialer socket.Dialer) Client { return libvirt.NewWithDialer(dialer) }
)

type Client interface {
	Capabilities() ([]byte, error)
	Connect() error
	Disconnected() <-chan struct{}
	Disconnect() error
	DomainCreate(Dom libvirt.Domain) (err error)
	DomainDefineXML(XML string) (rDom libvirt.Domain, err error)
	DomainGetState(Dom libvirt.Domain, Flags uint32) (rState int32, rReason int32, err error)
	DomainGetXMLDesc(Dom libvirt.Domain, Flags libvirt.DomainXMLFlags) (rXML string, err error)
	DomainLookupByName(Name string) (rDom libvirt.Domain, err error)
	DomainShutdown(Dom libvirt.Domain) (err error)
	DomainUndefineFlags(Dom libvirt.Domain, Flags libvirt.DomainUndefineFlagsValues) (err error)
	IsConnected() bool
	NetworkIsActive(Net libvirt.Network) (rActive int32, err error)
	NetworkLookupByName(Name string) (rNet libvirt.Network, err error)
	NodeDeviceGetXMLDesc(Name string, Flags uint32) (rXML string, err error)
	NodeDeviceIsActive(Name string) (rActive int32, err error)
	NodeDeviceLookupByName(Name string) (rDev libvirt.NodeDevice, err error)
	NodeGetMemoryStats(Nparams int32, CellNum int32, Flags uint32) (rParams []libvirt.NodeGetMemoryStats, rNparams int32, err error)
	StoragePoolGetInfo(Pool libvirt.StoragePool) (rState uint8, rCapacity uint64, rAllocation uint64, rAvailable uint64, err error)
	StoragePoolIsActive(Pool libvirt.StoragePool) (rActive int32, err error)
	StoragePoolLookupByName(name string) (rPool libvirt.StoragePool, err error)
	StorageVolLookupByKey(Key string) (rVol libvirt.StorageVol, err error)
	StorageVolLookupByName(pool libvirt.StoragePool, name string) (rVol libvirt.StorageVol, err error)
	StorageVolGetXMLDesc(vol libvirt.StorageVol, flags uint32) (rXML string, err error)
	StorageVolCreateXML(pool libvirt.StoragePool, xml string, flags libvirt.StorageVolCreateFlags) (rVol libvirt.StorageVol, err error)
	StorageVolUpload(vol libvirt.StorageVol, outStream io.Reader, offset uint64, length uint64, flags libvirt.StorageVolUploadFlags) (err error)
	StorageVolResize(vol libvirt.StorageVol, capacity uint64, flags libvirt.StorageVolResizeFlags) (err error)
	StorageVolDelete(vol libvirt.StorageVol, flags libvirt.StorageVolDeleteFlags) (err error)
}
