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

type PoolSpec struct {
	// +kubebuilder:validation:Required
	Name string `json:"name"`
	// +kubebuilder:validation:Required
	HostRef ResourceRef `json:"hostRef"`
}

type PoolCapacity struct {
	// +kubebuilder:validation:Required
	Capacity int64 `json:"capacity"`
	// +kubebuilder:validation:Required
	Allocation int64 `json:"allocation"`
	// +kubebuilder:validation:Required
	Available int64 `json:"available"`
}

// +kubebuilder:validation:Optional
type PoolStatus struct {
	Capacity   *PoolCapacity      `json:"capacity,omitempty"`
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
type Pool struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PoolSpec   `json:"spec,omitempty"`
	Status PoolStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type PoolList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Pool `json:"items"`
}

func (p *Pool) ResourceName() string {
	return p.Spec.Name
}

func (p *Pool) HostRef() types.NamespacedName {
	return types.NamespacedName{Name: p.Spec.HostRef.Name, Namespace: p.Namespace}
}

func init() {
	SchemeBuilder.Register(&Pool{}, &PoolList{})
}
