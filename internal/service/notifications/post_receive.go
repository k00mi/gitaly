package notifications

import (
	"fmt"
	"log"

	"gitlab.com/gitlab-org/gitaly/internal/helper"

	pb "gitlab.com/gitlab-org/gitaly-proto/go"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

func (s *server) PostReceive(ctx context.Context, in *pb.PostReceiveRequest) (*pb.PostReceiveResponse, error) {
	// TODO: Invalidate InfoRefs cache
	repoPath, err := helper.GetRepoPath(in.GetRepository())
	if err != nil {
		message := fmt.Sprintf("PostReceive: %v", err)
		log.Print(message)
		return &pb.PostReceiveResponse{}, grpc.Errorf(codes.InvalidArgument, message)
	}

	log.Printf("PostReceive: RepoPath=%q", repoPath)
	return &pb.PostReceiveResponse{}, nil
}
