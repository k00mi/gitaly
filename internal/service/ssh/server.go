package ssh

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"gitlab.com/gitlab-org/gitaly/internal/prometheus/metrics"
	"gitlab.com/gitlab-org/gitaly/proto/go/gitalypb"
)

var (
	defaultUploadPackRequestTimeout    = 10 * time.Minute
	defaultUploadArchiveRequestTimeout = time.Minute
)

type server struct {
	uploadPackRequestTimeout    time.Duration
	uploadArchiveRequestTimeout time.Duration
	havesMetric                 metrics.Counter
	deepensMetric               metrics.Counter
	filtersMetric               metrics.Counter
	gitalypb.UnimplementedSSHServiceServer
}

// NewServer creates a new instance of a grpc SSHServer
func NewServer(serverOpts ...ServerOpt) gitalypb.SSHServiceServer {
	s := &server{
		uploadPackRequestTimeout:    defaultUploadPackRequestTimeout,
		uploadArchiveRequestTimeout: defaultUploadArchiveRequestTimeout,
		deepensMetric:               prometheus.NewCounter(prometheus.CounterOpts{}),
		filtersMetric:               prometheus.NewCounter(prometheus.CounterOpts{}),
		havesMetric:                 prometheus.NewCounter(prometheus.CounterOpts{}),
	}

	for _, serverOpt := range serverOpts {
		serverOpt(s)
	}

	return s
}

// ServerOpt is a self referential option for server
type ServerOpt func(s *server)

// WithUploadPackRequestTimeout sets the upload pack request timeout
func WithUploadPackRequestTimeout(d time.Duration) ServerOpt {
	return func(s *server) {
		s.uploadPackRequestTimeout = d
	}
}

// WithArchiveRequestTimeout sets the upload pack request timeout
func WithArchiveRequestTimeout(d time.Duration) ServerOpt {
	return func(s *server) {
		s.uploadArchiveRequestTimeout = d
	}
}

func WithHavesMetric(c metrics.Counter) ServerOpt {
	return func(s *server) {
		s.havesMetric = c
	}
}

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
