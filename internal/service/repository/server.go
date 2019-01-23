package repository

import (
	"gitlab.com/gitlab-org/gitaly-proto/go/gitalypb"
	"gitlab.com/gitlab-org/gitaly/internal/rubyserver"
)

type server struct {
	*rubyserver.Server
}

// NewServer creates a new instance of a gRPC repo server
func NewServer(rs *rubyserver.Server) gitalypb.RepositoryServiceServer {
	return &server{Server: rs}
}
