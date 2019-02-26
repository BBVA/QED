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

	"github.com/bbva/qed/log"
	"github.com/stretchr/testify/assert"
)

func TestPruneToFind(t *testing.T) {

	testCases := []struct {
		version    uint64
		expectedOp operation
	}{
		{
			version:    0,
			expectedOp: leafnil(pos(0, 0)),
		},
		{
			version: 1,
			expectedOp: inner(pos(0, 1),
				collect(getCache(pos(0, 0))),
				leafnil(pos(1, 0)),
			),
		},
		{
			version: 2,
			expectedOp: inner(pos(0, 2),
				collect(getCache(pos(0, 1))),
				partial(pos(2, 1),
					leafnil(pos(2, 0)),
				),
			),
		},
		{
			version: 3,
			expectedOp: inner(pos(0, 2),
				collect(getCache(pos(0, 1))),
				inner(pos(2, 1),
					collect(getCache(pos(2, 0))),
					leafnil(pos(3, 0)),
				),
			),
		},
		{
			version: 4,
			expectedOp: inner(pos(0, 3),
				collect(getCache(pos(0, 2))),
				partial(pos(4, 2),
					partial(pos(4, 1),
						leafnil(pos(4, 0)),
					),
				),
			),
		},
		{
			version: 5,
			expectedOp: inner(pos(0, 3),
				collect(getCache(pos(0, 2))),
				partial(pos(4, 2),
					inner(pos(4, 1),
						collect(getCache(pos(4, 0))),
						leafnil(pos(5, 0)),
					),
				),
			),
		},
		{
			version: 6,
			expectedOp: inner(pos(0, 3),
				collect(getCache(pos(0, 2))),
				inner(pos(4, 2),
					collect(getCache(pos(4, 1))),
					partial(pos(6, 1),
						leafnil(pos(6, 0)),
					),
				),
			),
		},
		{
			version: 7,
			expectedOp: inner(pos(0, 3),
				collect(getCache(pos(0, 2))),
				inner(pos(4, 2),
					collect(getCache(pos(4, 1))),
					inner(pos(6, 1),
						collect(getCache(pos(6, 0))),
						leafnil(pos(7, 0)),
					),
				),
			),
		},
	}

	for i, c := range testCases {
		prunedOp := pruneToFind(c.version)
		assert.Equalf(t, c.expectedOp, prunedOp, "The pruned operation should match for test case %d", i)
	}

}

func BenchmarkPruneToFind(b *testing.B) {

	log.SetLogger("BenchmarkPruneToFind", log.SILENT)

	b.ResetTimer()
	for i := uint64(0); i < uint64(b.N); i++ {
		pruned := pruneToFind(i)
		assert.NotNil(b, pruned)
	}

}
