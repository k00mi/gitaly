package remote

import (
	"fmt"

	pb "gitlab.com/gitlab-org/gitaly-proto/go"
	"gitlab.com/gitlab-org/gitaly/internal/rubyserver"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *server) UpdateRemoteMirror(stream pb.RemoteService_UpdateRemoteMirrorServer) error {
	firstRequest, err := stream.Recv()
	if err != nil {
		return err
	}

	if err = validateUpdateRemoteMirrorRequest(firstRequest); err != nil {
		return status.Errorf(codes.InvalidArgument, "UpdateRemoteMirror: %v", err)
	}

	ctx := stream.Context()
	client, err := s.RemoteServiceClient(ctx)
	if err != nil {
		return err
	}

	clientCtx, err := rubyserver.SetHeaders(ctx, firstRequest.GetRepository())
	if err != nil {
		return err
	}

	rubyStream, err := client.UpdateRemoteMirror(clientCtx)
	if err != nil {
		return err
	}

	if err := rubyStream.Send(firstRequest); err != nil {
		return err
	}

	err = rubyserver.Proxy(func() error {
		request, err := stream.Recv()
		if err != nil {
			return err
		}
		return rubyStream.Send(request)
	})

	if err != nil {
		return err
	}

	response, err := rubyStream.CloseAndRecv()
	if err != nil {
		return err
	}

	return stream.SendAndClose(response)
}

func validateUpdateRemoteMirrorRequest(req *pb.UpdateRemoteMirrorRequest) error {
	if req.GetRepository() == nil {
		return fmt.Errorf("empty Repository")
	}
	if req.GetRefName() == "" {
		return fmt.Errorf("empty RefName")
	}

	return nil
}
