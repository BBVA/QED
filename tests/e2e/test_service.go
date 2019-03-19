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

package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"sync"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/bbva/qed/api/metricshttp"
	"github.com/bbva/qed/log"
	"github.com/bbva/qed/protocol"
)

var (

	// Prometheus

	QedStoreInstancesCount = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "qed_store_instances_count",
			Help: "Number of store services running.",
		},
	)

	QedStoreBatchesStoredTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "qed_store_batches_stored_total",
			Help: "Number of batches received (POST from publishers).",
		},
	)

	QedStoreSnapshotsRetrievedTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "qed_store_snapshots_retrieved_total",
			Help: "Number of snapshots retrieved (GET from auditors).",
		},
	)

	QedStoreAlertsGeneratedTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "qed_store_alerts_generated_total",
			Help: "Number of alerts generated.",
		},
	)

	QedStoreEventsStoredTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "qed_store_events_stored_total",
			Help: "Number of events stored.",
		},
	)

	metricsList = []prometheus.Collector{
		QedStoreInstancesCount,
		QedStoreBatchesStoredTotal,
		QedStoreSnapshotsRetrievedTotal,
		QedStoreAlertsGeneratedTotal,
		QedStoreEventsStoredTotal,
	}

	registerMetrics sync.Once
)

// Register all metrics.
func Register(r *prometheus.Registry) {
	registerMetrics.Do(
		func() {
			for _, metric := range metricsList {
				r.MustRegister(metric)
			}
		},
	)
}

type alertStore struct {
	sync.Mutex
	d []string
}

func newAlertStore() *alertStore {
	return &alertStore{d: make([]string, 0)}
}

func (a *alertStore) Append(msg string) {
	a.Lock()
	defer a.Unlock()
	a.d = append(a.d, msg)
}

func (a *alertStore) GetAll() []string {
	a.Lock()
	defer a.Unlock()
	n := make([]string, len(a.d))
	copy(n, a.d)
	return n
}

const segmentSize uint64 = 1 << 20

type Segment [segmentSize]*protocol.SignedSnapshot

type snapStore struct {
	segments []Segment
	sync.Mutex
}

func newSnapStore() *snapStore {
	return &snapStore{segments: append([]Segment{}, Segment{})}
}

func (s *snapStore) Put(b *protocol.BatchSnapshots) {
	s.Lock()
	defer s.Unlock()
	lastVersion := b.Snapshots[len(b.Snapshots)-1].Snapshot.Version
	maxSegment := lastVersion / segmentSize
	for i := uint64(len(s.segments)); i <= maxSegment; i++ {
		s.segments = append(s.segments, Segment{})
	}
	for _, snap := range b.Snapshots {
		targetSegment := snap.Snapshot.Version / segmentSize
		targetIndex := snap.Snapshot.Version - (targetSegment * segmentSize)
		s.segments[targetSegment][targetIndex] = snap
		QedStoreEventsStoredTotal.Inc()
	}
}

func (s *snapStore) Get(version uint64) *protocol.SignedSnapshot {
	s.Lock()
	defer s.Unlock()
	targetSegment := version / segmentSize
	if targetSegment >= uint64(len(s.segments)) {
		return nil
	}
	targetIndex := version - (targetSegment * segmentSize)
	return s.segments[targetSegment][targetIndex]
}

type Service struct {
	snaps  *snapStore
	alerts *alertStore

	metricsServer      *http.Server
	prometheusRegistry *prometheus.Registry
	httpServer         *http.Server

	quitCh chan bool
}

func NewService() *Service {
	return &Service{
		snaps:  newSnapStore(),
		alerts: newAlertStore(),
		quitCh: make(chan bool),
	}
}

func (s *Service) Start(foreground bool) {

	// Metrics server.
	r := prometheus.NewRegistry()
	Register(r)
	s.prometheusRegistry = r
	metricsMux := metricshttp.NewMetricsHTTP(r)
	s.metricsServer = &http.Server{Addr: ":18888", Handler: metricsMux}

	QedStoreInstancesCount.Inc()

	go func() {
		log.Debugf("	* Starting metrics HTTP server ")
		if err := s.metricsServer.ListenAndServe(); err != http.ErrServerClosed {
			log.Errorf("Can't start metrics HTTP server: %s", err)
		}
	}()

	// Snapshot/alert store server.
	router := http.NewServeMux()
	router.HandleFunc("/batch", s.postBatchHandler())
	router.HandleFunc("/snapshot", s.getSnapshotHandler())
	router.HandleFunc("/alert", s.alertHandler())

	s.httpServer = &http.Server{Addr: ":8888", Handler: router}
	fmt.Println("Starting test service...")

	go func() {
		for {
			select {
			case <-s.quitCh:
				log.Debugf("\nShutting down the server...")
				_ = s.httpServer.Shutdown(context.Background())
				return
			}
		}
	}()

	if foreground {
		if err := s.httpServer.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatal(err)
		}
	} else {
		go (func() {
			if err := s.httpServer.ListenAndServe(); err != http.ErrServerClosed {
				log.Fatal(err)
			}
		})()
	}
}

func (s *Service) Shutdown() {

	// Metrics
	QedStoreInstancesCount.Dec()

	log.Debugf("Metrics enabled: stopping server...")
	if err := s.metricsServer.Shutdown(context.Background()); err != nil { // TODO include timeout instead nil
		log.Error(err)
	}
	log.Debugf("Done.\n")

	s.quitCh <- true
	close(s.quitCh)
}

func (s *Service) postBatchHandler() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			QedStoreBatchesStoredTotal.Inc()
			// Decode batch to get signed snapshots and batch version.
			var b protocol.BatchSnapshots
			buf, err := ioutil.ReadAll(r.Body)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			err = b.Decode(buf)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if len(b.Snapshots) < 1 {
				log.Infof("Empty batch recevied!")
				http.Error(w, "Empty batch recevied!", http.StatusInternalServerError)
				return
			}
			s.snaps.Put(&b)
			return
		}
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
	}
}

func (s *Service) getSnapshotHandler() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			QedStoreSnapshotsRetrievedTotal.Inc()
			q := r.URL.Query()
			version, err := strconv.ParseInt(q.Get("v"), 10, 64)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			b := s.snaps.Get(uint64(version))
			if b == nil {
				http.Error(w, fmt.Sprintf("Version not found: %v", version), http.StatusUnprocessableEntity)
				return
			}
			buf, err := b.Encode()
			if err != nil {
				fmt.Printf("ERROR: %v", err)
			}

			_, err = w.Write(buf)
			if err != nil {
				fmt.Printf("ERROR: %v", err)
			}
			return
		}
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
	}
}

func (s *Service) alertHandler() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			b, err := json.Marshal(s.alerts.GetAll())
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			_, err = w.Write(b)
			if err != nil {
				fmt.Printf("ERROR: %v", err)
			}
			return
		} else if r.Method == "POST" {
			QedStoreAlertsGeneratedTotal.Inc()

			buf, err := ioutil.ReadAll(r.Body)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			s.alerts.Append(string(buf))
			return
		}
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
	}
}
