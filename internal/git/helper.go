package git

import (
	"time"

	"github.com/golang/protobuf/ptypes/timestamp"
	pb "gitlab.com/gitlab-org/gitaly-proto/go"
)

// NewCommit creates a commit based on the given elements
func NewCommit(id, subject, authorName, authorEmail, authorDate,
	committerName, committerEmail, committerDate []byte) (*pb.GitCommit, error) {
	authorDateTime, err := time.Parse(time.RFC3339, string(authorDate))
	if err != nil {
		return nil, err
	}

	committerDateTime, err := time.Parse(time.RFC3339, string(committerDate))
	if err != nil {
		return nil, err
	}

	author := pb.CommitAuthor{
		Name:  authorName,
		Email: authorEmail,
		Date:  &timestamp.Timestamp{Seconds: authorDateTime.Unix()},
	}
	committer := pb.CommitAuthor{
		Name:  committerName,
		Email: committerEmail,
		Date:  &timestamp.Timestamp{Seconds: committerDateTime.Unix()},
	}

	return &pb.GitCommit{
		Id:        string(id),
		Subject:   subject,
		Author:    &author,
		Committer: &committer,
	}, nil
}
