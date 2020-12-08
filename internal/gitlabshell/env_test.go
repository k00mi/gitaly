package gitlabshell_test

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/gitaly/internal/gitaly/config"
	"gitlab.com/gitlab-org/gitaly/internal/gitlabshell"
	"gitlab.com/gitlab-org/gitaly/internal/testhelper"
)

func TestGitHooksConfig(t *testing.T) {
	defer func(cfg config.Cfg) {
		config.Config = cfg
	}(config.Config)

	require.NoError(t, testhelper.ConfigureRuby(&config.Config))

	loggingDir, err := ioutil.TempDir("", t.Name())
	require.NoError(t, err)
	defer func() { os.RemoveAll(loggingDir) }()

	config.Config.Logging.Dir = loggingDir
	config.Config.Logging.Level = "fatal"
	config.Config.Logging.Format = "my-custom-format"
	config.Config.GitlabShell.Dir = "../../ruby/gitlab-shell"
	config.Config.Hooks.CustomHooksDir = "/path/to/custom_hooks"
	config.Config.Gitlab = config.Gitlab{
		URL: "http://gitlaburl.com",
		HTTPSettings: config.HTTPSettings{
			ReadTimeout: 100,
			User:        "user_name",
			Password:    "pwpw",
			CAFile:      "/ca_file_path",
			CAPath:      "/ca_path",
			SelfSigned:  true,
		},
		SecretFile: "secret_file_path",
	}

	env, err := gitlabshell.EnvFromConfig(config.Config)
	require.NoError(t, err)

	require.Contains(t, env, "GITALY_GITLAB_SHELL_DIR="+config.Config.GitlabShell.Dir)
	require.Contains(t, env, "GITALY_LOG_DIR="+config.Config.Logging.Dir)
	require.Contains(t, env, "GITALY_LOG_FORMAT="+config.Config.Logging.Format)
	require.Contains(t, env, "GITALY_LOG_LEVEL="+config.Config.Logging.Level)
	require.Contains(t, env, "GITALY_BIN_DIR="+config.Config.BinDir)

	jsonShellConfig := ""
	for _, envVar := range env {
		if strings.HasPrefix(envVar, "GITALY_GITLAB_SHELL_CONFIG=") {
			jsonShellConfig = strings.SplitN(envVar, "=", 2)[1]
			break
		}
	}

	var configMap map[string]interface{}

	require.NoError(t, json.Unmarshal([]byte(jsonShellConfig), &configMap))
	require.Equal(t, config.Config.Logging.Level, configMap["log_level"])
	require.Equal(t, config.Config.Logging.Format, configMap["log_format"])
	require.Equal(t, config.Config.Gitlab.SecretFile, configMap["secret_file"])
	require.Equal(t, config.Config.Hooks.CustomHooksDir, configMap["custom_hooks_dir"])
	require.Equal(t, config.Config.Gitlab.URL, configMap["gitlab_url"])

	// HTTP Settings
	httpSettings, ok := configMap["http_settings"].(map[string]interface{})
	require.True(t, ok)
	require.Equal(t, float64(config.Config.Gitlab.HTTPSettings.ReadTimeout), httpSettings["read_timeout"])
	require.Equal(t, config.Config.Gitlab.HTTPSettings.User, httpSettings["user"])
	require.Equal(t, config.Config.Gitlab.HTTPSettings.Password, httpSettings["password"])
	require.Equal(t, config.Config.Gitlab.HTTPSettings.CAFile, httpSettings["ca_file"])
	require.Equal(t, config.Config.Gitlab.HTTPSettings.CAPath, httpSettings["ca_path"])
	require.Equal(t, config.Config.Gitlab.HTTPSettings.SelfSigned, httpSettings["self_signed_cert"])
}
