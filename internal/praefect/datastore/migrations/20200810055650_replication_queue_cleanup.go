package migrations

import migrate "github.com/rubenv/sql-migrate"

func init() {
	m := &migrate.Migration{
		Id: "20200810055650_replication_queue_cleanup",
		Up: []string{
			`CREATE TEMP TABLE ids(id BIGINT)`,
			`
			-- +migrate StatementBegin
			DO $$DECLARE
			    found_amount BIGINT;
			BEGIN
			    LOOP
				TRUNCATE TABLE ids;
				INSERT INTO ids
				SELECT id
				FROM replication_queue
				WHERE state = ANY (ARRAY['dead'::REPLICATION_JOB_STATE, 'completed'::REPLICATION_JOB_STATE])
				LIMIT 100000;
				SELECT COUNT(*) INTO found_amount FROM ids;
				IF found_amount > 0 THEN
				    DELETE FROM replication_queue WHERE id IN (SELECT id FROM ids);
				    COMMIT;
				ELSE
				    RETURN;
				END IF;
			    END LOOP;
			END$$;
			-- +migrate StatementEnd`,
			`DROP TABLE ids`,
		},
		Down:                 []string{},
		DisableTransactionUp: true,
	}

	allMigrations = append(allMigrations, m)
}
