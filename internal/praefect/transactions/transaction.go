package transactions

import (
	"crypto/sha1"
	"errors"
	"fmt"
)

type transaction struct {
	node string
}

func newTransaction(nodes []string) (*transaction, error) {
	// We only accept a single node in transactions right now, which is
	// usually the primary. This limitation will be lifted at a later point
	// to allow for real transaction voting and multi-phase commits.
	if len(nodes) != 1 {
		return nil, errors.New("transaction requires exactly one node")
	}

	return &transaction{
		node: nodes[0],
	}, nil
}

func (t *transaction) vote(node string, hash []byte) error {
	// While the reference updates hash is not used yet, we already verify
	// it's there. At a later point, the hash will be used to verify that
	// all voting nodes agree on the same updates.
	if len(hash) != sha1.Size {
		return fmt.Errorf("invalid reference hash: %q", hash)
	}

	if t.node != node {
		return fmt.Errorf("invalid node for transaction: %q", node)
	}

	return nil
}
