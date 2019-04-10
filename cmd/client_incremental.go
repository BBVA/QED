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
	"encoding/hex"
	"fmt"

	"github.com/bbva/qed/client"
	"github.com/bbva/qed/hashing"
	"github.com/bbva/qed/log"
	"github.com/bbva/qed/protocol"
	"github.com/octago/sflags/gen/gpflag"

	"github.com/spf13/cobra"
)

var clientIncrementalCmd *cobra.Command = &cobra.Command{
	Use:   "incremental",
	Short: "Query for incremental proof",
	Long: `Query for an incremental proof to the authenticated data structure.
It also verifies the proofs provided by the server if flag enabled.`,
	RunE: runClientIncremental,
}

var clientIncrementalCtx context.Context

func init() {
	clientIncrementalCtx = configClientIncremental()
	clientCmd.AddCommand(clientIncrementalCmd)
}

type incrementalParams struct {
	Start  uint64
	End    uint64
	Verify bool
}

func configClientIncremental() context.Context {

	conf := &incrementalParams{}

	err := gpflag.ParseTo(conf, clientIncrementalCmd.PersistentFlags())
	if err != nil {
		log.Fatalf("err: %v", err)
	}
	return context.WithValue(Ctx, k("client.incremental.params"), conf)
}

func runClientIncremental(cmd *cobra.Command, args []string) error {

	// SilenceUsage is set to true -> https://github.com/spf13/cobra/issues/340
	cmd.SilenceUsage = true
	params := clientIncrementalCtx.Value(k("client.incremental.params")).(*incrementalParams)
	fmt.Printf("\nQuerying incremental between versions [ %d ] and [ %d ]\n", params.Start, params.End)

	clientConfig := clientCtx.Value(k("client.config")).(*client.Config)

	client, err := client.NewHTTPClientFromConfig(clientConfig)
	if err != nil {
		return err
	}

	proof, err := client.Incremental(params.Start, params.End)
	if err != nil {
		return err
	}

	fmt.Printf("\nReceived incremental proof: \n\n")
	fmt.Printf(" Start version: %d\n", proof.Start)
	fmt.Printf(" End version: %d\n", proof.End)
	fmt.Printf(" Incremental audit path: <TRUNCATED>\n\n")

	if params.Verify {

		var startDigest, endDigest string
		for {
			startDigest = readLine(fmt.Sprintf("Please, provide the starting historyDigest for version [ %d ]: ", params.Start))
			if startDigest != "" {
				break
			}
		}
		for {
			endDigest = readLine(fmt.Sprintf("Please, provide the ending historyDigest for version [ %d ] : ", params.End))
			if endDigest != "" {
				break
			}
		}

		sdBytes, _ := hex.DecodeString(startDigest)
		edBytes, _ := hex.DecodeString(endDigest)
		startSnapshot := &protocol.Snapshot{sdBytes, nil, params.Start, nil}
		endSnapshot := &protocol.Snapshot{edBytes, nil, params.End, nil}

		fmt.Printf("\nVerifying with snapshots: \n")
		fmt.Printf(" HistoryDigest for start version [ %d ]: %s\n", params.Start, startDigest)
		fmt.Printf(" HistoryDigest for end version [ %d ]: %s\n", params.End, endDigest)

		if client.VerifyIncremental(proof, startSnapshot, endSnapshot, hashing.NewSha256Hasher()) {
			fmt.Printf("\nVerify: OK\n\n")
		} else {
			fmt.Printf("\nVerify: KO\n\n")
		}
	}

	return nil
}

