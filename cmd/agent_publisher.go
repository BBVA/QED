/*
   Copyright 2018 Banco Bilbao Vizcaya Argentaria, n.A.
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
	"github.com/bbva/qed/gossip"
	"github.com/bbva/qed/gossip/publisher"
	"github.com/bbva/qed/log"
	"github.com/bbva/qed/util"
	"github.com/spf13/cobra"
	"github.com/valyala/fasthttp"
)

func newAgentPublisherCommand(ctx *agentContext) *cobra.Command {

	var endpoints []string

	cmd := &cobra.Command{
		Use:   "publisher",
		Short: "Start a QED publisher",
		Long: `Start a QED publisher that reacts to snapshot batches 
		propagated by QED servers and periodically publishes them to
		a certain log storage.`,
		Run: func(cmd *cobra.Command, args []string) {

			log.SetLogger("QedPublisher", logLevel)

			config := ctx.config
			publisherConfig := publisher.NewConfig(&fasthttp.Client{}, endpoints)

			agent, err := gossip.NewAgent(config, []gossip.Processor{publisher.NewPublisher(publisherConfig)})
			if err != nil {
				log.Fatalf("Failed to start the QED publisher: %v", err)
			}

			contacted, err := agent.Join(config.StartJoin)
			if err != nil {
				log.Fatalf("Failed to join the cluster: %v", err)
			}
			log.Debugf("Number of nodes contacted: %d", contacted)

			agent.Start()
			defer agent.Shutdown()
			util.AwaitTermSignal(agent.Leave)
		},
	}

	cmd.Flags().StringSliceVarP(&endpoints, "endpoints", "", []string{}, "Comma-delimited list of end-publishers ([host]:port), through which an publisher can send requests")
	cmd.MarkFlagRequired("endpoints")

	return cmd
}
