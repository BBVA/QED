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

package sign

import (
	"crypto/rand"
	"errors"
	"io/ioutil"
	"sync"

	"golang.org/x/crypto/ed25519"
	"golang.org/x/crypto/ssh"
)

type Signable interface {
	Sign(message []byte) ([]byte, error)
	Verify(message, sig []byte) (bool, error)
}

type EdSigner struct {
	privateKey ed25519.PrivateKey
	publicKey  ed25519.PublicKey

	sync.RWMutex
}

func NewSigner() Signable {
	var m sync.RWMutex

	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		panic(err)
	}

	return &EdSigner{
		privateKey,
		publicKey,
		m,
	}

}

func NewSignerFromFile(privateKeyPath string) (Signable, error) {
	var m sync.RWMutex

	privateKeyBytes, err := ioutil.ReadFile(privateKeyPath)
	if err != nil {
		return nil, err
	}

	pk, err := ssh.ParseRawPrivateKey(privateKeyBytes)
	privateKey := *pk.(*ed25519.PrivateKey)
	if err != nil {
		return nil, err
	}

	signer := &EdSigner{
		privateKey,
		privateKey.Public().(ed25519.PublicKey),
		m,
	}

	message := []byte("test message")
	sig, _ := signer.Sign(message)
	result, _ := signer.Verify(message, sig)
	if result != true {
		return nil, errors.New("key is unusable")
	}

	return signer, nil

}

func (s *EdSigner) Sign(message []byte) ([]byte, error) {
	s.Lock()
	defer s.Unlock()

	return ed25519.Sign(s.privateKey, message), nil
}

func (s *EdSigner) Verify(message, sig []byte) (bool, error) {
	s.Lock()
	defer s.Unlock()

	return ed25519.Verify(s.publicKey, message, sig), nil
}
