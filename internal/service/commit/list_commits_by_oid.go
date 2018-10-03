package commit

import (
	"gitlab.com/gitlab-org/gitaly-proto/go/gitalypb"
	"gitlab.com/gitlab-org/gitaly/internal/git/catfile"
	gitlog "gitlab.com/gitlab-org/gitaly/internal/git/log"
)

const batchSizeListCommitsByOid = 20

func (s *server) ListCommitsByOid(in *gitalypb.ListCommitsByOidRequest, stream gitalypb.CommitService_ListCommitsByOidServer) error {
	ctx := stream.Context()

	c, err := catfile.New(ctx, in.Repository)
	if err != nil {
		return err
	}

	send := func(commits []*gitalypb.GitCommit) error {
		return stream.Send(&gitalypb.ListCommitsByOidResponse{Commits: commits})
	}

	var commits []*gitalypb.GitCommit
	for _, oid := range in.Oid {
		commit, err := gitlog.GetCommitCatfile(c, oid)
		if err != nil {
			return err
		}

		if commit == nil {
			continue
		}

		commits = append(commits, commit)

		if len(commits) == batchSizeListCommitsByOid {
			if err := send(commits); err != nil {
				return err
			}
			commits = nil
		}
	}

	if len(commits) > 0 {
		return send(commits)
	}

	return nil
}
