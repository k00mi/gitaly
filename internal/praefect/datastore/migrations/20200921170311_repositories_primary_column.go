package migrations

import migrate "github.com/rubenv/sql-migrate"

func init() {
	m := &migrate.Migration{
		Id:   "20200921170311_repositories_primary_column",
		Up:   []string{`ALTER TABLE repositories ADD COLUMN "primary" TEXT`},
		Down: []string{`ALTER TABLE repositories DROP COLUMN "primary"`},
	}

	allMigrations = append(allMigrations, m)
}
