package repository

import (
	pb "gitlab.com/gitlab-org/gitaly-proto/go"
	"gitlab.com/gitlab-org/gitaly/internal/helper"
	"golang.org/x/net/context"
)

func (*server) WriteConfig(context.Context, *pb.WriteConfigRequest) (*pb.WriteConfigResponse, error) {
	return nil, helper.Unimplemented
}
