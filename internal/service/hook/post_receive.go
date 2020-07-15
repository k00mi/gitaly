package hook

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"os/exec"
	"strings"

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

const (
	// A standard terminal window is (at least) 80 characters wide.
	terminalWidth                = 80
	gitRemoteMessagePrefixLength = len("remote: ")
	terminalMessagePadding       = 2

	// Git prefixes remote messages with "remote: ", so this width is subtracted
	// from the width available to us.
	maxMessageWidth = terminalWidth - gitRemoteMessagePrefixLength

	// Our centered text shouldn't start or end right at the edge of the window,
	// so we add some horizontal padding: 2 chars on either side.
	maxMessageTextWidth = maxMessageWidth - 2*terminalMessagePadding
)

func printMessages(messages []PostReceiveMessage, w io.Writer) error {
	for _, message := range messages {
		if _, err := w.Write([]byte("\n")); err != nil {
			return err
		}

		switch message.Type {
		case "basic":
			if _, err := w.Write([]byte(message.Message)); err != nil {
				return err
			}
		case "alert":
			if err := printAlert(message, w); err != nil {
				return err
			}
		default:
			return fmt.Errorf("invalid message type: %v", message.Type)
		}

		if _, err := w.Write([]byte("\n\n")); err != nil {
			return err
		}
	}

	return nil
}

func centerLine(b []byte) []byte {
	b = bytes.TrimSpace(b)
	linePadding := int(math.Max((float64(maxMessageWidth)-float64(len(b)))/2, 0))
	return append(bytes.Repeat([]byte(" "), linePadding), b...)
}

func printAlert(m PostReceiveMessage, w io.Writer) error {
	if _, err := w.Write(bytes.Repeat([]byte("="), maxMessageWidth)); err != nil {
		return err
	}

	if _, err := w.Write([]byte("\n\n")); err != nil {
		return err
	}

	words := strings.Fields(m.Message)

	line := bytes.NewBufferString("")

	for _, word := range words {
		if line.Len()+1+len(word) > maxMessageTextWidth {
			if _, err := w.Write(append(centerLine(line.Bytes()), '\n')); err != nil {
				return err
			}
			line.Reset()
		}

		if _, err := line.WriteString(word + " "); err != nil {
			return err
		}
	}

	if _, err := w.Write(centerLine(line.Bytes())); err != nil {
		return err
	}

	if _, err := w.Write([]byte("\n\n")); err != nil {
		return err
	}

	if _, err := w.Write(bytes.Repeat([]byte("="), maxMessageWidth)); err != nil {
		return err
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

	hookEnv = append(hookEnv, hooks.GitPushOptions(firstRequest.GetGitPushOptions())...)

	stdin := streamio.NewReader(func() ([]byte, error) {
		req, err := stream.Recv()
		return req.GetStdin(), err
	})
	stdout := streamio.NewWriter(func(p []byte) error { return stream.Send(&gitalypb.PostReceiveHookResponse{Stdout: p}) })
	stderr := streamio.NewWriter(func(p []byte) error { return stream.Send(&gitalypb.PostReceiveHookResponse{Stderr: p}) })

	changes, err := ioutil.ReadAll(stdin)
	if err != nil {
		return helper.ErrInternalf("reading stdin from request: %w", err)
	}

	glID, glRepo := getEnvVar("GL_ID", hookEnv), getEnvVar("GL_REPOSITORY", hookEnv)

	ok, messages, err := s.gitlabAPI.PostReceive(glRepo, glID, string(changes), firstRequest.GetGitPushOptions()...)
	if err != nil {
		return postReceiveHookResponse(stream, 1, fmt.Sprintf("GitLab: %v", err))
	}

	if err := printMessages(messages, stdout); err != nil {
		return helper.ErrInternalf("error writing messages to stream: %v", err)
	}

	if !ok {
		return postReceiveHookResponse(stream, 1, "")
	}

	// custom hooks execution
	repoPath, err := helper.GetRepoPath(firstRequest.GetRepository())
	if err != nil {
		return err
	}
	executor, err := newCustomHooksExecutor(repoPath, s.hooksConfig.CustomHooksDir, "post-receive")
	if err != nil {
		return helper.ErrInternalf("creating custom hooks executor: %v", err)
	}

	if err = executor(
		stream.Context(),
		nil,
		hookEnv,
		bytes.NewReader(changes),
		stdout,
		stderr,
	); err != nil {
		var exitError *exec.ExitError
		if errors.As(err, &exitError) {
			return postReceiveHookResponse(stream, int32(exitError.ExitCode()), "")
		}

		return helper.ErrInternalf("executing custom hooks: %v", err)
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
