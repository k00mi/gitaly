package ref

import (
	"bufio"
	"bytes"
	"fmt"

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

	if repo == nil {
		err := status.Errorf(codes.InvalidArgument, "Repository does not exist")
		return nil, err
	}

	var branchName []byte

	if bytes.HasPrefix(refName, []byte("refs/heads/")) {
		branchName = bytes.TrimPrefix(refName, []byte("refs/heads/"))
	} else if bytes.HasPrefix(refName, []byte("heads/")) {
		branchName = bytes.TrimPrefix(refName, []byte("heads/"))
	} else {
		branchName = refName
	}

	cmd, err := git.Command(ctx, repo, "for-each-ref", "--format", "'%(objectname) %(refname)'", fmt.Sprintf("refs/heads/%s", string(branchName)))

	if err != nil {
		return nil, err
	}

	reader := bufio.NewReader(cmd)

	line, _, err := reader.ReadLine()

	if err != nil {
		return nil, err
	}

	var name []byte
	var revision []byte

	if len(line) > 0 {
		splitLine := bytes.Split(line[1:len(line)-1], []byte(" "))
		revision = splitLine[0]
		name = bytes.TrimPrefix(splitLine[1], []byte("refs/heads/"))
	}

	commit, err := log.GetCommit(ctx, repo, string(revision))

	if err != nil {
		return nil, err
	}

	err = cmd.Wait()

	if err != nil {
		return nil, err
	}

	return &gitalypb.FindBranchResponse{
		Branch: &gitalypb.Branch{
			Name:         name,
			TargetCommit: commit,
		},
	}, nil

}
