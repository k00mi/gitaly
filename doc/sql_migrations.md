# SQL migrations in Praefect

SQL migration files are stored in `/internal/praefect/datastore/migrations`.

The underlying migration engine we use is [github.com/rubenv/sql-migrate](https://github.com/rubenv/sql-migrate).

To generate a new migration, run the `_support/new-migration` script from the top level of your Gitaly checkout.

Praefect SQL migrations should be applied automatically when you deploy Praefect. If you want to run them manually, run:

```
praefect -config /path/to/config.toml sql-migrate
```

## Rolling back migrations

Rolling back SQL migrations in Praefect works a little differently
from ActiveRecord. It is a three step process.

### 1. Decide how many steps you want to roll back

Count the number of migrations you want to roll back.

### 2. Perform a dry run and verify that the right migrations are getting rolled back

```
praefect -config /path/to/config.toml sql-migrate-down NUM_ROLLBACK
```

This will perform a dry run and print the list of migrations that
would be rolled back. Verify that these are the migrations you want to
roll back.

### 3. Perform the rollback

We use the same command as before, but we pass `-f` to indicate we
want destructive changes (the rollbacks) to happen.

```
praefect -config /path/to/config.toml sql-migrate-down -f NUM_ROLLBACK
```
