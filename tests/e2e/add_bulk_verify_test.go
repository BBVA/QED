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

package e2e

import (
	"testing"

	"github.com/bbva/qed/hashing"
	"github.com/bbva/qed/log"
	"github.com/bbva/qed/protocol"
	"github.com/bbva/qed/testutils/rand"
	"github.com/bbva/qed/testutils/spec"
)

func TestAddBulkAndVerify(t *testing.T) {
	before, after := newServerSetup(0, false)
	let, report := spec.New()
	defer func() {
		after()
		t.Logf(report())
	}()
	log.SetLogger("e2e", log.ERROR)

	events := []string{
		rand.RandomString(10),
		rand.RandomString(10),
		rand.RandomString(10),
		rand.RandomString(10),
		rand.RandomString(10),
		rand.RandomString(10),
		rand.RandomString(10),
		rand.RandomString(10),
		rand.RandomString(10),
		rand.RandomString(10),
	}
	err := before()
	spec.NoError(t, err, "Error starting server")

	client, err := newQedClient(0)
	spec.NoError(t, err, "Error creating a new qed client")
	defer func() { client.Close() }()

	let(t, "Add one event and get its membership proof", func(t *testing.T) {
		var snapshotBulk []*protocol.Snapshot
		var membershipBulk []*protocol.MembershipResult
		var err error

		let(t, "Add event", func(t *testing.T) {
			snapshotBulk, err = client.AddBulk(events)
			spec.NoError(t, err, "Error calling client.AddBulk")

			for i, snapshot := range snapshotBulk {
				spec.Equal(t, snapshot.EventDigest, hashing.NewSha256Hasher().Do([]byte(events[i])), "The snapshot's event doesn't match")
				spec.False(t, snapshot.Version < 0, "The snapshot's version must be greater or equal to 0")
				spec.False(t, len(snapshot.HyperDigest) == 0, "The snapshot's hyperDigest cannot be empty")
				spec.False(t, len(snapshot.HistoryDigest) == 0, "The snapshot's hyperDigest cannot be empt")
			}
		})

		let(t, "Get membership proof for each inserted event", func(t *testing.T) {
			lastVersion := uint64(len(events) - 1)
			for i, snapshot := range snapshotBulk {

				result, err := client.Membership([]byte(events[i]), snapshot.Version)
				spec.NoError(t, err, "Error getting membership proof")

				spec.True(t, result.Exists, "The queried key should be a member")
				spec.Equal(t, result.QueryVersion, snapshot.Version, "The query version doest't match the queried one")
				spec.Equal(t, result.ActualVersion, snapshot.Version, "The actual version should match the queried one")
				spec.Equal(t, result.CurrentVersion, lastVersion, "The current version should match the queried one")
				spec.Equal(t, []byte(events[i]), result.Key, "The returned event doesn't math the original one")
				spec.False(t, len(result.KeyDigest) == 0, "The key digest cannot be empty")
				spec.False(t, len(result.Hyper) == 0, "The hyper proof cannot be empty")
				spec.False(t, result.ActualVersion > 0 && len(result.History) == 0, "The history proof cannot be empty when version is greater than 0")

				membershipBulk = append(membershipBulk, result)
			}
		})

		let(t, "Verify each membership", func(t *testing.T) {
			for i, result := range membershipBulk {
				snap := snapshotBulk[i]
				res := client.DigestVerify(result, snap, hashing.NewSha256Hasher)
				spec.True(t, res, "result should be valid")
			}
		})
	})
}
