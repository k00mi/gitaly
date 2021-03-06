package repository

import (
	"bytes"
	"context"

	"gitlab.com/gitlab-org/gitaly/internal/git"
	"gitlab.com/gitlab-org/gitaly/internal/git/alternates"
	"gitlab.com/gitlab-org/gitaly/proto/go/gitalypb"
)

func (s *server) Fsck(ctx context.Context, req *gitalypb.FsckRequest) (*gitalypb.FsckResponse, error) {
	var stdout, stderr bytes.Buffer

	repo := req.GetRepository()
	repoPath, err := s.locator.GetRepoPath(repo)
	if err != nil {
		return nil, err
	}

	env := alternates.Env(repoPath, repo.GetGitObjectDirectory(), repo.GetGitAlternateObjectDirectories())

	cmd, err := git.SafeBareCmd(ctx, env,
		[]git.GlobalOption{git.ValueFlag{"--git-dir", repoPath}},
		git.SubCmd{Name: "fsck"},
		git.WithStdout(&stdout),
		git.WithStderr(&stderr),
	)
	if err != nil {
		return nil, err
	}

	if err = cmd.Wait(); err != nil {
		return &gitalypb.FsckResponse{Error: append(stdout.Bytes(), stderr.Bytes()...)}, nil
	}

	return &gitalypb.FsckResponse{}, nil
}
