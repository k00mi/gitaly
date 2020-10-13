package hook

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"gitlab.com/gitlab-org/gitaly/internal/praefect/metadata"
	"gitlab.com/gitlab-org/gitaly/proto/go/gitalypb"
	"google.golang.org/grpc"
)

func isPrimary(env []string) (bool, error) {
	tx, err := metadata.TransactionFromEnv(env)
	if err != nil {
		if errors.Is(err, metadata.ErrTransactionNotFound) {
			// If there is no transaction, then we only ever write
			// to the primary. Thus, we return true.
			return true, nil
		}
		return false, err
	}

	return tx.Primary, nil
}

func (m *GitLabHookManager) getPraefectConn(ctx context.Context, server *metadata.PraefectServer) (*grpc.ClientConn, error) {
	address, err := server.Address()
	if err != nil {
		return nil, err
	}
	return m.conns.Dial(ctx, address, server.Token)
}

// transactionHandler is a callback invoked on a transaction if it exists.
type transactionHandler func(ctx context.Context, tx metadata.Transaction, client gitalypb.RefTransactionClient) error

// runWithTransaction runs the given function if the environment identifies a transaction. No error
// is returned if no transaction exists. If a transaction exists and the function is executed on it,
// then its error will ber returned directly.
func (m *GitLabHookManager) runWithTransaction(ctx context.Context, env []string, handler transactionHandler) error {
	tx, err := metadata.TransactionFromEnv(env)
	if err != nil {
		if errors.Is(err, metadata.ErrTransactionNotFound) {
			// No transaction being present is valid, e.g. in case
			// there is no Praefect server or the transactions
			// feature flag is not set.
			return nil
		}
		return fmt.Errorf("could not extract transaction: %w", err)
	}

	praefectServer, err := metadata.PraefectFromEnv(env)
	if err != nil {
		return fmt.Errorf("could not extract Praefect server: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	praefectConn, err := m.getPraefectConn(ctx, praefectServer)
	if err != nil {
		return err
	}

	praefectClient := gitalypb.NewRefTransactionClient(praefectConn)

	if err := handler(ctx, tx, praefectClient); err != nil {
		return err
	}

	return nil
}

func (m *GitLabHookManager) voteOnTransaction(ctx context.Context, hash []byte, env []string) error {
	return m.runWithTransaction(ctx, env, func(ctx context.Context, tx metadata.Transaction, client gitalypb.RefTransactionClient) error {
		defer prometheus.NewTimer(m.votingDelayMetric).ObserveDuration()

		response, err := client.VoteTransaction(ctx, &gitalypb.VoteTransactionRequest{
			TransactionId:        tx.ID,
			Node:                 tx.Node,
			ReferenceUpdatesHash: hash,
		})
		if err != nil {
			return err
		}

		switch response.State {
		case gitalypb.VoteTransactionResponse_COMMIT:
			return nil
		case gitalypb.VoteTransactionResponse_ABORT:
			return errors.New("transaction was aborted")
		case gitalypb.VoteTransactionResponse_STOP:
			return errors.New("transaction was stopped")
		default:
			return errors.New("invalid transaction state")
		}
	})
}

func (m *GitLabHookManager) stopTransaction(ctx context.Context, env []string) error {
	return m.runWithTransaction(ctx, env, func(ctx context.Context, tx metadata.Transaction, client gitalypb.RefTransactionClient) error {
		_, err := client.StopTransaction(ctx, &gitalypb.StopTransactionRequest{
			TransactionId: tx.ID,
		})
		if err != nil {
			return err
		}

		return nil
	})
}
