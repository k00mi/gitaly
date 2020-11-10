package migrations

import migrate "github.com/rubenv/sql-migrate"

func init() {
	m := &migrate.Migration{
		Id: "20201102171914_virtual_storage_configuration",
		Up: []string{
			`CREATE TABLE virtual_storages (
				virtual_storage TEXT PRIMARY KEY,
				repositories_imported BOOLEAN DEFAULT FALSE NOT NULL
			)`,
		},
		Down: []string{
			"DROP TABLE virtual_storages",
		},
	}

	allMigrations = append(allMigrations, m)
}
