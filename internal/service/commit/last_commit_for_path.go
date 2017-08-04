package commit

import (
	"fmt"

	pb "gitlab.com/gitlab-org/gitaly-proto/go"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

type lastCommitForPathSender struct {
	commit *pb.GitCommit
}

func (s *server) LastCommitForPath(ctx context.Context, in *pb.LastCommitForPathRequest) (*pb.LastCommitForPathResponse, error) {
	if err := validateLastCommitForPathRequest(in); err != nil {
		return nil, grpc.Errorf(codes.InvalidArgument, "LastCommitForPath: %v", err)
	}

	sender := &lastCommitForPathSender{}
	writer := newCommitsWriter(sender)

	path := string(in.GetPath())
	if len(path) == 0 {
		path = "."
	}

	if err := gitLog(ctx, writer, in.GetRepository(), []string{string(in.GetRevision())}, []string{path}, "-1"); err != nil {
		return nil, err
	}

	return &pb.LastCommitForPathResponse{Commit: sender.commit}, nil
}

func validateLastCommitForPathRequest(in *pb.LastCommitForPathRequest) error {
	if len(in.Revision) == 0 {
		return fmt.Errorf("empty Revision")
	}
	return nil
}

func (sender *lastCommitForPathSender) Send(commits []*pb.GitCommit) error {
	// Since LastCommitForPath's response is not streamed this is not actually
	// _sending_ anything. We just set the commit for the caller to return it.
	if len(commits) > 0 {
		sender.commit = commits[0]
	}
	return nil
}
