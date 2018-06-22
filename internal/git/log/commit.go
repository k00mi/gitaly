package log

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"

	pb "gitlab.com/gitlab-org/gitaly-proto/go"
	"gitlab.com/gitlab-org/gitaly/internal/git"
	"gitlab.com/gitlab-org/gitaly/internal/git/catfile"
	"gitlab.com/gitlab-org/gitaly/internal/helper"

	"github.com/golang/protobuf/ptypes/timestamp"
)

// GetCommit tries to resolve revision to a Git commit. Returns nil if
// no object is found at revision.
func GetCommit(ctx context.Context, repo *pb.Repository, revision string) (*pb.GitCommit, error) {
	c, err := catfile.New(ctx, repo)
	if err != nil {
		return nil, err
	}

	return getCommitCatfile(c, revision)
}

func getCommitCatfile(c *catfile.Batch, revision string) (*pb.GitCommit, error) {
	info, err := c.Info(revision)
	if err != nil {
		if catfile.IsNotFound(err) {
			return nil, nil
		}

		return nil, err
	}

	// If we found a tag object, resolve it to a commit. Repeat if needed but
	// not in an infinite loop.
	for i := 0; info.Type == "tag" && i < 100; i++ {
		info, err = c.Info(info.Oid + "^{commit}")
		if err != nil {
			return nil, err
		}
	}

	if info.Type != "commit" {
		return nil, fmt.Errorf("expected %s to resolve to commit, got %s", revision, info.Type)
	}

	r, err := c.Commit(info.Oid)
	if err != nil {
		return nil, err
	}

	raw, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	return parseRawCommit(raw, info)
}

func parseRawCommit(raw []byte, info *catfile.ObjectInfo) (*pb.GitCommit, error) {
	split := bytes.SplitN(raw, []byte("\n\n"), 2)
	if len(split) != 2 {
		return nil, fmt.Errorf("commit %q has no message", info.Oid)
	}

	header, body := split[0], split[1]

	commit := &pb.GitCommit{
		Id:       info.Oid,
		Body:     body,
		Subject:  subjectFromBody(body),
		BodySize: int64(len(body)),
	}
	if max := helper.MaxCommitOrTagMessageSize; len(commit.Body) > max {
		commit.Body = commit.Body[:max]
	}

	scanner := bufio.NewScanner(bytes.NewReader(header))
	for scanner.Scan() {
		line := scanner.Text()
		if len(line) == 0 || line[0] == ' ' {
			continue
		}

		headerSplit := strings.SplitN(line, " ", 2)
		if len(headerSplit) != 2 {
			continue
		}

		var err error
		switch headerSplit[0] {
		case "parent":
			commit.ParentIds = append(commit.ParentIds, headerSplit[1])
		case "author":
			commit.Author, err = parseCommitAuthor(headerSplit[1])
			if err != nil {
				return nil, err
			}
		case "committer":
			commit.Committer, err = parseCommitAuthor(headerSplit[1])
			if err != nil {
				return nil, err
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return commit, nil
}

const maxUnixCommitDate = 1 << 53

func parseCommitAuthor(line string) (*pb.CommitAuthor, error) {
	author := &pb.CommitAuthor{}

	splitName := strings.SplitN(line, "<", 2)
	if len(splitName) < 2 {
		return nil, fmt.Errorf("missing '<' in %q", line)
	}

	author.Name = []byte(strings.TrimSuffix(splitName[0], " "))

	line = splitName[1]
	splitEmail := strings.SplitN(line, ">", 2)
	if len(splitName) < 2 {
		return nil, fmt.Errorf("missing '>' in %q", line)
	}

	author.Email = []byte(splitEmail[0])

	sec, err := strconv.ParseInt(strings.Fields(splitEmail[1])[0], 10, 64)
	if err != nil || sec > maxUnixCommitDate {
		sec = git.FallbackTimeValue.Unix()
	}

	author.Date = &timestamp.Timestamp{Seconds: sec}

	return author, nil
}
