package commit

import (
	"golang.org/x/net/context"

	pb "gitlab.com/gitlab-org/gitaly-proto/go"
)

type server struct{}

// NewServer creates a new instance of a grpc CommitServiceServer
func NewServer() pb.CommitServiceServer {
	return &server{}
}

func (server) FindCommit(ctx context.Context, in *pb.FindCommitRequest) (*pb.FindCommitResponse, error) {
	return nil, nil
}
