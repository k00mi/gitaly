package repository

import (
	"gitlab.com/gitlab-org/gitaly/client"
	"gitlab.com/gitlab-org/gitaly/internal/gitaly/config"
	"gitlab.com/gitlab-org/gitaly/internal/gitaly/rubyserver"
	"gitlab.com/gitlab-org/gitaly/internal/storage"
	"gitlab.com/gitlab-org/gitaly/proto/go/gitalypb"
)

type server struct {
	ruby       *rubyserver.Server
	conns      *client.Pool
	locator    storage.Locator
	cfg        config.Cfg
	binDir     string
	loggingCfg config.Logging
}

// NewServer creates a new instance of a gRPC repo server
func NewServer(cfg config.Cfg, rs *rubyserver.Server, locator storage.Locator) gitalypb.RepositoryServiceServer {
	return &server{
		ruby:       rs,
		locator:    locator,
		conns:      client.NewPool(),
		cfg:        cfg,
		binDir:     cfg.BinDir,
		loggingCfg: cfg.Logging,
	}
}
