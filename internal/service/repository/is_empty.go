package repository

import (
	pb "gitlab.com/gitlab-org/gitaly-proto/go"
	"gitlab.com/gitlab-org/gitaly/internal/helper"
	"golang.org/x/net/context"
)

func (*server) RepositoryIsEmpty(ctx context.Context, in *pb.RepositoryIsEmptyRequest) (*pb.RepositoryIsEmptyResponse, error) {
	return nil, helper.Unimplemented
}
