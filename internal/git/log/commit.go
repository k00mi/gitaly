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

	return GetCommitCatfile(c, revision)
}

// GetCommitCatfile looks up a commit by revision using an existing *catfile.Batch instance.
func GetCommitCatfile(c *catfile.Batch, revision string) (*pb.GitCommit, error) {
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

	header := split[0]
	var body []byte
	if len(split) == 2 {
		body = split[1]
	}

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

		switch headerSplit[0] {
		case "parent":
			commit.ParentIds = append(commit.ParentIds, headerSplit[1])
		case "author":
			commit.Author = parseCommitAuthor(headerSplit[1])
		case "committer":
			commit.Committer = parseCommitAuthor(headerSplit[1])
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return commit, nil
}

const maxUnixCommitDate = 1 << 53

func parseCommitAuthor(line string) *pb.CommitAuthor {
	author := &pb.CommitAuthor{}

	splitName := strings.SplitN(line, "<", 2)
	author.Name = []byte(strings.TrimSuffix(splitName[0], " "))

	if len(splitName) < 2 {
		return author
	}

	line = splitName[1]
	splitEmail := strings.SplitN(line, ">", 2)
	if len(splitEmail) < 2 {
		return author
	}

	author.Email = []byte(splitEmail[0])

	secSplit := strings.Fields(splitEmail[1])
	if len(secSplit) < 1 {
		return author
	}

	sec, err := strconv.ParseInt(secSplit[0], 10, 64)
	if err != nil || sec > maxUnixCommitDate {
		sec = git.FallbackTimeValue.Unix()
	}

	author.Date = &timestamp.Timestamp{Seconds: sec}

	return author
}

func subjectFromBody(body []byte) []byte {
	return bytes.TrimRight(bytes.SplitN(body, []byte("\n"), 2)[0], "\r\n")
}
