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

package controller

// Finalizer
const (
	Finalizer = "libvirt.karelvanhecke.com/finalizer"
)

const (
	// #nosec G101
	LabelKeySecret = "libvirt.karelvanhecke.com/secret"
	LabelKeyAuth   = "libvirt.karelvanhecke.com/auth"
	LabelKeyHost   = "libvirt.karelvanhecke.com/host"
)

// Condition types

const (
	ConditionTypeCreated              = "Created"
	ConditionTypeDeletionProbihibited = "DeletionProhibited"
)

// Condition reasons

const (
	ConditionReasonInProgress = "InProgress"
	ConditionReasonFailed     = "Failed"
	ConditionReasonSucceeded  = "Succeeded"
)

// Condition messages
const (
	ConditionMessageHostNotFound     = "Host not found"
	ConditionMessageWaitingForHost   = "Waiting for host to become available"
	ConditionMessagePoolNotAvailable = "Pool is not available"
)

// Auth file names
const (
	PrivateKey = "privatekey"
	KnownHosts = "known_hosts"
	ClientCert = "clientcert.pem"
	ClientKey  = "clientkey.pem"
	CaCert     = "cacert.pem"
)
