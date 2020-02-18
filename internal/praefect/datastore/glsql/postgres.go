// Package glsql (Gitaly SQL) is a helper package to work with plain SQL queries.
package glsql

import (
	"context"
	"database/sql"

	// Blank import to enable integration of github.com/lib/pq into database/sql
	_ "github.com/lib/pq"
	migrate "github.com/rubenv/sql-migrate"
	"github.com/sirupsen/logrus"
	"gitlab.com/gitlab-org/gitaly/internal/praefect/config"
	"gitlab.com/gitlab-org/gitaly/internal/praefect/datastore/migrations"
)

// OpenDB returns connection pool to the database.
func OpenDB(conf config.DB) (*sql.DB, error) {
	db, err := sql.Open("postgres", conf.ToPQString())
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	return db, nil
}

// Migrate will apply all pending SQL migrations.
func Migrate(db *sql.DB) (int, error) {
	migrationSource := &migrate.MemoryMigrationSource{Migrations: migrations.All()}
	return migrate.Exec(db, "postgres", migrationSource, migrate.Up)
}

// TxQuery runs operations inside transaction and commits|rollbacks on Done.
type TxQuery interface {
	// Exec calls op function with provided ctx.
	// Returns true on success and false in case operation failed or wasn't called because of previously failed op.
	Exec(ctx context.Context, op func(context.Context, *sql.Tx) error) bool
	// Done must be called after work is finished to complete transaction.
	// errPtr must not be nil.
	// COMMIT will be executed if no errors happen during TxQuery usage.
	// Otherwise it will be ROLLBACK operation.
	Done(errPtr *error)
}

// NewTxQuery creates entity that allows to run queries in scope of a transaction.
// It always returns non-nil value.
func NewTxQuery(ctx context.Context, logger logrus.FieldLogger, db *sql.DB) TxQuery {
	tx, err := db.BeginTx(ctx, nil)
	return &txQuery{
		tx:     tx,
		err:    err,
		logger: logger,
	}
}

type txQuery struct {
	tx     *sql.Tx
	err    error
	logger logrus.FieldLogger
}

// Exec calls op function with provided ctx.
// Returns true on success and false in case operation failed or wasn't called because of previously failed op.
func (txq *txQuery) Exec(ctx context.Context, op func(context.Context, *sql.Tx) error) bool {
	if txq.err != nil {
		return false
	}

	txq.err = op(ctx, txq.tx)
	return txq.err == nil
}

// Done must be called after work is finished to complete transaction.
// errPtr must not be nil.
// COMMIT will be executed if no errors happen during txQuery usage.
// Otherwise it will be ROLLBACK operation.
func (txq *txQuery) Done(errPtr *error) {
	if txq.err == nil {
		txq.err = txq.tx.Commit()
		if txq.err != nil {
			txq.log(txq.err, "commit failed")
		}
	} else {
		// Don't overwrite txq.err because it's already non-nil
		if err := txq.tx.Rollback(); err != nil {
			txq.log(err, "rollback failed")
		}
	}

	if *errPtr == nil {
		*errPtr = txq.err
	}
}

func (txq *txQuery) log(err error, msg string) {
	if txq.logger != nil {
		txq.logger.WithError(err).Error(msg)
	}
}
