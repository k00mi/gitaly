package repository

import (
	"context"

	"gitlab.com/gitlab-org/gitaly-proto/go/gitalypb"
	"gitlab.com/gitlab-org/gitaly/internal/helper"
	"gitlab.com/gitlab-org/gitaly/internal/rubyserver"
)

type server struct {
	*rubyserver.Server
}

// NewServer creates a new instance of a gRPC repo server
func NewServer(rs *rubyserver.Server) gitalypb.RepositoryServiceServer {
	return &server{Server: rs}
}

func (*server) FetchHTTPRemote(context.Context, *gitalypb.FetchHTTPRemoteRequest) (*gitalypb.FetchHTTPRemoteResponse, error) {
	return nil, helper.Unimplemented
}

func (*server) CloneFromPool(context.Context, *gitalypb.CloneFromPoolRequest) (*gitalypb.CloneFromPoolResponse, error) {
	return nil, helper.Unimplemented
}

func (*server) CloneFromPoolInternal(context.Context, *gitalypb.CloneFromPoolInternalRequest) (*gitalypb.CloneFromPoolInternalResponse, error) {
	return nil, helper.Unimplemented
}
