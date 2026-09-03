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

package hyper

import (
	"bytes"
	"encoding/binary"
	"math/bits"
	"sync"

	"github.com/bbva/qed/storage"
	"github.com/bbva/qed/util"
)

const (
	treeHeight  uint16 = 256           // Height for a SHA256-based hyper tree
	batchHeight uint16 = 4             // Fixed size of a single batch
	batchSize   uint64 = (31 * 33) + 4 // 31 nodes * 33 bytes/node + 4 bytes of bitmap
	flagSize    uint64 = 1             // to mark (non-)empty buckets
	// flags
	empty  byte = 0x0
	filled byte = 0x1
)

// BatchCache is a specific tailor-made cache for a Hyper tree
// that uses a SHA256 hasher.
// It stores all batches in a single fixed byte slice.
type BatchCache struct {
	buf        []byte
	entryCount int
	offsets    []uint64
	bucketSize uint64

	sync.RWMutex
}

// NewBatchCache is a constructor for a BatchCache.
func NewBatchCache(batchLevels uint8) *BatchCache {

	// First, we calculate the offsets for every depth in the
	// cache. The offset indicates the number of batches stored
	// before that position.
	// For a 6 level cache, we will the following:
	// [0 1 17 273 4369 69905 1118481]
	// Note that the last element corresponds to the depth 7.
	offsets := make([]uint64, batchLevels+1)
	n := uint64(0)
	for i := uint8(0); i < uint8(batchLevels); i++ {
		n = n + (1 << (i * uint8(batchHeight)))
		offsets[i+1] = n
	}

	// The number of buckets is the last element of the offset slice.
	numBuckets := offsets[len(offsets)-1]

	// The bucket size is the sum of the batch size and a byte flag.
	bucketSize := batchSize + flagSize

	// We calculate the size of the buffer by multiplying the number
	// of buckets by the bucket size.
	return &BatchCache{
		buf:        make([]byte, numBuckets*bucketSize),
		entryCount: 0,
		offsets:    offsets,
		bucketSize: bucketSize,
	}
}

// Get returns the value for the given a tree and a flag
// indicating if the key exists in the cache.
func (c *BatchCache) Get(key []byte) ([]byte, bool) {

	offset := c.seek(key)

	c.RLock()
	defer c.RUnlock()

	if c.buf[offset] == filled {
		value := make([]byte, batchSize)
		copy(value, c.buf[offset+flagSize:offset+c.bucketSize])
		// IMPORTANT: we are returning the whole batch size although it could
		// include 0x0s at the end. We could use two flag bytes to indicate
		// the actual size of the batch, but the cache will tend to be
		// filled up and thus, it seems to be an unnecessary overhead
		return value, true
	}

	return nil, false
}

// Put sets a key and value for a cache entry and stores it
// in the cache.
func (c *BatchCache) Put(key []byte, value []byte) {

	offset := c.seek(key)

	c.Lock()
	defer c.Unlock()

	if c.buf[offset] == empty {
		c.entryCount++
	}

	c.buf[offset] = filled
	copy(c.buf[offset+flagSize:], value)
	for i := offset + flagSize + uint64(len(value)); i < offset+c.bucketSize; i++ {
		c.buf[i] = 0x0
	}

}

// Fill takes a storage reader and loads its contents in the cache.
func (c *BatchCache) Fill(r storage.KVPairReader) (err error) {
	defer r.Close()
	for {
		entries := make([]*storage.KVPair, 100)
		n, err := r.Read(entries)
		if err != nil || n == 0 {
			break
		}
		for _, entry := range entries {
			if entry != nil {
				c.Put(entry.Key, entry.Value)
			}
		}
	}
	return nil
}

// Size returns the number of entries in the cache.
func (c *BatchCache) Size() int {
	return c.entryCount
}

// Equal is useful to compare the contents of two caches.
func (c *BatchCache) Equal(o *BatchCache) bool {
	return bytes.Equal(c.buf, o.buf)
}

func (c *BatchCache) seek(key []byte) uint64 {

	// First, we extract the height and the index from the key
	height := util.BytesAsUint16(key[0:2])
	index := key[2:]

	// In order to calculate the exact offset where the desired
	// key is stored into, we can combine the previously calculated
	// offsets by depth and the index that locates every batch at its depth.
	// This comes from the intuition that, at every depth of the tree,
	// the n (when n = depth) most significant bit of the key corresponds
	// with the ordered index of all nodes at such depth.
	//
	// For example, in the following tree of 4 bits:
	//
	// tree height - node height -> number of bits to take into account
	//
	//
	//                         0000                              4 - 4 = 0 bits => index 0
	//                          +
	//                          |
	//             +------------+-------------+
	//             |                          |
	//             +                          +
	//           0000                        1000                4 - 3 = 1 bit => indexes 0 and 1
	//             +                          +
	//             |                          |
	//      +------+------+            +------+------+
	//      |             |            |             |
	//      +             +            +             +
	//    0000          0100         1000          1100          4 - 2 = 2 bits => indexes 0..3
	//      +             +            +             +
	//      |             |            |             |
	//  +---+---+     +---+---+   +----+----+   +----+----
	//  |       |     |       |   |         |   |        |
	//  +       +     +       +   +         +   +        +
	//0000    0010  0100    0110 1000     1010 1100    1110      4 - 1 = 3 bits => indexes 0..7
	//
	//
	//                       . . . . .                           and so on...
	//
	// So, if we have the global offset of the first batch of that depth
	// (which also indicates the number of previous batches) and
	// the index of the batch at that depth, we can calculate the offset
	// this way:
	//
	// ( offset at depth + index at depth) * bucket size

	// calculate index at depth
	iPos := uint64(bits.Reverse32(binary.BigEndian.Uint32(index[0:4]))) // 32 levels max -> 4

	// calculate the tree depth for the given key
	depth := (treeHeight - height) / batchHeight

	// we get the offset for that depth. This number indicates
	// how many batches are stored before that position.
	iPos += c.offsets[depth]

	// we calculate the final offset multiplying the offset by
	// the bucket size
	return iPos * c.bucketSize
}
