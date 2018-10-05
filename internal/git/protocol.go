package git

import (
	"github.com/prometheus/client_golang/prometheus"
)

// RequestWithGitProtocol holds requests that respond to GitProtocol
type RequestWithGitProtocol interface {
	GetGitProtocol() string
}

var (
	gitProtocolRequests = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gitaly_protocol_requests_total",
			Help: "Counter of Git protocol requests",
		},
		[]string{"protocol"},
	)
)

func init() {
	prometheus.MustRegister(gitProtocolRequests)
}
