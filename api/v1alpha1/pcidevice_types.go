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

type PCIDeviceSpec struct {
	// +kubebuilder:validation:Pattern="^pci_"
	// +kubebuilder:validation:Required
	Name string `json:"name"`
	// +kubebuilder:validation:Required
	HostRef ResourceRef `json:"hostRef"`
}

type PCIDeviceAddress struct {
	Domain   int32 `json:"domain"`
	Bus      int32 `json:"bus"`
	Slot     int32 `json:"slot"`
	Function int32 `json:"function"`
}

// +kubebuilder:validation:Optional
type PCIDeviceStatus struct {
	Address    *PCIDeviceAddress  `json:"address,omitempty"`
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
type PCIDevice struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PCIDeviceSpec   `json:"spec,omitempty"`
	Status PCIDeviceStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type PCIDeviceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []PCIDevice `json:"items"`
}

func (pd *PCIDevice) ResourceName() string {
	return pd.Spec.Name
}

func init() {
	SchemeBuilder.Register(&PCIDevice{}, &PCIDeviceList{})
}
