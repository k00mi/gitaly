package notifications

import (
	"log"

	pb "gitlab.com/gitlab-org/gitaly/protos/go"
	"golang.org/x/net/context"
)

func (s *server) PostReceive(ctx context.Context, in *pb.PostReceiveRequest) (*pb.PostReceiveResponse, error) {
	// TODO: Invalidate InfoRefs cache
	log.Printf("PostReceive: RepoPath=%q", in.Repository.Path)
	return &pb.PostReceiveResponse{}, nil
}
