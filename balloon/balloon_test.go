// Copyright © 2018 Banco Bilbao Vizcaya Argentaria S.A.  All rights reserved.
// Use of this source code is governed by an Apache 2 License
// that can be found in the LICENSE file

package balloon

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"os"
	"testing"
	"verifiabledata/balloon/hashing"
	"verifiabledata/balloon/history"
	"verifiabledata/balloon/hyper"
	"verifiabledata/balloon/storage/badger"
	"verifiabledata/balloon/storage/bolt"
	"verifiabledata/balloon/storage/bplus"
	"verifiabledata/balloon/storage/cache"
)

func TestAdd(t *testing.T) {

	frozen, frozenCloseF := openBPlusStorage()
	leaves, leavesCloseF := openBPlusStorage()
	defer frozenCloseF()
	defer leavesCloseF()

	cache := cache.NewSimpleCache(5000)
	hasher := hashing.XorHasher

	hyperT := hyper.NewTree(string(0x0), 2, cache, leaves, hasher, hyper.FakeLeafHasherF(hasher), hyper.FakeInteriorHasherF(hasher))
	historyT := history.NewTree(frozen, history.FakeLeafHasherF(hasher), history.FakeInteriorHasherF(hasher))
	balloon := NewHyperBalloon(hasher, historyT, hyperT)

	var testCases = []struct {
		event         string
		indexDigest   []byte
		historyDigest []byte
		version       uint
	}{
		{"test event 0", []byte{0x0}, []byte{0x4a}, 0},
		{"test event 1", []byte{0x1}, []byte{0x00}, 1},
		{"test event 2", []byte{0x3}, []byte{0x48}, 2},
		{"test event 3", []byte{0x0}, []byte{0x01}, 3},
		{"test event 4", []byte{0x4}, []byte{0x4e}, 4},
		{"test event 5", []byte{0x1}, []byte{0x01}, 5},
		{"test event 6", []byte{0x7}, []byte{0x4c}, 6},
		{"test event 7", []byte{0x0}, []byte{0x01}, 7},
		{"test event 8", []byte{0x8}, []byte{0x43}, 8},
		{"test event 9", []byte{0x1}, []byte{0x00}, 9},
	}

	for i, e := range testCases {

		commitment := <-balloon.Add([]byte(e.event))
		fmt.Println(commitment)

		if commitment.Version != e.version {
			t.Fatalf("Wrong version for test %d: expected %d, actual %d", i, e.version, commitment.Version)
		}

		if bytes.Compare(commitment.IndexDigest, e.indexDigest) != 0 {
			t.Fatalf("Wrong index digest for test %d: expected: %x, Actual: %x", i, e.indexDigest, commitment.IndexDigest)
		}

		if bytes.Compare(commitment.HistoryDigest, e.historyDigest) != 0 {
			t.Fatalf("Wrong history digest for test %d: expected: %x, Actual: %x", i, e.historyDigest, commitment.HistoryDigest)
		}
	}

}

//https://play.golang.org/p/nP241T7HXBj
// test event 0 : 4a [1001010] - 00 [0]
// test event 1 : 4b [1001011] - 01 [1]
// test event 2 : 48 [1001000] - 02 [10]
// test event 3 : 49 [1001001] - 03 [11]
// test event 4 : 4e [1001110] - 04 [100]
// test event 5 : 4f [1001111] - 05 [101]
// test event 6 : 4c [1001100] - 06 [110]
// test event 7 : 4d [1001101] - 07 [111]
// test event 8 : 42 [1000010] - 08 [1000]
// test event 9 : 43 [1000011] - 09 [1001]

func randomBytes(n int) []byte {
	bytes := make([]byte, n)
	_, err := rand.Read(bytes)
	if err != nil {
		panic(err)
	}

	return bytes
}

func deleteFilesInDir(path string) {
	os.RemoveAll(fmt.Sprintf("%s/leaves.db", path))
	os.RemoveAll(fmt.Sprintf("%s/frozen.db", path))
}

func BenchmarkAddBolt(b *testing.B) {
	path := "/tmp/benchAdd"
	os.MkdirAll(path, os.FileMode(0755))

	frozen, frozenCloseF := openBoltStorage(path)
	leaves, leavesCloseF := openBoltStorage(path)
	defer frozenCloseF()
	defer leavesCloseF()
	defer deleteFilesInDir(path)

	cache := cache.NewSimpleCache(5000)
	hasher := hashing.XorHasher

	hyperT := hyper.NewTree(string(0x0), 2, cache, leaves, hasher, hyper.FakeLeafHasherF(hasher), hyper.FakeInteriorHasherF(hasher))
	historyT := history.NewTree(frozen, history.FakeLeafHasherF(hasher), history.FakeInteriorHasherF(hasher))
	balloon := NewHyperBalloon(hasher, historyT, hyperT)

	b.ResetTimer()
	b.N = 10000
	for i := 0; i < b.N; i++ {
		event := randomBytes(128)
		r := balloon.Add(event)
		<-r
	}

}

func BenchmarkAddBadger(b *testing.B) {
	path := "/tmp/benchAdd"
	os.MkdirAll(path, os.FileMode(0755))

	frozen, frozenCloseF := openBadgerStorage(path)
	leaves, leavesCloseF := openBadgerStorage(path)
	defer frozenCloseF()
	defer leavesCloseF()
	defer deleteFilesInDir(path)

	cache := cache.NewSimpleCache(5000)
	hasher := hashing.XorHasher

	hyperT := hyper.NewTree(string(0x0), 2, cache, leaves, hasher, hyper.FakeLeafHasherF(hasher), hyper.FakeInteriorHasherF(hasher))
	historyT := history.NewTree(frozen, history.FakeLeafHasherF(hasher), history.FakeInteriorHasherF(hasher))
	balloon := NewHyperBalloon(hasher, historyT, hyperT)

	b.ResetTimer()
	b.N = 10000
	for i := 0; i < b.N; i++ {
		event := randomBytes(128)
		r := balloon.Add(event)
		<-r
	}

}

func openBPlusStorage() (*bplus.BPlusTreeStorage, func()) {
	store := bplus.NewBPlusTreeStorage()
	return store, func() {
		store.Close()
	}
}

func openBoltStorage(path string) (*bolt.BoltStorage, func()) {
	store := bolt.NewBoltStorage(path, "test")
	return store, func() {
		store.Close()
		deleteFile(path)
	}
}

func openBadgerStorage(path string) (*badger.BadgerStorage, func()) {
	store := badger.NewBadgerStorage(path)
	return store, func() {
		store.Close()
		deleteFile(path)
	}
}

func deleteFile(path string) {
	err := os.RemoveAll(path)
	if err != nil {
		fmt.Printf("Unable to remove db file %s", err)
	}
}
