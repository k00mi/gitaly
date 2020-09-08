package hook

import (
	"gitlab.com/gitlab-org/gitaly/internal/gitaly/config"
	gitalyhook "gitlab.com/gitlab-org/gitaly/internal/gitaly/hook"
	"gitlab.com/gitlab-org/gitaly/proto/go/gitalypb"
)

type server struct {
	manager     *gitalyhook.Manager
	hooksConfig config.Hooks
	gitlabAPI   gitalyhook.GitlabAPI
}

// NewServer creates a new instance of a gRPC namespace server
func NewServer(manager *gitalyhook.Manager, gitlab gitalyhook.GitlabAPI, hooksConfig config.Hooks) gitalypb.HookServiceServer {
	return &server{
		manager:     manager,
		gitlabAPI:   gitlab,
		hooksConfig: hooksConfig,
	}
}
