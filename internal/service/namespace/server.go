package namespace

import (
	pb "gitlab.com/gitlab-org/gitaly-proto/go"
)

type server struct{}

// NewServer creates a new instance of a gRPC namespace server
func NewServer() pb.NamespaceServiceServer {
	return &server{}
}
