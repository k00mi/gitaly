package repository

import (
	"golang.org/x/net/context"

	"gitlab.com/gitlab-org/gitaly/internal/rubyserver"

	pb "gitlab.com/gitlab-org/gitaly-proto/go"
)

func (s *server) FindLicense(ctx context.Context, in *pb.FindLicenseRequest) (*pb.FindLicenseResponse, error) {
	client, err := s.RepositoryServiceClient(ctx)
	if err != nil {
		return nil, err
	}

	clientCtx, err := rubyserver.SetHeaders(ctx, in.GetRepository())
	if err != nil {
		return nil, err
	}

	return client.FindLicense(clientCtx, in)
}
