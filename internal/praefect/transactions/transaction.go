package transactions

import (
	"context"
	"crypto/sha1"
	"errors"
	"fmt"
	"sync"
)

var (
	ErrDuplicateNodes        = errors.New("transactions cannot have duplicate nodes")
	ErrMissingNodes          = errors.New("transaction requires at least one node")
	ErrTransactionVoteFailed = errors.New("transaction vote failed")
	ErrTransactionCanceled   = errors.New("transaction was canceled")
)

// Voter is a participant in a given transaction that may cast a vote.
type Voter struct {
	// Name of the voter, usually Gitaly's storage name.
	Name string

	vote vote
}

type vote [sha1.Size]byte

func voteFromHash(hash []byte) (vote, error) {
	var vote vote

	if len(hash) != sha1.Size {
		return vote, fmt.Errorf("invalid voting hash: %q", hash)
	}

	copy(vote[:], hash)
	return vote, nil
}

func (v vote) isEmpty() bool {
	return v == vote{}
}

type transaction struct {
	doneCh   chan interface{}
	cancelCh chan interface{}

	lock         sync.Mutex
	votersByNode map[string]*Voter
}

func newTransaction(voters []Voter) (*transaction, error) {
	if len(voters) == 0 {
		return nil, ErrMissingNodes
	}

	votersByNode := make(map[string]*Voter, len(voters))
	for _, voter := range voters {
		if _, ok := votersByNode[voter.Name]; ok {
			return nil, ErrDuplicateNodes
		}

		voter := voter // rescope loop variable
		votersByNode[voter.Name] = &voter
	}

	return &transaction{
		doneCh:       make(chan interface{}),
		cancelCh:     make(chan interface{}),
		votersByNode: votersByNode,
	}, nil
}

func (t *transaction) cancel() {
	close(t.cancelCh)
}

func (t *transaction) vote(node string, hash []byte) error {
	vote, err := voteFromHash(hash)
	if err != nil {
		return err
	}

	t.lock.Lock()
	defer t.lock.Unlock()

	// Cast our vote. In case the node doesn't exist or has already cast a
	// vote, we need to abort.
	voter, ok := t.votersByNode[node]
	if !ok {
		return fmt.Errorf("invalid node for transaction: %q", node)
	}
	if !voter.vote.isEmpty() {
		return fmt.Errorf("node already cast a vote: %q", node)
	}
	voter.vote = vote

	// Count votes to see if we're done. If there are no more votes, then
	// we must notify other voters (and ourselves) by closing the `done`
	// channel.
	for _, voter := range t.votersByNode {
		if voter.vote.isEmpty() {
			return nil
		}
	}

	// As only the last voter may see that all participants have cast their
	// vote, this can really only be called by a single goroutine.
	close(t.doneCh)

	return nil
}

func (t *transaction) collectVotes(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.cancelCh:
		return ErrTransactionCanceled
	case <-t.doneCh:
		break
	}

	// Count votes to see whether we reached agreement or not. There should
	// be no need to lock as nobody will modify the votes anymore.
	var firstVote vote
	for _, voter := range t.votersByNode {
		if firstVote.isEmpty() {
			firstVote = voter.vote
		} else if firstVote != voter.vote {
			return ErrTransactionVoteFailed
		}
	}

	return nil
}
