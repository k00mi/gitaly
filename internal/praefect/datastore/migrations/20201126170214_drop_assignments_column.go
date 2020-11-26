package migrations

import migrate "github.com/rubenv/sql-migrate"

func init() {
	m := &migrate.Migration{
		Id: "20201126170214_drop_assignments_column",
		Up: []string{
			"ALTER TABLE storage_repositories DROP COLUMN assigned",
		},
		Down: []string{
			"ALTER TABLE storage_repositories ADD COLUMN assigned BOOLEAN NOT NULL DEFAULT TRUE",
		},
	}

	allMigrations = append(allMigrations, m)
}
