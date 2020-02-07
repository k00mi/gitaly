// Package glsql (Gitaly SQL) is a helper package to work with plain SQL queries.
package glsql

import (
	"database/sql"

	// Blank import to enable integration of github.com/lib/pq into database/sql
	_ "github.com/lib/pq"
	migrate "github.com/rubenv/sql-migrate"
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
