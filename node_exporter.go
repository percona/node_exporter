// Copyright 2015 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"flag"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/log"
	"github.com/prometheus/common/version"
	"github.com/prometheus/node_exporter/collector"

	"github.com/percona/exporter_shared"
)

var (
	scrapeDurationDesc = prometheus.NewDesc(
		prometheus.BuildFQName(collector.Namespace, "scrape", "collector_duration_seconds"),
		"node_exporter: Duration of a collector scrape.",
		[]string{"collector"},
		nil,
	)
	scrapeSuccessDesc = prometheus.NewDesc(
		prometheus.BuildFQName(collector.Namespace, "scrape", "collector_success"),
		"node_exporter: Whether a collector succeeded.",
		[]string{"collector"},
		nil,
	)
)

// NodeCollector implements the prometheus.Collector interface.
type NodeCollector struct {
	collectors map[string]collector.Collector
}

// Describe implements the prometheus.Collector interface.
func (n NodeCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- scrapeDurationDesc
	ch <- scrapeSuccessDesc
}

// Collect implements the prometheus.Collector interface.
func (n NodeCollector) Collect(ch chan<- prometheus.Metric) {
	wg := sync.WaitGroup{}
	wg.Add(len(n.collectors))
	for name, c := range n.collectors {
		go func(name string, c collector.Collector) {
			execute(name, c, ch)
			wg.Done()
		}(name, c)
	}
	wg.Wait()
}

func filterAvailableCollectors(collectors string) string {
	var availableCollectors []string
	for _, c := range strings.Split(collectors, ",") {
		_, ok := collector.Factories[c]
		if ok {
			availableCollectors = append(availableCollectors, c)
		}
	}
	return strings.Join(availableCollectors, ",")
}

func execute(name string, c collector.Collector, ch chan<- prometheus.Metric) {
	begin := time.Now()
	err := c.Update(ch)
	duration := time.Since(begin)
	var success float64

	if err != nil {
		log.Errorf("ERROR: %s collector failed after %fs: %s", name, duration.Seconds(), err)
		success = 0
	} else {
		log.Debugf("OK: %s collector succeeded after %fs.", name, duration.Seconds())
		success = 1
	}
	ch <- prometheus.MustNewConstMetric(scrapeDurationDesc, prometheus.GaugeValue, duration.Seconds(), name)
	ch <- prometheus.MustNewConstMetric(scrapeSuccessDesc, prometheus.GaugeValue, success, name)
}

func loadCollectors(list []string) (map[string]collector.Collector, error) {
	collectors := map[string]collector.Collector{}
	//	for _, name := range strings.Split(list, ",") {
	for _, name := range list {
		fn, ok := collector.Factories[name]
		if !ok {
			fmt.Printf("collector '%s' not available", name)
			return nil, fmt.Errorf("collector '%s' not available", name)
		}
		c, err := fn()
		if err != nil {
			return nil, err
		}
		collectors[name] = c
	}
	return collectors, nil
}

func init() {
	prometheus.MustRegister(version.NewCollector("node_exporter"))
}

func handler(w http.ResponseWriter, r *http.Request) {
	filters := r.URL.Query()["collect[]"]
	log.Debugln("collect query:", filters)

	nc, err := collector.NewNodeCollector(filters...)
	if err != nil {
		log.Warnln("Couldn't create", err)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(fmt.Sprintf("Couldn't create %s", err)))
		return
	}

	registry := prometheus.NewRegistry()
	err = registry.Register(nc)
	if err != nil {
		log.Errorln("Couldn't register collector:", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("Couldn't register collector: %s", err)))
		return
	}

	gatherers := prometheus.Gatherers{
		prometheus.DefaultGatherer,
		registry,
	}
	// Delegate http serving to Prometheus client library, which will call collector.Collect.
	h := promhttp.InstrumentMetricHandler(
		registry,
		promhttp.HandlerFor(gatherers,
			promhttp.HandlerOpts{
				ErrorLog:      log.NewErrorLogger(),
				ErrorHandling: promhttp.ContinueOnError,
			}),
	)
	h.ServeHTTP(w, r)
}

func main() {
	var (
		showVersion = flag.Bool("version", false, "Print version information.")
		listenAddress = flag.String("web.listen-address", ":9100", "Address on which to expose metrics and web interface.")
		metricsPath = flag.String("web.telemetry-path", "/metrics", "Path under which to expose metrics.")
	)

	flag.Parse()

	if *showVersion {
		fmt.Fprintln(os.Stdout, version.Print("node_exporter"))
		os.Exit(0)
	}

	log.Infoln("Starting node_exporter", version.Info())
	log.Infoln("Build context", version.BuildContext())

	// This instance is only used to check collector creation and logging.
	nc, err := collector.NewNodeCollector()
	if err != nil {
		log.Fatalf("Couldn't create collector: %s", err)
	}
	log.Infof("Enabled collectors:")
	collectors := []string{}
	for n := range nc.Collectors {
		collectors = append(collectors, n)
	}
	sort.Strings(collectors)
	for _, n := range collectors {
		log.Infof(" - %s", n)
	}

	promcollectors, err := loadCollectors(collectors)

	if err := prometheus.Register(NodeCollector{collectors: promcollectors}); err != nil {
		log.Fatalf("Couldn't register collector: %s", err)
	}

	// Use our shared code to run server and exit on error. Upstream's code below will not be executed.
	exporter_shared.RunServer("Node", *listenAddress, *metricsPath, promhttp.ContinueOnError)

	http.HandleFunc(*metricsPath, handler)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
			<head><title>Node Exporter</title></head>
			<body>
			<h1>Node Exporter</h1>
			<p><a href="` + *metricsPath + `">Metrics</a></p>
			</body>
			</html>`))
	})

	log.Infoln("Listening on", *listenAddress)
	err = http.ListenAndServe(*listenAddress, nil)
	if err != nil {
		log.Fatal(err)
	}
}
