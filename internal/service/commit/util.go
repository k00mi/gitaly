package commit

import (
	"bytes"

	pb "gitlab.com/gitlab-org/gitaly-proto/go"
	"gitlab.com/gitlab-org/gitaly/internal/git"
	"gitlab.com/gitlab-org/gitaly/internal/helper/lines"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

func newCommitsBetweenWriter(stream pb.CommitService_CommitsBetweenServer) lines.Sender {
	return func(refs [][]byte) error {
		var commits []*pb.GitCommit

		for _, ref := range refs {
			elements := bytes.Split(ref, []byte("\x00"))
			if len(elements) != 8 {
				return grpc.Errorf(codes.Internal, "error parsing ref %q", ref)
			}

			commit, err := git.NewCommit(elements[0], elements[1], elements[2],
				elements[3], elements[4], elements[5], elements[6], elements[7])
			if err != nil {
				return err
			}

			commits = append(commits, commit)
		}
		return stream.Send(&pb.CommitsBetweenResponse{Commits: commits})
	}
}
