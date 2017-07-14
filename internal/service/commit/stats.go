package commit

import (
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"

	pb "gitlab.com/gitlab-org/gitaly-proto/go"
)

func (server) CommitStats(ctx context.Context, in *pb.CommitStatsRequest) (*pb.CommitStatsResponse, error) {
	return nil, grpc.Errorf(codes.Unimplemented, "not implemented")
}
