package repository

import (
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/logrus"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/context"

	pb "gitlab.com/gitlab-org/gitaly-proto/go"
	"gitlab.com/gitlab-org/gitaly/internal/rubyserver"
)

func (s *server) FetchRemote(ctx context.Context, in *pb.FetchRemoteRequest) (*pb.FetchRemoteResponse, error) {
	grpc_logrus.Extract(ctx).WithFields(log.Fields{
		"Remote":     in.GetRemote(),
		"Force":      in.GetForce(),
		"NoTags":     in.GetNoTags(),
		"Timeout":    in.GetTimeout(),
		"SSHKey":     in.GetSshKey(),
		"KnownHosts": in.GetKnownHosts(),
	}).Debug("FetchRemote")

	client, err := s.RepositoryServiceClient(ctx)
	if err != nil {
		return nil, err
	}

	clientCtx, err := rubyserver.SetHeaders(ctx, in.GetRepository())
	if err != nil {
		return nil, err
	}

	return client.FetchRemote(clientCtx, in)
}
