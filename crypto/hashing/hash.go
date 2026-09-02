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

// Package hashing implements different hashers and their functionality.
package hashing

import (
	"crypto/sha256"
	"fmt"
	"hash"

	"golang.org/x/crypto/blake2b"
)

type Digest []byte

type Hasher interface {
	Salted([]byte, ...[]byte) Digest
	Do(...[]byte) Digest
	Len() uint16
}

// XorHasher implements the Hasher interface and computes a 2 bit hash
// function. Handy for testing hash tree implementations.
type XorHasher struct{}

func NewXorHasher() Hasher {
	return new(XorHasher)
}

// Salted function adds a seed to the input data before hashing it.
func (x XorHasher) Salted(salt []byte, data ...[]byte) Digest {
	data = append(data, salt)
	return x.Do(data...)
}

// Do function hashes input data using the XOR hash function.
func (x XorHasher) Do(data ...[]byte) Digest {
	var result byte
	for _, elem := range data {
		var sum byte
		for _, b := range elem {
			sum = sum ^ b
		}
		result = result ^ sum
	}
	return []byte{result}
}

// Len function returns the size of the resulting hash.
func (s XorHasher) Len() uint16 { return uint16(8) }

type KeyHasher struct {
	underlying hash.Hash
}

// NewBlake2bHasher implements the Hasher interface and computes a 256 bit hash
// function using the Blake2 hashing algorithm.
func NewBlake2bHasher() Hasher {
	hasher, err := blake2b.New256(nil)
	if err != nil {
		panic(fmt.Sprintf("Error creating BLAKE2b hasher %v", err))
	}
	return &KeyHasher{underlying: hasher}
}

// NewSha256Hasher implements the Hasher interface and computes a 256 bit hash
// function using the SHA256 hashing algorithm.
func NewSha256Hasher() Hasher {
	return &KeyHasher{underlying: sha256.New()}
}

// Salted function adds a seed to the input data before hashing it.
func (s *KeyHasher) Salted(salt []byte, data ...[]byte) Digest {
	data = append(data, salt)
	return s.Do(data...)
}

// Do function hashes input data using the hashing function given by the KeyHasher.
func (s *KeyHasher) Do(data ...[]byte) Digest {
	s.underlying.Reset()
	for i := 0; i < len(data); i++ {
		_, _ = s.underlying.Write(data[i])
	}
	return s.underlying.Sum(nil)[:]
}

// Len function returns the size of the resulting hash.
func (s KeyHasher) Len() uint16 { return uint16(256) }

// PearsonHasher implements the Hasher interface and computes a 8 bit hash
// function. Handy for testing hash tree implementations.
type PearsonHasher struct{}

func NewPearsonHasher() Hasher {
	return new(PearsonHasher)
}

// Salted function adds a seed to the input data before hashing it.
func (h *PearsonHasher) Salted(salt []byte, data ...[]byte) Digest {
	data = append(data, salt)
	return h.Do(data...)
}

// Do function hashes input data using the Pearson hash function.
func (p *PearsonHasher) Do(data ...[]byte) Digest {
	lookupTable := [...]uint8{
		// 0-255 shuffled in any (random) order suffices
		0x62, 0x06, 0x55, 0x96, 0x24, 0x17, 0x70, 0xa4, 0x87, 0xcf, 0xa9, 0x05, 0x1a, 0x40, 0xa5, 0xdb, //  1
		0x3d, 0x14, 0x44, 0x59, 0x82, 0x3f, 0x34, 0x66, 0x18, 0xe5, 0x84, 0xf5, 0x50, 0xd8, 0xc3, 0x73, //  2
		0x5a, 0xa8, 0x9c, 0xcb, 0xb1, 0x78, 0x02, 0xbe, 0xbc, 0x07, 0x64, 0xb9, 0xae, 0xf3, 0xa2, 0x0a, //  3
		0xed, 0x12, 0xfd, 0xe1, 0x08, 0xd0, 0xac, 0xf4, 0xff, 0x7e, 0x65, 0x4f, 0x91, 0xeb, 0xe4, 0x79, //  4
		0x7b, 0xfb, 0x43, 0xfa, 0xa1, 0x00, 0x6b, 0x61, 0xf1, 0x6f, 0xb5, 0x52, 0xf9, 0x21, 0x45, 0x37, //  5
		0x3b, 0x99, 0x1d, 0x09, 0xd5, 0xa7, 0x54, 0x5d, 0x1e, 0x2e, 0x5e, 0x4b, 0x97, 0x72, 0x49, 0xde, //  6
		0xc5, 0x60, 0xd2, 0x2d, 0x10, 0xe3, 0xf8, 0xca, 0x33, 0x98, 0xfc, 0x7d, 0x51, 0xce, 0xd7, 0xba, //  7
		0x27, 0x9e, 0xb2, 0xbb, 0x83, 0x88, 0x01, 0x31, 0x32, 0x11, 0x8d, 0x5b, 0x2f, 0x81, 0x3c, 0x63, //  8
		0x9a, 0x23, 0x56, 0xab, 0x69, 0x22, 0x26, 0xc8, 0x93, 0x3a, 0x4d, 0x76, 0xad, 0xf6, 0x4c, 0xfe, //  9
		0x85, 0xe8, 0xc4, 0x90, 0xc6, 0x7c, 0x35, 0x04, 0x6c, 0x4a, 0xdf, 0xea, 0x86, 0xe6, 0x9d, 0x8b, // 10
		0xbd, 0xcd, 0xc7, 0x80, 0xb0, 0x13, 0xd3, 0xec, 0x7f, 0xc0, 0xe7, 0x46, 0xe9, 0x58, 0x92, 0x2c, // 11
		0xb7, 0xc9, 0x16, 0x53, 0x0d, 0xd6, 0x74, 0x6d, 0x9f, 0x20, 0x5f, 0xe2, 0x8c, 0xdc, 0x39, 0x0c, // 12
		0xdd, 0x1f, 0xd1, 0xb6, 0x8f, 0x5c, 0x95, 0xb8, 0x94, 0x3e, 0x71, 0x41, 0x25, 0x1b, 0x6a, 0xa6, // 13
		0x03, 0x0e, 0xcc, 0x48, 0x15, 0x29, 0x38, 0x42, 0x1c, 0xc1, 0x28, 0xd9, 0x19, 0x36, 0xb3, 0x75, // 14
		0xee, 0x57, 0xf0, 0x9b, 0xb4, 0xaa, 0xf2, 0xd4, 0xbf, 0xa3, 0x4e, 0xda, 0x89, 0xc2, 0xaf, 0x6e, // 15
		0x2b, 0x77, 0xe0, 0x47, 0x7a, 0x8e, 0x2a, 0xa0, 0x68, 0x30, 0xf7, 0x67, 0x0f, 0x0b, 0x8a, 0xef, // 16
	}

	ih := make([]byte, 0)
	for _, k := range data {
		h := uint8(0)
		for _, v := range k {
			h = lookupTable[h^v]
		}
		ih = append(ih, h)
	}

	r := uint8(0)
	for _, v := range ih {
		r = lookupTable[r^v]
	}
	return Digest{r}

}

// Len function returns the size of the resulting hash.
func (p PearsonHasher) Len() uint16 { return uint16(8) }

// FakeHasher implements the Hasher interface and computes a hash
// function depending on the caller.
// Here, 'Salted' function does nothing but act as a passthrough to 'Do' function.
// Handy for testing hash tree implementations.
type FakeHasher struct {
	underlying Hasher
}

// Salted function directly hashes data, similarly to Do function.
func (h *FakeHasher) Salted(salt []byte, data ...[]byte) Digest {
	return h.underlying.Do(data...)
}

// Do function hashes input data using the hashing function given by the KeyHasher.
func (h *FakeHasher) Do(data ...[]byte) Digest {
	return h.underlying.Do(data...)
}

// Len function returns the size of the resulting hash.
func (h FakeHasher) Len() uint16 {
	return h.underlying.Len()
}

func NewFakeXorHasher() Hasher {
	return &FakeHasher{NewXorHasher()}
}

func NewFakeSha256Hasher() Hasher {
	return &FakeHasher{NewSha256Hasher()}
}

func NewFakePearsonHasher() Hasher {
	return &FakeHasher{NewPearsonHasher()}
}
