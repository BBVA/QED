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

package hyper

import (
	"bytes"
	"fmt"
	"runtime"

	"github.com/bbva/qed/balloon/hashing"
)

// Constant Empty is a constant for empty leaves
var Empty = []byte{0x00}

// Constant Set is a constant for non-empty leaves
var Set = []byte{0x01}

// leafHasher is the internal interface to be used in the hyper tree.
type leafHasher func([]byte, []byte, []byte) []byte

// interiorHasher is the internal interface to be used in the hyper tree.
type interiorHasher func([]byte, []byte, []byte, []byte) []byte

// leafHasherF is a closure to create a leafHasher function with a
// switchable hasher.
func leafHasherF(hasher hashing.Hasher) leafHasher {
	return func(id, a, base []byte) []byte {
		if bytes.Equal(a, Empty) {
			return hasher(id)
		}

		return hasher(id, base)
	}
}

// interiorHasherF is a closure to create a interiorHasher function with a
// switchable hasher.
func interiorHasherF(hasher hashing.Hasher) interiorHasher {
	return func(left, right, base, height []byte) []byte {
		if bytes.Equal(left, right) {
			return hasher(left, right)
		}

		return hasher(left, right, base, height)
	}
}
