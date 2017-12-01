package repository

import (
	"gitlab.com/gitlab-org/gitaly/internal/helper"

	pb "gitlab.com/gitlab-org/gitaly-proto/go"

	"golang.org/x/net/context"
)

func (s *server) WriteRef(ctx context.Context, req *pb.WriteRefRequest) (*pb.WriteRefResponse, error) {
	return nil, helper.Unimplemented
}
