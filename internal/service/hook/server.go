package hook

import (
	"gitlab.com/gitlab-org/gitaly/client"
	"gitlab.com/gitlab-org/gitaly/internal/config"
	"gitlab.com/gitlab-org/gitaly/proto/go/gitalypb"
)

type server struct {
	conns       *client.Pool
	hooksConfig config.Hooks
	gitlabAPI   GitlabAPI
}

// NewServer creates a new instance of a gRPC namespace server
func NewServer(gitlab GitlabAPI, hooksConfig config.Hooks) gitalypb.HookServiceServer {
	return &server{
		gitlabAPI:   gitlab,
		hooksConfig: hooksConfig,
		conns:       client.NewPool(),
	}
}
