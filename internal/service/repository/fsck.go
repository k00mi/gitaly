package repository

import (
	"gitlab.com/gitlab-org/gitaly/internal/helper"

	pb "gitlab.com/gitlab-org/gitaly-proto/go"

	"golang.org/x/net/context"
)

func (s *server) Fsck(ctx context.Context, req *pb.FsckRequest) (*pb.FsckResponse, error) {
	return nil, helper.Unimplemented
}
