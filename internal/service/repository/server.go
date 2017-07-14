package repository

import (
	pb "gitlab.com/gitlab-org/gitaly-proto/go"

	"golang.org/x/net/context"
)

type server struct{}

// NewServer creates a new instance of a gRPC repo server
func NewServer() pb.RepositoryServiceServer {
	return &server{}
}

func (s *server) GarbageCollect(ctx context.Context, in *pb.GarbageCollectRequest) (*pb.GarbageCollectResponse, error) {
	return nil, nil
}

func (s *server) RepackFull(ctx context.Context, in *pb.RepackFullRequest) (*pb.RepackFullResponse, error) {
	return nil, nil
}
func (s *server) RepackIncremental(ctx context.Context, in *pb.RepackIncrementalRequest) (*pb.RepackIncrementalResponse, error) {
	return nil, nil
}
