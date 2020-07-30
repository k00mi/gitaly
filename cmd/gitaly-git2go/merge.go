// +build static,system_libgit2

package main

import (
	"flag"
)

type mergeSubcommand struct {
}

func (cmd *mergeSubcommand) Flags() *flag.FlagSet {
	flags := flag.NewFlagSet("merge", flag.ExitOnError)
	return flags
}

func (cmd *mergeSubcommand) Run() error {
	return nil
}
