package objectpool

import (
	"context"

	"gitlab.com/gitlab-org/gitaly-proto/go/gitalypb"
	"gitlab.com/gitlab-org/gitaly/internal/helper"
)

type server struct{}

// NewServer creates a new instance of a gRPC repo server
func NewServer() gitalypb.ObjectPoolServiceServer {
	return &server{}
}

func (s *server) DisconnectGitAlternates(ctx context.Context, req *gitalypb.DisconnectGitAlternatesRequest) (*gitalypb.DisconnectGitAlternatesResponse, error) {
	return nil, helper.Unimplemented
}
