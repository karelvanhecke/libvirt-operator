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

type VolumeSize struct {
	// +kubebuilder:validation:Enum=bytes;B;KB;K;KiB;MB;M;MiB;GB;G;GiB;TB;T;TiB;PB;P;PiB;EB;E;EiB
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Default=bytes
	Unit *string `json:"unit,omitempty"`

	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validaton:XValidation:rule="self > oldSelf",message="volume can only be expanded"
	// +kubebuilder:validation:Required
	Value int64 `json:"value"`
}

type VolumeSource struct {
	// +kubebuilder:validation:XValidation:rule="url(self).getScheme() == 'https' || url(self).getScheme() == 'http'",message="must be a valid http(s) url"
	// +kubebuilder:validation:Required
	URL string `json:"url"`

	// +kubebuilder:validation:XValidation:rule="self.matches(\"^sha256:[a-fA-F0-9]{64}|sha512:[a-fA-F0-9]{128}$\")",message="must be a valid SHA256 or SHA512 checksum"
	// +kubebuilder:validation:Optional
	Checksum *string `json:"checksum,omitempty"`
}

type BackingStoreRef struct {
	// +kubebuilder:validation:Pattern="^[a-z0-9][a-z0-9\\-.]{0,251}[a-z0-9]|[a-z0-9]$"
	// +kubebuilder:validation:Required
	Name string `json:"name"`
}

// +kubebuilder:validation:XValidation:rule="has(self.size) ? true : has(self.source) || has(self.backingStoreRef)",message="size can only be omitted when a backing store is defined"
// +kubebuilder:validation:XValidation:rule="has(self.source) ? !has(self.backingStoreRef) : true",message="source and backingstore can not be defined at the same time"
type VolumeSpec struct {
	// +kubebuilder:validation:Enum=qcow2;raw
	// +kubebuilder:validation:Required
	Format string `json:"format"`
	// +kubebuilder:validation:Required
	HostRef ResourceRef `json:"hostRef"`

	// +kubebuilder:validation:Optional
	Size *VolumeSize `json:"size,omitempty"`

	// +kubebuilder:validation:Optional
	Pool *string `json:"pool,omitempty"`

	// +kubebuilder:validation:Optional
	Source *VolumeSource `json:"source,omitempty"`

	// +kubebuilder:validation:Optional
	BackingStoreRef *ResourceRef `json:"backingStoreRef,omitempty"`
}

// +kubebuilder:validation:Optional
type VolumeStatus struct {
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
type Volume struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VolumeSpec   `json:"spec,omitempty"`
	Status VolumeStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type VolumeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metdata,omitempty"`

	Items []Volume `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Volume{}, &VolumeList{})
}
