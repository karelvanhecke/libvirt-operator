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

import "time"

// Finalizer
const (
	Finalizer = "libvirt.karelvanhecke.com/finalizer"
)

const (
	// #nosec G101
	LabelKeySecret = "libvirt.karelvanhecke.com/secret"
	LabelKeyAuth   = "libvirt.karelvanhecke.com/auth"
	LabelKeyHost   = "libvirt.karelvanhecke.com/host"
	LabelKeyPool   = "libvirt.karelvanhecke.com/pool"
)

// Condition types

const (
	CondTypeCreated              = "Created"
	CondTypeDeletionProbihibited = "DeletionProhibited"
	CondTypeDataRetrieved        = "DataRetrieved"
)

// Condition reasons

const (
	CondReasonInProgress = "InProgress"
	CondReasonFailed     = "Failed"
	CondReasonSucceeded  = "Succeeded"
	CondReasonInUse      = "InUse"
)

// Condition messages
const (
	CondMsgHostNotFound   = "Host not found"
	CondMsgWaitingForHost = "Waiting for host to become available"
	CondMsgPoolNotFound   = "Pool not found"
	CondMsgWaitingForPool = "Waiting for pool to become available"
)

// Auth file names
const (
	PrivateKey = "privatekey"
	KnownHosts = "known_hosts"
	ClientCert = "clientcert.pem"
	ClientKey  = "clientkey.pem"
	CaCert     = "cacert.pem"
)

const (
	DataRefreshInterval = 1 * time.Minute
)
