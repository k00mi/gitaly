package remoterepo

import (
	"context"
	"fmt"

	"gitlab.com/gitlab-org/gitaly/client"
	"gitlab.com/gitlab-org/gitaly/internal/git"
	"gitlab.com/gitlab-org/gitaly/internal/helper"
	"gitlab.com/gitlab-org/gitaly/internal/storage"
	"gitlab.com/gitlab-org/gitaly/proto/go/gitalypb"
)

// Repository represents a Git repository on a different Gitaly storage
type Repository struct {
	repo   *gitalypb.Repository
	server storage.ServerInfo
	pool   *client.Pool
}

// New creates a new remote Repository from its protobuf representation.
func New(ctx context.Context, repo *gitalypb.Repository, pool *client.Pool) (Repository, error) {
	server, err := helper.ExtractGitalyServer(ctx, repo.GetStorageName())
	if err != nil {
		return Repository{}, fmt.Errorf("remote repository: %w", err)
	}

	return Repository{
		repo:   repo,
		server: server,
		pool:   pool,
	}, nil
}

// ResolveRefish will dial to the remote repository and attempt to resolve the
// refish string via the gRPC interface
func (rr Repository) ResolveRefish(ctx context.Context, ref string) (string, error) {
	cc, err := rr.pool.Dial(ctx, rr.server.Address, rr.server.Token)
	if err != nil {
		return "", err
	}

	cli := gitalypb.NewCommitServiceClient(cc)
	resp, err := cli.FindCommit(ctx, &gitalypb.FindCommitRequest{
		Repository: rr.repo,
		Revision:   []byte(ref),
	})
	if err != nil {
		return "", err
	}

	oid := resp.GetCommit().GetId()
	if oid == "" {
		return "", git.ErrReferenceNotFound
	}

	return oid, nil
}

func (rr Repository) HasBranches(ctx context.Context) (bool, error) {
	cc, err := rr.pool.Dial(ctx, rr.server.Address, rr.server.Token)
	if err != nil {
		return false, err
	}

	resp, err := gitalypb.NewRepositoryServiceClient(cc).HasLocalBranches(
		ctx, &gitalypb.HasLocalBranchesRequest{Repository: rr.repo})
	if err != nil {
		return false, fmt.Errorf("has local branches: %w", err)
	}

	return resp.Value, nil
}
