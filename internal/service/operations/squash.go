package operations

import (
	pb "gitlab.com/gitlab-org/gitaly-proto/go"
	"gitlab.com/gitlab-org/gitaly/internal/helper"
	"golang.org/x/net/context"
)

func (*server) UserSquash(context.Context, *pb.UserSquashRequest) (*pb.UserSquashResponse, error) {
	return nil, helper.Unimplemented
}
