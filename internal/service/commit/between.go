package commit

import (
	"bytes"
	"fmt"
	"strings"

	log "github.com/Sirupsen/logrus"

	pb "gitlab.com/gitlab-org/gitaly-proto/go"
	"gitlab.com/gitlab-org/gitaly/internal/helper"
	"gitlab.com/gitlab-org/gitaly/internal/helper/lines"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

var commitLogFormatFields = []string{
	"%H",  // commit hash
	"%s",  // subject
	"%B",  // raw body (subject + body)
	"%an", // author name
	"%ae", // author email
	"%aI", // author date, strict ISO 8601 format
	"%cn", // committer name
	"%ce", // committer email
	"%cI", // committer date, strict ISO 8601 format
	"%P",  // parent hashes
}

func gitLog(writer lines.Sender, repo *pb.Repository, from string, to string) error {
	repoPath, err := helper.GetRepoPath(repo)
	if err != nil {
		return err
	}

	log.WithFields(log.Fields{
		"RepoPath": repoPath,
		"From":     from,
		"To":       to,
	}).Debug("GitLog")

	revisionRange := string(from) + ".." + string(to)
	// Use \x1f (ASCII field separator) as the field delimiter
	formatFlag := "--pretty=format:" + strings.Join(commitLogFormatFields, "%x1f")

	args := []string{
		"--git-dir",
		repoPath,
		"log",
		"-z", // use 0x00 as the entry terminator (instead of \n)
		"--reverse",
		formatFlag,
		revisionRange,
	}
	cmd, err := helper.GitCommandReader(args...)
	if err != nil {
		return err
	}
	defer cmd.Kill()

	split := lines.ScanWithDelimiter([]byte("\x00"))
	if err := lines.Send(cmd, writer, split); err != nil {
		return err
	}

	if err := cmd.Wait(); err != nil {
		// We expect this error to be caused by non-existing references. In that
		// case, we just log the error and send no commits to the `writer`.
		log.WithFields(log.Fields{"error": err}).Info("ignoring git-log error")
	}

	return nil
}

func parseCommitsBetweenRevision(revision []byte) (string, error) {
	if len(revision) == 0 {
		return "", fmt.Errorf("empty revision")
	}
	if bytes.HasPrefix(revision, []byte("-")) {
		return "", fmt.Errorf("revision can't start with '-'")
	}

	return string(revision), nil
}

func (s *server) CommitsBetween(in *pb.CommitsBetweenRequest, stream pb.CommitService_CommitsBetweenServer) error {
	from, err := parseCommitsBetweenRevision(in.GetFrom())
	if err != nil {
		return grpc.Errorf(codes.InvalidArgument, "CommitsBetween: from: %v", err)
	}
	to, err := parseCommitsBetweenRevision(in.GetTo())
	if err != nil {
		return grpc.Errorf(codes.InvalidArgument, "CommitsBetween: to: %v", err)
	}

	writer := newCommitsBetweenWriter(stream)

	return gitLog(writer, in.GetRepository(), from, to)
}
