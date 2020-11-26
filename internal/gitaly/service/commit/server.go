package commit

import (
	"gitlab.com/gitlab-org/gitaly/internal/gitaly/config"
	"gitlab.com/gitlab-org/gitaly/internal/gitaly/service/ref"
	"gitlab.com/gitlab-org/gitaly/internal/storage"
	"gitlab.com/gitlab-org/gitaly/proto/go/gitalypb"
)

type server struct {
	locator storage.Locator
	cfg     config.Cfg
}

var (
	defaultBranchName = ref.DefaultBranchName
)

// NewServer creates a new instance of a grpc CommitServiceServer
func NewServer(cfg config.Cfg, locator storage.Locator) gitalypb.CommitServiceServer {
	return &server{cfg: cfg, locator: locator}
}
