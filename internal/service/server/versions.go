package server

import (
	pb "gitlab.com/gitlab-org/gitaly-proto/go"
	"gitlab.com/gitlab-org/gitaly/internal/git"
	"gitlab.com/gitlab-org/gitaly/internal/version"

	"golang.org/x/net/context"
)

func (s *server) ServerVersion(ctx context.Context, in *pb.ServerVersionRequest) (*pb.ServerVersionResponse, error) {
	return &pb.ServerVersionResponse{Version: version.GetVersion()}, nil
}

func (s *server) ServerGitVersion(ctx context.Context, in *pb.ServerGitVersionRequest) (*pb.ServerGitVersionResponse, error) {
	version, err := git.Version()
	return &pb.ServerGitVersionResponse{Version: version}, err
}
