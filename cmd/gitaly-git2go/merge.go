// +build static,system_libgit2

package main

import (
	"errors"
	"flag"
	"fmt"
	"strings"
	"time"

	git "github.com/libgit2/git2go/v30"
)

type mergeSubcommand struct {
	repository string
	authorName string
	authorMail string
	authorDate string
	message    string
	ours       string
	theirs     string
}

func (cmd *mergeSubcommand) Flags() *flag.FlagSet {
	flags := flag.NewFlagSet("merge", flag.ExitOnError)
	flags.StringVar(&cmd.repository, "repository", "", "repository to merge branches in")
	flags.StringVar(&cmd.authorName, "author-name", "", "author name to use for the merge commit")
	flags.StringVar(&cmd.authorMail, "author-mail", "", "author mail to use for the merge commit")
	flags.StringVar(&cmd.authorDate, "author-date", "", "author date to use for the merge commit")
	flags.StringVar(&cmd.message, "message", "", "the commit message to use for the merge commit")
	flags.StringVar(&cmd.ours, "ours", "", "the commit that reflects the destination tree")
	flags.StringVar(&cmd.theirs, "theirs", "", "the commit to merge into the destination tree")
	return flags
}

func (cmd *mergeSubcommand) verifyOptions() error {
	if cmd.repository == "" {
		return errors.New("missing repository")
	}
	if cmd.authorName == "" {
		return errors.New("missing author name")
	}
	if cmd.authorMail == "" {
		return errors.New("missing author mail")
	}
	if cmd.message == "" {
		return errors.New("missing message")
	}
	if cmd.ours == "" {
		return errors.New("missing ours")
	}
	if cmd.theirs == "" {
		return errors.New("missing theirs")
	}
	return nil
}

func lookupCommit(repo *git.Repository, ref string) (*git.Commit, error) {
	object, err := repo.RevparseSingle(ref)
	if err != nil {
		return nil, fmt.Errorf("could not lookup reference: %w", err)
	}

	peeled, err := object.Peel(git.ObjectCommit)
	if err != nil {
		return nil, fmt.Errorf("could not peel reference: %w", err)
	}

	commit, err := peeled.AsCommit()
	if err != nil {
		return nil, fmt.Errorf("could not cast to commit: %w", err)
	}

	return commit, nil
}

func sanitizeSignatureInfo(info string) string {
	return strings.Map(func(r rune) rune {
		switch r {
		case '<', '>', '\n':
			return -1
		default:
			return r
		}
	}, info)
}

func (cmd *mergeSubcommand) Run() error {
	if err := cmd.verifyOptions(); err != nil {
		return fmt.Errorf("invalid options: %w", err)
	}

	var date time.Time = time.Now()
	if cmd.authorDate != "" {
		var err error
		date, err = time.Parse("Mon Jan 2 15:04:05 2006 -0700", cmd.authorDate)
		if err != nil {
			return err
		}
	}

	repo, err := git.OpenRepository(cmd.repository)
	if err != nil {
		return fmt.Errorf("could not open repository: %w", err)
	}

	ours, err := lookupCommit(repo, cmd.ours)
	if err != nil {
		return fmt.Errorf("could not lookup commit %q: %w", cmd.ours, err)
	}

	theirs, err := lookupCommit(repo, cmd.theirs)
	if err != nil {
		return fmt.Errorf("could not lookup commit %q: %w", cmd.theirs, err)
	}

	index, err := repo.MergeCommits(ours, theirs, nil)
	if err != nil {
		return fmt.Errorf("could not merge commits: %w", err)
	}

	if index.HasConflicts() {
		return errors.New("could not auto-merge due to conflicts")
	}

	tree, err := index.WriteTreeTo(repo)
	if err != nil {
		return fmt.Errorf("could not write tree: %w", err)
	}

	committer := git.Signature{
		Name:  sanitizeSignatureInfo(cmd.authorName),
		Email: sanitizeSignatureInfo(cmd.authorMail),
		When:  date,
	}

	commit, err := repo.CreateCommitFromIds("", &committer, &committer, cmd.message, tree, ours.Id(), theirs.Id())
	if err != nil {
		return fmt.Errorf("could not create merge commit: %w", err)
	}

	fmt.Println(commit.String())

	return nil
}
