package repository

import (
	"context"

	"gitlab.com/gitlab-org/gitaly-proto/go/gitalypb"
	"gitlab.com/gitlab-org/gitaly/internal/git"
	"gitlab.com/gitlab-org/gitaly/internal/helper"
)

func (s *server) CreateRepository(ctx context.Context, req *gitalypb.CreateRepositoryRequest) (*gitalypb.CreateRepositoryResponse, error) {
	diskPath, err := helper.GetPath(req.GetRepository())
	if err != nil {
		return nil, helper.ErrInvalidArgument(err)
	}

	cmd, err := git.CommandWithoutRepo(ctx, "init", "--bare", "--quiet", diskPath)
	if err != nil {
		return nil, helper.ErrInternal(err)
	}

	if err := cmd.Wait(); err != nil {
		return nil, helper.ErrInternal(err)
	}

	return &gitalypb.CreateRepositoryResponse{}, nil
}
