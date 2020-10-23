package migrations

import migrate "github.com/rubenv/sql-migrate"

func init() {
	m := &migrate.Migration{
		Id: "20201006125956_trigger_repository_update_generation",
		Up: []string{
			`-- +migrate StatementBegin
			CREATE OR REPLACE FUNCTION notify_on_change() RETURNS TRIGGER AS $$
				DECLARE
				    old_val JSON DEFAULT NULL;
				    new_val JSON DEFAULT NULL;
				BEGIN
				    CASE TG_OP
					WHEN 'INSERT' THEN
						SELECT JSON_AGG(ROW_TO_JSON(t.*)) INTO new_val FROM NEW AS t;
					WHEN 'UPDATE' THEN
						SELECT JSON_AGG(ROW_TO_JSON(t.*)) INTO old_val FROM OLD AS t;
						SELECT JSON_AGG(ROW_TO_JSON(t.*)) INTO new_val FROM NEW AS t;
					WHEN 'DELETE' THEN
						SELECT JSON_AGG(ROW_TO_JSON(t.*)) INTO old_val FROM OLD AS t;
					END CASE;

				    PERFORM PG_NOTIFY(TG_ARGV[TG_NARGS-1], JSON_BUILD_OBJECT('old', old_val, 'new', new_val)::TEXT);
				    RETURN NULL;
				END;
				$$ LANGUAGE plpgsql;
			-- +migrate StatementEnd`,

			// for repositories table
			`CREATE TRIGGER notify_on_delete AFTER DELETE ON repositories
				REFERENCING OLD TABLE AS OLD
		 		FOR EACH STATEMENT
				EXECUTE FUNCTION notify_on_change('repositories_updates')`,

			// for storage_repositories table
			`CREATE TRIGGER notify_on_insert AFTER INSERT ON storage_repositories
				REFERENCING NEW TABLE AS NEW
		 		FOR EACH STATEMENT
				EXECUTE FUNCTION notify_on_change('storage_repositories_updates')`,

			`CREATE TRIGGER notify_on_update AFTER UPDATE ON storage_repositories
				REFERENCING OLD TABLE AS OLD NEW TABLE AS NEW
		 		FOR EACH STATEMENT
				EXECUTE FUNCTION notify_on_change('storage_repositories_updates')`,

			`CREATE TRIGGER notify_on_delete AFTER DELETE ON storage_repositories
				REFERENCING OLD TABLE AS OLD
		 		FOR EACH STATEMENT
				EXECUTE FUNCTION notify_on_change('storage_repositories_updates')`,
		},
		Down: []string{
			`DROP TRIGGER IF EXISTS notify_on_delete ON repositories`,

			`DROP TRIGGER IF EXISTS notify_on_insert ON storage_repositories`,
			`DROP TRIGGER IF EXISTS notify_on_update ON storage_repositories`,
			`DROP TRIGGER IF EXISTS notify_on_delete ON storage_repositories`,

			`DROP FUNCTION IF EXISTS notify_on_change`,
		},
	}

	allMigrations = append(allMigrations, m)
}
