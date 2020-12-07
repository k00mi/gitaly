package hook

import (
	"context"
	"crypto/sha1"
	"io"
	"io/ioutil"

	"gitlab.com/gitlab-org/gitaly/internal/git"
	"gitlab.com/gitlab-org/gitaly/internal/helper"
)

func (m *GitLabHookManager) ReferenceTransactionHook(ctx context.Context, state ReferenceTransactionState, env []string, stdin io.Reader) error {
	payload, err := git.HooksPayloadFromEnv(env)
	if err != nil {
		return helper.ErrInternalf("extracting hooks payload: %w", err)
	}

	changes, err := ioutil.ReadAll(stdin)
	if err != nil {
		return helper.ErrInternalf("reading stdin from request: %w", err)
	}

	// We're only voting in prepared state as this is the only stage in
	// Git's reference transaction which allows us to abort the
	// transaction.
	if state != ReferenceTransactionPrepared {
		return nil
	}

	hash := sha1.Sum(changes)

	if err := m.voteOnTransaction(ctx, hash[:], payload); err != nil {
		return helper.ErrInternalf("error voting on transaction: %v", err)
	}

	return nil
}
