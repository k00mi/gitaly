package commit

import (
	"bytes"

	pb "gitlab.com/gitlab-org/gitaly-proto/go"
	"gitlab.com/gitlab-org/gitaly/internal/git"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

type commitsBetweenSender struct {
	stream pb.CommitService_CommitsBetweenServer
}

func (w commitsBetweenSender) SendRefs(refs [][]byte) error {
	var commits []*pb.GitCommit

	for _, ref := range refs {
		elements := bytes.Split(ref, []byte("\x00"))
		if len(elements) != 8 {
			return grpc.Errorf(codes.Internal, "error parsing ref %q", ref)
		}
		commit, err := git.BuildCommit(elements)
		if err != nil {
			return err
		}
		commits = append(commits, commit)
	}
	return w.stream.Send(&pb.CommitsBetweenResponse{Commits: commits})
}

func newCommitsBetweenWriter(stream pb.CommitService_CommitsBetweenServer, maxMsgSize int) git.RefsWriter {
	return git.RefsWriter{
		RefsSender: commitsBetweenSender{stream},
		MaxMsgSize: maxMsgSize,
	}
}
