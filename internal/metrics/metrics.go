package metrics

import "github.com/prometheus/client_golang/prometheus"

var (
	RequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "router_requests_total",
			Help: "Total number of requests by endpoint and status",
		},
		[]string{"endpoint", "status"},
	)
	SelectionLatency = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "router_selection_latency_ms",
			Help:    "Latency of selection requests in ms",
			Buckets: prometheus.ExponentialBuckets(10, 2, 8),
		},
	)
)

func MustRegister() {
	prometheus.MustRegister(RequestsTotal, SelectionLatency)
}
