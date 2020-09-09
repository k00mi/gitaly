package operations

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"path"
	"strings"

	"gitlab.com/gitlab-org/gitaly/internal/command"
	"gitlab.com/gitlab-org/gitaly/internal/git"
	"gitlab.com/gitlab-org/gitaly/internal/git/updateref"
	"gitlab.com/gitlab-org/gitaly/internal/gitaly/config"
	"gitlab.com/gitlab-org/gitaly/internal/gitaly/rubyserver"
	"gitlab.com/gitlab-org/gitaly/internal/gitlabshell"
	"gitlab.com/gitlab-org/gitaly/internal/helper"
	"gitlab.com/gitlab-org/gitaly/internal/helper/text"
	"gitlab.com/gitlab-org/gitaly/internal/metadata/featureflag"
	"gitlab.com/gitlab-org/gitaly/proto/go/gitalypb"
)

type preReceiveError struct {
	message string
}

func (e preReceiveError) Error() string {
	return e.message
}

type updateRefError struct {
	reference string
}

func (e updateRefError) Error() string {
	return fmt.Sprintf("Could not update %s. Please refresh and try again.", e.reference)
}

func (s *server) UserMergeBranch(bidi gitalypb.OperationService_UserMergeBranchServer) error {
	ctx := bidi.Context()

	if featureflag.IsEnabled(ctx, featureflag.GoUserMergeBranch) {
		return s.userMergeBranch(bidi)
	}

	firstRequest, err := bidi.Recv()
	if err != nil {
		return err
	}

	client, err := s.ruby.OperationServiceClient(ctx)
	if err != nil {
		return err
	}

	clientCtx, err := rubyserver.SetHeaders(ctx, firstRequest.GetRepository())
	if err != nil {
		return err
	}

	rubyBidi, err := client.UserMergeBranch(clientCtx)
	if err != nil {
		return err
	}

	if err := rubyBidi.Send(firstRequest); err != nil {
		return err
	}

	return rubyserver.ProxyBidi(
		func() error {
			request, err := bidi.Recv()
			if err != nil {
				return err
			}

			return rubyBidi.Send(request)
		},
		rubyBidi,
		func() error {
			response, err := rubyBidi.Recv()
			if err != nil {
				return err
			}

			return bidi.Send(response)
		},
	)
}

func validateMergeBranchRequest(request *gitalypb.UserMergeBranchRequest) error {
	if request.User == nil {
		return fmt.Errorf("empty user")
	}

	if len(request.Branch) == 0 {
		return fmt.Errorf("empty branch name")
	}

	if request.CommitId == "" {
		return fmt.Errorf("empty commit ID")
	}

	if len(request.Message) == 0 {
		return fmt.Errorf("empty message")
	}

	return nil
}

func (s *server) updateReferenceWithHooks(ctx context.Context, repo *gitalypb.Repository, user *gitalypb.User, reference, newrev, oldrev string) error {
	gitlabshellEnv, err := gitlabshell.Env()
	if err != nil {
		return err
	}

	env := append([]string{
		fmt.Sprintf("GL_ID=%s", user.GetGlId()),
		fmt.Sprintf("GL_USERNAME=%s", user.GetGlUsername()),
		fmt.Sprintf("GL_REPOSITORY=%s", repo.GetGlRepository()),
		fmt.Sprintf("GL_PROJECT_PATH=%s", repo.GetGlProjectPath()),
		fmt.Sprintf("GITALY_SOCKET=" + config.GitalyInternalSocketPath()),
		fmt.Sprintf("GITALY_REPO=%s", repo),
		fmt.Sprintf("GITALY_TOKEN=%s", s.cfg.Auth.Token),
	}, gitlabshellEnv...)

	stdin := bytes.NewBufferString(fmt.Sprintf("%s %s %s\n", oldrev, newrev, reference))
	var stdout, stderr bytes.Buffer

	if err := s.hookManager.PreReceiveHook(ctx, repo, env, stdin, &stdout, &stderr); err != nil {
		return preReceiveError{message: stdout.String()}
	}
	if err := s.hookManager.UpdateHook(ctx, repo, reference, oldrev, newrev, env, &stdout, &stderr); err != nil {
		return preReceiveError{message: stdout.String()}
	}

	updater, err := updateref.New(ctx, repo)
	if err != nil {
		return err
	}

	if err := updater.Update(reference, newrev, oldrev); err != nil {
		return err
	}

	if err := updater.Wait(); err != nil {
		return updateRefError{reference: reference}
	}

	if err := s.hookManager.PostReceiveHook(ctx, repo, nil, env, stdin, &stdout, &stderr); err != nil {
		return err
	}

	return nil
}

// parseRevision parses a Git revision and returns its OID.
func parseRevision(ctx context.Context, repo *gitalypb.Repository, revision string) (string, error) {
	revParse, err := git.SafeCmd(ctx, repo, nil, git.SubCmd{
		Name:  "rev-parse",
		Flags: []git.Option{git.Flag{"--verify"}},
		Args:  []string{revision},
	})
	if err != nil {
		return "", err
	}

	var stdout bytes.Buffer
	if _, err := io.Copy(&stdout, revParse); err != nil {
		return "", err
	}

	if err := revParse.Wait(); err != nil {
		return "", err
	}

	return text.ChompBytes(stdout.Bytes()), nil
}

func (s *server) userMergeBranch(stream gitalypb.OperationService_UserMergeBranchServer) error {
	ctx := stream.Context()

	firstRequest, err := stream.Recv()
	if err != nil {
		return err
	}

	if err := validateMergeBranchRequest(firstRequest); err != nil {
		return helper.ErrInvalidArgument(err)
	}

	revision, err := parseRevision(ctx, firstRequest.Repository, string(firstRequest.Branch))
	if err != nil {
		return err
	}

	repoPath, err := s.locator.GetPath(firstRequest.Repository)
	if err != nil {
		return err
	}

	binary := path.Join(s.cfg.BinDir, "gitaly-git2go")
	args := []string{
		"merge",
		"-repository", repoPath,
		"-author-name", string(firstRequest.User.Name),
		"-author-mail", string(firstRequest.User.Email),
		"-message", string(firstRequest.Message),
		"-ours", firstRequest.CommitId,
		"-theirs", revision,
	}

	var stderr, stdout bytes.Buffer
	mergeCommand, err := command.New(ctx, exec.Command(binary, args...), nil, &stdout, &stderr)
	if err != nil {
		return err
	}

	if err := mergeCommand.Wait(); err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			return fmt.Errorf("%w: %s", err, stderr.String())
		}
		return err
	}

	mergeCommit := text.ChompBytes(stdout.Bytes())

	if err := stream.Send(&gitalypb.UserMergeBranchResponse{
		CommitId: mergeCommit,
	}); err != nil {
		return err
	}

	secondRequest, err := stream.Recv()
	if err != nil {
		return err
	}
	if !secondRequest.Apply {
		return helper.ErrPreconditionFailedf("merge aborted by client")
	}

	branch := "refs/heads/" + text.ChompBytes(firstRequest.Branch)
	if err := s.updateReferenceWithHooks(ctx, firstRequest.Repository, firstRequest.User, branch, mergeCommit, revision); err != nil {
		var preReceiveError preReceiveError
		var updateRefError updateRefError

		if errors.As(err, &preReceiveError) {
			err = stream.Send(&gitalypb.UserMergeBranchResponse{
				PreReceiveError: preReceiveError.message,
			})
		} else if errors.As(err, &updateRefError) {
			// When an error happens updating the reference, e.g. because of a race
			// with another update, then Ruby code didn't send an error but just an
			// empty response.
			err = stream.Send(&gitalypb.UserMergeBranchResponse{})
		}

		return err
	}

	if err := stream.Send(&gitalypb.UserMergeBranchResponse{
		BranchUpdate: &gitalypb.OperationBranchUpdate{
			CommitId:      mergeCommit,
			RepoCreated:   false,
			BranchCreated: false,
		},
	}); err != nil {
		return err
	}

	return nil
}

func validateFFRequest(in *gitalypb.UserFFBranchRequest) error {
	if len(in.Branch) == 0 {
		return fmt.Errorf("empty branch name")
	}

	if in.User == nil {
		return fmt.Errorf("empty user")
	}

	if in.CommitId == "" {
		return fmt.Errorf("empty commit id")
	}

	return nil
}

func (s *server) UserFFBranch(ctx context.Context, in *gitalypb.UserFFBranchRequest) (*gitalypb.UserFFBranchResponse, error) {
	if err := validateFFRequest(in); err != nil {
		return nil, helper.ErrInvalidArgument(err)
	}

	client, err := s.ruby.OperationServiceClient(ctx)
	if err != nil {
		return nil, err
	}

	clientCtx, err := rubyserver.SetHeaders(ctx, in.GetRepository())
	if err != nil {
		return nil, err
	}

	return client.UserFFBranch(clientCtx, in)
}

func validateUserMergeToRefRequest(in *gitalypb.UserMergeToRefRequest) error {
	if len(in.FirstParentRef) == 0 && len(in.Branch) == 0 {
		return fmt.Errorf("empty first parent ref and branch name")
	}

	if in.User == nil {
		return fmt.Errorf("empty user")
	}

	if in.SourceSha == "" {
		return fmt.Errorf("empty source SHA")
	}

	if len(in.TargetRef) == 0 {
		return fmt.Errorf("empty target ref")
	}

	if !strings.HasPrefix(string(in.TargetRef), "refs/merge-requests") {
		return fmt.Errorf("invalid target ref")
	}

	return nil
}

func (s *server) UserMergeToRef(ctx context.Context, in *gitalypb.UserMergeToRefRequest) (*gitalypb.UserMergeToRefResponse, error) {
	if err := validateUserMergeToRefRequest(in); err != nil {
		return nil, helper.ErrInvalidArgument(err)
	}

	client, err := s.ruby.OperationServiceClient(ctx)
	if err != nil {
		return nil, helper.ErrInternal(err)
	}

	clientCtx, err := rubyserver.SetHeaders(ctx, in.GetRepository())
	if err != nil {
		return nil, err
	}

	return client.UserMergeToRef(clientCtx, in)
}
