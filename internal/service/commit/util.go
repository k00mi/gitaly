package commit

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"

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
			elements := bytes.Split(ref, []byte("\x1f"))
			if len(elements) != 10 {
				return grpc.Errorf(codes.Internal, "error parsing ref %q", ref)
			}
			parentIds := strings.Split(string(elements[9]), " ")

			commit, err := git.NewCommit(elements[0], elements[1], elements[2],
				elements[3], elements[4], elements[5], elements[6], elements[7],
				elements[8], parentIds...)
			if err != nil {
				return err
			}

			commits = append(commits, commit)
		}
		return stream.Send(&pb.CommitsBetweenResponse{Commits: commits})
	}
}

func newTreeEntry(commitOid, rootOid string, path, oidBytes, modeBytes []byte) (*pb.TreeEntry, error) {
	var objectType pb.TreeEntry_EntryType

	mode, err := strconv.ParseInt(string(modeBytes), 8, 32)
	if err != nil {
		return nil, fmt.Errorf("parse mode: %v", err)
	}

	oid := fmt.Sprintf("%02x", oidBytes)

	// Based on https://github.com/git/git/blob/v2.13.1/builtin/ls-tree.c#L67-L87
	switch mode & 0xf000 {
	case 0160000:
		objectType = pb.TreeEntry_COMMIT
	case 040000:
		objectType = pb.TreeEntry_TREE
	default:
		objectType = pb.TreeEntry_BLOB
	}

	return &pb.TreeEntry{
		CommitOid: commitOid,
		RootOid:   rootOid,
		Oid:       oid,
		Path:      path,
		Type:      objectType,
		Mode:      int32(mode),
	}, nil
}
