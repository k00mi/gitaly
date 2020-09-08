package hook

import (
	"crypto/sha1"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os/exec"
	"path/filepath"
	"strings"

	"gitlab.com/gitlab-org/gitaly/internal/git/hooks"
	"gitlab.com/gitlab-org/gitaly/internal/gitaly/config"
	"gitlab.com/gitlab-org/gitaly/internal/gitlabshell"
	"gitlab.com/gitlab-org/gitaly/internal/helper"
	"gitlab.com/gitlab-org/gitaly/internal/metadata/featureflag"
	"gitlab.com/gitlab-org/gitaly/internal/praefect/metadata"
	"gitlab.com/gitlab-org/gitaly/proto/go/gitalypb"
	"gitlab.com/gitlab-org/gitaly/streamio"
)

type hookRequest interface {
	GetEnvironmentVariables() []string
	GetRepository() *gitalypb.Repository
}

type prePostRequest interface {
	hookRequest
	GetGitPushOptions() []string
}

func hookRequestEnv(req hookRequest) ([]string, error) {
	gitlabshellEnv, err := gitlabshell.Env()
	if err != nil {
		return nil, err
	}
	return append(gitlabshellEnv, req.GetEnvironmentVariables()...), nil
}

func preReceiveEnv(req prePostRequest) ([]string, error) {
	env, err := hookRequestEnv(req)
	if err != nil {
		return nil, err
	}

	env = append(env, hooks.GitPushOptions(req.GetGitPushOptions())...)

	return env, nil
}

func gitlabShellHook(hookName string) string {
	return filepath.Join(config.Config.Ruby.Dir, "gitlab-shell", "hooks", hookName)
}

func isPrimary(env []string) (bool, error) {
	tx, err := metadata.TransactionFromEnv(env)
	if err != nil {
		if errors.Is(err, metadata.ErrTransactionNotFound) {
			// If there is no transaction, then we only ever write
			// to the primary. Thus, we return true.
			return true, nil
		}
		return false, err
	}

	return tx.Primary, nil
}

func (s *server) PreReceiveHook(stream gitalypb.HookService_PreReceiveHookServer) error {
	firstRequest, err := stream.Recv()
	if err != nil {
		return helper.ErrInternalf("receiving first request: %w", err)
	}

	if err := validatePreReceiveHookRequest(firstRequest); err != nil {
		return helper.ErrInvalidArgument(err)
	}
	reqEnvVars := firstRequest.GetEnvironmentVariables()
	repository := firstRequest.GetRepository()

	if !useGoPreReceiveHook(reqEnvVars) {
		return s.preReceiveHookRuby(firstRequest, stream)
	}

	stdin := streamio.NewReader(func() ([]byte, error) {
		req, err := stream.Recv()
		return req.GetStdin(), err
	})
	stdout := streamio.NewWriter(func(p []byte) error { return stream.Send(&gitalypb.PreReceiveHookResponse{Stdout: p}) })
	stderr := streamio.NewWriter(func(p []byte) error { return stream.Send(&gitalypb.PreReceiveHookResponse{Stderr: p}) })

	env, err := preReceiveEnv(firstRequest)
	if err != nil {
		return fmt.Errorf("getting env vars from request: %v", err)
	}

	if err := s.manager.PreReceiveHook(
		stream.Context(),
		repository,
		append(env, reqEnvVars...),
		stdin,
		stdout,
		stderr,
	); err != nil {
		var exitError *exec.ExitError
		if errors.As(err, &exitError) {
			return preReceiveHookResponse(stream, int32(exitError.ExitCode()), "")
		}

		return preReceiveHookResponse(stream, 1, fmt.Sprintf("%s", err))
	}

	return preReceiveHookResponse(stream, 0, "")
}

func validatePreReceiveHookRequest(in *gitalypb.PreReceiveHookRequest) error {
	if in.GetRepository() == nil {
		return errors.New("repository is empty")
	}

	return nil
}

func useGoPreReceiveHook(env []string) bool {
	return getEnvVar(featureflag.GoPreReceiveHookEnvVar, env) == "true"
}

func preReceiveHookResponse(stream gitalypb.HookService_PreReceiveHookServer, code int32, stderr string) error {
	if err := stream.Send(&gitalypb.PreReceiveHookResponse{
		ExitStatus: &gitalypb.ExitStatus{Value: code},
		Stderr:     []byte(stderr),
	}); err != nil {
		return helper.ErrInternalf("sending response: %v", err)
	}

	return nil
}

func (s *server) preReceiveHookRuby(firstRequest *gitalypb.PreReceiveHookRequest, stream gitalypb.HookService_PreReceiveHookServer) error {
	referenceUpdatesHasher := sha1.New()

	stdin := streamio.NewReader(func() ([]byte, error) {
		req, err := stream.Recv()
		if err != nil {
			return nil, err
		}

		stdin := req.GetStdin()
		if _, err := referenceUpdatesHasher.Write(stdin); err != nil {
			return stdin, err
		}

		return stdin, nil
	})

	env, err := preReceiveEnv(firstRequest)
	if err != nil {
		return helper.ErrInternal(err)
	}

	primary, err := isPrimary(env)
	if err != nil {
		return helper.ErrInternalf("could not check role: %w", err)
	}

	var status int32

	// Only the primary should execute hooks.
	if primary {
		stdout := streamio.NewWriter(func(p []byte) error { return stream.Send(&gitalypb.PreReceiveHookResponse{Stdout: p}) })
		stderr := streamio.NewWriter(func(p []byte) error { return stream.Send(&gitalypb.PreReceiveHookResponse{Stderr: p}) })

		repoPath, err := helper.GetRepoPath(firstRequest.GetRepository())
		if err != nil {
			return helper.ErrInternal(err)
		}

		c := exec.Command(gitlabShellHook("pre-receive"))
		c.Dir = repoPath

		status, err = streamCommandResponse(
			stream.Context(),
			stdin,
			stdout, stderr,
			c,
			env,
		)
		if err != nil {
			return helper.ErrInternal(err)
		}
	} else {
		// We need to read all of stdin on secondaries so that we
		// arrive at the same hash as the primary.
		_, err := io.Copy(ioutil.Discard, stdin)
		if err != nil {
			return helper.ErrInternal(err)
		}
	}

	if err := s.manager.VoteOnTransaction(stream.Context(), referenceUpdatesHasher.Sum(nil), env); err != nil {
		return helper.ErrInternalf("error voting on transaction: %w", err)
	}

	if err := stream.SendMsg(&gitalypb.PreReceiveHookResponse{
		ExitStatus: &gitalypb.ExitStatus{Value: status},
	}); err != nil {
		return helper.ErrInternal(err)
	}

	return nil
}

func getEnvVar(key string, vars []string) string {
	for _, varPair := range vars {
		kv := strings.SplitN(varPair, "=", 2)
		if kv[0] == key {
			return kv[1]
		}
	}

	return ""
}
