package git2go

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"gitlab.com/gitlab-org/gitaly/internal/gitaly/config"
)

// RevertResult contains results from a revert.
type RevertResult struct {
	// CommitID is the object ID of the generated revert commit.
	CommitID string `json:"commit_id"`
}

// SerializeTo serializes the revert result and writes it into the writer.
func (r RevertResult) SerializeTo(w io.Writer) error {
	return serializeTo(w, r)
}

type RevertCommand struct {
	// Repository is the path to execute the revert in.
	Repository string `json:"repository"`
	// AuthorName is the author name of revert commit.
	AuthorName string `json:"author_name"`
	// AuthorMail is the author mail of revert commit.
	AuthorMail string `json:"author_mail"`
	// AuthorDate is the author date of revert commit.
	AuthorDate time.Time `json:"author_date"`
	// Message is the message to be used for the revert commit.
	Message string `json:"message"`
	// Ours is the commit that the revert is applied to.
	Ours string `json:"ours"`
	// Revert is the commit to be reverted.
	Revert string `json:"revert"`
	// Mainline is the parent to be considered the mainline
	Mainline uint `json:"mainline"`
}

func (r RevertCommand) Run(ctx context.Context, cfg config.Cfg) (RevertResult, error) {
	if err := r.verify(); err != nil {
		return RevertResult{}, fmt.Errorf("revert: %w: %s", ErrInvalidArgument, err.Error())
	}

	serialized, err := serialize(r)
	if err != nil {
		return RevertResult{}, err
	}

	stdout, err := run(ctx, binaryPathFromCfg(cfg), nil, "revert", "-request", serialized)
	if err != nil {
		return RevertResult{}, err
	}

	var response RevertResult
	if err := deserialize(stdout.String(), &response); err != nil {
		return RevertResult{}, err
	}

	return response, nil
}

func (r RevertCommand) verify() error {
	if r.Repository == "" {
		return errors.New("missing repository")
	}
	if r.AuthorName == "" {
		return errors.New("missing author name")
	}
	if r.AuthorMail == "" {
		return errors.New("missing author mail")
	}
	if r.Message == "" {
		return errors.New("missing message")
	}
	if r.Ours == "" {
		return errors.New("missing ours")
	}
	if r.Revert == "" {
		return errors.New("missing revert")
	}
	return nil
}

// RevertCommandFromSerialized deserializes the revert request from its JSON representation encoded with base64.
func RevertCommandFromSerialized(serialized string) (RevertCommand, error) {
	var request RevertCommand
	if err := deserialize(serialized, &request); err != nil {
		return RevertCommand{}, err
	}

	if err := request.verify(); err != nil {
		return RevertCommand{}, fmt.Errorf("revert: %w: %s", ErrInvalidArgument, err.Error())
	}

	return request, nil
}
