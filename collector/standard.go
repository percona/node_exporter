package collector

import (
	"github.com/prometheus/client_golang/prometheus"
)

func init() {
	registerCollector("standard.go", defaultDisabled, NewStandardGoCollector)
	registerCollector("standard.proccess", defaultDisabled, NewStandardProccessCollector)
}

type standardGoCollector struct {
	origin prometheus.Collector
}

// NewStandardGoCollector.
func NewStandardGoCollector() (Collector, error) {
	c := prometheus.NewGoCollector()
	return &standardGoCollector{origin: c}, nil
}

func (c *standardGoCollector) Update(ch chan<- prometheus.Metric) error {
	c.origin.Collect(ch)
	return nil
}

type standardProcessCollector struct {
	origin prometheus.Collector
}

// NewStandardGoCollector.
func NewStandardProccessCollector() (Collector, error) {
	c := prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{})
	return &standardProcessCollector{origin: c}, nil
}

func (c *standardProcessCollector) Update(ch chan<- prometheus.Metric) error {
	c.origin.Collect(ch)
	return nil
}
