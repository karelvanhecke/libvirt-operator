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

type ResourceRef struct {
	// +kubebuilder:validation:Pattern="^[a-z0-9][a-z0-9\\-.]{0,251}[a-z0-9]|[a-z0-9]$"
	// +kubebuilder:validation:Required
	Name string `json:"name"`
}

// +kubebuilder:validation:XValidation:rule="!has(self.name) ? has(self.uuid) : true",message="at least the name or uuid need to be provided"
type LibvirtLookup struct {
	// +kubebuilder:validation:Optional
	Name *string `json:"name,omitempty"`
	// +kubebuilder:validation:Optional
	UUID *string `json:"uuid,omitempty"`
	// +kubebuilder:validation:Required
	HostRef ResourceRef `json:"hostRef"`
}

type LibvirtIdentifier struct {
	// +kubebuilder:validation:Required
	Name string `json:"name"`
}

type LibvirtIdentifierWithUUID struct {
	LibvirtIdentifier `json:",inline"`
	// +kubebuilder:validation:Required
	UUID string `json:"uuid"`
}
