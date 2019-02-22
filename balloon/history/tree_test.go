/*
   Copyright 2018 Banco Bilbao Vizcaya Argentaria, S.A.

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

package history

import (
	"testing"

	"github.com/bbva/qed/balloon/history/navigation"
	"github.com/bbva/qed/hashing"
	"github.com/bbva/qed/log"
	"github.com/bbva/qed/storage/bplus"
	"github.com/bbva/qed/testutils/rand"
	storage_utils "github.com/bbva/qed/testutils/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdd(t *testing.T) {

	log.SetLogger("TestAdd", log.INFO)

	testCases := []struct {
		eventDigest          hashing.Digest
		expectedRootHash     hashing.Digest
		expectedMutationsLen int
	}{
		{
			eventDigest:          hashing.Digest{0x0},
			expectedRootHash:     hashing.Digest{0x0},
			expectedMutationsLen: 1,
		},
		{
			eventDigest:          hashing.Digest{0x1},
			expectedRootHash:     hashing.Digest{0x1},
			expectedMutationsLen: 2,
		},
		{
			eventDigest:          hashing.Digest{0x2},
			expectedRootHash:     hashing.Digest{0x3},
			expectedMutationsLen: 1,
		},
		{
			eventDigest:          hashing.Digest{0x3},
			expectedRootHash:     hashing.Digest{0x0},
			expectedMutationsLen: 3,
		},
		{
			eventDigest:          hashing.Digest{0x4},
			expectedRootHash:     hashing.Digest{0x4},
			expectedMutationsLen: 1,
		},
		{
			eventDigest:          hashing.Digest{0x5},
			expectedRootHash:     hashing.Digest{0x1},
			expectedMutationsLen: 2,
		},
		{
			eventDigest:          hashing.Digest{0x6},
			expectedRootHash:     hashing.Digest{0x7},
			expectedMutationsLen: 1,
		},
		{
			eventDigest:          hashing.Digest{0x7},
			expectedRootHash:     hashing.Digest{0x0},
			expectedMutationsLen: 4,
		},
		{
			eventDigest:          hashing.Digest{0x8},
			expectedRootHash:     hashing.Digest{0x8},
			expectedMutationsLen: 1,
		},
		{
			eventDigest:          hashing.Digest{0x9},
			expectedRootHash:     hashing.Digest{0x1},
			expectedMutationsLen: 2,
		},
	}

	store := bplus.NewBPlusTreeStore()
	tree := NewHistoryTree(hashing.NewFakeXorHasher, store, 30)

	for i, c := range testCases {
		index := uint64(i)
		rootHash, mutations, err := tree.Add(c.eventDigest, index)
		require.NoError(t, err)
		assert.Equalf(t, c.expectedRootHash, rootHash, "Incorrect root hash for test case %d", i)
		assert.Equalf(t, c.expectedMutationsLen, len(mutations), "The mutations should match for test case %d", i)
		store.Mutate(mutations)
	}

}

func TestProveMembership(t *testing.T) {

	log.SetLogger("TestProveMembership", log.INFO)

	testCases := []struct {
		index, version    uint64
		eventDigest       hashing.Digest
		expectedAuditPath navigation.AuditPath
	}{
		{
			index:             0,
			version:           0,
			eventDigest:       hashing.Digest{0x0},
			expectedAuditPath: navigation.AuditPath{},
		},
		{
			index:       1,
			version:     1,
			eventDigest: hashing.Digest{0x1},
			expectedAuditPath: navigation.AuditPath{
				pos(0, 0).FixedBytes(): hashing.Digest{0x0},
			},
		},
		{
			index:       2,
			version:     2,
			eventDigest: hashing.Digest{0x2},
			expectedAuditPath: navigation.AuditPath{
				pos(0, 1).FixedBytes(): hashing.Digest{0x1},
			},
		},
		{
			index:       3,
			version:     3,
			eventDigest: hashing.Digest{0x3},
			expectedAuditPath: navigation.AuditPath{
				pos(0, 1).FixedBytes(): hashing.Digest{0x1},
				pos(2, 0).FixedBytes(): hashing.Digest{0x2},
			},
		},
		{
			index:       4,
			version:     4,
			eventDigest: hashing.Digest{0x4},
			expectedAuditPath: navigation.AuditPath{
				pos(0, 2).FixedBytes(): hashing.Digest{0x0},
			},
		},
		{
			index:       5,
			version:     5,
			eventDigest: hashing.Digest{0x5},
			expectedAuditPath: navigation.AuditPath{
				pos(0, 2).FixedBytes(): hashing.Digest{0x0},
				pos(4, 0).FixedBytes(): hashing.Digest{0x4},
			},
		},
		{
			index:       6,
			version:     6,
			eventDigest: hashing.Digest{0x6},
			expectedAuditPath: navigation.AuditPath{
				pos(0, 2).FixedBytes(): hashing.Digest{0x0},
				pos(4, 1).FixedBytes(): hashing.Digest{0x1},
			},
		},
		{
			index:       7,
			version:     7,
			eventDigest: hashing.Digest{0x7},
			expectedAuditPath: navigation.AuditPath{
				pos(0, 2).FixedBytes(): hashing.Digest{0x0},
				pos(4, 1).FixedBytes(): hashing.Digest{0x1},
				pos(6, 0).FixedBytes(): hashing.Digest{0x6},
			},
		},
		{
			index:       8,
			version:     8,
			eventDigest: hashing.Digest{0x8},
			expectedAuditPath: navigation.AuditPath{
				pos(0, 3).FixedBytes(): hashing.Digest{0x0},
			},
		},
		{
			index:       9,
			version:     9,
			eventDigest: hashing.Digest{0x9},
			expectedAuditPath: navigation.AuditPath{
				pos(0, 3).FixedBytes(): hashing.Digest{0x0},
				pos(8, 0).FixedBytes(): hashing.Digest{0x8},
			},
		},
		{
			index:       0,
			version:     1,
			eventDigest: hashing.Digest{0x0},
			expectedAuditPath: navigation.AuditPath{
				pos(1, 0).FixedBytes(): hashing.Digest{0x1},
			},
		},
		{
			index:       0,
			version:     2,
			eventDigest: hashing.Digest{0x0},
			expectedAuditPath: navigation.AuditPath{
				pos(1, 0).FixedBytes(): hashing.Digest{0x1},
				pos(2, 1).FixedBytes(): hashing.Digest{0x2},
			},
		},
		{
			index:       0,
			version:     3,
			eventDigest: hashing.Digest{0x0},
			expectedAuditPath: navigation.AuditPath{
				pos(1, 0).FixedBytes(): hashing.Digest{0x1},
				pos(2, 1).FixedBytes(): hashing.Digest{0x1},
			},
		},
		{
			index:       0,
			version:     4,
			eventDigest: hashing.Digest{0x0},
			expectedAuditPath: navigation.AuditPath{
				pos(1, 0).FixedBytes(): hashing.Digest{0x1},
				pos(2, 1).FixedBytes(): hashing.Digest{0x1},
				pos(4, 2).FixedBytes(): hashing.Digest{0x4},
			},
		},
		{
			index:       0,
			version:     5,
			eventDigest: hashing.Digest{0x0},
			expectedAuditPath: navigation.AuditPath{
				pos(1, 0).FixedBytes(): hashing.Digest{0x1},
				pos(2, 1).FixedBytes(): hashing.Digest{0x1},
				pos(4, 2).FixedBytes(): hashing.Digest{0x1},
			},
		},
		{
			index:       0,
			version:     6,
			eventDigest: hashing.Digest{0x0},
			expectedAuditPath: navigation.AuditPath{
				pos(1, 0).FixedBytes(): hashing.Digest{0x1},
				pos(2, 1).FixedBytes(): hashing.Digest{0x1},
				pos(4, 2).FixedBytes(): hashing.Digest{0x7},
			},
		},
		{
			index:       0,
			version:     7,
			eventDigest: hashing.Digest{0x0},
			expectedAuditPath: navigation.AuditPath{
				pos(1, 0).FixedBytes(): hashing.Digest{0x1},
				pos(2, 1).FixedBytes(): hashing.Digest{0x1},
				pos(4, 2).FixedBytes(): hashing.Digest{0x0},
			},
		},
	}

	store := bplus.NewBPlusTreeStore()
	tree := NewHistoryTree(hashing.NewFakeXorHasher, store, 30)

	for _, c := range testCases {
		_, mutations, err := tree.Add(c.eventDigest, c.index)
		store.Mutate(mutations)
		require.NoError(t, err)
	}

	for _, c := range testCases {
		mp, err := tree.ProveMembership(c.index, c.version)
		require.NoError(t, err)
		assert.Equalf(t, c.expectedAuditPath, mp.AuditPath, "Incorrect audit path for test case with index %d and version %d", c.index, c.version)
		assert.Equal(t, c.index, mp.Index, "The index should math")
		assert.Equal(t, c.version, mp.Version, "The version should match")
	}

}

func TestProveConsistency(t *testing.T) {

	log.SetLogger("TestProveConsistency", log.INFO)

	testCases := []struct {
		eventDigest       hashing.Digest
		expectedAuditPath navigation.AuditPath
	}{
		{
			eventDigest: hashing.Digest{0x0},
			expectedAuditPath: navigation.AuditPath{
				pos(0, 0).FixedBytes(): hashing.Digest{0x0},
			},
		},
		{
			eventDigest: hashing.Digest{0x1},
			expectedAuditPath: navigation.AuditPath{
				pos(0, 0).FixedBytes(): hashing.Digest{0x0},
				pos(1, 0).FixedBytes(): hashing.Digest{0x1},
			},
		},
		{
			eventDigest: hashing.Digest{0x2},
			expectedAuditPath: navigation.AuditPath{
				pos(0, 0).FixedBytes(): hashing.Digest{0x0},
				pos(1, 0).FixedBytes(): hashing.Digest{0x1},
				pos(2, 0).FixedBytes(): hashing.Digest{0x2},
			},
		},
		{
			eventDigest: hashing.Digest{0x3},
			expectedAuditPath: navigation.AuditPath{
				pos(0, 1).FixedBytes(): hashing.Digest{0x1},
				pos(2, 0).FixedBytes(): hashing.Digest{0x2},
				pos(3, 0).FixedBytes(): hashing.Digest{0x3},
			},
		},
		{
			eventDigest: hashing.Digest{0x4},
			expectedAuditPath: navigation.AuditPath{
				pos(0, 1).FixedBytes(): hashing.Digest{0x1},
				pos(2, 0).FixedBytes(): hashing.Digest{0x2},
				pos(3, 0).FixedBytes(): hashing.Digest{0x3},
				pos(4, 0).FixedBytes(): hashing.Digest{0x4},
			},
		},
		{
			eventDigest: hashing.Digest{0x5},
			expectedAuditPath: navigation.AuditPath{
				pos(0, 2).FixedBytes(): hashing.Digest{0x0},
				pos(4, 0).FixedBytes(): hashing.Digest{0x4},
				pos(5, 0).FixedBytes(): hashing.Digest{0x5},
			},
		},
		{
			eventDigest: hashing.Digest{0x6},
			expectedAuditPath: navigation.AuditPath{
				pos(0, 2).FixedBytes(): hashing.Digest{0x0},
				pos(4, 0).FixedBytes(): hashing.Digest{0x4},
				pos(5, 0).FixedBytes(): hashing.Digest{0x5},
				pos(6, 0).FixedBytes(): hashing.Digest{0x6},
			},
		},
		{
			eventDigest: hashing.Digest{0x7},
			expectedAuditPath: navigation.AuditPath{
				pos(0, 2).FixedBytes(): hashing.Digest{0x0},
				pos(4, 1).FixedBytes(): hashing.Digest{0x1},
				pos(6, 0).FixedBytes(): hashing.Digest{0x6},
				pos(7, 0).FixedBytes(): hashing.Digest{0x7},
			},
		},
		{
			eventDigest: hashing.Digest{0x8},
			expectedAuditPath: navigation.AuditPath{
				pos(0, 2).FixedBytes(): hashing.Digest{0x0},
				pos(4, 1).FixedBytes(): hashing.Digest{0x1},
				pos(6, 0).FixedBytes(): hashing.Digest{0x6},
				pos(7, 0).FixedBytes(): hashing.Digest{0x7},
				pos(8, 0).FixedBytes(): hashing.Digest{0x8},
			},
		},
		{
			eventDigest: hashing.Digest{0x9},
			expectedAuditPath: navigation.AuditPath{
				pos(0, 3).FixedBytes(): hashing.Digest{0x0},
				pos(8, 0).FixedBytes(): hashing.Digest{0x8},
				pos(9, 0).FixedBytes(): hashing.Digest{0x9},
			},
		},
	}

	store := bplus.NewBPlusTreeStore()
	tree := NewHistoryTree(hashing.NewFakeXorHasher, store, 30)

	for i, c := range testCases {
		index := uint64(i)
		_, mutations, err := tree.Add(c.eventDigest, index)
		require.NoError(t, err)
		store.Mutate(mutations)

		start := uint64(max(0, i-1))
		end := index
		proof, err := tree.ProveConsistency(start, end)
		require.NoError(t, err)
		assert.Equalf(t, start, proof.StartVersion, "The start version should match for test case %d", i)
		assert.Equalf(t, end, proof.EndVersion, "The start version should match for test case %d", i)
		assert.Equal(t, c.expectedAuditPath, proof.AuditPath, "Invalid audit path in test case: %d", i)
	}

}

func TestProveConsistencySameVersions(t *testing.T) {

	log.SetLogger("TestProveConsistencySameVersions", log.INFO)

	testCases := []struct {
		index             uint64
		eventDigest       hashing.Digest
		expectedAuditPath navigation.AuditPath
	}{
		{
			index:       0,
			eventDigest: hashing.Digest{0x0},
			expectedAuditPath: navigation.AuditPath{
				pos(0, 0).FixedBytes(): hashing.Digest{0x0},
			},
		},
		{
			index:       1,
			eventDigest: hashing.Digest{0x1},
			expectedAuditPath: navigation.AuditPath{
				pos(0, 0).FixedBytes(): hashing.Digest{0x0},
				pos(1, 0).FixedBytes(): hashing.Digest{0x1},
			},
		},
		{
			index:       2,
			eventDigest: hashing.Digest{0x2},
			expectedAuditPath: navigation.AuditPath{
				pos(0, 1).FixedBytes(): hashing.Digest{0x1},
				pos(2, 0).FixedBytes(): hashing.Digest{0x2},
			},
		},
		{
			index:       3,
			eventDigest: hashing.Digest{0x3},
			expectedAuditPath: navigation.AuditPath{
				pos(0, 1).FixedBytes(): hashing.Digest{0x1},
				pos(2, 0).FixedBytes(): hashing.Digest{0x2},
				pos(3, 0).FixedBytes(): hashing.Digest{0x3},
			},
		},
		{
			index:       4,
			eventDigest: hashing.Digest{0x4},
			expectedAuditPath: navigation.AuditPath{
				pos(0, 2).FixedBytes(): hashing.Digest{0x0},
				pos(4, 0).FixedBytes(): hashing.Digest{0x4},
			},
		},
	}

	store := bplus.NewBPlusTreeStore()
	tree := NewHistoryTree(hashing.NewFakeXorHasher, store, 30)

	for i, c := range testCases {
		_, mutations, err := tree.Add(c.eventDigest, c.index)
		require.NoError(t, err)
		store.Mutate(mutations)

		proof, err := tree.ProveConsistency(c.index, c.index)
		require.NoError(t, err)
		assert.Equalf(t, c.index, proof.StartVersion, "The start version should match for test case %d", i)
		assert.Equalf(t, c.index, proof.EndVersion, "The start version should match for test case %d", i)
		assert.Equal(t, c.expectedAuditPath, proof.AuditPath, "Invalid audit path in test case: %d", i)
	}
}

func max(x, y int) int {
	if x > y {
		return x
	}
	return y
}

func BenchmarkAdd(b *testing.B) {

	log.SetLogger("BenchmarkAdd", log.SILENT)

	store, closeF := storage_utils.OpenBadgerStore(b, "/var/tmp/history_tree_test.db")
	defer closeF()

	tree := NewHistoryTree(hashing.NewSha256Hasher, store, 300)

	b.N = 100000
	b.ResetTimer()
	for i := uint64(0); i < uint64(b.N); i++ {
		key := rand.Bytes(64)
		_, mutations, _ := tree.Add(key, i)
		store.Mutate(mutations)
	}
}
