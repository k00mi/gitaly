package transactions

import (
	"context"
	"errors"
)

var (
	ErrDuplicateNodes   = errors.New("transactions cannot have duplicate nodes")
	ErrMissingNodes     = errors.New("transaction requires at least one node")
	ErrInvalidThreshold = errors.New("transaction has invalid threshold")
)

// Voter is a participant in a given transaction that may cast a vote.
type Voter struct {
	// Name of the voter, usually Gitaly's storage name.
	Name string
	// Votes is the number of votes available to this voter in the voting
	// process. `0` means the outcome of the vote will not be influenced by
	// this voter.
	Votes uint

	vote   vote
	result voteResult
}

// transaction is a session where a set of voters votes on one or more
// subtransactions. Subtransactions are a sequence of sessions, where each node
// needs to go through the same sequence and agree on the same thing in the end
// in order to have the complete transaction succeed.
type transaction struct {
	threshold      uint
	voters         []Voter
	subtransaction *subtransaction
}

func newTransaction(voters []Voter, threshold uint) (*transaction, error) {
	if len(voters) == 0 {
		return nil, ErrMissingNodes
	}

	var totalVotes uint
	votersByNode := make(map[string]interface{}, len(voters))
	for _, voter := range voters {
		if _, ok := votersByNode[voter.Name]; ok {
			return nil, ErrDuplicateNodes
		}
		votersByNode[voter.Name] = nil
		totalVotes += voter.Votes
	}

	// If the given threshold is smaller than the total votes, then we
	// cannot ever reach quorum.
	if totalVotes < threshold {
		return nil, ErrInvalidThreshold
	}

	// If the threshold is less or equal than half of all node's votes,
	// it's possible to reach multiple different quorums that settle on
	// different outcomes.
	if threshold*2 <= totalVotes {
		return nil, ErrInvalidThreshold
	}

	subtransaction, err := newSubtransaction(voters, threshold)
	if err != nil {
		return nil, err
	}

	return &transaction{
		threshold:      threshold,
		voters:         voters,
		subtransaction: subtransaction,
	}, nil
}

func (t *transaction) cancel() map[string]bool {
	return t.subtransaction.cancel()
}

func (t *transaction) vote(ctx context.Context, node string, hash []byte) error {
	if err := t.subtransaction.vote(node, hash); err != nil {
		return err
	}
	return t.subtransaction.collectVotes(ctx, node)
}
