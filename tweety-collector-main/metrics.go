// Package main initializes and run Tweety-Collector
// application and its methods.
package main

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metric structure contains all required counters for data representation.
type Metric struct {
	HttpRequests    *prometheus.CounterVec
	MethodDurations *prometheus.HistogramVec
}

// Collector metrics constructor.
func NewMetric() Metric {
	metric := Metric{
		HttpRequests: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "http_requests_total",
				Help: "How many HTTP requests were processed at different endpoints.",
			},
			[]string{"service", "jobtype"},
		),
		MethodDurations: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "method_durations_total",
				Help:    "Durations of Tweety-Collector methods differed by name and tag of data for which they are measured.",
				Buckets: prometheus.LinearBuckets(0, 0.2, 20),
			},
			[]string{"methodname"},
		),
	}
	return metric
}

// Functions starts server that handles metrics in real time using prometheus package.
func startMetrics() {
	http.Handle("/metrics", promhttp.Handler())
	http.ListenAndServe(":2112", nil)
}
