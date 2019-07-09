/*
   copyright 2018-2019 Banco Bilbao Vizcaya Argentaria, S.A.

   licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   you may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   withouT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   see the License for the specific language governing permissions and
   limitations under the License.
*/

package workload

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/imdario/mergo"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/bbva/qed/api/apihttp"
	"github.com/bbva/qed/api/metricshttp"
	"github.com/bbva/qed/client"
	"github.com/bbva/qed/crypto/hashing"
	"github.com/bbva/qed/log"
	"github.com/bbva/qed/util"
)

// Register all metrics.
func Register(r *prometheus.Registry) {
	// Register the metrics.
	registerMetrics.Do(
		func() {
			for _, metric := range metricsList {
				r.MustRegister(metric)
			}
		},
	)
}

type Workload struct {
	Config Config

	httpServer         *http.Server
	metricsServer      *http.Server
	prometheusRegistry *prometheus.Registry
}

type Plan [][]Config

type kind string

const (
	add         kind = "add"
	bulk        kind = "bulk"
	membership  kind = "membership"
	incremental kind = "incremental"
)

type Attack struct {
	kind           kind
	balloonVersion uint64

	config  Config
	client  *client.HTTPClient
	senChan chan Task
}

type Task struct {
	kind kind

	events              []string
	version, start, end uint64
}

func (workload *Workload) Start(APIMode bool) {
	if APIMode {
		workload.Serve()
		util.AwaitTermSignal(workload.Stop)
	} else {
		workload.RunOnce()
	}

	log.Debug("Stopping workload, about to exit...")
}

func (workload *Workload) RunOnce() {
	newAttack(workload.Config)
}

func (workload *Workload) MergeConf(newConf Config) Config {
	conf := workload.Config
	_ = mergo.Merge(&conf, newConf)
	return conf
}

func (workload *Workload) Serve() {

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, WorkloadHelp)
	})
	mux.HandleFunc("/run", func(w http.ResponseWriter, r *http.Request) {
		var err error
		w, r, err = apihttp.PostReqSanitizer(w, r)
		if err != nil {
			return
		}

		var newConf Config
		err = json.NewDecoder(r.Body).Decode(&newConf)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		newAttack(workload.MergeConf(newConf))
	})

	mux.HandleFunc("/plan", func(w http.ResponseWriter, r *http.Request) {
		var wg sync.WaitGroup
		var err error
		w, r, err = apihttp.PostReqSanitizer(w, r)
		if err != nil {
			return
		}

		var plan Plan
		err = json.NewDecoder(r.Body).Decode(&plan)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		for _, batch := range plan {
			for _, conf := range batch {
				wg.Add(1)
				go func(conf Config) {
					newAttack(workload.MergeConf(conf))
					wg.Done()
				}(conf)

			}
			wg.Wait()
		}
	})

	// Metrics server
	r := prometheus.NewRegistry()
	Register(r)
	workload.prometheusRegistry = r
	metricsMux := metricshttp.NewMetricsHTTP(r)
	log.Debug("	* Starting workload Metrics server at :17700")
	workload.metricsServer = &http.Server{Addr: ":17700", Handler: metricsMux}

	go func() {
		if err := workload.metricsServer.ListenAndServe(); err != http.ErrServerClosed {
			log.Errorf("Can't start metrics HTTP server: %s", err)
		}
	}()

	// API server
	workload.httpServer = &http.Server{Addr: ":7700", Handler: mux}
	log.Debug("	* Starting workload HTTP server at :7700")
	go func() {
		if err := workload.httpServer.ListenAndServe(); err != http.ErrServerClosed {
			log.Errorf("Can't start workload API HTTP server: %s", err)
		}
	}()
}

func (workload *Workload) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	log.Debug("Stopping metrics server...")
	err := workload.metricsServer.Shutdown(ctx)
	if err != nil {
		return err
	}

	log.Debug("Stopping HTTP server...")
	err = workload.httpServer.Shutdown(ctx)
	if err != nil {
		return err
	}

	return nil
}

func newAttack(conf Config) {
	// QED client
	transport := http.DefaultTransport.(*http.Transport)
	transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: conf.Insecure}
	httpClient := http.DefaultClient
	httpClient.Transport = transport
	client, err := client.NewHTTPClient(
		client.SetHttpClient(httpClient),
		client.SetURLs(conf.Endpoint[0], conf.Endpoint[1:]...),
		client.SetAPIKey(conf.APIKey),
		client.SetReadPreference(client.Any),
		client.SetMaxRetries(1),
		client.SetTopologyDiscovery(true),
		client.SetHealthChecks(true),
		client.SetHealthCheckTimeout(2*time.Second),   // default value
		client.SetHealthCheckInterval(60*time.Second), // default value
		client.SetAttemptToReviveEndpoints(true),
		client.SetHasherFunction(hashing.NewSha256Hasher),
	)

	if err != nil {
		panic(err)
	}
	attack := Attack{
		client:         client,
		config:         conf,
		kind:           kind(conf.Kind),
		balloonVersion: uint64(conf.NumRequests + conf.Offset - 1),
	}

	if err := attack.client.Ping(); err != nil {
		panic(err)
	}

	attack.Run()
}

func (a *Attack) Run() {
	var wg sync.WaitGroup
	a.senChan = make(chan Task)

	for rID := uint(0); rID < a.config.MaxGoRoutines; rID++ {
		wg.Add(1)
		go func(rID uint) {
			for {
				task, ok := <-a.senChan
				if !ok {
					log.Debugf("!!! close: %d", rID)
					wg.Done()
					return
				}

				switch task.kind {
				case add:
					log.Debugf("Adding: %s", task.events[0])
					_, err := a.client.Add(task.events[0])
					if err != nil {
						workloadEventAddFail.Inc()
						log.Debugf("Error adding event: version %d. Error: %s", task.version, err)
					} else {
						workloadEventAdd.Inc()
					}
				case bulk:
					bulkSize := len(task.events)
					log.Debugf("Inserting bulk: version %d, size %d, first event: %s", task.version, bulkSize, task.events[0])
					_, err := a.client.AddBulk(task.events)
					if err != nil {
						workloadEventAddFail.Add(float64(bulkSize))
						log.Debugf("Error inserting bulk: version %d, size %d. Error: %s", task.version, bulkSize, err)
					} else {
						workloadEventAdd.Add(float64(bulkSize))
					}
				case membership:
					log.Debugf("Querying membership: event %s", task.events[0])
					_, _ = a.client.Membership([]byte(task.events[0]), &task.version)
					workloadQueryMembership.Inc()
				case incremental:
					log.Debugf("Querying incremental: start %d, end %d", task.start, task.end)
					_, _ = a.client.Incremental(task.start, task.end)
					workloadQueryIncremental.Inc()
				}
			}
		}(rID)
	}

	hasReqs := func(i uint) bool {
		return i < a.config.Offset+a.config.NumRequests
	}

	hasBulk := func(j, i uint) bool {
		return i < j+a.config.BulkSize && hasReqs(i)
	}

	for i := a.config.Offset; hasReqs(i); i++ {
		task := Task{
			kind:    a.kind,
			events:  []string{},
			version: a.balloonVersion,
			start:   uint64(i),
			end:     uint64(i + a.config.IncrementalDelta),
		}

		for j := i; hasBulk(j, i); i++ {
			task.events = append(task.events, fmt.Sprintf("event %d", i))
		}

		a.senChan <- task
	}

	close(a.senChan)
	wg.Wait()
}
