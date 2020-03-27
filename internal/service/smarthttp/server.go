package smarthttp

import (
	"github.com/prometheus/client_golang/prometheus"
	"gitlab.com/gitlab-org/gitaly/internal/prometheus/metrics"
	"gitlab.com/gitlab-org/gitaly/proto/go/gitalypb"
)

type server struct {
	deepensMetric metrics.Counter
	filtersMetric metrics.Counter
	havesMetric   metrics.Counter
	gitalypb.UnimplementedSmartHTTPServiceServer
}

// NewServer creates a new instance of a grpc SmartHTTPServer
func NewServer(serverOpts ...ServerOpt) gitalypb.SmartHTTPServiceServer {
	s := &server{
		deepensMetric: prometheus.NewCounter(prometheus.CounterOpts{}),
		filtersMetric: prometheus.NewCounter(prometheus.CounterOpts{}),
		havesMetric:   prometheus.NewCounter(prometheus.CounterOpts{}),
	}

	for _, serverOpt := range serverOpts {
		serverOpt(s)
	}

	return s
}

// ServerOpt is a self referential option for server
type ServerOpt func(s *server)

func WithDeepensMetric(c metrics.Counter) ServerOpt {
	return func(s *server) {
		s.deepensMetric = c
	}
}

func WithFiltersMetric(c metrics.Counter) ServerOpt {
	return func(s *server) {
		s.filtersMetric = c
	}
}

func WithHavesMetric(c metrics.Counter) ServerOpt {
	return func(s *server) {
		s.havesMetric = c
	}
}
