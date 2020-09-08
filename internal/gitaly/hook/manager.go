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
func NewManager(gitlabAPI GitlabAPI, hooksConfig config.Hooks, opts ...ManagerOpt) *Manager {
	m := &Manager{
		gitlabAPI:         gitlabAPI,
		hooksConfig:       hooksConfig,
		conns:             client.NewPool(),
		votingDelayMetric: prometheus.NewHistogram(prometheus.HistogramOpts{}),
	}

	for _, opt := range opts {
		opt(m)
	}

	return m
}

// ManagerOpt is a self referential option for manager
type ManagerOpt func(m *Manager)

func WithVotingDelayMetric(metric prometheus.Histogram) ManagerOpt {
	return func(m *Manager) {
		m.votingDelayMetric = metric
	}
}
