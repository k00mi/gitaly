package repository

// GitRepo supplies an interface for executing `git.Command`s
type GitRepo interface {
	GetStorageName() string
	GetRelativePath() string
	GetGitObjectDirectory() string
	GetGitAlternateObjectDirectories() []string
}
