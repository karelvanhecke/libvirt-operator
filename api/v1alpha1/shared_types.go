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

// +kubebuilder:validation:Required
// +kubebuilder:validation:XValidation:rule="has(self.name) || has(self.generateName)",message="name or generateName must be defined"
type ExternalResourceMeta struct {
	// +kubebuilder:validation:XValidation:rule="oldSelf == self",message="can not change name of external resource"
	// +kubebuilder:validation:Pattern="^[A-Za-z0-9\\-.:_]+$"
	// +kubebuilder:validation:Optional
	Name string `json:"name,omitempty"`
	// +kubebuilder:validation:XValidation:rule="oldSelf == self",message="can not change generateName of external resource"
	// +kubebuilder:validation:Pattern="^[A-Za-z0-9\\-.:_]+$"
	// +kubebuilder:validation:Optional
	GenerateName string `json:"generateName,omitempty"`
}
