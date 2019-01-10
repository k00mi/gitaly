package log

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"strings"

	"gitlab.com/gitlab-org/gitaly-proto/go/gitalypb"
	"gitlab.com/gitlab-org/gitaly/internal/git/catfile"
	"gitlab.com/gitlab-org/gitaly/internal/helper"
)

// GetTagCatfile looks up a commit by revision using an existing *catfile.Batch instance.
func GetTagCatfile(c *catfile.Batch, tagName string) (*gitalypb.Tag, error) {
	info, err := c.Info(tagName)
	if err != nil {
		return nil, err
	}

	r, err := c.Tag(info.Oid)
	if err != nil {
		return nil, err
	}

	header, body, err := splitRawTag(r)
	if err != nil {
		return nil, err
	}

	tag, err := buildAnnotatedTag(c, info.Oid, tagName, header, body)
	if err != nil {
		return nil, err
	}

	return tag, nil
}

type tagHeader struct {
	oid     string
	tagType string
}

func splitRawTag(r io.Reader) (*tagHeader, []byte, error) {
	raw, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, nil, err
	}

	split := bytes.SplitN(raw, []byte("\n\n"), 2)
	if len(split) != 2 {
		return nil, nil, errors.New("invalid tag object")
	}

	// Remove trailing newline, if any, to preserve existing behavior the old GitLab tag finding code.
	// See https://gitlab.com/gitlab-org/gitaly/blob/5e94dc966ac1900c11794b107a77496552591f9b/ruby/lib/gitlab/git/repository.rb#L211.
	// Maybe this belongs in the FindAllTags handler, or even on the gitlab-ce client side, instead of here?
	body := bytes.TrimRight(split[1], "\n")

	var header tagHeader
	s := bufio.NewScanner(bytes.NewReader(split[0]))
	for s.Scan() {
		headerSplit := strings.SplitN(s.Text(), " ", 2)
		if len(headerSplit) != 2 {
			continue
		}

		key, value := headerSplit[0], headerSplit[1]
		switch key {
		case "object":
			header.oid = value
		case "type":
			header.tagType = value
		}
	}

	return &header, body, nil
}

func buildAnnotatedTag(b *catfile.Batch, tagID, name string, header *tagHeader, body []byte) (*gitalypb.Tag, error) {
	tag := &gitalypb.Tag{
		Id:          tagID,
		Name:        []byte(name),
		MessageSize: int64(len(body)),
		Message:     body,
	}

	if max := helper.MaxCommitOrTagMessageSize; len(body) > max {
		tag.Message = tag.Message[:max]
	}

	if header.tagType == "commit" {
		commit, err := GetCommitCatfile(b, header.oid)
		if err != nil {
			return nil, fmt.Errorf("buildAnnotatedTag error when getting target commit: %v", err)
		}

		tag.TargetCommit = commit
	}

	return tag, nil
}
