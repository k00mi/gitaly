package datastore

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	migrate "github.com/rubenv/sql-migrate"
	"gitlab.com/gitlab-org/gitaly/internal/praefect/config"
	"gitlab.com/gitlab-org/gitaly/internal/praefect/datastore/glsql"
	"gitlab.com/gitlab-org/gitaly/internal/praefect/datastore/migrations"
)

// CheckPostgresVersion checks the server version of the Postgres DB
// specified in conf. This is a diagnostic for the Praefect Postgres
// rollout. https://gitlab.com/gitlab-org/gitaly/issues/1755
func CheckPostgresVersion(db *sql.DB) error {
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

const sqlMigrateDialect = "postgres"

// MigrateDownPlan does a dry run for rolling back at most max migrations.
func MigrateDownPlan(conf config.Config, max int) ([]string, error) {
	db, err := glsql.OpenDB(conf.DB)
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
	db, err := glsql.OpenDB(conf.DB)
	if err != nil {
		return 0, fmt.Errorf("sql open: %v", err)
	}
	defer db.Close()

	return migrate.ExecMax(db, sqlMigrateDialect, migrationSource(), migrate.Down, max)
}

func migrationSource() *migrate.MemoryMigrationSource {
	return &migrate.MemoryMigrationSource{Migrations: migrations.All()}
}
