package commit

import (
	"golang.org/x/net/context"

	pb "gitlab.com/gitlab-org/gitaly-proto/go"
	"gitlab.com/gitlab-org/gitaly/internal/rubyserver"
)

func (server) CommitStats(ctx context.Context, in *pb.CommitStatsRequest) (*pb.CommitStatsResponse, error) {
	client, err := rubyserver.CommitServiceClient(ctx)
	if err != nil {
		return nil, err
	}

	clientCtx, err := rubyserver.SetHeaders(ctx, in.GetRepository())
	if err != nil {
		return nil, err
	}

	return client.CommitStats(clientCtx, in)
}
