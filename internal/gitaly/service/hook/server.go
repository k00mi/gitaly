package hook

import (
	"github.com/prometheus/client_golang/prometheus"
	"gitlab.com/gitlab-org/gitaly/client"
	"gitlab.com/gitlab-org/gitaly/internal/gitaly/config"
	"gitlab.com/gitlab-org/gitaly/proto/go/gitalypb"
)

type server struct {
	conns             *client.Pool
	hooksConfig       config.Hooks
	gitlabAPI         GitlabAPI
	votingDelayMetric prometheus.Histogram
}

// NewServer creates a new instance of a gRPC namespace server
func NewServer(gitlab GitlabAPI, hooksConfig config.Hooks, serverOpts ...ServerOpt) gitalypb.HookServiceServer {
	s := &server{
		gitlabAPI:         gitlab,
		hooksConfig:       hooksConfig,
		conns:             client.NewPool(),
		votingDelayMetric: prometheus.NewHistogram(prometheus.HistogramOpts{}),
	}

	for _, serverOpt := range serverOpts {
		serverOpt(s)
	}

	return s
}

// ServerOpt is a self referential option for server
type ServerOpt func(s *server)

func WithVotingDelayMetric(metric prometheus.Histogram) ServerOpt {
	return func(s *server) {
		s.votingDelayMetric = metric
	}
}
