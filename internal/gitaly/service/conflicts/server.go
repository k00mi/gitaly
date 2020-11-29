package conflicts

import (
	"gitlab.com/gitlab-org/gitaly/client"
	"gitlab.com/gitlab-org/gitaly/internal/gitaly/config"
	"gitlab.com/gitlab-org/gitaly/internal/gitaly/rubyserver"
	"gitlab.com/gitlab-org/gitaly/internal/storage"
	"gitlab.com/gitlab-org/gitaly/proto/go/gitalypb"
)

type server struct {
	ruby    *rubyserver.Server
	cfg     config.Cfg
	locator storage.Locator
	pool    *client.Pool
}

// NewServer creates a new instance of a grpc ConflictsServer
func NewServer(rs *rubyserver.Server, cfg config.Cfg, locator storage.Locator) gitalypb.ConflictsServiceServer {
	return &server{
		ruby:    rs,
		cfg:     cfg,
		locator: locator,
		pool:    client.NewPoolWithOptions(client.WithDialOptions(client.FailOnNonTempDialError()...)),
	}
}
