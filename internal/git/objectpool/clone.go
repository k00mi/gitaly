package objectpool

import (
	"context"
	"io"
	"os"
	"path"

	"gitlab.com/gitlab-org/gitaly-proto/go/gitalypb"
	"gitlab.com/gitlab-org/gitaly/internal/git"
	"gitlab.com/gitlab-org/gitaly/internal/helper"
)

// Clone a repository to a pool, without setting the alternates, is not the
// resposibility of this function.
func (o *ObjectPool) clone(ctx context.Context, repo *gitalypb.Repository) error {
	repoPath, err := helper.GetRepoPath(repo)
	if err != nil {
		return err
	}

	targetDir := o.FullPath()
	targetName := path.Base(targetDir)
	if err != nil {
		return err
	}

	cloneArgs := []string{"-C", path.Dir(targetDir), "clone", "--quiet", "--bare", "--local", repoPath, targetName}
	cmd, err := git.CommandWithoutRepo(ctx, cloneArgs...)
	if err != nil {
		return err
	}

	return cmd.Wait()
}

func (o *ObjectPool) removeRefs(ctx context.Context) error {
	pipeReader, pipeWriter := io.Pipe()
	defer pipeReader.Close()
	defer pipeWriter.Close()

	cmd, err := git.BareCommand(ctx, nil, pipeWriter, os.Stderr, nil, "--git-dir", o.FullPath(), "for-each-ref", "--format=delete %(refname)")
	if err != nil {
		return err
	}

	updateRefCmd, err := git.BareCommand(ctx, pipeReader, nil, os.Stderr, nil, "-C", o.FullPath(), "update-ref", "--stdin")
	if err != nil {
		return err
	}

	if err := cmd.Wait(); err != nil {
		return err
	}

	pipeWriter.Close()

	return updateRefCmd.Wait()
}

func (o *ObjectPool) removeHooksDir() error {
	return os.RemoveAll(path.Join(o.FullPath(), "hooks"))
}
