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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

type DomainVCPU struct {
	// +kubebuilder:validation:Required
	Value int32 `json:"value"`
	// +kubebuilder:validation:Enum=auto;static
	// +kubebuilder:validation:Optional
	Placement *string `json:"placement,omitempty"`
}

type DomainCPU struct {
	// +kubebuilder:validation:Enum=host-model;host-passthrough;maximum
	// +kubebuilder:validation:Optional
	Mode *string `json:"mode,omitempty"`
	// +kubebuilder:validation:Enum=emulate;passthrough;disable
	// +kubebuilder:validation:Optional
	CacheMode *string `json:"cacheMode,omitempty"`
}

type DomainMemory struct {
	// +kubebuilder:validation:Enum=bytes;B;KB;K;KiB;MB;M;MiB;GB;G;GiB;TB;T;TiB
	// +kubebuilder:validation:Optional
	Unit string `json:"unit,omitempty"`

	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validaton:XValidation:rule="self > oldSelf",message="volume can only be expanded"
	// +kubebuilder:validation:Required
	Value int64 `json:"value"`
}

type DomainDisk struct {
	// +kubebuilder:validation:Required
	VolumeRef ResourceRef `json:"volumeRef"`
	// +kubebuilder:validation:Optional
	WWN *string `json:"wwn,omitempty"`
}

type DomainInterface struct {
	// +kubebuilder:validation:Required
	NetworkRef ResourceRef `json:"networkRef"`
	// +kubebuilder:validation:Optional
	Queues *int32 `json:"queues,omitempty"`
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Pattern="^(?:[0-9a-fA-F]{2}:){5}[0-9a-fA-F]{2}$"
	MacAddress *string `json:"macAddress,omitempty"`
}

type DomainCloudInit struct {
	// +kubebuilder:validation:Required
	CloudInitRef *ResourceRef `json:"cloudInitRef"`
}

type DomainPCIPassthrough struct {
	// +kubebuilder:validation:Required
	PCIDeviceRef ResourceRef `json:"pciDeviceRef"`
}

// +kubebuilder:validation:XValidation:rule="has(self.secureBoot) && self.secureBoot ? !has(self.uefi) || self.uefi : true",message="SecureBoot requires UEFI to be enabled"
type DomainSpec struct {
	// +kubebuilder:validation:XValidation:rule="oldSelf == self",message="can not change host of existing domain"
	// +kubebuilder:validation:Required
	HostRef ResourceRef `json:"hostRef"`
	// +kubebuilder:validation:XValidation:rule="oldSelf == self",message="can not change vcpu of existing domain"
	// +kubebuilder:validation:Required
	VCPU DomainVCPU `json:"vcpu"`
	// +kubebuilder:validation:XValidation:rule="oldSelf == self",message="can not change cpu of existing domain"
	// +kubebuilder:validation:Optional
	CPU *DomainCPU `json:"cpu,omitempty"`
	// +kubebuilder:validation:XValidation:rule="oldSelf == self",message="can not change memory of existing domain"
	// +kubebuilder:validation:Required
	Memory DomainMemory `json:"memory"`
	// +kubebuilder:validation:XValidation:rule="oldSelf == self",message="can not change SCSI queues of existing domain"
	// +kubebuilder:validation:Optional
	SCSIQueues *int32 `json:"scsiQueues,omitempty"`
	// +kubebuilder:validation:XValidation:rule="oldSelf == self",message="can not change disks of existing domain"
	// +kubebuilder:validation:Required
	Disks []DomainDisk `json:"disks"`
	// +kubebuilder:validation:XValidation:rule="oldSelf == self",message="can not change interfaces of existing domain"
	// +kubebuilder:validation:Optional
	Interfaces []DomainInterface `json:"interfaces,omitempty"`
	// +kubebuilder:validation:XValidation:rule="oldSelf == self",message="can not change cloud-init of existing domain"
	// +kubebuilder:validation:Optional
	CloudInit *DomainCloudInit `json:"cloudInit,omitempty"`
	// +kubebuilder:validation:XValidation:rule="oldSelf == self",message="can not change numatune mode of existing domain"
	// +kubebuilder:validation:Enum=interleave;strict;preferred;restrictive
	// +kubebuilder:validation:Optional
	NumatuneMode *string `json:"numatuneMode,omitempty"`
	// +kubebuilder:validation:XValidation:rule="oldSelf == self",message="can not change pci passthrough of existing domain"
	// +kubebuilder:validation:Optional
	PCIPassthrough []DomainPCIPassthrough `json:"pciPassthrough,omitempty"`
	// +kubebuilder:validation:XValidation:rule="oldSelf == self",message="can not change uefi of existing domain"
	// +kubebuilder:validation:Optional
	UEFI *bool `json:"uefi,omitempty"`
	// +kubebuilder:validation:XValidation:rule="oldSelf == self",message="can not change secureboot of existing domain"
	// +kubebuilder:validation:Optional
	SecureBoot *bool `json:"secureBoot,omitempty"`
	// +kubebuilder:validation:Optional
	Shutoff *bool `json:"shutoff,omitempty"`
}

// +kubebuilder:validation:Optional
type DomainStatus struct {
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
type Domain struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DomainSpec   `json:"spec,omitempty"`
	Status DomainStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type DomainList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Domain `json:"items"`
}

func (d *Domain) ResourceName() string {
	return d.Name
}

func (d *Domain) HostRef() types.NamespacedName {
	return types.NamespacedName{Name: d.Spec.HostRef.Name, Namespace: d.Namespace}
}

func (d *Domain) VolumeRef(disk DomainDisk) types.NamespacedName {
	return types.NamespacedName{Name: disk.VolumeRef.Name, Namespace: d.Namespace}
}

func (d *Domain) NetworkRef(in DomainInterface) types.NamespacedName {
	return types.NamespacedName{Name: in.NetworkRef.Name, Namespace: d.Namespace}
}

func (d *Domain) PCIDeviceRef(pci DomainPCIPassthrough) types.NamespacedName {
	return types.NamespacedName{Name: pci.PCIDeviceRef.Name, Namespace: d.Namespace}
}

func (d *Domain) CloudInitRef() types.NamespacedName {
	return types.NamespacedName{Name: d.Spec.CloudInit.CloudInitRef.Name, Namespace: d.Namespace}
}

func init() {
	SchemeBuilder.Register(&Domain{}, &DomainList{})
}
