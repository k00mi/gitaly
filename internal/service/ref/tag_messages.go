package ref

import (
	"fmt"

	"gitlab.com/gitlab-org/gitaly/internal/rubyserver"

	pb "gitlab.com/gitlab-org/gitaly-proto/go"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *server) GetTagMessages(request *pb.GetTagMessagesRequest, stream pb.RefService_GetTagMessagesServer) error {
	if err := validateGetTagMessagesRequest(request); err != nil {
		return status.Errorf(codes.InvalidArgument, "GetTagMessages: %v", err)
	}

	ctx := stream.Context()

	client, err := s.RefServiceClient(ctx)
	if err != nil {
		return err
	}

	clientCtx, err := rubyserver.SetHeaders(ctx, request.GetRepository())
	if err != nil {
		return err
	}

	rubyStream, err := client.GetTagMessages(clientCtx, request)
	if err != nil {
		return err
	}

	return rubyserver.Proxy(func() error {
		resp, err := rubyStream.Recv()
		if err != nil {
			md := rubyStream.Trailer()
			stream.SetTrailer(md)
			return err
		}
		return stream.Send(resp)
	})
}

func validateGetTagMessagesRequest(request *pb.GetTagMessagesRequest) error {
	if request.GetRepository() == nil {
		return fmt.Errorf("empty Repository")
	}

	return nil
}
