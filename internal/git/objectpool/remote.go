package objectpool

import (
	"context"

	"gitlab.com/gitlab-org/gitaly/internal/git"
)

func (o *ObjectPool) removeRemote(ctx context.Context, name string) error {
	cmd, err := git.Command(ctx, o, "remote", "remove", name)
	if err != nil {
		return err
	}

	return cmd.Wait()
}
