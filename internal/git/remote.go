package git

import (
	"context"
	"fmt"

	"gitlab.com/gitlab-org/gitaly/client"
	"gitlab.com/gitlab-org/gitaly/internal/helper"
	"gitlab.com/gitlab-org/gitaly/internal/storage"
	"gitlab.com/gitlab-org/gitaly/proto/go/gitalypb"
)

// remoteRepository represents a Git repository on a different Gitaly storage
type remoteRepository struct {
	UnimplementedRepo
	repo   *gitalypb.Repository
	server storage.ServerInfo
	pool   *client.Pool
}

// NewRepository creates a new remote Repository from its protobuf representation.
func NewRemoteRepository(ctx context.Context, repo *gitalypb.Repository, pool *client.Pool) (Repository, error) {
	server, err := helper.ExtractGitalyServer(ctx, repo.GetStorageName())
	if err != nil {
		return nil, fmt.Errorf("remote repository: %w", err)
	}

	return remoteRepository{
		repo:   repo,
		server: server,
		pool:   pool,
	}, nil
}

// ResolveRefish will dial to the remote repository and attempt to resolve the
// refish string via the gRPC interface
func (rr remoteRepository) ResolveRefish(ctx context.Context, ref string, verify bool) (string, error) {
	if verify {
		return "", ErrUnimplemented
	}

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

	return resp.GetCommit().GetId(), nil
}
