package migrations

import migrate "github.com/rubenv/sql-migrate"

func init() {
	m := &migrate.Migration{
		Id:   "20201102115118_variable_replication_factor",
		Up:   []string{"ALTER TABLE storage_repositories ADD COLUMN assigned BOOLEAN NOT NULL DEFAULT TRUE"},
		Down: []string{"ALTER TABLE storage_repositories DROP COLUMN assigned"},
	}

	allMigrations = append(allMigrations, m)
}
