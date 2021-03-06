package operations

import (
	"context"
	"errors"
	"fmt"

	"gitlab.com/gitlab-org/gitaly/internal/git"
	"gitlab.com/gitlab-org/gitaly/internal/git/remoterepo"
	"gitlab.com/gitlab-org/gitaly/internal/git2go"
	"gitlab.com/gitlab-org/gitaly/internal/gitaly/rubyserver"
	"gitlab.com/gitlab-org/gitaly/internal/helper"
	"gitlab.com/gitlab-org/gitaly/internal/metadata/featureflag"
	"gitlab.com/gitlab-org/gitaly/proto/go/gitalypb"
)

func (s *Server) UserRevert(ctx context.Context, req *gitalypb.UserRevertRequest) (*gitalypb.UserRevertResponse, error) {
	if err := validateCherryPickOrRevertRequest(req); err != nil {
		return nil, helper.ErrInvalidArgument(err)
	}
	if featureflag.IsDisabled(ctx, featureflag.GoUserRevert) {
		return s.rubyUserRevert(ctx, req)
	}

	startRevision, err := s.fetchStartRevision(ctx, req)
	if err != nil {
		return nil, err
	}

	localRepo := git.NewRepository(req.Repository)
	repoHadBranches, err := localRepo.HasBranches(ctx)
	if err != nil {
		return nil, err
	}

	repoPath, err := s.locator.GetPath(req.Repository)
	if err != nil {
		return nil, helper.ErrInternalf("get path: %w", err)
	}

	var mainline uint
	if len(req.Commit.ParentIds) > 1 {
		mainline = 1
	}

	newrev, err := git2go.RevertCommand{
		Repository: repoPath,
		AuthorName: string(req.User.Name),
		AuthorMail: string(req.User.Email),
		Message:    string(req.Message),
		Ours:       startRevision,
		Revert:     req.Commit.Id,
		Mainline:   mainline,
	}.Run(ctx, s.cfg)
	if err != nil {
		if errors.As(err, &git2go.RevertConflictError{}) {
			return &gitalypb.UserRevertResponse{
				CreateTreeError:     err.Error(),
				CreateTreeErrorCode: gitalypb.UserRevertResponse_CONFLICT,
			}, nil
		} else if errors.Is(err, git2go.ErrInvalidArgument) {
			return nil, helper.ErrInvalidArgument(err)
		} else {
			return nil, helper.ErrInternalf("revert command: %w", err)
		}
	}

	branch := fmt.Sprintf("refs/heads/%s", req.BranchName)

	branchCreated := false
	oldrev, err := localRepo.ResolveRefish(ctx, fmt.Sprintf("%s^{commit}", branch))
	if errors.Is(err, git.ErrReferenceNotFound) {
		branchCreated = true
		oldrev = git.NullSHA
	} else if err != nil {
		return nil, helper.ErrInvalidArgumentf("resolve ref: %w", err)
	}

	if req.DryRun {
		newrev = startRevision
	}

	if !branchCreated {
		ancestor, err := isAncestor(ctx, req.Repository, oldrev, newrev)
		if err != nil {
			return nil, err
		}
		if !ancestor {
			return &gitalypb.UserRevertResponse{
				CommitError: "Branch diverged",
			}, nil
		}
	}

	if err := s.updateReferenceWithHooks(ctx, req.Repository, req.User, branch, newrev, oldrev); err != nil {
		var preReceiveError preReceiveError
		if errors.As(err, &preReceiveError) {
			return &gitalypb.UserRevertResponse{
				PreReceiveError: preReceiveError.message,
			}, nil
		}

		return nil, fmt.Errorf("update reference with hooks: %w", err)
	}

	return &gitalypb.UserRevertResponse{
		BranchUpdate: &gitalypb.OperationBranchUpdate{
			CommitId:      newrev,
			BranchCreated: branchCreated,
			RepoCreated:   !repoHadBranches,
		},
	}, nil
}

func (s *Server) rubyUserRevert(ctx context.Context, req *gitalypb.UserRevertRequest) (*gitalypb.UserRevertResponse, error) {
	client, err := s.ruby.OperationServiceClient(ctx)
	if err != nil {
		return nil, err
	}

	clientCtx, err := rubyserver.SetHeaders(ctx, s.locator, req.GetRepository())
	if err != nil {
		return nil, err
	}

	return client.UserRevert(clientCtx, req)
}

func (s *Server) fetchStartRevision(ctx context.Context, req *gitalypb.UserRevertRequest) (string, error) {
	startBranchName := req.StartBranchName
	if len(startBranchName) == 0 {
		startBranchName = req.BranchName
	}

	startRepository := req.StartRepository
	if startRepository == nil {
		startRepository = req.Repository
	}

	remote, err := remoterepo.New(ctx, startRepository, s.conns)
	if err != nil {
		return "", helper.ErrInternal(err)
	}
	startRevision, err := remote.ResolveRefish(ctx, fmt.Sprintf("%s^{commit}", startBranchName))
	if err != nil {
		return "", helper.ErrInvalidArgumentf("resolve start ref: %w", err)
	}

	if req.StartRepository == nil {
		return startRevision, nil
	}

	_, err = git.NewRepository(req.Repository).ResolveRefish(ctx, fmt.Sprintf("%s^{commit}", startRevision))
	if errors.Is(err, git.ErrReferenceNotFound) {
		if err := s.fetchRemoteObject(ctx, req.Repository, req.StartRepository, startRevision); err != nil {
			return "", helper.ErrInternalf("fetch start: %w", err)
		}
	} else if err != nil {
		return "", helper.ErrInvalidArgumentf("resolve start: %w", err)
	}

	return startRevision, nil
}
