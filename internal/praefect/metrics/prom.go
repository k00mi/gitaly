package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	promconfig "gitlab.com/gitlab-org/gitaly/internal/config/prometheus"
)

var (
	replicationLatency prometheus.Histogram

	replicationJobsInFlight = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "gitaly",
			Subsystem: "praefect",
			Name:      "replication_jobs",
		},
	)

	// RecordReplicationLatency records replication latency
	RecordReplicationLatency = func(d float64) {
		go replicationLatency.Observe(d)
	}

	// IncReplicationJobsInFlight increases the gauge that keeps track of in flight replication jobs
	IncReplicationJobsInFlight = func() {
		go replicationJobsInFlight.Inc()
	}

	// DecReplicationJobsInFlight decreases the gauge that keeps track of in flight replication jobs
	DecReplicationJobsInFlight = func() {
		go replicationJobsInFlight.Dec()
	}
)

// Register registers praefect prometheus metrics
func Register(conf promconfig.Config) {
	replicationLatency = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: "gitaly",
			Subsystem: "praefect",
			Name:      "replication_latency",
			Buckets:   conf.GRPCLatencyBuckets,
		},
	)

	prometheus.MustRegister(replicationLatency)
	prometheus.MustRegister(replicationJobsInFlight)
}
