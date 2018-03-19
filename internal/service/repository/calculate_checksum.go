package repository

import (
	pb "gitlab.com/gitlab-org/gitaly-proto/go"

	"gitlab.com/gitlab-org/gitaly/internal/helper"

	"golang.org/x/net/context"
)

func (s *server) CalculateChecksum(ctx context.Context, in *pb.CalculateChecksumRequest) (*pb.CalculateChecksumResponse, error) {
	return nil, helper.Unimplemented
}
