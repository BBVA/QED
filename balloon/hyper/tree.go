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

// Package hyper implements the history tree (a sparse Merkle tree)
// life cycle, its operations, as well as
// the functionality of request and verify a membership proof.
package hyper

import (
	"sync"

	"github.com/bbva/qed/balloon/cache"
	"github.com/bbva/qed/crypto/hashing"
	"github.com/bbva/qed/log"
	"github.com/bbva/qed/storage"
	"github.com/bbva/qed/util"
)

type HyperTree struct {
	store   storage.Store
	cache   cache.ModifiableCache
	hasherF func() hashing.Hasher

	hasher           hashing.Hasher
	cacheHeightLimit uint16
	defaultHashes    []hashing.Digest
	batchLoader      batchLoader

	log log.Logger

	sync.RWMutex
}

func NewHyperTree(hasherF func() hashing.Hasher, store storage.Store, cache cache.ModifiableCache) *HyperTree {
	return NewHyperTreeWithLogger(hasherF, store, cache, log.L())
}

func NewHyperTreeWithLogger(hasherF func() hashing.Hasher, store storage.Store, cache cache.ModifiableCache, logger log.Logger) *HyperTree {

	hasher := hasherF()
	numBits := hasher.Len()
	cacheHeightLimit := numBits - min(24, numBits/8*4)

	tree := &HyperTree{
		store:            store,
		cache:            cache,
		hasherF:          hasherF,
		hasher:           hasher,
		cacheHeightLimit: cacheHeightLimit,
		defaultHashes:    make([]hashing.Digest, numBits),
		batchLoader:      NewDefaultBatchLoaderWithLogger(store, cache, cacheHeightLimit, logger.Named("loader")),
		log:              logger,
	}

	tree.defaultHashes[0] = tree.hasher.Do([]byte{0x0}, []byte{0x0})
	for i := uint16(1); i < hasher.Len(); i++ {
		tree.defaultHashes[i] = tree.hasher.Do(tree.defaultHashes[i-1], tree.defaultHashes[i-1])
	}

	// warm-up cache
	tree.RebuildCache()

	return tree
}

// Add function adds an event digest into the hyper tree.
// It builds a stack of operations and then interpret it to calculates the expected
// root hash, and returns it along with the storage mutations to be done at balloon level.
func (t *HyperTree) Add(eventDigest hashing.Digest, version uint64) (hashing.Digest, []*storage.Mutation, error) {
	t.Lock()
	defer t.Unlock()

	//t.log.Tracef("Adding new event digest %x with version %d", eventDigest, version)

	versionAsBytes := util.Uint64AsBytes(version)

	// build a stack of operations and then interpret it to generate the root hash
	ops := pruneToInsert(eventDigest, versionAsBytes, t.cacheHeightLimit, t.batchLoader)
	ctx := &pruningContext{
		Hasher:         t.hasher,
		Cache:          t.cache,
		RecoveryHeight: t.cacheHeightLimit + 4,
		DefaultHashes:  t.defaultHashes,
		Mutations:      make([]*storage.Mutation, 0),
	}

	rh, err := ops.Pop().Interpret(ops, ctx)
	if err != nil {
		t.log.Fatalf("Invalid operation: %v", err)
	}

	return rh, ctx.Mutations, nil
}

// AddBulk function adds a bulk of event digests into the hyper tree.
// It builds a stack of operations and then interpret it to calculates the expected
// root hash, and returns it along with the storage mutations to be done at balloon level.
func (t *HyperTree) AddBulk(eventDigests []hashing.Digest, initialVersion uint64) (hashing.Digest, []*storage.Mutation, error) {
	t.Lock()
	defer t.Unlock()

	versionsAsBytes := make([][]byte, 0)
	digestsAsBytes := make([][]byte, 0)
	for i, eventDigest := range eventDigests {
		versionsAsBytes = append(versionsAsBytes, util.Uint64AsBytes(initialVersion+uint64(i)))
		digestsAsBytes = append(digestsAsBytes, []byte(eventDigest))
	}

	// build a stack of operations and then interpret it to generate the root hash
	ops := pruneToInsertBulk(digestsAsBytes, versionsAsBytes, t.cacheHeightLimit, t.batchLoader)
	ctx := &pruningContext{
		Hasher:         t.hasher,
		Cache:          t.cache,
		RecoveryHeight: t.cacheHeightLimit + 4,
		DefaultHashes:  t.defaultHashes,
		Mutations:      make([]*storage.Mutation, 0),
	}

	rh, err := ops.Pop().Interpret(ops, ctx)
	if err != nil {
		t.log.Fatalf("Invalid operation: %v", err)
	}

	return rh, ctx.Mutations, nil
}

// QueryMembership function builds the membership proof of the given event digest.
// It builds a stack of operations and then interpret it to generate and return the audit
// path.
func (t *HyperTree) QueryMembership(eventDigest hashing.Digest) (proof *QueryProof, err error) {
	t.Lock()
	defer t.Unlock()

	//t.log.Tracef("Proving membership for index %d", eventDigest)

	// build a stack of operations and then interpret it to generate the audit path
	ops := pruneToFind(eventDigest, t.batchLoader)
	ctx := &pruningContext{
		Hasher:         t.hasher,
		Cache:          t.cache,
		RecoveryHeight: t.cacheHeightLimit + 4,
		DefaultHashes:  t.defaultHashes,
		AuditPath:      make(AuditPath, 0),
	}

	_, err = ops.Pop().Interpret(ops, ctx)
	if err != nil {
		t.log.Fatalf("Invalid operation: %v", err)
	}

	// ctx.Value is nil if the digest does not exist
	return NewQueryProof(eventDigest, ctx.Value, ctx.AuditPath, t.hasherF()), nil
}

// RebuildCache function reads the hypercache rocksDB table to create indexes and cache.
// It builds a stack of operations and then interpret it to rebuild the cache.
func (t *HyperTree) RebuildCache() {
	t.Lock()
	defer t.Unlock()

	// warm up cache
	t.log.Info("Warming up hyper cache...")

	indexes := make([][]byte, 0)

	tileReader := t.store.GetAll(storage.HyperCacheTable)
	tiles := make([]*storage.KVPair, 1000)
	for {
		n, err := tileReader.Read(tiles)
		if n == 0 || err != nil {
			break
		}

		for i := 0; i < n; i++ {
			indexes = append(indexes, tiles[i].Key[2:])
			t.cache.Put(tiles[i].Key, tiles[i].Value)
		}
	}
	// if there are no elements, we start with a clean
	// cache
	if len(indexes) == 0 {
		t.log.Info("Warming up finished: empty cache")
		return
	}

	ops := pruneToRebuild(indexes, t.cacheHeightLimit+4, t.batchLoader)
	ctx := &pruningContext{
		Hasher:         t.hasher,
		Cache:          t.cache,
		RecoveryHeight: t.cacheHeightLimit + 4,
		DefaultHashes:  t.defaultHashes,
	}
	_, err := ops.Pop().Interpret(ops, ctx)
	if err != nil {
		t.log.Fatalf("Invalid operation: %v", err)
	}

	t.log.Info("Warming up finished")
}

// Close function resets all hyper tree stuff.
func (t *HyperTree) Close() {
	t.Lock()
	defer t.Unlock()

	t.cache = nil
	t.hasher = nil
	t.defaultHashes = nil
	t.store = nil
	t.batchLoader = nil
}

func min(x, y uint16) uint16 {
	if x < y {
		return x
	}
	return y
}
