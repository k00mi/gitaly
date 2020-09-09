package hook

import (
	"github.com/prometheus/client_golang/prometheus"
	"gitlab.com/gitlab-org/gitaly/client"
	"gitlab.com/gitlab-org/gitaly/internal/gitaly/config"
)

// Manager is a hook manager containing Git hook business logic.
type Manager struct {
	gitlabAPI         GitlabAPI
	hooksConfig       config.Hooks
	conns             *client.Pool
	votingDelayMetric prometheus.Histogram
}

// NewManager returns a new hook manager
func NewManager(gitlabAPI GitlabAPI, cfg config.Cfg) *Manager {
	return &Manager{
		gitlabAPI:   gitlabAPI,
		hooksConfig: cfg.Hooks,
		conns:       client.NewPool(),
		votingDelayMetric: prometheus.NewHistogram(
			prometheus.HistogramOpts{
				Name:    "gitaly_hook_transaction_voting_delay_seconds",
				Help:    "Delay between calling out to transaction service and receiving a response",
				Buckets: cfg.Prometheus.GRPCLatencyBuckets,
			},
		),
	}
}

func (m *Manager) Describe(descs chan<- *prometheus.Desc) {
	prometheus.DescribeByCollect(m, descs)
}

func (m *Manager) Collect(metrics chan<- prometheus.Metric) {
	m.votingDelayMetric.Collect(metrics)
}
