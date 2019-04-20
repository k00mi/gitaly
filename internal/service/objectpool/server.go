package objectpool

import (
	"context"
	"errors"

	"gitlab.com/gitlab-org/gitaly-proto/go/gitalypb"
)

type server struct{}

// NewServer creates a new instance of a gRPC repo server
func NewServer() gitalypb.ObjectPoolServiceServer {
	return &server{}
}

func (s *server) DisconnectGitAlternates(ctx context.Context, req *gitalypb.DisconnectGitAlternatesRequest) (*gitalypb.DisconnectGitAlternatesResponse, error) {
	return nil, errors.New("not implemented yet")
}
