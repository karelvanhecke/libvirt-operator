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
)

type DomainVCPU struct {
	// +kubebuilder:validation:Required
	Count int32 `json:"count"`
	// +kubebuilder:validation:Enum=auto;static
	// +kubebuilder:validation:Optional
	Placement *string `json:"placement,omitempty"`
}

type DomainMemory struct {
	// +kubebuilder:validation:Enum=bytes;B;KB;K;KiB;MB;M;MiB;GB;G;GiB;TB;T;TiB
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Default=bytes
	Unit *string `json:"unit,omitempty"`

	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validaton:XValidation:rule="self > oldSelf",message="volume can only be expanded"
	// +kubebuilder:validation:Required
	Value int64 `json:"value"`
}

type DomainDisk struct {
	// +kubebuilder:validation:Required
	VolumeRefs []ResourceRef `json:"volumeRefs"`
	// +kubebuilder:validation:Optional
	WWN *string `json:"wwn,omitempty"`
}

// +kubebuilder:validation:XValidation:rule="has(self.secureBoot) && self.secureBoot ? !has(self.uefi) || self.uefi : true",message="SecureBoot requires UEFI to be enabled"
type DomainSpec struct {
	// +kubebuilder:validation:Required
	HostRef ResourceRef `json:"hostRef"`
	// +kubebuilder:validation:Required
	VCPU int32 `json:"vcpu"`
	// +kubebuilder:validation:Required
	Memory DomainMemory `json:"memory"`
	// +kubebuilder:validation:Required
	Disks []DomainDisk `json:"disks"`
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Default=true
	VirtioSCSIMultiqueue *bool `json:"VirtioSCSIMultiQueue,omitempty"`
	// +kubebuilder:validation:Optional
	Networks []string `json:"network,omitempty"`
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Default=true
	VirtioNetMultiqueue *bool `json:"VirtioNetMultiQueue,omitempty"`
	// +kubebuilder:validation:Optional
	CloudInit *ResourceRef `json:"cloudInitRef,omitempty"`
	// +kubebuilder:validation:Enum=interleave;strict;preferred;restrictive
	// +kubebuilder:validation:Optional
	NumatuneMode *string `json:"numatuneMode,omitempty"`
	// +kubebuilder:validation:Optional
	PCIPassthrough []string `json:"pciPassthrough,omitempty"`
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Default=true
	UEFI *bool `json:"uefi,omitempty"`
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Default=true
	SecureBoot *bool `json:"secureBoot,omitempty"`
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Default=false
	Shutdown *bool `json:"shutdown,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
type Domain struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DomainSpec `json:"spec,omitempty"`
	Status Status     `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type DomainList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Domain `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Domain{}, &DomainList{})
}
