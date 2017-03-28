package smarthttp

import pb "gitlab.com/gitlab-org/gitaly-proto/go"

type server struct{}

// PostUploadPack is a mock. TODO: Replace with actual implementation
func (s *server) PostUploadPack(stream pb.SmartHTTP_PostUploadPackServer) error {
	return nil
}

// PostReceivePack is a mock. TODO: Replace with actual implementation
func (s *server) PostReceivePack(stream pb.SmartHTTP_PostReceivePackServer) error {
	return nil
}

// NewServer creates a new instance of a grpc SmartHTTPServer
func NewServer() pb.SmartHTTPServer {
	return &server{}
}
