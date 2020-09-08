package hook

import (
	"gitlab.com/gitlab-org/gitaly/internal/gitaly/config"
)

// Manager is a hook manager containing Git hook business logic.
type Manager struct {
	gitlabAPI   GitlabAPI
	hooksConfig config.Hooks
}

// NewManager returns a new hook manager
func NewManager(gitlabAPI GitlabAPI, hooksConfig config.Hooks) *Manager {
	return &Manager{
		gitlabAPI:   gitlabAPI,
		hooksConfig: hooksConfig,
	}
}
