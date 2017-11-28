package operations

import (
	"golang.org/x/net/context"

	"gitlab.com/gitlab-org/gitaly/internal/helper"

	pb "gitlab.com/gitlab-org/gitaly-proto/go"
)

func (s *server) UserCherryPick(ctx context.Context, in *pb.UserCherryPickRequest) (*pb.UserCherryPickResponse, error) {
	return nil, helper.Unimplemented
}
