package hook

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os/exec"

	"gitlab.com/gitlab-org/gitaly/internal/git/hooks"
	"gitlab.com/gitlab-org/gitaly/internal/helper"
	"gitlab.com/gitlab-org/gitaly/internal/metadata/featureflag"
	"gitlab.com/gitlab-org/gitaly/proto/go/gitalypb"
	"gitlab.com/gitlab-org/gitaly/streamio"
)

func isGoPostReceiveHookUsed(env []string) bool {
	return getEnvVar(featureflag.GoPostReceiveHookEnvVar, env) == "true"
}

func postReceiveHookResponse(stream gitalypb.HookService_PostReceiveHookServer, code int32, stderr string) error {
	if err := stream.Send(&gitalypb.PostReceiveHookResponse{
		ExitStatus: &gitalypb.ExitStatus{Value: code},
		Stderr:     []byte(stderr),
	}); err != nil {
		return helper.ErrInternalf("sending response: %v", err)
	}

	return nil
}

func (s *server) PostReceiveHook(stream gitalypb.HookService_PostReceiveHookServer) error {
	firstRequest, err := stream.Recv()
	if err != nil {
		return helper.ErrInternal(err)
	}

	if err := validatePostReceiveHookRequest(firstRequest); err != nil {
		return helper.ErrInvalidArgument(err)
	}

	if !isGoPostReceiveHookUsed(firstRequest.GetEnvironmentVariables()) {
		return postReceiveHookRuby(firstRequest, stream)
	}

	hookEnv, err := hookRequestEnv(firstRequest)
	if err != nil {
		return helper.ErrInternal(err)
	}

	stdin := streamio.NewReader(func() ([]byte, error) {
		req, err := stream.Recv()
		return req.GetStdin(), err
	})
	stdout := streamio.NewWriter(func(p []byte) error { return stream.Send(&gitalypb.PostReceiveHookResponse{Stdout: p}) })
	stderr := streamio.NewWriter(func(p []byte) error { return stream.Send(&gitalypb.PostReceiveHookResponse{Stderr: p}) })

	if err := s.manager.PostReceiveHook(
		stream.Context(),
		firstRequest.Repository,
		hooks.GitPushOptions(firstRequest.GetGitPushOptions()),
		hookEnv,
		stdin,
		stdout,
		stderr,
	); err != nil {
		var exitError *exec.ExitError
		if errors.As(err, &exitError) {
			return postReceiveHookResponse(stream, int32(exitError.ExitCode()), "")
		}

		return postReceiveHookResponse(stream, 1, fmt.Sprintf("%s", err))
	}

	return postReceiveHookResponse(stream, 0, "")
}

func postReceiveHookRuby(firstRequest *gitalypb.PostReceiveHookRequest, stream gitalypb.HookService_PostReceiveHookServer) error {
	hookEnv, err := hookRequestEnv(firstRequest)
	if err != nil {
		return helper.ErrInternal(err)
	}

	hookEnv = append(hookEnv, hooks.GitPushOptions(firstRequest.GetGitPushOptions())...)

	primary, err := isPrimary(hookEnv)
	if err != nil {
		return helper.ErrInternalf("could not check role: %w", err)
	}

	stdin := streamio.NewReader(func() ([]byte, error) {
		req, err := stream.Recv()
		return req.GetStdin(), err
	})

	var status int32
	if primary {
		stdout := streamio.NewWriter(func(p []byte) error { return stream.Send(&gitalypb.PostReceiveHookResponse{Stdout: p}) })
		stderr := streamio.NewWriter(func(p []byte) error { return stream.Send(&gitalypb.PostReceiveHookResponse{Stderr: p}) })

		repoPath, err := helper.GetRepoPath(firstRequest.GetRepository())
		if err != nil {
			return helper.ErrInternal(err)
		}

		c := exec.Command(gitlabShellHook("post-receive"))
		c.Dir = repoPath

		status, err = streamCommandResponse(
			stream.Context(),
			stdin,
			stdout, stderr,
			c,
			hookEnv,
		)

		if err != nil {
			return helper.ErrInternal(err)
		}
	} else {
		_, err := io.Copy(ioutil.Discard, stdin)
		if err != nil {
			return helper.ErrInternal(err)
		}
	}

	if err := stream.SendMsg(&gitalypb.PostReceiveHookResponse{
		ExitStatus: &gitalypb.ExitStatus{Value: status},
	}); err != nil {
		return helper.ErrInternal(err)
	}

	return nil
}

func validatePostReceiveHookRequest(in *gitalypb.PostReceiveHookRequest) error {
	if in.GetRepository() == nil {
		return errors.New("repository is empty")
	}

	return nil
}
