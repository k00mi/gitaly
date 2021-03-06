package ssh

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"gitlab.com/gitlab-org/gitaly/internal/storage"
	"gitlab.com/gitlab-org/gitaly/proto/go/gitalypb"
)

var (
	defaultUploadPackRequestTimeout    = 10 * time.Minute
	defaultUploadArchiveRequestTimeout = time.Minute
)

type server struct {
	locator                     storage.Locator
	uploadPackRequestTimeout    time.Duration
	uploadArchiveRequestTimeout time.Duration
	packfileNegotiationMetrics  *prometheus.CounterVec
}

// NewServer creates a new instance of a grpc SSHServer
func NewServer(locator storage.Locator, serverOpts ...ServerOpt) gitalypb.SSHServiceServer {
	s := &server{
		locator:                     locator,
		uploadPackRequestTimeout:    defaultUploadPackRequestTimeout,
		uploadArchiveRequestTimeout: defaultUploadArchiveRequestTimeout,
		packfileNegotiationMetrics: prometheus.NewCounterVec(
			prometheus.CounterOpts{},
			[]string{"git_negotiation_feature"},
		),
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

func WithPackfileNegotiationMetrics(c *prometheus.CounterVec) ServerOpt {
	return func(s *server) {
		s.packfileNegotiationMetrics = c
	}
}
