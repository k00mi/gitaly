package commit

import (
	"fmt"

	"gitlab.com/gitlab-org/gitaly/internal/helper"

	pb "gitlab.com/gitlab-org/gitaly-proto/go"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

type commitsByMessageSender struct {
	stream pb.CommitService_CommitsByMessageServer
}

func (s *server) CommitsByMessage(in *pb.CommitsByMessageRequest, stream pb.CommitService_CommitsByMessageServer) error {
	if err := validateCommitsByMessageRequest(in); err != nil {
		return grpc.Errorf(codes.InvalidArgument, "CommitsByMessage: %v", err)
	}

	ctx := stream.Context()
	writer := newCommitsWriter(&commitsByMessageSender{stream})

	gitLogExtraOptions := []string{
		"--grep=" + in.GetQuery(),
		"--regexp-ignore-case",
	}
	if offset := in.GetOffset(); offset > 0 {
		gitLogExtraOptions = append(gitLogExtraOptions, fmt.Sprintf("--skip=%d", offset))
	}
	if limit := in.GetLimit(); limit > 0 {
		gitLogExtraOptions = append(gitLogExtraOptions, fmt.Sprintf("--max-count=%d", limit))
	}

	revision := in.GetRevision()
	if len(revision) == 0 {
		var err error

		repoPath, err := helper.GetRepoPath(in.GetRepository())
		if err != nil {
			return err
		}

		revision, err = defaultBranchName(ctx, repoPath)
		if err != nil {
			return grpc.Errorf(codes.Internal, "CommitsByMessage: defaultBranchName: %v", err)
		}
	}

	var paths []string
	if path := in.GetPath(); len(path) > 0 {
		paths = append(paths, string(path))
	}

	return gitLog(ctx, writer, in.GetRepository(), []string{string(revision)}, paths, gitLogExtraOptions...)
}

func validateCommitsByMessageRequest(in *pb.CommitsByMessageRequest) error {
	if in.GetQuery() == "" {
		return fmt.Errorf("empty Query")
	}

	return nil
}

func (sender *commitsByMessageSender) Send(commits []*pb.GitCommit) error {
	return sender.stream.Send(&pb.CommitsByMessageResponse{Commits: commits})
}
