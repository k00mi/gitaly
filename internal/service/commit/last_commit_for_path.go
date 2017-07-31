package commit

import (
	"fmt"

	pb "gitlab.com/gitlab-org/gitaly-proto/go"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

type lastCommitForPathSender struct {
	commitPointer **pb.GitCommit
}

func (s *server) LastCommitForPath(ctx context.Context, in *pb.LastCommitForPathRequest) (*pb.LastCommitForPathResponse, error) {
	if err := validateLastCommitForPathRequest(in); err != nil {
		return nil, grpc.Errorf(codes.InvalidArgument, "LastCommitForPath: %v", err)
	}

	var commit *pb.GitCommit
	writer := newCommitsWriter(&lastCommitForPathSender{&commit})

	path := in.GetPath()
	if len(path) == 0 {
		path = []byte(".")
	}

	if err := gitLog(ctx, writer, in.GetRepository(), [][]byte{in.GetRevision(), []byte("--"), path}, "-1"); err != nil {
		return nil, err
	}

	return &pb.LastCommitForPathResponse{Commit: commit}, nil
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
		*sender.commitPointer = commits[0]
	}
	return nil
}
