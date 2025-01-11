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

// Condition types
const (
	ConditionReady             = "Ready"
	ConditionProbed            = "Probed"
	ConditionDeletionPrevented = "DeletionPrevented"
)

// Condition Reasons
const (
	ConditionCreated           = "Created"
	ConditionError             = "Error"
	ConditionUnmetRequirements = "UnmetRequirements"
	ConditionActive            = "Active"
	ConditionNotActive         = "NotActive"
	ConditionNotExist          = "NotExist"
	ConditionInUse             = "InUse"
	ConditionRequired          = "Required"
	ConditionCompleted         = "Completed"
)
