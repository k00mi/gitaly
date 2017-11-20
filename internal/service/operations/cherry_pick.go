package operations

import (
	"fmt"

	pb "gitlab.com/gitlab-org/gitaly-proto/go"

	"gitlab.com/gitlab-org/gitaly/internal/rubyserver"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

func (s *server) UserCherryPick(ctx context.Context, req *pb.UserCherryPickRequest) (*pb.UserCherryPickResponse, error) {
	if err := validateUserCherryPickRequest(req); err != nil {
		return nil, grpc.Errorf(codes.InvalidArgument, "UserCherryPick: %v", err)
	}

	client, err := s.OperationServiceClient(ctx)
	if err != nil {
		return nil, err
	}

	clientCtx, err := rubyserver.SetHeaders(ctx, req.GetRepository())
	if err != nil {
		return nil, err
	}

	return client.UserCherryPick(clientCtx, req)
}

func validateUserCherryPickRequest(req *pb.UserCherryPickRequest) error {
	if req.User == nil {
		return fmt.Errorf("empty User")
	}

	if req.Commit == nil {
		return fmt.Errorf("empty Commit")
	}

	if len(req.BranchName) == 0 {
		return fmt.Errorf("empty BranchName")
	}

	if len(req.Message) == 0 {
		return fmt.Errorf("empty Message")
	}

	return nil
}
