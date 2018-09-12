package commit

import (
	"fmt"
	"io"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "gitlab.com/gitlab-org/gitaly-proto/go"
	"gitlab.com/gitlab-org/gitaly/internal/command"
	"gitlab.com/gitlab-org/gitaly/internal/git"
	"gitlab.com/gitlab-org/gitaly/internal/git/log"
	"gitlab.com/gitlab-org/gitaly/internal/git/lstree"
)

var (
	maxNumStatBatchSize = 25
)

func (s *server) ListLastCommitsForTree(in *pb.ListLastCommitsForTreeRequest, stream pb.CommitService_ListLastCommitsForTreeServer) error {
	if err := validateListLastCommitsForTreeRequest(in); err != nil {
		return status.Errorf(codes.InvalidArgument, "ListLastCommitsForTree: %v", err)
	}

	cmd, parser, err := newLSTreeParser(in, stream)
	if err != nil {
		return err
	}

	var batch []*pb.ListLastCommitsForTreeResponse_CommitForTree

	for {
		entry, err := parser.NextEntry()
		if err != nil {
			if err == io.EOF {
				break
			}

			return err
		}

		commit, err := log.LastCommitForPath(stream.Context(), in.GetRepository(), string(in.GetRevision()), entry.Path)
		if err != nil {
			return err
		}

		commitForTree := &pb.ListLastCommitsForTreeResponse_CommitForTree{
			Path:   entry.Path,
			Commit: commit,
		}

		batch = append(batch, commitForTree)
		if len(batch) == maxNumStatBatchSize {
			if err := sendCommitsForTree(batch, stream); err != nil {
				return err
			}

			batch = nil
		}
	}

	if err := cmd.Wait(); err != nil {
		return status.Errorf(codes.Unavailable, "%s: %v", "ListLastCommitsForTree", err)
	}

	return sendCommitsForTree(batch, stream)
}

func newLSTreeParser(in *pb.ListLastCommitsForTreeRequest, stream pb.CommitService_ListLastCommitsForTreeServer) (*command.Command, *lstree.Parser, error) {
	path := string(in.GetPath())
	if path == "" || path == "/" {
		path = "."
	}

	cmdArgs := []string{"ls-tree", "-z", "--full-name", string(in.GetRevision()), path}
	cmd, err := git.Command(stream.Context(), in.GetRepository(), cmdArgs...)
	if err != nil {
		return nil, nil, err
	}

	return cmd, lstree.NewParser(cmd), nil
}

func sendCommitsForTree(batch []*pb.ListLastCommitsForTreeResponse_CommitForTree, stream pb.CommitService_ListLastCommitsForTreeServer) error {
	if len(batch) == 0 {
		return nil
	}

	if err := stream.Send(&pb.ListLastCommitsForTreeResponse{Commits: batch}); err != nil {
		return status.Errorf(codes.Unavailable, "ListLastCommitsForTree: send: %v", err)
	}

	return nil
}

func validateListLastCommitsForTreeRequest(in *pb.ListLastCommitsForTreeRequest) error {
	if in.Revision == "" {
		return fmt.Errorf("empty Revision")
	}
	return nil
}
