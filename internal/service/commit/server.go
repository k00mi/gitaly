package commit

import pb "gitlab.com/gitlab-org/gitaly-proto/go"

type server struct{}

// TreeEntry is a stub. Substitute it with the real implementation
func (s *server) TreeEntry(in *pb.TreeEntryRequest, stream pb.Commit_TreeEntryServer) error {
	return nil
}

// NewServer creates a new instance of a grpc CommitServer
func NewServer() pb.CommitServer {
	return &server{}
}
