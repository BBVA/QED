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
package gossip

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestRunLen(t *testing.T) {
	tm := NewDefaultTasksManager(100*time.Millisecond, 100*time.Millisecond, 1)
	tm.Start()
	executions := 0
	tm.Add(func() error {
		executions++
		return nil
	})
	time.Sleep(1 * time.Second)
	require.Equal(t, 0, tm.Len(), "Pending tasks must be 0")
	tm.Stop()
	require.Equal(t, 1, executions, "Executions must be 1")
}
