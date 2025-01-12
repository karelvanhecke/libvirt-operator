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

type HostSpec struct {
	// +kubebuilder:validation:Required
	Address string `json:"address"`
	// +kubebuilder:validation:Required
	AuthRef ResourceRef `json:"authRef"`

	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	// +kubebuilder:validation:Optional
	Port *int32 `json:"port,omitempty"`
}

type HostMemory struct {
	// +kubebuilder:validation:Required
	Total int64 `json:"total"`
	// +kubebuilder:validation:Required
	Free int64 `json:"free"`
}

type HostCapability struct {
	// +kubebuilder:validation:Required
	Arch string `json:"arch"`
	// +kubebuilder:validation:Required
	CPUS int32 `json:"cpus"`
	// +kubebuilder:validation:Required
	Memory HostMemory `json:"memory"`
	// +kubebuilder:validation:Required
	NUMA bool `json:"numa"`
}

// +kubebuilder:validation:Optional
type HostStatus struct {
	Capacity   *HostCapability    `json:"capability,omitempty"`
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
type Host struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   HostSpec   `json:"spec,omitempty"`
	Status HostStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type HostList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Host `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Host{}, &HostList{})
}
