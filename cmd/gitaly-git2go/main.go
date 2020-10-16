// +build static,system_libgit2

package main

import (
	"flag"
	"fmt"
	"os"
)

type subcmd interface {
	Flags() *flag.FlagSet
	Run() error
}

var subcommands = map[string]subcmd{
	"conflicts": &conflictsSubcommand{},
	"merge":     &mergeSubcommand{},
	"revert":    &revertSubcommand{},
}

const programName = "gitaly-git2go"

func fatalf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}

func main() {
	flags := flag.NewFlagSet(programName, flag.ExitOnError)
	flags.Parse(os.Args)

	if flags.NArg() < 2 {
		fatalf("missing subcommand")
	}

	subcmd, ok := subcommands[flags.Arg(1)]
	if !ok {
		fatalf("unknown subcommand: %q", flags.Arg(1))
	}

	subcmdFlags := subcmd.Flags()
	subcmdFlags.Parse(flags.Args()[2:])

	if subcmdFlags.NArg() != 0 {
		fatalf("%s: trailing arguments", subcmdFlags.Name())
	}

	if err := subcmd.Run(); err != nil {
		fatalf("%s: %s", subcmdFlags.Name(), err)
	}
}
