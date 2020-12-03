package git

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/golang/protobuf/jsonpb"
	"gitlab.com/gitlab-org/gitaly/internal/git/hooks"
	"gitlab.com/gitlab-org/gitaly/internal/gitaly/config"
	"gitlab.com/gitlab-org/gitaly/internal/gitlabshell"
	"gitlab.com/gitlab-org/gitaly/internal/metadata/featureflag"
	"gitlab.com/gitlab-org/gitaly/internal/praefect/metadata"
	"gitlab.com/gitlab-org/gitaly/proto/go/gitalypb"
)

const (
	// ErrHooksPayloadNotFound is the name of the environment variable used
	// to hold the hooks payload.
	ErrHooksPayloadNotFound = "GITALY_HOOKS_PAYLOAD"
)

var (
	jsonpbMarshaller   = &jsonpb.Marshaler{}
	jsonpbUnmarshaller = &jsonpb.Unmarshaler{}

	// ErrPayloadNotFound is returned by HooksPayloadFromEnv if the given
	// environment variables don't have a hooks payload.
	ErrPayloadNotFound = errors.New("no hooks payload found in environment")
)

// HooksPayload holds parameters required for all hooks.
type HooksPayload struct {
	// Repo is the repository in which the hook is running.
	Repo *gitalypb.Repository `json:"-"`
	// BinDir is the binary directory of Gitaly.
	BinDir string `json:"binary_directory"`
	// InternalSocket is the path to Gitaly's internal socket.
	InternalSocket string `json:"internal_socket"`
	// InternalSocketToken is the token required to authenticate with
	// Gitaly's internal socket.
	InternalSocketToken string `json:"internal_socket_token"`
}

// jsonHooksPayload wraps the HooksPayload such that we can manually encode the
// repository protobuf message.
type jsonHooksPayload struct {
	HooksPayload
	Repo string `json:"repository"`
}

// NewHooksPayload creates a new hooks payload which can then be encoded and
// passed to Git hooks.
func NewHooksPayload(cfg config.Cfg, repo *gitalypb.Repository) HooksPayload {
	return HooksPayload{
		Repo:                repo,
		BinDir:              cfg.BinDir,
		InternalSocket:      cfg.GitalyInternalSocketPath(),
		InternalSocketToken: cfg.Auth.Token,
	}
}

func lookupEnv(envs []string, key string) (string, bool) {
	for _, env := range envs {
		kv := strings.SplitN(env, "=", 2)
		if len(kv) != 2 {
			continue
		}

		if kv[0] == key {
			return kv[1], true
		}
	}

	return "", false
}

// HooksPayloadFromEnv extracts the HooksPayload from the given environment
// variables. If no HooksPayload exists, it returns a ErrPayloadNotFound
// error.
func HooksPayloadFromEnv(envs []string) (HooksPayload, error) {
	encoded, ok := lookupEnv(envs, ErrHooksPayloadNotFound)
	if !ok {
		return fallbackPayloadFromEnv(envs)
	}

	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return HooksPayload{}, err
	}

	var jsonPayload jsonHooksPayload
	if err := json.Unmarshal(decoded, &jsonPayload); err != nil {
		return HooksPayload{}, err
	}

	var repo gitalypb.Repository
	err = jsonpbUnmarshaller.Unmarshal(strings.NewReader(jsonPayload.Repo), &repo)
	if err != nil {
		return HooksPayload{}, err
	}

	payload := jsonPayload.HooksPayload
	payload.Repo = &repo

	return payload, nil
}

// fallbackPayloadFromEnv is a compatibility layer for upgrades where it may
// happen that the new gitaly-hooks binary has already been installed while the
// old server is still running. As a result, there'd be some time where we
// don't yet have GITALY_HOOKS_PAYLOAD set up in our environment, and we'll
// have to cope with this. We thus fall back to the old envvars here.
func fallbackPayloadFromEnv(envs []string) (HooksPayload, error) {
	var payload HooksPayload

	marshalledRepo, ok := lookupEnv(envs, "GITALY_REPOSITORY")
	if !ok {
		return payload, ErrPayloadNotFound
	}

	var repo gitalypb.Repository
	if err := jsonpbUnmarshaller.Unmarshal(strings.NewReader(marshalledRepo), &repo); err != nil {
		return HooksPayload{}, err
	}
	payload.Repo = &repo

	binDir, ok := lookupEnv(envs, "GITALY_BIN_DIR")
	if !ok {
		return payload, ErrPayloadNotFound
	}
	payload.BinDir = binDir

	socket, ok := lookupEnv(envs, "GITALY_SOCKET")
	if !ok {
		return payload, ErrPayloadNotFound
	}
	payload.InternalSocket = socket

	// The token may be unset, which is fine if no authentication is
	// required.
	token, _ := lookupEnv(envs, "GITALY_TOKEN")
	payload.InternalSocketToken = token

	return payload, nil
}

// Env encodes the given HooksPayload into an environment variable.
func (p HooksPayload) Env() (string, error) {
	repo, err := jsonpbMarshaller.MarshalToString(p.Repo)
	if err != nil {
		return "", err
	}

	jsonPayload := jsonHooksPayload{p, repo}
	marshalled, err := json.Marshal(jsonPayload)
	if err != nil {
		return "", err
	}

	encoded := base64.StdEncoding.EncodeToString(marshalled)

	return fmt.Sprintf("%s=%s", ErrHooksPayloadNotFound, encoded), nil
}

// WithRefTxHook returns an option that populates the safe command with the
// environment variables necessary to properly execute a reference hook for
// repository changes that may possibly update references
func WithRefTxHook(ctx context.Context, repo *gitalypb.Repository, cfg config.Cfg) CmdOpt {
	return func(cc *cmdCfg) error {
		if repo == nil {
			return fmt.Errorf("missing repo: %w", ErrInvalidArg)
		}

		rfEnvs, err := refHookEnv(ctx, repo, cfg)
		if err != nil {
			return fmt.Errorf("ref hook env var: %w", err)
		}

		cc.env = append(cc.env, rfEnvs...)
		cc.globals = append(cc.globals, ValueFlag{"-c", fmt.Sprintf("core.hooksPath=%s", hooks.Path(cfg))})
		cc.refHookConfigured = true

		return nil
	}
}

// refHookEnv returns all env vars required by the reference transaction hook
func refHookEnv(ctx context.Context, repo *gitalypb.Repository, cfg config.Cfg) ([]string, error) {
	payload, err := NewHooksPayload(cfg, repo).Env()
	if err != nil {
		return nil, err
	}

	return []string{
		payload,
		"GITALY_BIN_DIR=" + cfg.BinDir,
		fmt.Sprintf("%s=true", featureflag.ReferenceTransactionHookEnvVar),
	}, nil
}

// ReceivePackRequest abstracts away the different requests that end up
// spawning git-receive-pack.
type ReceivePackRequest interface {
	GetGlId() string
	GetGlUsername() string
	GetGlRepository() string
	GetRepository() *gitalypb.Repository
}

// WithReceivePackHooks returns an option that populates the safe command with the environment
// variables necessary to properly execute the pre-receive, update and post-receive hooks for
// git-receive-pack(1).
func WithReceivePackHooks(ctx context.Context, cfg config.Cfg, req ReceivePackRequest, protocol string) CmdOpt {
	return func(cc *cmdCfg) error {
		env, err := receivePackHookEnv(ctx, cfg, req, protocol)
		if err != nil {
			return fmt.Errorf("receive-pack hook envvars: %w", err)
		}

		cc.env = append(cc.env, env...)
		return nil
	}
}

func receivePackHookEnv(ctx context.Context, cfg config.Cfg, req ReceivePackRequest, protocol string) ([]string, error) {
	gitlabshellEnv, err := gitlabshell.EnvFromConfig(cfg)
	if err != nil {
		return nil, err
	}

	env, err := refHookEnv(ctx, req.GetRepository(), config.Config)
	if err != nil {
		return nil, err
	}

	env = append(env,
		fmt.Sprintf("GL_ID=%s", req.GetGlId()),
		fmt.Sprintf("GL_USERNAME=%s", req.GetGlUsername()),
		fmt.Sprintf("GL_REPOSITORY=%s", req.GetGlRepository()),
		fmt.Sprintf("GL_PROJECT_PATH=%s", req.GetRepository().GetGlProjectPath()),
		fmt.Sprintf("GL_PROTOCOL=%s", protocol),
		fmt.Sprintf("%s=true", featureflag.ReferenceTransactionHookEnvVar),
	)
	env = append(env, gitlabshellEnv...)

	transaction, err := metadata.TransactionFromContext(ctx)
	if err == nil {
		praefect, err := metadata.PraefectFromContext(ctx)
		if err != nil {
			return nil, err
		}

		praefectEnv, err := praefect.Env()
		if err != nil {
			return nil, err
		}

		transactionEnv, err := transaction.Env()
		if err != nil {
			return nil, err
		}

		env = append(env, praefectEnv, transactionEnv)
	} else if !errors.Is(err, metadata.ErrTransactionNotFound) {
		return nil, err
	}

	return env, nil
}
