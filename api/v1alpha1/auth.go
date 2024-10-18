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

// +kubebuilder:validation:Enum=TLS;SSH
type AuthType string

const (
	TLSAuth = "TLS"
	SSHAuth = "SSH"
)

// +kubebuilder:validation:XValidation:rule="self.type == \"SSH\" ? has(self.username) && (has(self.knownHosts) || (has(self.verify) && !self.verify)) && !has(self.ca) : true",message="type SSH requires username and knownHosts (unless verify is disabled) to be set"
// +kubebuilder:validation:XValidation:rule="self.type == \"TLS\" ? has(self.ca) || (has(self.verify) && !self.verify) && !has(self.username) && !has(self.knownHosts) : true",message="type TLS requires ca (unless verify is disabled) to be set"
type AuthSpec struct {
	// +kubebuilder:validation:XValidation:rule="oldSelf == self",message="type is immutable"
	// +kubebuilder:validation:Required
	Type AuthType `json:"type"`
	// +kubebuilder:validation:Required
	SecretRef ResourceRef `json:"secretRef"`

	// +kubebuilder:validation:Optional
	Verify *bool `json:"verify,omitempty"`
	// +kubebuilder:validation:Optional
	Ca *string `json:"ca,omitempty"`
	// +kubebuilder:validation:Optional
	KnownHosts *string `json:"knownHosts,omitempty"`
	// +kubebuilder:validation:Optional
	Username *string `json:"username,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
type Auth struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec AuthSpec `json:"spec,omitempty"`
}

// +kubebuilder:object:root=true
type AuthList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Auth `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Auth{}, &AuthList{})
}
