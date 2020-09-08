package hook

import (
	"errors"

	"gitlab.com/gitlab-org/gitaly/internal/praefect/metadata"
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
