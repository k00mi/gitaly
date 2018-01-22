package repository

import (
	pb "gitlab.com/gitlab-org/gitaly-proto/go"
	"gitlab.com/gitlab-org/gitaly/internal/helper"
	"golang.org/x/net/context"
)

func (*server) IsSquashInProgress(context.Context, *pb.IsSquashInProgressRequest) (*pb.IsSquashInProgressResponse, error) {
	return nil, helper.Unimplemented
}
