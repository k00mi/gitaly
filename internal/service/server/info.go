package server

import (
	pb "gitlab.com/gitlab-org/gitaly-proto/go"
	"gitlab.com/gitlab-org/gitaly/internal/git"
	"gitlab.com/gitlab-org/gitaly/internal/version"

	"golang.org/x/net/context"
)

func (s *server) ServerInfo(ctx context.Context, in *pb.ServerInfoRequest) (*pb.ServerInfoResponse, error) {
	gitVersion, err := git.Version()

	return &pb.ServerInfoResponse{
		ServerVersion: version.GetVersion(),
		GitVersion:    gitVersion,
	}, err
}
