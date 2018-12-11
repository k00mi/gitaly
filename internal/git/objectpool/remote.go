package objectpool

import (
	"bufio"
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

// hasRemote will always return a boolean value, but should only be depended on
// when the error value is nil
func (o *ObjectPool) hasRemote(ctx context.Context, name string) (bool, error) {
	cmd, err := git.Command(ctx, o, "remote")
	if err != nil {
		return false, err
	}

	found := false
	scanner := bufio.NewScanner(cmd)
	for scanner.Scan() {
		if scanner.Text() == name {
			found = true
			break
		}
	}

	return found, cmd.Wait()
}
