package service

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"gitlab.com/gitlab-org/gitaly/client"
	"gitlab.com/gitlab-org/gitaly/internal/gitaly/config"
	gitalyhook "gitlab.com/gitlab-org/gitaly/internal/gitaly/hook"
	"gitlab.com/gitlab-org/gitaly/internal/gitaly/rubyserver"
	"gitlab.com/gitlab-org/gitaly/internal/gitaly/service/blob"
	"gitlab.com/gitlab-org/gitaly/internal/gitaly/service/cleanup"
	"gitlab.com/gitlab-org/gitaly/internal/gitaly/service/commit"
	"gitlab.com/gitlab-org/gitaly/internal/gitaly/service/conflicts"
	"gitlab.com/gitlab-org/gitaly/internal/gitaly/service/diff"
	"gitlab.com/gitlab-org/gitaly/internal/gitaly/service/hook"
	"gitlab.com/gitlab-org/gitaly/internal/gitaly/service/internalgitaly"
	"gitlab.com/gitlab-org/gitaly/internal/gitaly/service/namespace"
	"gitlab.com/gitlab-org/gitaly/internal/gitaly/service/objectpool"
	"gitlab.com/gitlab-org/gitaly/internal/gitaly/service/operations"
	"gitlab.com/gitlab-org/gitaly/internal/gitaly/service/ref"
	"gitlab.com/gitlab-org/gitaly/internal/gitaly/service/remote"
	"gitlab.com/gitlab-org/gitaly/internal/gitaly/service/repository"
	"gitlab.com/gitlab-org/gitaly/internal/gitaly/service/server"
	"gitlab.com/gitlab-org/gitaly/internal/gitaly/service/smarthttp"
	"gitlab.com/gitlab-org/gitaly/internal/gitaly/service/ssh"
	"gitlab.com/gitlab-org/gitaly/internal/gitaly/service/wiki"
	"gitlab.com/gitlab-org/gitaly/internal/storage"
	"gitlab.com/gitlab-org/gitaly/proto/go/gitalypb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

var (
	once sync.Once

	smarthttpPackfileNegotiationMetrics *prometheus.CounterVec
	sshPackfileNegotiationMetrics       *prometheus.CounterVec
)

func registerMetrics(cfg config.Cfg) {
	smarthttpPackfileNegotiationMetrics = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "gitaly",
			Subsystem: "smarthttp",
			Name:      "packfile_negotiation_requests_total",
			Help:      "Total number of features used for packfile negotiations",
		},
		[]string{"git_negotiation_feature"},
	)

	sshPackfileNegotiationMetrics = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "gitaly",
			Subsystem: "ssh",
			Name:      "packfile_negotiation_requests_total",
			Help:      "Total number of features used for packfile negotiations",
		},
		[]string{"git_negotiation_feature"},
	)
}

// RegisterAll will register all the known grpc services with
// the specified grpc service instance
func RegisterAll(grpcServer *grpc.Server, cfg config.Cfg, rubyServer *rubyserver.Server, hookManager gitalyhook.Manager, locator storage.Locator, conns *client.Pool) {
	once.Do(func() {
		registerMetrics(cfg)
	})

	gitalypb.RegisterBlobServiceServer(grpcServer, blob.NewServer(rubyServer, locator))
	gitalypb.RegisterCleanupServiceServer(grpcServer, cleanup.NewServer(locator))
	gitalypb.RegisterCommitServiceServer(grpcServer, commit.NewServer(cfg, locator))
	gitalypb.RegisterDiffServiceServer(grpcServer, diff.NewServer(locator))
	gitalypb.RegisterNamespaceServiceServer(grpcServer, namespace.NewServer(locator))
	gitalypb.RegisterOperationServiceServer(grpcServer, operations.NewServer(cfg, rubyServer, hookManager, locator, conns))
	gitalypb.RegisterRefServiceServer(grpcServer, ref.NewServer(locator))
	gitalypb.RegisterRepositoryServiceServer(grpcServer, repository.NewServer(cfg, rubyServer, locator))
	gitalypb.RegisterSSHServiceServer(grpcServer, ssh.NewServer(
		locator,
		ssh.WithPackfileNegotiationMetrics(sshPackfileNegotiationMetrics),
	))
	gitalypb.RegisterSmartHTTPServiceServer(grpcServer, smarthttp.NewServer(
		locator,
		smarthttp.WithPackfileNegotiationMetrics(smarthttpPackfileNegotiationMetrics),
	))
	gitalypb.RegisterWikiServiceServer(grpcServer, wiki.NewServer(rubyServer, locator))
	gitalypb.RegisterConflictsServiceServer(grpcServer, conflicts.NewServer(rubyServer, cfg, locator))
	gitalypb.RegisterRemoteServiceServer(grpcServer, remote.NewServer(rubyServer, locator))
	gitalypb.RegisterServerServiceServer(grpcServer, server.NewServer(cfg.Storages))
	gitalypb.RegisterObjectPoolServiceServer(grpcServer, objectpool.NewServer(cfg, locator))
	gitalypb.RegisterHookServiceServer(grpcServer, hook.NewServer(cfg, hookManager))
	gitalypb.RegisterInternalGitalyServer(grpcServer, internalgitaly.NewServer(cfg.Storages))

	healthpb.RegisterHealthServer(grpcServer, health.NewServer())
}
