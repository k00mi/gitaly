package remote

import (
	"fmt"
	"strings"

	"google.golang.org/grpc/codes"

	"golang.org/x/net/context"
	"google.golang.org/grpc"

	pb "gitlab.com/gitlab-org/gitaly-proto/go"
	"gitlab.com/gitlab-org/gitaly/internal/helper"
	"gitlab.com/gitlab-org/gitaly/internal/rubyserver"
)

// AddRemote adds a remote to the repository
func (s *server) AddRemote(ctx context.Context, req *pb.AddRemoteRequest) (*pb.AddRemoteResponse, error) {
	if err := validateAddRemoteRequest(req); err != nil {
		return nil, grpc.Errorf(codes.InvalidArgument, "AddRemote: %v", err)
	}

	client, err := s.RemoteServiceClient(ctx)
	if err != nil {
		return nil, err
	}

	clientCtx, err := rubyserver.SetHeaders(ctx, req.GetRepository())
	if err != nil {
		return nil, err
	}

	return client.AddRemote(clientCtx, req)
}

func validateAddRemoteRequest(req *pb.AddRemoteRequest) error {
	if strings.TrimSpace(req.GetName()) == "" {
		return fmt.Errorf("empty remote name")
	}
	if req.GetUrl() == "" {
		return fmt.Errorf("empty remote url")
	}

	return nil
}

// RemoveRemote removes the given remote
func (s *server) RemoveRemote(context.Context, *pb.RemoveRemoteRequest) (*pb.RemoveRemoteResponse, error) {
	return nil, helper.Unimplemented
}
