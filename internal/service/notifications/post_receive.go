package notifications

import (
	"errors"
	"log"

	pb "gitlab.com/gitlab-org/gitaly/protos/go"
	"golang.org/x/net/context"
)

func (s *server) PostReceive(ctx context.Context, in *pb.PostReceiveRequest) (*pb.PostReceiveResponse, error) {
	// TODO: Invalidate InfoRefs cache
	if in.Repository == nil {
		message := "Bad Request (empty repository)"
		log.Printf("PostReceive: %q", message)
		return &pb.PostReceiveResponse{}, errors.New(message)
	}

	log.Printf("PostReceive: RepoPath=%q", in.Repository.Path)
	return &pb.PostReceiveResponse{}, nil
}
