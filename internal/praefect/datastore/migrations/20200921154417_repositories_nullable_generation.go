package migrations

import migrate "github.com/rubenv/sql-migrate"

func init() {
	m := &migrate.Migration{
		Id:   "20200921154417_repositories_nullable_generation",
		Up:   []string{`ALTER TABLE repositories ALTER COLUMN generation DROP NOT NULL`},
		Down: []string{},
	}

	allMigrations = append(allMigrations, m)
}
