package remote

import (
	"golang.org/x/net/context"

	pb "gitlab.com/gitlab-org/gitaly-proto/go"
	"gitlab.com/gitlab-org/gitaly/internal/helper"
)

// AddRemote adds a remote to the repository
func (s *server) AddRemote(context.Context, *pb.AddRemoteRequest) (*pb.AddRemoteResponse, error) {
	return nil, helper.Unimplemented
}

// RemoveRemote removes the given remote
func (s *server) RemoveRemote(context.Context, *pb.RemoveRemoteRequest) (*pb.RemoveRemoteResponse, error) {
	return nil, helper.Unimplemented
}
