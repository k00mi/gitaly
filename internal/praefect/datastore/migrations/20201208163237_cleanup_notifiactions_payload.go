package migrations

import migrate "github.com/rubenv/sql-migrate"

func init() {
	m := &migrate.Migration{
		Id: "20201208163237_cleanup_notifiactions_payload",
		Up: []string{
			`-- +migrate StatementBegin
			CREATE OR REPLACE FUNCTION notify_on_change() RETURNS TRIGGER AS $$
				DECLARE
					msg JSONB;
				BEGIN
				    	CASE TG_OP
					WHEN 'INSERT' THEN
						SELECT JSON_AGG(obj) INTO msg
						FROM (
							SELECT JSONB_BUILD_OBJECT('virtual_storage', virtual_storage, 'relative_paths', ARRAY_AGG(DISTINCT relative_path)) AS obj
							FROM NEW
							GROUP BY virtual_storage
						) t;
					WHEN 'UPDATE' THEN
						SELECT JSON_AGG(obj) INTO msg
						FROM (
							SELECT JSONB_BUILD_OBJECT('virtual_storage', virtual_storage, 'relative_paths', ARRAY_AGG(DISTINCT relative_path)) AS obj
							FROM NEW
							FULL JOIN OLD USING (virtual_storage, relative_path)
							GROUP BY virtual_storage
						) t;
					WHEN 'DELETE' THEN
						SELECT JSON_AGG(obj) INTO msg
						FROM (
							SELECT JSONB_BUILD_OBJECT('virtual_storage', virtual_storage, 'relative_paths', ARRAY_AGG(DISTINCT relative_path)) AS obj
							FROM OLD
							GROUP BY virtual_storage
						) t;
					END CASE;

				    	CASE WHEN JSONB_ARRAY_LENGTH(msg) > 0 THEN
						PERFORM PG_NOTIFY(TG_ARGV[TG_NARGS-1], msg::TEXT);
					ELSE END CASE;

					RETURN NULL;
				END;
				$$ LANGUAGE plpgsql;
			-- +migrate StatementEnd`,
		},
		Down: []string{
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
		},
	}

	allMigrations = append(allMigrations, m)
}
