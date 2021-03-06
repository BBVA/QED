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
	"fmt"
	"os"

	"github.com/bbva/qed/gossip"
	"github.com/octago/sflags/gen/gpflag"
	"github.com/spf13/cobra"
)

var agentCmd *cobra.Command = &cobra.Command{
	Use:   "agent",
	Short: "Provides access to the QED gossip agents",
	Long: `QED provides standalone agents to help maintain QED security. We have included
three agents into the distribution:
	* Monitor agent: checks the lag of the system between the QED Log and the
	  Snapshot Store as seen by the gossip network
	* Auditor agent: verifies QED membership proofs of the snapshots received
	  throught the  gossip network
	* Publisher agent: publish snapshots to the snapshot store`,
	TraverseChildren:  true,
	PersistentPreRunE: runAgent,
}

var agentCtx context.Context

func init() {
	agentCtx = configAgent()
	agentCmd.SilenceUsage = true
	agentCmd.MarkFlagRequired("bind-addr")
	agentCmd.MarkFlagRequired("metrics-addr")
	agentCmd.MarkFlagRequired("node-name")
	agentCmd.MarkFlagRequired("role")
	agentCmd.MarkFlagRequired("log")
	Root.AddCommand(agentCmd)
}

func configAgent() context.Context {
	conf := gossip.DefaultConfig()
	err := gpflag.ParseTo(conf, agentCmd.PersistentFlags())
	if err != nil {
		fmt.Printf("Cannot parse agent flags: %v\n", err)
		os.Exit(1)
	}

	return context.WithValue(Ctx, k("agent.config"), conf)
}

func runAgent(cmd *cobra.Command, args []string) error {
	// URL parsing
	var err error

	gossipStartJoin, _ := cmd.Flags().GetStringSlice("start-join")
	err = urlParseNoSchemaRequired(gossipStartJoin...)
	if err != nil {
		return fmt.Errorf("Gosspip start join: %v", err)
	}

	bindAddress, _ := cmd.Flags().GetString("bind-addr")
	err = urlParseNoSchemaRequired(bindAddress)
	if err != nil {
		return fmt.Errorf("Bind address: %v", err)
	}

	advertiseAddress, _ := cmd.Flags().GetString("advertise-addr")
	err = urlParseNoSchemaRequired(advertiseAddress)
	if err != nil {
		return fmt.Errorf("Advertise address: %v", err)
	}

	return nil
}
