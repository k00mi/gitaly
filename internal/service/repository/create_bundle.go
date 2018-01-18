package repository

import (
	"io"

	"gitlab.com/gitlab-org/gitaly/internal/git"
	"gitlab.com/gitlab-org/gitaly/streamio"

	pb "gitlab.com/gitlab-org/gitaly-proto/go"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

func (s *server) CreateBundle(req *pb.CreateBundleRequest, stream pb.RepositoryService_CreateBundleServer) error {
	repo := req.GetRepository()
	if repo == nil {
		return grpc.Errorf(codes.InvalidArgument, "CreateBundle: empty Repository")
	}

	ctx := stream.Context()

	cmd, err := git.Command(ctx, repo, "bundle", "create", "-", "--all")
	if err != nil {
		return grpc.Errorf(codes.Internal, "CreateBundle: cmd start failed: %v", err)
	}

	writer := streamio.NewWriter(func(p []byte) error {
		return stream.Send(&pb.CreateBundleResponse{Data: p})
	})

	_, err = io.Copy(writer, cmd)
	if err != nil {
		return grpc.Errorf(codes.Internal, "CreateBundle: stream writer failed: %v", err)
	}

	if err := cmd.Wait(); err != nil {
		return grpc.Errorf(codes.Internal, "CreateBundle: cmd wait failed: %v", err)
	}

	return nil
}
