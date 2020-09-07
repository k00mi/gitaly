package ref

import (
	"gitlab.com/gitlab-org/gitaly/proto/go/gitalypb"
)

type server struct {
	gitalypb.UnimplementedRefServiceServer
}

// NewServer creates a new instance of a grpc RefServer
func NewServer() gitalypb.RefServiceServer {
	return &server{}
}
