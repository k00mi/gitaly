package datastore

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	// Blank import to enable integration of github.com/lib/pq into database/sql
	_ "github.com/lib/pq"
	migrate "github.com/rubenv/sql-migrate"
	"gitlab.com/gitlab-org/gitaly/internal/praefect/config"
	"gitlab.com/gitlab-org/gitaly/internal/praefect/datastore/migrations"
)

// CheckPostgresVersion checks the server version of the Postgres DB
// specified in conf. This is a diagnostic for the Praefect Postgres
// rollout. https://gitlab.com/gitlab-org/gitaly/issues/1755
func CheckPostgresVersion(conf config.Config) error {
	db, err := openDB(conf)
	if err != nil {
		return fmt.Errorf("sql open: %v", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	var serverVersion int
	if err := db.QueryRowContext(ctx, "SHOW server_version_num").Scan(&serverVersion); err != nil {
		return fmt.Errorf("get postgres server version: %v", err)
	}

	const minimumServerVersion = 90600 // Postgres 9.6
	if serverVersion < minimumServerVersion {
		return fmt.Errorf("postgres server version too old: %d", serverVersion)
	}

	return nil
}

func openDB(conf config.Config) (*sql.DB, error) { return sql.Open("postgres", conf.DB.ToPQString()) }

const sqlMigrateDialect = "postgres"

// Migrate will apply all pending SQL migrations
func Migrate(conf config.Config) (int, error) {
	db, err := openDB(conf)
	if err != nil {
		return 0, fmt.Errorf("sql open: %v", err)
	}
	defer db.Close()

	return migrate.Exec(db, sqlMigrateDialect, migrationSource(), migrate.Up)
}

// MigrateDownPlan does a dry run for rolling back at most max migrations.
func MigrateDownPlan(conf config.Config, max int) ([]string, error) {
	db, err := openDB(conf)
	if err != nil {
		return nil, fmt.Errorf("sql open: %v", err)
	}
	defer db.Close()

	planned, _, err := migrate.PlanMigration(db, sqlMigrateDialect, migrationSource(), migrate.Down, max)
	if err != nil {
		return nil, err
	}

	var result []string
	for _, m := range planned {
		result = append(result, m.Id)
	}

	return result, nil
}

// MigrateDown rolls back at most max migrations.
func MigrateDown(conf config.Config, max int) (int, error) {
	db, err := openDB(conf)
	if err != nil {
		return 0, fmt.Errorf("sql open: %v", err)
	}
	defer db.Close()

	return migrate.ExecMax(db, sqlMigrateDialect, migrationSource(), migrate.Down, max)
}

func migrationSource() *migrate.MemoryMigrationSource {
	return &migrate.MemoryMigrationSource{Migrations: migrations.All()}
}
