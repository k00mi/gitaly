package objectpool

import (
	"context"

	"gitlab.com/gitlab-org/gitaly-proto/go/gitalypb"
	"gitlab.com/gitlab-org/gitaly/internal/helper"
)

func (s *server) LinkRepositoryToObjectPool(ctx context.Context, req *gitalypb.LinkRepositoryToObjectPoolRequest) (*gitalypb.LinkRepositoryToObjectPoolResponse, error) {
	return nil, helper.Unimplemented
}

func (s *server) UnlinkRepositoryFromObjectPool(ctx context.Context, req *gitalypb.UnlinkRepositoryFromObjectPoolRequest) (*gitalypb.UnlinkRepositoryFromObjectPoolResponse, error) {
	return nil, helper.Unimplemented
}
