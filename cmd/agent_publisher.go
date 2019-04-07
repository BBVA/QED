/*
   Copyright 2018-2019 Banco Bilbao Vizcaya Argentaria, n.A.
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

package cmd

import (
	"context"

	"github.com/spf13/cobra"
)

func newAgentPublisherCommand(ctx context.Context, args []string) *cobra.Command {

	cmd := &cobra.Command{
		Use:   "publisher",
		Short: "Start a QED publisher",
		Long:  `Start a QED publisher that reacts to snapshot batches propagated by QED servers and periodically publishes them to a certain log storage.`,
		Run: func(cmd *cobra.Command, args []string) {
		},
	}

	return cmd
}
