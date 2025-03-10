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

// Common condition messages
const (
	conditionCreationSucceeded  = "Creation succeeded"
	conditionProbeCompleted     = "Probe completed"
	conditionHostClientNotReady = "Host client is not ready"
	conditionPoolNotReady       = "Pool is not ready"
)

// Auth file names
const (
	privateKey = "privatekey"
	knownHosts = "known_hosts"
	clientCert = "clientcert.pem"
	clientKey  = "clientkey.pem"
	caCert     = "cacert.pem"
)

const (
	dataRefreshInterval = 1 * time.Minute
)
