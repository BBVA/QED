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

	"github.com/bbva/qed/balloon"
	"github.com/bbva/qed/client"
	"github.com/bbva/qed/gossip"
	"github.com/bbva/qed/log"
	"github.com/bbva/qed/protocol"
	"github.com/bbva/qed/util"
	"github.com/octago/sflags/gen/gpflag"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/spf13/cobra"
)

var (
	QedAuditorInstancesCount = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "qed_auditor_instances_count",
			Help: "Number of auditor agents running.",
		},
	)

	QedAuditorBatchesProcessSeconds = prometheus.NewSummary(
		prometheus.SummaryOpts{
			Name: "qed_auditor_batches_process_seconds",
			Help: "Duration of Auditor batch processing",
		},
	)

	QedAuditorBatchesReceivedTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "qed_auditor_batches_received_total",
			Help: "Number of batches received by auditors.",
		},
	)

	QedAuditorGetMembershipProofErrTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "qed_auditor_get_membership_proof_err_total",
			Help: "Number of errors trying to get membership proofs by auditors.",
		},
	)
)

var agentAuditorCmd = &cobra.Command{
	Use:   "auditor",
	Short: "Provides access to the QED gossip auditor agent",
	Long: `Start a QED auditor that reacts to snapshot batches propagated
by QED servers and periodically executes membership queries to verify
the inclusion of events`,
	RunE: runAgentAuditor,
}

var agentAuditorCtx context.Context

func init() {
	agentAuditorCtx = configAuditor()
	agentCmd.AddCommand(agentAuditorCmd)
}

type auditorConfig struct {
	Qed      *client.Config
	Notifier *gossip.SimpleNotifierConfig
	Store    *gossip.RestSnapshotStoreConfig
	Tasks    *gossip.SimpleTasksManagerConfig
}

func newAuditorConfig() *auditorConfig {
	conf := client.DefaultConfig()
	conf.AttemptToReviveEndpoints = true
	conf.ReadPreference = client.Any
	conf.MaxRetries = 1
	return &auditorConfig{
		Qed:      conf,
		Notifier: gossip.DefaultSimpleNotifierConfig(),
		Store:    gossip.DefaultRestSnapshotStoreConfig(),
		Tasks:    gossip.DefaultSimpleTasksManagerConfig(),
	}
}

func configAuditor() context.Context {
	conf := newAuditorConfig()
	err := gpflag.ParseTo(conf, agentAuditorCmd.PersistentFlags())
	if err != nil {
		fmt.Printf("Cannot parse auditor flags: %v\n", err)
		os.Exit(1)
	}

	ctx := context.WithValue(agentCtx, k("auditor.config"), conf)

	return ctx
}

func runAgentAuditor(cmd *cobra.Command, args []string) error {
	agentConfig := agentAuditorCtx.Value(k("agent.config")).(*gossip.Config)
	conf := agentAuditorCtx.Value(k("auditor.config")).(*auditorConfig)

	// create main logger
	logOpts := &log.LoggerOptions{
		Name:            "qed.auditor",
		IncludeLocation: true,
		Level:           log.LevelFromString(agentConfig.Log),
		Output:          log.DefaultOutput,
		TimeFormat:      log.DefaultTimeFormat,
	}
	log.SetDefault(log.New(logOpts))

	// URL parse
	err := checkAuditorParams(conf)
	if err != nil {
		return err
	}

	notifier := gossip.NewSimpleNotifierFromConfig(conf.Notifier, log.L().Named("agent.notifier"))
	qed, err := client.NewHTTPClientFromConfig(conf.Qed)
	if err != nil {
		return err
	}
	tm := gossip.NewSimpleTasksManagerFromConfig(conf.Tasks, log.L().Named("agent.task-manager"))
	store := gossip.NewRestSnapshotStoreFromConfig(conf.Store)

	agent, err := gossip.NewDefaultAgent(agentConfig, qed, store, tm, notifier, log.L().Named("agent"))
	if err != nil {
		return err
	}

	memF := membershipFactory{log.L().Named("agent.membership-factory")}
	bp := gossip.NewBatchProcessor(agent, []gossip.TaskFactory{memF}, log.L().Named("agent.processor"))
	agent.In.Subscribe(gossip.BatchMessageType, bp, 255)
	defer bp.Stop()

	agent.Start()

	QedAuditorInstancesCount.Inc()

	util.AwaitTermSignal(agent.Shutdown)
	return nil
}

func checkAuditorParams(conf *auditorConfig) error {
	var err error
	err = urlParse(conf.Notifier.Endpoint...)
	if err != nil {
		return fmt.Errorf("Notifier endpoint: %v", err)
	}

	err = urlParse(conf.Store.Endpoint...)
	if err != nil {
		return fmt.Errorf("Store endpoint: %v", err)
	}

	err = urlParse(conf.Qed.Endpoints...)
	if err != nil {
		return fmt.Errorf("QED endpoint: %v", err)
	}

	return nil
}

type membershipFactory struct {
	log log.Logger
}

func (m membershipFactory) Metrics() []prometheus.Collector {
	return []prometheus.Collector{
		QedAuditorInstancesCount,
		QedAuditorBatchesProcessSeconds,
		QedAuditorBatchesReceivedTotal,
		QedAuditorGetMembershipProofErrTotal,
	}
}

func (i membershipFactory) New(ctx context.Context) gossip.Task {
	a := ctx.Value("agent").(*gossip.Agent)
	b := ctx.Value("batch").(*protocol.BatchSnapshots)

	s := b.Snapshots[0]

	QedAuditorBatchesReceivedTotal.Inc()

	return func() error {
		timer := prometheus.NewTimer(QedAuditorBatchesProcessSeconds)
		defer timer.ObserveDuration()

		// TODO Get hasher via negotiation between agent and QED
		proof, err := a.Qed.MembershipDigest(s.Snapshot.EventDigest, &s.Snapshot.Version)
		if err != nil {
			i.log.Infof("Auditor is unable to get membership proof from QED server: %v", err)

			switch fmt.Sprintf("%T", err) {
			case "*errors.errorString":
				_ = a.Notifier.Alert(fmt.Sprintf("Auditor is unable to get membership proof from QED server: %v", err))
			default:
				QedAuditorGetMembershipProofErrTotal.Inc()
			}

			return err
		}

		storedSnap, err := a.SnapshotStore.GetSnapshot(proof.CurrentVersion)
		if err != nil {
			i.log.Infof("Unable to get snapshot with version %d from storage: %v", proof.CurrentVersion, err)
			return err
		}

		checkSnap := &balloon.Snapshot{
			HistoryDigest: s.Snapshot.HistoryDigest,
			HyperDigest:   storedSnap.Snapshot.HyperDigest,
			Version:       s.Snapshot.Version,
			EventDigest:   s.Snapshot.EventDigest,
		}

		ok, err := a.Qed.MembershipVerify(s.Snapshot.EventDigest, proof, checkSnap)
		if err != nil {
			return err
		}
		if !ok {
			_ = a.Notifier.Alert(fmt.Sprintf("Unable to verify snapshot %v", s.Snapshot))
			i.log.Infof("Unable to verify snapshot %v", s.Snapshot)
		}

		i.log.Infof("Snapshot %v has been verified by QED", s.Snapshot)
		return nil
	}
}
