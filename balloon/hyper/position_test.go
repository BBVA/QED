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
	"testing"

	"github.com/bbva/qed/balloon/position"
	assert "github.com/stretchr/testify/require"
)

func TestString(t *testing.T) {

	p := NewPosition([]byte{0x00}, 0, 8, 0)

	assert.Equal(t, "base: [0] , split: [0] , height: 0 , numBits: 8", p.String(), "Invalid hyper position")
}

func TestPositionId(t *testing.T) {
	p := NewPosition([]byte{0x00}, 0, 8, 0)
	assert.Equal(t, p.Id(), make([]byte, 9), "Invalid hyper position")
}

func TestPositionStringId(t *testing.T) {
	p := NewPosition([]byte{0x00}, 0, 8, 0)
	assert.Equal(t, p.StringId(), "00|0", "Invalid hyper position")
}

func TestPositionLeft(t *testing.T) {
	p := NewPosition([]byte{0x00}, 1, 8, 0)
	l := p.Left()
	assert.Equal(t, []byte{0x0}, l.Key(), "Invalid index")
	assert.Equal(t, uint64(0), l.Height(), "Invalid height")
	assert.Equal(t, "base: [0] , split: [0] , height: 0 , numBits: 8", l.String(), "Invalid hyper position")
}

func TestPositionRight(t *testing.T) {
	p := NewPosition([]byte{0x00}, 1, 8, 0)
	r := p.Right()
	assert.Equal(t, []byte{0x1}, r.Key(), "Invalid index")
	assert.Equal(t, uint64(0), r.Height(), "Invalid height")
	assert.Equal(t, "base: [1] , split: [1] , height: 0 , numBits: 8", r.String(), "Invalid hyper position")
}

func TestPositionDirection(t *testing.T) {
	p := NewPosition([]byte{0x00}, 2, 8, 0)
	assert.Equal(t, position.Left, p.Direction(p.Left().Key()), "Invalid direction")
	assert.Equal(t, position.Right, p.Direction(p.Right().Key()), "Invalid direction")
}

func TestPositionIsLeaf(t *testing.T) {
	p := NewPosition([]byte{0x00}, 0, 8, 0)
	assert.True(t, p.IsLeaf(), "Position should be a leaf")
	p = NewPosition([]byte{0x00}, 1, 8, 0)
	assert.False(t, p.IsLeaf(), "Position shouldn't be a leaf")
}

func TestPositionShouldBeCached(t *testing.T) {
	p := NewPosition([]byte{0x00}, 8, 8, 1)
	assert.True(t, p.ShouldBeCached(), "Position should be cached")
	p = NewPosition([]byte{0x00}, 1, 8, 1)
	assert.False(t, p.ShouldBeCached(), "Position shouldn't be cached")
}
