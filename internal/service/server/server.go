package server

import (
	pb "gitlab.com/gitlab-org/gitaly-proto/go"
)

type server struct{}

// NewServer creates a new instance of a grpc ServerServiceServer
func NewServer() pb.ServerServiceServer {
	return &server{}
}
