package blob

import (
	"gitlab.com/gitlab-org/gitaly/internal/rubyserver"

	pb "gitlab.com/gitlab-org/gitaly-proto/go"
)

type server struct {
	*rubyserver.Server
}

// NewServer creates a new instance of a grpc BlobServer
func NewServer(rs *rubyserver.Server) pb.BlobServiceServer {
	return &server{rs}
}
