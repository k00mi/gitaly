package hook

import (
	"context"
	"io"
	"os/exec"

	"gitlab.com/gitlab-org/gitaly/internal/command"
	"gitlab.com/gitlab-org/gitaly/internal/helper"
)

func streamCommandResponse(
	ctx context.Context,
	stdin io.Reader,
	stdout, stderr io.Writer,
	c *exec.Cmd,
	env []string,
) (bool, error) {
	cmd, err := command.New(ctx, c, stdin, stdout, stderr, env...)
	if err != nil {
		return false, helper.ErrInternal(err)
	}

	err = cmd.Wait()
	if err == nil {
		return true, nil
	}

	code, ok := command.ExitStatus(err)
	if ok && code != 0 {
		return false, nil
	}

	return false, err
}
