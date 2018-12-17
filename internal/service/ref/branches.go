package ref

import (
	"bufio"
	"bytes"
	"fmt"
	"io"

	"gitlab.com/gitlab-org/gitaly-proto/go/gitalypb"
	"gitlab.com/gitlab-org/gitaly/internal/git"
	"gitlab.com/gitlab-org/gitaly/internal/git/log"
	"gitlab.com/gitlab-org/gitaly/internal/rubyserver"
	"golang.org/x/net/context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *server) CreateBranch(ctx context.Context, req *gitalypb.CreateBranchRequest) (*gitalypb.CreateBranchResponse, error) {
	client, err := s.RefServiceClient(ctx)
	if err != nil {
		return nil, err
	}

	clientCtx, err := rubyserver.SetHeaders(ctx, req.GetRepository())
	if err != nil {
		return nil, err
	}

	return client.CreateBranch(clientCtx, req)
}

func (s *server) DeleteBranch(ctx context.Context, req *gitalypb.DeleteBranchRequest) (*gitalypb.DeleteBranchResponse, error) {
	client, err := s.RefServiceClient(ctx)
	if err != nil {
		return nil, err
	}

	clientCtx, err := rubyserver.SetHeaders(ctx, req.GetRepository())
	if err != nil {
		return nil, err
	}

	return client.DeleteBranch(clientCtx, req)
}

func (s *server) FindBranch(ctx context.Context, req *gitalypb.FindBranchRequest) (*gitalypb.FindBranchResponse, error) {
	refName := req.GetName()
	if len(refName) == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "Branch name cannot be empty")
	}
	repo := req.GetRepository()

	if bytes.HasPrefix(refName, []byte("refs/heads/")) {
		refName = bytes.TrimPrefix(refName, []byte("refs/heads/"))
	} else if bytes.HasPrefix(refName, []byte("heads/")) {
		refName = bytes.TrimPrefix(refName, []byte("heads/"))
	}

	cmd, err := git.Command(ctx, repo, "for-each-ref", "--format", "%(objectname)", fmt.Sprintf("refs/heads/%s", string(refName)))
	if err != nil {
		return nil, err
	}

	reader := bufio.NewReader(cmd)
	revision, _, err := reader.ReadLine()
	if err != nil {
		if err == io.EOF {
			return &gitalypb.FindBranchResponse{}, nil
		}
		return nil, err
	}

	commit, err := log.GetCommit(ctx, repo, string(revision))
	if err != nil {
		return nil, err
	}

	if err := cmd.Wait(); err != nil {
		return nil, err
	}

	return &gitalypb.FindBranchResponse{
		Branch: &gitalypb.Branch{
			Name:         refName,
			TargetCommit: commit,
		},
	}, nil
}
