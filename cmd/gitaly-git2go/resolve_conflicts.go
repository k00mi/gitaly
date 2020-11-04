// +build static,system_libgit2

package main

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	git "github.com/libgit2/git2go/v30"
	"gitlab.com/gitlab-org/gitaly/internal/git/conflict"
	"gitlab.com/gitlab-org/gitaly/internal/git2go"
)

type resolveSubcommand struct {
	request string
}

func (cmd *resolveSubcommand) Flags() *flag.FlagSet {
	flags := flag.NewFlagSet("resolve", flag.ExitOnError)
	flags.StringVar(&cmd.request, "request", "", "git2go.ResolveCommand")
	return flags
}

func (cmd resolveSubcommand) Run(context.Context, io.Reader, io.Writer) error {
	request, err := git2go.ResolveCommandFromSerialized(cmd.request)
	if err != nil {
		return err
	}

	if request.AuthorDate.IsZero() {
		request.AuthorDate = time.Now()
	}

	repo, err := git.OpenRepository(request.Repository)
	if err != nil {
		return fmt.Errorf("could not open repository: %w", err)
	}

	ours, err := lookupCommit(repo, request.Ours)
	if err != nil {
		return fmt.Errorf("could not lookup commit %q: %w", request.Ours, err)
	}

	theirs, err := lookupCommit(repo, request.Theirs)
	if err != nil {
		return fmt.Errorf("could not lookup commit %q: %w", request.Theirs, err)
	}

	index, err := repo.MergeCommits(ours, theirs, nil)
	if err != nil {
		return fmt.Errorf("could not merge commits: %w", err)
	}

	ci, err := index.ConflictIterator()
	if err != nil {
		return err
	}

	type paths struct {
		theirs, ours string
	}
	conflicts := map[paths]git.IndexConflict{}

	for {
		c, err := ci.Next()
		if git.IsErrorCode(err, git.ErrIterOver) {
			break
		}
		if err != nil {
			return err
		}

		if c.Our.Path == "" || c.Their.Path == "" {
			return errors.New("conflict side missing")
		}

		k := paths{
			theirs: c.Their.Path,
			ours:   c.Our.Path,
		}
		conflicts[k] = c
	}

	odb, err := repo.Odb()
	if err != nil {
		return err
	}

	for _, r := range request.Resolutions {
		c, ok := conflicts[paths{
			theirs: r.OldPath,
			ours:   r.NewPath,
		}]
		if !ok {
			// Note: this emulates the Ruby error that occurs when
			// there are no conflicts for a resolution
			return errors.New("NoMethodError: undefined method `resolve_lines' for nil:NilClass") //nolint
		}

		mfr, err := mergeFileResult(odb, c)
		if err != nil {
			return fmt.Errorf("merge file result for %q: %w", r.NewPath, err)
		}

		conflictFile, err := conflict.Parse(
			bytes.NewReader(mfr.Contents),
			c.Our.Path,
			c.Their.Path,
			c.Ancestor.Path,
		)
		if err != nil {
			return fmt.Errorf("parse conflict for %q: %w", c.Ancestor.Path, err)
		}

		resolvedBlob, err := conflictFile.Resolve(r)
		if err != nil {
			return err // do not decorate this error to satisfy old test
		}

		resolvedBlobOID, err := odb.Write(resolvedBlob, git.ObjectBlob)
		if err != nil {
			return fmt.Errorf("write object for %q: %w", c.Ancestor.Path, err)
		}

		ourResolvedEntry := *c.Our // copy by value
		ourResolvedEntry.Id = resolvedBlobOID
		if err := index.Add(&ourResolvedEntry); err != nil {
			return fmt.Errorf("add index for %q: %w", c.Ancestor.Path, err)
		}

		if err := index.RemoveConflict(ourResolvedEntry.Path); err != nil {
			return fmt.Errorf("remove conflict from index for %q: %w", c.Ancestor.Path, err)
		}
	}

	if index.HasConflicts() {
		ci, err := index.ConflictIterator()
		if err != nil {
			return fmt.Errorf("iterating unresolved conflicts: %w", err)
		}

		var conflictPaths []string
		for {
			c, err := ci.Next()
			if git.IsErrorCode(err, git.ErrIterOver) {
				break
			}
			if err != nil {
				return fmt.Errorf("next unresolved conflict: %w", err)
			}
			conflictPaths = append(conflictPaths, c.Ancestor.Path)
		}

		return fmt.Errorf("Missing resolutions for the following files: %s", strings.Join(conflictPaths, ", ")) //nolint
	}

	tree, err := index.WriteTreeTo(repo)
	if err != nil {
		return fmt.Errorf("write tree to repo: %w", err)
	}

	signature := git2go.NewSignature(request.AuthorName, request.AuthorMail, request.AuthorDate)
	committer := &git.Signature{
		Name:  signature.Name,
		Email: signature.Email,
		When:  request.AuthorDate,
	}

	commit, err := repo.CreateCommitFromIds("", committer, committer, request.Message, tree, ours.Id(), theirs.Id())
	if err != nil {
		return fmt.Errorf("could not create resolve conflict commit: %w", err)
	}

	response := git2go.ResolveResult{
		git2go.MergeResult{
			CommitID: commit.String(),
		},
	}

	if err := response.SerializeTo(os.Stdout); err != nil {
		return fmt.Errorf("serializing response: %w", err)
	}

	return nil
}

func mergeFileResult(odb *git.Odb, c git.IndexConflict) (*git.MergeFileResult, error) {
	ancestorBlob, err := odb.Read(c.Ancestor.Id)
	if err != nil {
		return nil, err
	}

	ourBlob, err := odb.Read(c.Our.Id)
	if err != nil {
		return nil, err
	}

	theirBlob, err := odb.Read(c.Their.Id)
	if err != nil {
		return nil, err
	}

	mfr, err := git.MergeFile(
		git.MergeFileInput{
			Path:     c.Ancestor.Path,
			Mode:     uint(c.Ancestor.Mode),
			Contents: ancestorBlob.Data(),
		},
		git.MergeFileInput{
			Path:     c.Our.Path,
			Mode:     uint(c.Our.Mode),
			Contents: ourBlob.Data(),
		},
		git.MergeFileInput{
			Path:     c.Their.Path,
			Mode:     uint(c.Their.Mode),
			Contents: theirBlob.Data(),
		},
		nil,
	)
	if err != nil {
		return nil, err
	}

	return mfr, nil
}
