package git2go

import (
	"context"
	"fmt"
	"io"

	"gitlab.com/gitlab-org/gitaly/internal/git/conflict"
	"gitlab.com/gitlab-org/gitaly/internal/gitaly/config"
)

// ResolveCommand contains arguments to perform a merge commit and resolve any
// conflicts produced from that merge commit
type ResolveCommand struct {
	MergeCommand `json:"merge_command"`
	Resolutions  []conflict.Resolution `json:"conflict_files"`
}

// ResolveResult returns information about the successful merge and resolution
type ResolveResult struct {
	MergeResult `json:"merge_result"`
}

// ResolveResolveCommandFromSerialized deserializes a ResolveCommand and
// verifies the arguments are valid
func ResolveCommandFromSerialized(serialized string) (ResolveCommand, error) {
	var request ResolveCommand
	if err := deserialize(serialized, &request); err != nil {
		return ResolveCommand{}, err
	}

	if err := request.verify(); err != nil {
		return ResolveCommand{}, fmt.Errorf("resolve: %w: %s", ErrInvalidArgument, err.Error())
	}

	return request, nil
}

// Run will attempt merging and resolving conflicts for the provided request
func (r ResolveCommand) Run(ctx context.Context, cfg config.Cfg) (ResolveResult, error) {
	if err := r.verify(); err != nil {
		return ResolveResult{}, fmt.Errorf("resolve: %w: %s", ErrInvalidArgument, err.Error())
	}

	serialized, err := serialize(r)
	if err != nil {
		return ResolveResult{}, err
	}

	stdout, err := run(ctx, binaryPathFromCfg(cfg), nil, "resolve", "-request", serialized)
	if err != nil {
		return ResolveResult{}, err
	}

	var response ResolveResult
	if err := deserialize(stdout.String(), &response); err != nil {
		return ResolveResult{}, err
	}

	return response, nil
}

// SerializeTo serializes the resolve conflict and writes it to the writer
func (r ResolveResult) SerializeTo(w io.Writer) error {
	return serializeTo(w, r)
}
