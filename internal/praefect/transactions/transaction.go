package transactions

import (
	"context"
	"errors"
	"sync"
)

var (
	// ErrDuplicateNodes indicates a transaction was registered with two
	// voters having the same name.
	ErrDuplicateNodes = errors.New("transactions cannot have duplicate nodes")
	// ErrMissingNodes indicates a transaction was registered with no voters.
	ErrMissingNodes = errors.New("transaction requires at least one node")
	// ErrInvalidThreshold indicates a transaction was registered with an
	// invalid threshold that may either allow for multiple different
	// quorums or none at all.
	ErrInvalidThreshold = errors.New("transaction has invalid threshold")
	// ErrSubtransactionFailed indicates a vote was cast on a
	// subtransaction which failed already.
	ErrSubtransactionFailed = errors.New("subtransaction has failed")
	// ErrTransactionStopped indicates the transaction was gracefully stopped.
	ErrTransactionStopped = errors.New("transaction has been stopped")
)

type transactionState int

const (
	transactionOpen = transactionState(iota)
	transactionCanceled
	transactionStopped
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
	result VoteResult
}

// Transaction is a session where a set of voters votes on one or more
// subtransactions. Subtransactions are a sequence of sessions, where each node
// needs to go through the same sequence and agree on the same thing in the end
// in order to have the complete transaction succeed.
type Transaction struct {
	id        uint64
	threshold uint
	voters    []Voter

	lock            sync.Mutex
	state           transactionState
	subtransactions []*subtransaction
}

func newTransaction(id uint64, voters []Voter, threshold uint) (*Transaction, error) {
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

	return &Transaction{
		id:        id,
		threshold: threshold,
		voters:    voters,
		state:     transactionOpen,
	}, nil
}

func (t *Transaction) cancel() {
	t.lock.Lock()
	defer t.lock.Unlock()

	for _, subtransaction := range t.subtransactions {
		subtransaction.cancel()
	}

	t.state = transactionCanceled
}

func (t *Transaction) stop() error {
	t.lock.Lock()
	defer t.lock.Unlock()

	for _, subtransaction := range t.subtransactions {
		if err := subtransaction.stop(); err != nil {
			return err
		}
	}
	t.state = transactionStopped

	return nil
}

// ID returns the identifier used to uniquely identify a transaction.
func (t *Transaction) ID() uint64 {
	return t.id
}

// State returns the voting state mapped by voters. A voting state of `true`
// means all subtransactions were successful, a voting state of `false` means
// either no subtransactions were created or any of the subtransactions failed.
func (t *Transaction) State() (map[string]VoteResult, error) {
	t.lock.Lock()
	defer t.lock.Unlock()

	results := make(map[string]VoteResult, len(t.voters))

	if len(t.subtransactions) == 0 {
		for _, voter := range t.voters {
			switch t.state {
			case transactionOpen:
				results[voter.Name] = VoteUndecided
			case transactionCanceled:
				results[voter.Name] = VoteAborted
			case transactionStopped:
				results[voter.Name] = VoteStopped
			default:
				return nil, errors.New("invalid transaction state")
			}
		}
		return results, nil
	}

	// Collect all subtransactions. As they are ordered by reverse recency, we can simply
	// overwrite our own results.
	for _, subtransaction := range t.subtransactions {
		for voter, result := range subtransaction.state() {
			results[voter] = result
		}
	}

	return results, nil
}

// CountSubtransactions counts the number of subtransactions created as part of
// the transaction.
func (t *Transaction) CountSubtransactions() int {
	return len(t.subtransactions)
}

// getOrCreateSubtransaction gets an ongoing subtransaction on which the given
// node hasn't yet voted on or creates a new one if the node has succeeded on
// all subtransactions. In case the node has failed on any of the
// subtransactions, an error will be returned.
func (t *Transaction) getOrCreateSubtransaction(node string) (*subtransaction, error) {
	t.lock.Lock()
	defer t.lock.Unlock()

	switch t.state {
	case transactionOpen:
		// expected state, nothing to do
	case transactionCanceled:
		return nil, ErrTransactionCanceled
	case transactionStopped:
		return nil, ErrTransactionStopped
	default:
		return nil, errors.New("invalid transaction state")
	}

	for _, subtransaction := range t.subtransactions {
		result, err := subtransaction.getResult(node)
		if err != nil {
			return nil, err
		}

		switch result {
		case VoteUndecided:
			// An undecided vote means we should vote on this one.
			return subtransaction, nil
		case VoteCommitted:
			// If we have committed this subtransaction, we're good
			// to go.
			continue
		case VoteAborted:
			// If the subtransaction was aborted, then we need to
			// fail as we cannot proceed if the path leading to the
			// end result has intermittent failures.
			return nil, ErrSubtransactionFailed
		case VoteStopped:
			// If the transaction was stopped, then we need to fail
			// with a graceful error.
			return nil, ErrTransactionStopped
		}
	}

	// If we arrive here, then we know that all the node has voted and
	// reached quorum on all subtransactions. We can thus create a new one.
	subtransaction, err := newSubtransaction(t.voters, t.threshold)
	if err != nil {
		return nil, err
	}

	t.subtransactions = append(t.subtransactions, subtransaction)

	return subtransaction, nil
}

func (t *Transaction) vote(ctx context.Context, node string, hash []byte) error {
	subtransaction, err := t.getOrCreateSubtransaction(node)
	if err != nil {
		return err
	}

	if err := subtransaction.vote(node, hash); err != nil {
		return err
	}

	return subtransaction.collectVotes(ctx, node)
}
