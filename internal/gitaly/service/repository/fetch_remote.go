package repository

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/grpc-ecosystem/go-grpc-middleware/logging/logrus/ctxlogrus"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"gitlab.com/gitlab-org/gitaly/internal/command"
	"gitlab.com/gitlab-org/gitaly/internal/errors"
	"gitlab.com/gitlab-org/gitaly/internal/git"
	"gitlab.com/gitlab-org/gitaly/internal/gitaly/rubyserver"
	"gitlab.com/gitlab-org/gitaly/internal/helper"
	"gitlab.com/gitlab-org/gitaly/internal/metadata/featureflag"
	"gitlab.com/gitlab-org/gitaly/proto/go/gitalypb"
	"google.golang.org/grpc/status"
)

var (
	fetchRemoteImplCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gitaly_fetch_remote_counter",
			Help: "Number of calls to FetchRemote rpc for each implementation (ruby/go)",
		},
		[]string{"impl"},
	)
)

func init() {
	prometheus.MustRegister(fetchRemoteImplCounter)
}

func (s *server) FetchRemote(ctx context.Context, req *gitalypb.FetchRemoteRequest) (*gitalypb.FetchRemoteResponse, error) {
	if featureflag.IsDisabled(ctx, featureflag.GoFetchRemote) {
		fetchRemoteImplCounter.WithLabelValues("ruby").Inc()

		client, err := s.ruby.RepositoryServiceClient(ctx)
		if err != nil {
			return nil, err
		}

		clientCtx, err := rubyserver.SetHeaders(ctx, s.locator, req.GetRepository())
		if err != nil {
			return nil, err
		}

		return client.FetchRemote(clientCtx, req)
	}

	fetchRemoteImplCounter.WithLabelValues("go").Inc()

	if err := s.validateFetchRemoteRequest(req); err != nil {
		return nil, err
	}

	var stderr bytes.Buffer
	opts := git.FetchOpts{Stderr: &stderr, Force: req.Force, Prune: true, Tags: git.FetchOptsTagsAll}

	if req.GetNoTags() {
		opts.Tags = git.FetchOptsTagsNone
	}

	repo := git.NewRepository(req.GetRepository())
	params := req.GetRemoteParams()
	remoteName := req.GetRemote()

	if params != nil {
		remoteName = params.GetName()
		remoteURL := params.GetUrl()
		refspecs := s.getRefspecs(params.GetMirrorRefmaps())

		if err := s.setRemote(ctx, repo, remoteName, remoteURL); err != nil {
			return nil, fmt.Errorf("set remote: %w", err)
		}

		defer func(parentCtx context.Context) {
			ctx, cancel := context.WithCancel(command.SuppressCancellation(parentCtx))
			defer cancel()

			// we pass context as it may be overridden in case timeout is set for the call
			if err := s.removeRemote(ctx, repo, remoteName); err != nil {
				ctxlogrus.Extract(ctx).WithError(err).WithFields(logrus.Fields{
					"remote":  remoteName,
					"storage": req.GetRepository().GetStorageName(),
					"path":    req.GetRepository().GetRelativePath(),
				}).Error("removal of remote failed")
			}
		}(ctx)

		for _, refspec := range refspecs {
			opts.Global = append(opts.Global, git.ConfigPair{Key: "remote." + remoteName + ".fetch", Value: refspec})
		}

		opts.Global = append(opts.Global,
			git.ConfigPair{Key: "remote." + remoteName + ".mirror", Value: "true"},
			git.ConfigPair{Key: "remote." + remoteName + ".prune", Value: "true"},
			git.ConfigPair{Key: "http.followRedirects", Value: "false"},
		)

		if params.GetHttpAuthorizationHeader() != "" {
			client, err := s.ruby.RepositoryServiceClient(ctx)
			if err != nil {
				return nil, err
			}

			clientCtx, err := rubyserver.SetHeaders(ctx, s.locator, req.GetRepository())
			if err != nil {
				return nil, err
			}

			// currently it is only possible way to set config value without exposing it to outside (won't be listed in 'ps')
			extraHeaderKey := "http." + remoteURL + ".extraHeader"

			if _, err := client.SetConfig(clientCtx, &gitalypb.SetConfigRequest{
				Repository: req.GetRepository(),
				Entries: []*gitalypb.SetConfigRequest_Entry{{
					Key:   extraHeaderKey,
					Value: &gitalypb.SetConfigRequest_Entry_ValueStr{ValueStr: "Authorization: " + params.GetHttpAuthorizationHeader()},
				}},
			}); err != nil {
				return nil, helper.ErrInternal(fmt.Errorf("set extra header: %w", err))
			}

			defer func() {
				ctx, cancel := context.WithCancel(command.SuppressCancellation(clientCtx))
				defer cancel()

				// currently it is only possible way to set config value without exposing it to outside (won't be listed in 'ps')
				if _, err := client.DeleteConfig(ctx, &gitalypb.DeleteConfigRequest{
					Repository: req.Repository,
					Keys:       []string{extraHeaderKey},
				}); err != nil {
					ctxlogrus.Extract(ctx).WithError(err).WithFields(logrus.Fields{
						"remote":  remoteName,
						"storage": req.GetRepository().GetStorageName(),
						"path":    req.GetRepository().GetRelativePath(),
					}).Error("removal of extra header config failed")
				}
			}()
		}
	} else {
		envGitSSHCommand, cleanup, err := s.configureSSH(ctx, req.GetSshKey(), req.GetKnownHosts())
		if err != nil {
			return nil, err
		}
		defer cleanup()

		opts.Env = append(opts.Env, envGitSSHCommand)
	}

	if req.GetTimeout() > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(req.GetTimeout())*time.Second)
		defer cancel()
	}

	if err := repo.FetchRemote(ctx, remoteName, opts); err != nil {
		if _, ok := status.FromError(err); ok {
			// this check is used because of internal call to alternates.PathAndEnv
			// which may return gRPC status as an error result
			return nil, err
		}

		errMsg := stderr.String()
		if errMsg != "" {
			return nil, fmt.Errorf("fetch remote: %q: %w", errMsg, err)
		}

		return nil, fmt.Errorf("fetch remote: %w", err)
	}

	return &gitalypb.FetchRemoteResponse{}, nil
}

func (s *server) validateFetchRemoteRequest(req *gitalypb.FetchRemoteRequest) error {
	if req.GetRepository() == nil {
		return helper.ErrInvalidArgument(errors.ErrEmptyRepository)
	}

	params := req.GetRemoteParams()
	if params == nil {
		remote := req.GetRemote()
		if strings.TrimSpace(remote) == "" {
			return helper.ErrInvalidArgument(fmt.Errorf(`blank or empty "remote": %q`, remote))
		}
		return nil
	}

	remoteURL, err := url.ParseRequestURI(params.GetUrl())
	if err != nil {
		return helper.ErrInvalidArgument(fmt.Errorf(`invalid "remote_params.url": %q: %w`, params.GetUrl(), err))
	}

	if remoteURL.Host == "" {
		return helper.ErrInvalidArgumentf(`invalid "remote_params.url": %q: no host`, params.GetUrl())
	}

	remote := params.GetName()
	if strings.TrimSpace(remote) == "" {
		return helper.ErrInvalidArgument(fmt.Errorf(`blank or empty "remote_params.name": %q`, remote))
	}

	return nil
}

func (s *server) getRefspecs(refmaps []string) []string {
	refspecs := make([]string, 0, len(refmaps))

	for _, refmap := range refmaps {
		switch refmap {
		case "all_refs":
			// with `all_refs`, the repository is equivalent to the result of `git clone --mirror`
			refspecs = append(refspecs, "+refs/*:refs/*")
		case "heads":
			refspecs = append(refspecs, "+refs/heads/*:refs/heads/*")
		case "tags":
			refspecs = append(refspecs, "+refs/tags/*:refs/tags/*")
		default:
			refspecs = append(refspecs, refmap)
		}
	}
	return refspecs
}

func (s *server) setRemote(ctx context.Context, repo *git.LocalRepository, name, url string) error {
	if err := repo.Remote().Remove(ctx, name); err != nil {
		if err != git.ErrNotFound {
			return fmt.Errorf("remove remote: %w", err)
		}
	}

	if err := repo.Remote().Add(ctx, name, url, git.RemoteAddOpts{}); err != nil {
		return fmt.Errorf("add remote: %w", err)
	}

	return nil
}

func (s *server) removeRemote(ctx context.Context, repo *git.LocalRepository, name string) error {
	if err := repo.Remote().Remove(ctx, name); err != nil {
		if err != git.ErrNotFound {
			return fmt.Errorf("remove remote: %w", err)
		}
	}

	return nil
}

func (s *server) configureSSH(ctx context.Context, sshKey, knownHosts string) (string, func(), error) {
	sshKeyPresent := strings.TrimSpace(sshKey) != ""
	knownHostsPresent := strings.TrimSpace(knownHosts) != ""

	if !sshKeyPresent && !knownHostsPresent {
		return "", func() {}, nil
	}

	tmpdir, err := ioutil.TempDir("", "")
	if err != nil {
		return "", nil, err
	}

	cleanup := func() {
		if err := os.RemoveAll(tmpdir); err != nil {
			ctxlogrus.Extract(ctx).WithError(err).Error("failed to remove tmp directory with ssh key/config")
		}
	}

	var conf []string

	if sshKeyPresent {
		identityFilePath := filepath.Join(tmpdir, "gitlab-shell-key-file")

		if err := ioutil.WriteFile(identityFilePath, []byte(sshKey), 0400); err != nil {
			cleanup()
			return "", nil, err
		}
		conf = append(conf, "-oIdentitiesOnly=yes", "-oIdentityFile="+identityFilePath)
	}

	if knownHostsPresent {
		hostsFilePath := filepath.Join(tmpdir, "gitlab-shell-known-hosts")

		if err := ioutil.WriteFile(hostsFilePath, []byte(knownHosts), 0400); err != nil {
			cleanup()
			return "", nil, err
		}
		conf = append(conf, "-oStrictHostKeyChecking=yes", "-oUserKnownHostsFile="+hostsFilePath)
	}

	return "GIT_SSH_COMMAND=ssh " + strings.Join(conf, " "), cleanup, nil
}
