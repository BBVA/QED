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

package cmd

import (
	"context"

	"github.com/bbva/qed/gossip"
	"github.com/bbva/qed/log"
	"github.com/octago/sflags/gen/gpflag"
	"github.com/spf13/cobra"
)

var agentCmd *cobra.Command = &cobra.Command{
	Use:              "agent",
	Short:            "Provides access to the QED gossip agents",
	TraverseChildren: true,
}

var agentCtx context.Context = configAgent()

func init() {
	Root.AddCommand(agentCmd)
}

func configAgent() context.Context {

	conf := gossip.DefaultConfig()
	a := &struct{ Agent *gossip.Config }{conf}
	err := gpflag.ParseTo(a, agentCmd.PersistentFlags())
	if err != nil {
		log.Fatalf("err: %v", err)
	}

	return context.WithValue(Ctx, k("agent.config"), conf)
}

