package operations

import (
	"gitlab.com/gitlab-org/gitaly/internal/rubyserver"

	pb "gitlab.com/gitlab-org/gitaly-proto/go"

	"golang.org/x/net/context"
)

type server struct {
	*rubyserver.Server
}

// NewServer creates a new instance of a grpc OperationServiceServer
func NewServer(rs *rubyserver.Server) pb.OperationServiceServer {
	return &server{rs}
}

func (s *server) UserCreateBranch(ctx context.Context, req *pb.UserCreateBranchRequest) (*pb.UserCreateBranchResponse, error) {
	return nil, nil
}
