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

package fake

import (
	"errors"
	"io"
	"slices"

	"github.com/ARM-software/golang-utils/utils/safecast"
	"github.com/digitalocean/go-libvirt"
	"libvirt.org/go/libvirtxml"
)

const (
	ErrConnectFailed        = "connect to host failed"
	ErrDisconnectFailed     = "disconnect from host failed"
	ErrPoolNotExist         = "pool does not exist"
	ErrUnsupportedDialer    = "unsupported dialer"
	ErrDomainNotExist       = "domain does not exist"
	ErrDomainAlreadyExist   = "domain does not exist"
	ErrDomainAlreadyRunning = "domain is already running"
	ErrDomainAlreadyShutoff = "domain is already shutoff"
	ErrNetworkNotExist      = "network does not exist"
	ErrNodedevNotExist      = "Node device does not exist"
)

type Memory struct {
	Total uint64
	Free  uint64
}

type Fake struct {
	capabilities   *libvirtxml.Caps
	memory         Memory
	connectFail    bool
	disconnectFail bool
	disconnected   chan struct{}
	pools          []*Pool
	domains        []*Domain
	networks       []*Network
	nodedevs       []*Nodedev
}

func New() *Fake {
	f := &Fake{disconnected: make(chan struct{})}
	close(f.disconnected)
	return f
}

func (f *Fake) WithConnectFail() {
	f.connectFail = true
}

func (f *Fake) WithDisconnectFail() {
	f.disconnectFail = true
}

func (f *Fake) WithPool(pool *libvirtxml.StoragePool, state int32, volumes []*libvirtxml.StorageVolume) {
	p := &Pool{
		xml:     pool,
		state:   state,
		volumes: volumes,
	}
	f.pools = append(f.pools, p)
}

func (f *Fake) WithCapbilities(caps *libvirtxml.Caps) {
	f.capabilities = caps
}

func (f *Fake) Capabilities() ([]byte, error) {
	caps, err := f.capabilities.Marshal()
	return []byte(caps), err
}

func (f *Fake) Connect() error {
	if f.connectFail {
		return errors.New(ErrConnectFailed)
	}

	f.disconnected = make(chan struct{})

	return nil
}

func (f *Fake) Disconnected() <-chan struct{} {
	return f.disconnected
}

func (f *Fake) Disconnect() error {
	if f.disconnectFail {
		return errors.New(ErrDisconnectFailed)
	}

	close(f.disconnected)
	return nil
}

func (f *Fake) IsConnected() bool {
	select {
	case <-f.disconnected:
		return false
	default:
		return true
	}
}

func (f *Fake) StoragePoolLookupByName(name string) (rPool libvirt.StoragePool, err error) {
	pool, err := f.getPoolByName(name)
	if err != nil {
		return libvirt.StoragePool{}, err
	}

	return libvirt.StoragePool{Name: pool.xml.Name}, nil
}

func (f *Fake) StorageVolLookupByName(pool libvirt.StoragePool, name string) (rVol libvirt.StorageVol, err error) {
	p, err := f.getPoolByName(pool.Name)
	if err != nil {
		return libvirt.StorageVol{}, err
	}

	v, err := p.getVolumeByName(name)
	if err != nil {
		return libvirt.StorageVol{}, err
	}

	return libvirt.StorageVol{
		Pool: pool.Name,
		Name: v.Name,
		Key:  v.Key,
	}, nil
}

func (f *Fake) StorageVolLookupByKey(key string) (rVol libvirt.StorageVol, err error) {
	for _, pool := range f.pools {
		i := slices.IndexFunc(pool.volumes, func(v *libvirtxml.StorageVolume) bool { return v.Key == key })
		if i != -1 {
			volume := pool.volumes[i]
			return libvirt.StorageVol{Name: volume.Name, Pool: pool.xml.Name, Key: volume.Key}, nil
		}
	}
	return libvirt.StorageVol{}, libvirt.Error{
		Code:    uint32(libvirt.ErrNoStorageVol),
		Message: ErrVolumeNotExist,
	}
}

func (f *Fake) StorageVolGetXMLDesc(vol libvirt.StorageVol, flags uint32) (rXML string, err error) {
	p, err := f.getPoolByName(vol.Pool)
	if err != nil {
		return "", err
	}
	v, err := p.getVolumeByName(vol.Name)
	if err != nil {
		return "", err
	}

	return v.Marshal()
}

func (f *Fake) StorageVolCreateXML(pool libvirt.StoragePool, xml string, flags libvirt.StorageVolCreateFlags) (rVol libvirt.StorageVol, err error) {
	p, err := f.getPoolByName(pool.Name)
	if err != nil {
		return libvirt.StorageVol{}, err
	}

	v, err := p.createVol(xml)
	if err != nil {
		return libvirt.StorageVol{}, err
	}

	return libvirt.StorageVol{Pool: p.xml.Name, Name: v.Name, Key: v.Key}, nil
}

func (f *Fake) StorageVolUpload(vol libvirt.StorageVol, outStream io.Reader, offset uint64, length uint64, flags libvirt.StorageVolUploadFlags) (err error) {
	p, err := f.getPoolByName(vol.Pool)
	if err != nil {
		return err
	}
	v, err := p.getVolumeByName(vol.Name)
	if err != nil {
		return err
	}

	bytes, err := io.ReadAll(outStream)
	if err != nil {
		return err
	}

	v.Capacity.Unit = "bytes"
	v.Capacity.Value = uint64(len(bytes))

	return nil
}

func (f *Fake) StorageVolResize(vol libvirt.StorageVol, capacity uint64, flags libvirt.StorageVolResizeFlags) (err error) {
	p, err := f.getPoolByName(vol.Pool)
	if err != nil {
		return err
	}
	v, err := p.getVolumeByName(vol.Name)
	if err != nil {
		return err
	}

	v.Capacity.Unit = "bytes"
	v.Capacity.Value = uint64(capacity)

	return nil
}

func (f *Fake) StorageVolDelete(vol libvirt.StorageVol, flags libvirt.StorageVolDeleteFlags) (err error) {
	p, err := f.getPoolByName(vol.Pool)
	if err != nil {
		return err
	}

	return p.deleteVol(vol.Name)
}

func (f *Fake) getPoolByName(name string) (*Pool, error) {
	i := slices.IndexFunc(f.pools, func(pool *Pool) bool { return pool.xml.Name == name })
	if i == -1 {
		return nil, libvirt.Error{Code: uint32(libvirt.ErrNoStoragePool), Message: ErrPoolNotExist}
	}
	return f.pools[i], nil
}

func (f *Fake) WithDomain(domain *libvirtxml.Domain, state int32) {
	f.domains = append(f.domains, &Domain{
		state: state,
		xml:   domain,
	})
}

func (f *Fake) DomainBlockResize(Dom libvirt.Domain, Disk string, Size uint64, Flags libvirt.DomainBlockResizeFlags) (err error) {
	//TODO
	return nil
}

func (f *Fake) DomainCreate(Dom libvirt.Domain) (err error) {
	d, err := f.getDomainByName(Dom.Name)
	if err != nil {
		return err
	}

	if d.state == int32(libvirt.DomainRunning) {
		return errors.New(ErrDomainAlreadyRunning)
	}

	d.state = int32(libvirt.DomainRunning)
	d.reason = int32(libvirt.DomainRunningBooted)

	return nil
}

func (f *Fake) DomainDefineXML(XML string) (rDom libvirt.Domain, err error) {
	d := &libvirtxml.Domain{}
	if err := d.Unmarshal(XML); err != nil {
		return libvirt.Domain{}, err
	}

	_, err = f.getDomainByName(d.Name)
	if err == nil {
		return libvirt.Domain{}, errors.New(ErrDomainAlreadyExist)
	}
	if err.Error() != ErrDomainNotExist {
		return libvirt.Domain{}, err
	}

	f.domains = append(f.domains, &Domain{state: int32(libvirt.DomainShutoff), xml: d})

	return libvirt.Domain{Name: d.Name}, nil
}

func (f *Fake) DomainGetState(Dom libvirt.Domain, Flags uint32) (rState int32, rReason int32, err error) {
	d, err := f.getDomainByName(Dom.Name)
	if err != nil {
		return 0, 0, err
	}

	return d.state, d.reason, nil
}

func (f *Fake) DomainGetXMLDesc(Dom libvirt.Domain, Flags libvirt.DomainXMLFlags) (rXML string, err error) {
	d, err := f.getDomainByName(Dom.Name)
	if err != nil {
		return "", err
	}

	return d.xml.Marshal()
}

func (f *Fake) DomainLookupByName(Name string) (rDom libvirt.Domain, err error) {
	d, err := f.getDomainByName(Name)
	if err != nil {
		return libvirt.Domain{}, err
	}

	return libvirt.Domain{Name: d.xml.Name}, nil
}

func (f *Fake) DomainShutdown(Dom libvirt.Domain) (err error) {
	d, err := f.getDomainByName(Dom.Name)
	if err != nil {
		return err
	}

	if d.state == int32(libvirt.DomainShutoff) {
		return errors.New(ErrDomainAlreadyShutoff)
	}

	d.state = int32(libvirt.DomainShutoff)
	d.reason = int32(libvirt.DomainShutoffShutdown)

	return nil
}

func (f *Fake) DomainUndefineFlags(Dom libvirt.Domain, Flags libvirt.DomainUndefineFlagsValues) (err error) {
	f.domains = slices.DeleteFunc(f.domains, func(d *Domain) bool { return d.xml.Name == Dom.Name })
	return
}

func (f *Fake) getDomainByName(name string) (*Domain, error) {
	i := slices.IndexFunc(f.domains, func(domain *Domain) bool { return domain.xml.Name == name })
	if i == -1 {
		return nil, libvirt.Error{
			Code:    uint32(libvirt.ErrNoDomain),
			Message: ErrDomainNotExist,
		}
	}
	return f.domains[i], nil
}

func (f *Fake) WithNodeDev(nodedev *libvirtxml.NodeDevice, state int32) {
	f.nodedevs = append(f.nodedevs, &Nodedev{xml: nodedev, state: state})
}

func (f *Fake) WithNetwork(network *libvirtxml.Network, state int32) {
	f.networks = append(f.networks, &Network{xml: network, state: 1})
}

func (f *Fake) getNetworkByName(name string) (*Network, error) {
	i := slices.IndexFunc(f.networks, func(n *Network) bool { return n.xml.Name == name })
	if i == -1 {
		return nil, libvirt.Error{Code: uint32(libvirt.ErrNoNetwork), Message: ErrNetworkNotExist}
	}

	return f.networks[i], nil
}

func (f *Fake) getNodeDeviceByName(name string) (*Nodedev, error) {
	i := slices.IndexFunc(f.nodedevs, func(n *Nodedev) bool { return n.xml.Name == name })
	if i == -1 {
		return nil, libvirt.Error{Code: uint32(libvirt.ErrNoNodeDevice), Message: ErrNodedevNotExist}
	}

	return f.nodedevs[i], nil
}

func (f *Fake) NodeDeviceGetXMLDesc(Name string, Flags uint32) (rXML string, err error) {
	n, err := f.getNodeDeviceByName(Name)
	if err != nil {
		return "", err
	}

	return n.xml.Marshal()
}

func (f *Fake) NetworkIsActive(Net libvirt.Network) (rActive int32, err error) {
	n, err := f.getNetworkByName(Net.Name)
	if err != nil {
		return -1, err
	}
	return n.state, nil
}

func (f *Fake) NodeDeviceIsActive(Name string) (rActive int32, err error) {
	n, err := f.getNodeDeviceByName(Name)
	if err != nil {
		return -1, err
	}
	return n.state, nil
}

func (f *Fake) StoragePoolIsActive(Pool libvirt.StoragePool) (rActive int32, err error) {
	p, err := f.getPoolByName(Pool.Name)
	if err != nil {
		return -1, err
	}
	return p.state, nil
}

func (f *Fake) StoragePoolGetInfo(Pool libvirt.StoragePool) (rState uint8, rCapacity uint64, rAllocation uint64, rAvailable uint64, err error) {
	p, err := f.getPoolByName(Pool.Name)
	if err != nil {
		return 0, 0, 0, 0, err
	}
	return safecast.ToUint8(p.state), p.xml.Capacity.Value, p.xml.Allocation.Value, p.xml.Available.Value, nil
}

func (f *Fake) NetworkLookupByName(Name string) (rNet libvirt.Network, err error) {
	n, err := f.getNetworkByName(Name)
	if err != nil {
		return libvirt.Network{}, err
	}

	return libvirt.Network{Name: n.xml.Name}, nil
}

func (f *Fake) NodeDeviceLookupByName(Name string) (rDev libvirt.NodeDevice, err error) {
	d, err := f.getNodeDeviceByName(Name)
	if err != nil {
		return libvirt.NodeDevice{}, err
	}

	return libvirt.NodeDevice{Name: d.xml.Name}, nil
}

func (f *Fake) WithMemoryStats(memory Memory) {
	f.memory = memory
}

func (f *Fake) NodeGetMemoryStats(Nparams int32, CellNum int32, Flags uint32) (rParams []libvirt.NodeGetMemoryStats, rNparams int32, err error) {
	return []libvirt.NodeGetMemoryStats{
		{Field: libvirt.NodeMemoryStatsTotal, Value: f.memory.Total},
		{Field: libvirt.NodeMemoryStatsFree, Value: f.memory.Free},
	}, 2, nil
}
