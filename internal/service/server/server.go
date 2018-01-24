package server

import (
	pb "gitlab.com/gitlab-org/gitaly-proto/go"
)

type server struct{}

func NewServer() pb.ServerServiceServer {
	return &server{}
}
