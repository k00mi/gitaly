package main

import (
	"io/ioutil"
	"log"
	"os"

	git "github.com/libgit2/git2go"
)

// echo "https://remote.url.com" | gitaly-remote /path/to/repository remote-name
func main() {
	if n := len(os.Args); n != 3 {
		log.Fatalf("invalid number of arguments, expected 2, got %d", n-1)
	}

	repoPath := os.Args[1]
	remoteName := os.Args[2]

	remoteURLBytes, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		log.Fatalf("error while reading remote URL: %v", err)
	}
	remoteURL := string(remoteURLBytes)

	err = addOrUpdate(repoPath, remoteName, remoteURL)
	if err != nil {
		log.Fatalf("error while setting remote: %v", err)
	}

}

//addOrUpdate adds, or updates if exits, the remote to repository
func addOrUpdate(repoPath string, name, url string) error {
	if err := add(repoPath, name, url); err != nil {
		return update(repoPath, name, url)
	}

	return nil
}

func add(repoPath string, name, url string) error {
	git2repo, err := openGit2GoRepo(repoPath)
	if err != nil {
		return err
	}

	_, err = git2repo.Remotes.Create(name, url)
	return err
}

func update(repoPath string, name, url string) error {
	git2repo, err := openGit2GoRepo(repoPath)
	if err != nil {
		return err
	}

	return git2repo.Remotes.SetUrl(name, url)
}

func openGit2GoRepo(repoPath string) (*git.Repository, error) {
	git2repo, err := git.OpenRepository(repoPath)
	if err != nil {
		return nil, err
	}

	return git2repo, nil
}
