/*
   Copyright 2018-2019 Banco Bilbao Vizcaya Argentaria, S.A.

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

package client

import (
	"time"

	"github.com/bbva/qed/crypto/hashing"
)

// ReadPref specifies the preferred type of node in the cluster
// to send request to.
type ReadPref int

const (
	// Primary forces to read only from the primary node (or leader).
	Primary ReadPref = iota

	// PrimaryPreferred aims to read from the primary node (or leader).
	//
	// Use PrimaryPreferred if you want an application to read from the primary
	// under normal circumstances, but to allow stale reads from secondaries when
	// the primary is unavailable. This provides a "read-only mode" for your
	// application during a failover.
	PrimaryPreferred

	// Secondary force to read only from secondary nodes (or replicas).
	Secondary

	// SecondaryPreferred aims to read from secondary nodes (or replicas).
	//
	// In general, do not use SecondaryPreferred to provide extra capacity for reads,
	// because all members of a cluster have roughly equivalent write traffic; as
	// a result, secondaries will service reads at roughly the same rate as the
	// primary. In addition, although replication is synchronous, there is some amount
	// of delay between event replication to secondaries and change application
	// to the corresponding balloon. Reading from a secondary can return stale data.
	SecondaryPreferred

	// Any forces to read from any node in the cluster including the leader.
	Any
)

const (
	// DefaultTimeout is the default number of seconds to wait for a request to QED.
	DefaultTimeout = 10 * time.Second

	// DefaultDialTimeout is the default number of seconds to wait for the connection
	// to be established.
	DefaultDialTimeout = 5 * time.Second

	// DefaultHandshakeTimeout is the default number of seconds to wait for a handshake
	// negotiation.
	DefaultHandshakeTimeout = 5 * time.Second

	// DefaultInsecure sets if the client verifies, by default, the server's
	// certificate chain and host name, allowing MiTM vector attacks.
	DefaultInsecure = false

	// DefaultMaxRetries sets the default maximum number of retries before giving up
	// when performing an HTTP request to QED.
	DefaultMaxRetries = 0

	// DefaultHealthCheckEnabled specifies if healthchecks are enabled by default.
	DefaultHealthCheckEnabled = true

	// DefaultHealthCheckTimeout specifies the time the health checker waits for
	// a response from QED.
	DefaultHealthCheckTimeout = 2 * time.Second

	// DefaultHealthCheckInterval is the default interval between two health checks
	// of the nodes in the QED cluster.
	DefaultHealthCheckInterval = 60 * time.Second

	// DefaultTopologyDiscoveryEnabled specifies if the discoverer is enabled by default.
	DefaultTopologyDiscoveryEnabled = true

	// off is used to disable timeouts.
	off = -1 * time.Second
)

// Config sets the HTTP client configuration
type Config struct {
	// Log level
	Log string `desc:"Set log level to info, error or debug"`

	// Endpoints [host:port,host:port,...] to ask for QED cluster-topology.
	Endpoints []string `desc:"REST QED Log service endpoint list http://ip1:port1,http://ip2:port2... "`

	// Snapshot store [host:port] to ask for QED published signed snapshots.
	SnapshotStoreURL string `desc:"REST Snapshot store service endpoint http://ip:port "`

	// Insecure enables the verification of the server's certificate chain
	// and host name, allowing MiTM vector attacks.
	Insecure bool `desc:"Set it to true to disable the verification of the server's certificate chain"`

	// Timeout is the time to wait for a request to QED.
	Timeout time.Duration `desc:"Time to wait for a request to QED"`

	// DialTimeout is the time to wait for the connection to be established.
	DialTimeout time.Duration `desc:"Time to wait for the connection to be established"`

	// HandshakeTimeout is the time to wait for a handshake negotiation.
	HandshakeTimeout time.Duration `desc:"Time to wait for a handshake negotiation"`

	// Controls how the client will route all queries to members of the cluster.
	ReadPreference ReadPref `flag:"-"`

	// MaxRetries sets the maximum number of retries before giving up
	// when performing an HTTP request to QED.
	MaxRetries int `desc:"Sets the maximum number of retries before giving up"`

	// EnableTopologyDiscovery enables the process of discovering the cluster
	// topology when requests fail.
	EnableTopologyDiscovery bool `desc:"Enables the process of discovering the cluster topology when requests fail"`

	// EnableHealthChecks enables healthchecks of all endpoints in the current cluster topology.
	EnableHealthChecks bool `desc:"Enables healthchecks of all endpoints in the current cluster topology"`

	// HealthCheckTimeout is the time the healthcheck waits for a response
	// from a QED server.
	HealthCheckTimeout time.Duration `desc:"Time the healthcheck waits for a response from QED"`

	// HealthCheckInterval is the interval between two health checks of the nodes in the QED cluster.
	HealthCheckInterval time.Duration `desc:"Interval between two health checks of the nodes in the QED cluster"`

	// AttemptToReviveEndpoints sets if dead endpoints will be marked alive again after a
	// round-robin round. This way, they will be picked up in the next try.
	AttemptToReviveEndpoints bool `desc:"Set if dead endpoints will be marked alive again after a round-robin round"`

	// HasherFunction sets which function will use the client to do its work: verify, ask for proofs, ...
	HasherFunction func() hashing.Hasher `desc:"Hashing function to verify proofs"`
}

// DefaultConfig creates a Config structures with default values.
func DefaultConfig() *Config {
	return &Config{
		Endpoints:                []string{"http://127.0.0.1:8800"},
		SnapshotStoreURL:         "http://127.0.0.1:8888",
		Insecure:                 DefaultInsecure,
		Timeout:                  DefaultTimeout,
		DialTimeout:              DefaultDialTimeout,
		HandshakeTimeout:         DefaultHandshakeTimeout,
		ReadPreference:           Primary,
		MaxRetries:               DefaultMaxRetries,
		EnableTopologyDiscovery:  DefaultTopologyDiscoveryEnabled,
		EnableHealthChecks:       DefaultHealthCheckEnabled,
		HealthCheckTimeout:       DefaultHealthCheckTimeout,
		HealthCheckInterval:      DefaultHealthCheckInterval,
		AttemptToReviveEndpoints: false,
		HasherFunction:           hashing.NewSha256Hasher,
	}
}
