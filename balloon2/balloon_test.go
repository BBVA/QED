package balloon2

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bbva/qed/balloon2/common"
	"github.com/bbva/qed/db/bplus"
	"github.com/bbva/qed/testutils/rand"
	"github.com/bbva/qed/util"
)

func TestAdd(t *testing.T) {

	store := bplus.NewBPlusTreeStore()
	defer store.Close()

	balloon := NewBalloon(0, store, common.NewSha256Hasher)

	for i := uint64(0); i < 9; i++ {
		commitment, mutations, err := balloon.Add(rand.Bytes(128))
		store.Mutate(mutations...)

		require.NoError(t, err)
		assert.Truef(t, len(mutations) > 0, "There should be some mutations in test %d", i)
		assert.Equalf(t, i, commitment.Version, "Wrong version in test %d", i)
		assert.NotNil(t, commitment.HyperDigest, "The HyperDigest shouldn't be nil in test %d", i)
		assert.NotNil(t, commitment.HistoryDigest, "The HistoryDigest shouldn't be nil in test %d", i)
	}

}

func TestQueryMembership(t *testing.T) {

	store := bplus.NewBPlusTreeStore()
	defer store.Close()

	balloon := NewBalloon(0, store, common.NewFakeXorHasher)

	testCases := []struct {
		key     []byte
		version uint64
	}{
		{[]byte{0x5a}, uint64(0)},
	}

	for i, c := range testCases {
		_, mutations, err := balloon.Add(c.key)
		require.NoErrorf(t, err, "Error adding event %d", i)
		store.Mutate(mutations...)

		proof, err := balloon.QueryMembership(c.key, c.version)

		require.NoError(t, err)
		assert.True(t, proof.Exists, "The event should exist in test %d ", i)
		assert.Equalf(t, c.version, proof.QueryVersion, "The query version does not match in test %d : expected %d, actual %d", i, c.version, proof.QueryVersion)
		assert.Equalf(t, c.version, proof.ActualVersion, "The actual version does not match in test %d : expected %d, actual %d", i, c.version, proof.ActualVersion)
		assert.NotNil(t, proof.HyperProof, "The hyper proof should not be nil in test %d ", i)
		assert.NotNil(t, proof.HistoryProof, "The history proof should not be nil in test %d ", i)
	}

}

func TestMembershipProofVerify(t *testing.T) {

	testCases := []struct {
		exists         bool
		hyperOK        bool
		historyOK      bool
		currentVersion uint64
		queryVersion   uint64
		actualVersion  uint64
		expectedResult bool
	}{
		// Event exists, queryVersion <= actualVersion, and both trees verify it
		{true, true, true, uint64(0), uint64(0), uint64(0), true},
		// Event exists, queryVersion <= actualVersion, but HyperTree does not verify it
		{true, false, true, uint64(0), uint64(0), uint64(0), false},
		// Event exists, queryVersion <= actualVersion, but HistoryTree does not verify it
		{true, true, false, uint64(0), uint64(0), uint64(0), false},

		// Event exists, queryVersion > actualVersion, and both trees verify it
		{true, true, true, uint64(1), uint64(1), uint64(0), true},
		// Event exists, queryVersion > actualVersion, but HyperTree does not verify it
		{true, false, true, uint64(1), uint64(1), uint64(0), false},

		// Event does not exist, HyperTree verifies it
		{false, true, false, uint64(0), uint64(0), uint64(0), true},
		// Event does not exist, HyperTree does not verify it
		{false, false, false, uint64(0), uint64(0), uint64(0), false},
	}

	for i, c := range testCases {
		event := []byte("Yadda yadda")
		commitment := &Commitment{
			common.Digest("Some hyperDigest"),
			common.Digest("Some historyDigest"),
			c.actualVersion,
		}
		proof := NewMembershipProof(
			c.exists,
			common.NewFakeVerifiable(c.hyperOK),
			common.NewFakeVerifiable(c.historyOK),
			c.currentVersion,
			c.queryVersion,
			c.actualVersion,
			event,
			common.NewSha256Hasher(),
		)

		result := proof.Verify(event, commitment)

		require.Equalf(t, c.expectedResult, result, "Unexpected result '%v' in test case '%d'", result, i)
	}
}

func TestQueryConsistencyProof(t *testing.T) {

	testCases := []struct {
		start, end uint64
	}{
		{uint64(0), uint64(2)},
	}

	for i, c := range testCases {
		store := bplus.NewBPlusTreeStore()
		defer store.Close()
		balloon := NewBalloon(0, store, common.NewFakeXorHasher)

		for j := 0; j <= int(c.end); j++ {
			_, mutations, err := balloon.Add(util.Uint64AsBytes(uint64(j)))
			require.NoErrorf(t, err, "Error adding event %d", j)
			store.Mutate(mutations...)
		}

		proof, err := balloon.QueryConsistency(c.start, c.end)

		require.NoError(t, err)
		assert.Equalf(t, c.start, proof.Start, "The query start does not match in test %d: expected %d, actual %d", i, c.start, proof.Start)
		assert.Equalf(t, c.end, proof.End, "The query end does not match in test %d: expected %d, actual %d", i, c.end, proof.End)
		assert.Truef(t, len(proof.AuditPath) > 0, "The lenght of the audith path should be >0 in test %d ", i)
	}
}

func BenchmarkAddBadger(b *testing.B) {
	store, closeF := common.OpenBadgerStore("/var/tmp/ballon_bench.db")
	defer closeF()

	balloon := NewBalloon(0, store, common.NewSha256Hasher)

	b.ResetTimer()
	b.N = 10000
	for i := 0; i < b.N; i++ {
		event := rand.Bytes(128)
		_, mutations, _ := balloon.Add(event)
		store.Mutate(mutations...)
	}

}
