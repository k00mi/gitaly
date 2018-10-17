package hooks

import (
	"path"

	"gitlab.com/gitlab-org/gitaly/internal/config"
)

// Path returns the path where the global git hooks are located
func Path() string {
	return path.Join(config.Config.GitlabShell.Dir, "hooks")
}
