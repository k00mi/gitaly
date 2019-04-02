package objectpool

import (
	"bufio"
	"context"

	"gitlab.com/gitlab-org/gitaly-proto/go/gitalypb"
	"gitlab.com/gitlab-org/gitaly/internal/command"
	"gitlab.com/gitlab-org/gitaly/internal/git"
	"gitlab.com/gitlab-org/gitaly/internal/helper"
)

// FetchFromOrigin initializes the pool and fetches the objects from its origin repository
func (o *ObjectPool) FetchFromOrigin(ctx context.Context, origin *gitalypb.Repository) error {
	if err := o.Init(ctx); err != nil {
		return err
	}

	originPath, err := helper.GetPath(origin)

	if err != nil {
		return err
	}

	getRemotes, err := git.Command(ctx, o, "remote")
	if err != nil {
		return err
	}

	remoteReader := bufio.NewScanner(getRemotes)

	var originExists bool
	for remoteReader.Scan() {
		if remoteReader.Text() == "origin" {
			originExists = true
		}
	}
	if err := getRemotes.Wait(); err != nil {
		return err
	}

	var setOriginCmd *command.Command
	if originExists {
		setOriginCmd, err = git.Command(ctx, o, "remote", "set-url", "origin", originPath)
		if err != nil {
			return err
		}
	} else {
		setOriginCmd, err = git.Command(ctx, o, "remote", "add", "origin", originPath)
		if err != nil {
			return err
		}
	}

	if err := setOriginCmd.Wait(); err != nil {
		return err
	}

	fetchCmd, err := git.Command(ctx, o, "fetch", "--quiet", "origin")
	if err != nil {
		return err
	}

	if err := fetchCmd.Wait(); err != nil {
		return err
	}
	return nil
}
