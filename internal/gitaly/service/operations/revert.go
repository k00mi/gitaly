package operations

import (
	"context"
	"errors"

	"gitlab.com/gitlab-org/gitaly/internal/gitaly/rubyserver"
	"gitlab.com/gitlab-org/gitaly/internal/metadata/featureflag"
	"gitlab.com/gitlab-org/gitaly/proto/go/gitalypb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *Server) UserRevert(ctx context.Context, req *gitalypb.UserRevertRequest) (*gitalypb.UserRevertResponse, error) {
	if err := validateCherryPickOrRevertRequest(req); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "UserRevert: %v", err)
	}
	if featureflag.IsDisabled(ctx, featureflag.GoUserRevert) {
		return s.rubyUserRevert(ctx, req)
	}

	return nil, errors.New("not implemented")
}

func (s *server) rubyUserRevert(ctx context.Context, req *gitalypb.UserRevertRequest) (*gitalypb.UserRevertResponse, error) {
	client, err := s.ruby.OperationServiceClient(ctx)
	if err != nil {
		return nil, err
	}

	clientCtx, err := rubyserver.SetHeaders(ctx, s.locator, req.GetRepository())
	if err != nil {
		return nil, err
	}

	return client.UserRevert(clientCtx, req)
}
