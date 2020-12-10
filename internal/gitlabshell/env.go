package gitlabshell

import (
	"encoding/json"
	"fmt"

	"gitlab.com/gitlab-org/gitaly/internal/gitaly/config"
	"gitlab.com/gitlab-org/gitaly/internal/log"
)

type Config struct {
	CustomHooksDir string              `json:"custom_hooks_dir"`
	GitlabURL      string              `json:"gitlab_url"`
	HTTPSettings   config.HTTPSettings `json:"http_settings"`
	LogFormat      string              `json:"log_format"`
	LogLevel       string              `json:"log_level"`
	LogPath        string              `json:"log_path"`
	SecretFile     string              `json:"secret_file"`
}

// EnvFromConfig returns a set of environment variables from a config struct relevant to gitlab shell
func EnvFromConfig(cfg config.Cfg) ([]string, error) {
	gitlabShellConfig := Config{
		CustomHooksDir: cfg.Hooks.CustomHooksDir,
		GitlabURL:      cfg.Gitlab.URL,
		HTTPSettings:   cfg.Gitlab.HTTPSettings,
		LogFormat:      cfg.Logging.Format,
		LogLevel:       cfg.Logging.Level,
		LogPath:        cfg.Logging.Dir,
		SecretFile:     cfg.Gitlab.SecretFile,
	}

	gitlabShellConfigString, err := json.Marshal(&gitlabShellConfig)
	if err != nil {
		return nil, err
	}

	return []string{
		fmt.Sprintf("%s=%s", log.GitalyLogDirEnvKey, cfg.Logging.Dir),
		"GITALY_BIN_DIR=" + cfg.BinDir,
		"GITALY_GITLAB_SHELL_CONFIG=" + string(gitlabShellConfigString),
	}, nil
}
