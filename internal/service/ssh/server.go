package ssh

import pb "gitlab.com/gitlab-org/gitaly-proto/go"

const maxChunkSize = 1024

type server struct {
	ChunkSize int
}

// NewServer creates a new instance of a grpc SSHServer
func NewServer() pb.SSHServer {
	return &server{ChunkSize: maxChunkSize}
}
