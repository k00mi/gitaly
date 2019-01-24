package remote

import (
	"context"

	"gitlab.com/gitlab-org/gitaly-proto/go/gitalypb"
)

func (s *server) ListRemotes(ctx context.Context, in *gitalypb.ListRemotesRequest) (*gitalypb.ListRemotesResponse, error) {
	return nil, nil
}

