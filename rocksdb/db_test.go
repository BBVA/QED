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

package rocksdb

import (
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOpenDB(t *testing.T) {
	db := newTestDB(t, "TestOpenDB", nil)
	defer db.Close()
}

func TestDBCRUD(t *testing.T) {

	db := newTestDB(t, "TestDBCRUD", nil)
	defer db.Close()

	var (
		key    = []byte("key1")
		value1 = []byte("value1")
		value2 = []byte("value2")
		wo     = NewDefaultWriteOptions()
		ro     = NewDefaultReadOptions()
	)

	// put
	require.NoError(t, db.Put(wo, key, value1))

	// retrieve
	slice1, err := db.Get(ro, key)
	defer slice1.Free()
	require.NoError(t, err)
	require.Equal(t, slice1.Data(), value1)

	// update
	require.NoError(t, db.Put(wo, key, value2))
	slice2, err := db.Get(ro, key)
	defer slice2.Free()
	require.NoError(t, err)
	require.Equal(t, slice2.Data(), value2)

	// delete
	require.NoError(t, db.Delete(wo, key))
	slice3, err := db.Get(ro, key)
	defer slice3.Free()
	require.NoError(t, err)
	require.Nil(t, slice3.Data())

}

func newTestDB(t *testing.T, name string, applyOpts func(opts *Options)) *DB {
	dir, err := ioutil.TempDir("", "rocksdb-"+name)
	require.NoError(t, err)

	opts := NewDefaultOptions()
	opts.SetCreateIfMissing(true)
	if applyOpts != nil {
		applyOpts(opts)
	}

	db, err := OpenDB(dir, opts)
	require.NoError(t, err)

	return db
}
