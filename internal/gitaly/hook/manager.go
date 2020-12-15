package hook

import (
	"context"
	"io"

	"github.com/prometheus/client_golang/prometheus"
	"gitlab.com/gitlab-org/gitaly/client"
	"gitlab.com/gitlab-org/gitaly/internal/gitaly/config"
	"gitlab.com/gitlab-org/gitaly/internal/storage"
	"gitlab.com/gitlab-org/gitaly/proto/go/gitalypb"
)

// ReferenceTransactionState is the state of the Git reference transaction. It reflects the first
// parameter of the reference-transaction hook. See githooks(1) for more information.
type ReferenceTransactionState int

const (
	// ReferenceTransactionPrepared indicates all reference updates have been queued to the
	// transaction and were locked on disk.
	ReferenceTransactionPrepared = ReferenceTransactionState(iota)
	// ReferenceTransactionCommitted indicates the reference transaction was committed and all
	// references now have their respective new value.
	ReferenceTransactionCommitted
	// ReferenceTransactionAborted indicates the transaction was aborted, no changes were
	// performed and the reference locks have been released.
	ReferenceTransactionAborted
)

// Manager is an interface providing the ability to execute Git hooks.
type Manager interface {
	// PreReceiveHook executes the pre-receive Git hook and any installed custom hooks. stdin
	// must contain all references to be updated and match the format specified in githooks(5).
	PreReceiveHook(ctx context.Context, repo *gitalypb.Repository, pushOptions, env []string, stdin io.Reader, stdout, stderr io.Writer) error

	// PostReceiveHook executes the post-receive Git hook and any installed custom hooks. stdin
	// must contain all references to be updated and match the format specified in githooks(5).
	PostReceiveHook(ctx context.Context, repo *gitalypb.Repository, pushOptions, env []string, stdin io.Reader, stdout, stderr io.Writer) error

	// UpdateHook executes the update Git hook and any installed custom hooks for the reference
	// `ref` getting updated from `oldValue` to `newValue`.
	UpdateHook(ctx context.Context, repo *gitalypb.Repository, ref, oldValue, newValue string, env []string, stdout, stderr io.Writer) error

	// ReferenceTransactionHook executes the reference-transaction Git hook. stdin must contain
	// all references to be updated and match the format specified in githooks(5).
	ReferenceTransactionHook(ctx context.Context, state ReferenceTransactionState, env []string, stdin io.Reader) error
}

// GitLabHookManager is a hook manager containing Git hook business logic. It
// uses the GitLab API to authenticate and track ongoing hook calls.
type GitLabHookManager struct {
	locator           storage.Locator
	gitlabAPI         GitlabAPI
	hooksConfig       config.Hooks
	conns             *client.Pool
	votingDelayMetric prometheus.Histogram
}

// NewManager returns a new hook manager
func NewManager(locator storage.Locator, gitlabAPI GitlabAPI, cfg config.Cfg) *GitLabHookManager {
	return &GitLabHookManager{
		locator:     locator,
		gitlabAPI:   gitlabAPI,
		hooksConfig: cfg.Hooks,
		conns:       client.NewPoolWithOptions(client.WithDialOptions(client.FailOnNonTempDialError()...)),
		votingDelayMetric: prometheus.NewHistogram(
			prometheus.HistogramOpts{
				Name:    "gitaly_hook_transaction_voting_delay_seconds",
				Help:    "Delay between calling out to transaction service and receiving a response",
				Buckets: cfg.Prometheus.GRPCLatencyBuckets,
			},
		),
	}
}

func (m *GitLabHookManager) Describe(descs chan<- *prometheus.Desc) {
	prometheus.DescribeByCollect(m, descs)
}

func (m *GitLabHookManager) Collect(metrics chan<- prometheus.Metric) {
	m.votingDelayMetric.Collect(metrics)
}
