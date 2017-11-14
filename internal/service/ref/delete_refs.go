package ref

import (
	pb "gitlab.com/gitlab-org/gitaly-proto/go"
	"gitlab.com/gitlab-org/gitaly/internal/rubyserver"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

func (s *server) DeleteRefs(ctx context.Context, in *pb.DeleteRefsRequest) (*pb.DeleteRefsResponse, error) {
	if len(in.ExceptWithPrefix) == 0 { // You can't delete all refs
		return nil, grpc.Errorf(codes.InvalidArgument, "DeleteRefs: empty ExceptWithPrefix")
	}

	for _, prefix := range in.ExceptWithPrefix {
		if len(prefix) == 0 {
			return nil, grpc.Errorf(codes.InvalidArgument, "DeleteRefs: empty prefix for exclussion")
		}
	}

	client, err := s.RefServiceClient(ctx)
	if err != nil {
		return nil, err
	}

	clientCtx, err := rubyserver.SetHeaders(ctx, in.GetRepository())
	if err != nil {
		return nil, err
	}

	return client.DeleteRefs(clientCtx, in)
}
