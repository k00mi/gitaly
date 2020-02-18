package main

import (
	"flag"
	"fmt"
	"strconv"

	"gitlab.com/gitlab-org/gitaly/internal/praefect/config"
	"gitlab.com/gitlab-org/gitaly/internal/praefect/datastore"
)

const subCmdSQLMigrateDown = "sql-migrate-down"

func sqlMigrateDown(conf config.Config, args []string) int {
	cmd := &sqlMigrateDownCmd{Config: conf}
	return cmd.Run(args)
}

type sqlMigrateDownCmd struct{ config.Config }

func (*sqlMigrateDownCmd) prefix() string { return progname + " " + subCmdSQLMigrateDown }

func (*sqlMigrateDownCmd) invocation() string { return invocationPrefix + " " + subCmdSQLMigrateDown }

func (smd *sqlMigrateDownCmd) Run(args []string) int {
	flagset := flag.NewFlagSet(smd.prefix(), flag.ExitOnError)
	flagset.Usage = func() {
		printfErr("usage:  %s [-f] MAX_MIGRATIONS\n", smd.invocation())
	}
	force := flagset.Bool("f", false, "apply down-migrations (default is dry run)")

	_ = flagset.Parse(args) // No error check because flagset is set to ExitOnError

	if flagset.NArg() != 1 {
		flagset.Usage()
		return 1
	}

	if err := smd.run(*force, flagset.Arg(0)); err != nil {
		printfErr("%s: fail: %v\n", smd.prefix(), err)
		return 1
	}

	return 0
}

func (smd *sqlMigrateDownCmd) run(force bool, maxString string) error {
	maxMigrations, err := strconv.Atoi(maxString)
	if err != nil {
		return err
	}

	if maxMigrations < 1 {
		return fmt.Errorf("number of migrations to roll back must be 1 or more")
	}

	if force {
		n, err := datastore.MigrateDown(smd.Config, maxMigrations)
		if err != nil {
			return err
		}

		fmt.Printf("%s: OK (applied %d \"down\" migrations)\n", smd.prefix(), n)
		return nil
	}

	planned, err := datastore.MigrateDownPlan(smd.Config, maxMigrations)
	if err != nil {
		return err
	}

	fmt.Printf("%s: DRY RUN -- would roll back:\n\n", smd.prefix())
	for _, id := range planned {
		fmt.Printf("- %s\n", id)
	}
	fmt.Printf("\nTo apply these migrations run: %s -f %d\n", smd.invocation(), maxMigrations)

	return nil
}
