package linguist

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/gitaly/internal/gitaly/config"
	"gitlab.com/gitlab-org/gitaly/internal/testhelper"
)

func TestMain(m *testing.M) {
	os.Exit(testMain(m))
}

func testMain(m *testing.M) int {
	defer testhelper.MustHaveNoChildProcess()
	cleanup := testhelper.Configure()
	defer cleanup()
	return m.Run()
}

func TestStatsUnmarshalJSONError(t *testing.T) {
	ctx, cancel := testhelper.Context()
	defer cancel()

	// When an error occurs, this used to trigger JSON marshelling of a plain string
	// the new behaviour shouldn't do that, and return an command error
	_, err := Stats(ctx, config.Config, "/var/empty", "deadbeef")
	require.Error(t, err)

	_, ok := err.(*json.SyntaxError)
	require.False(t, ok, "expected the error not be a json Syntax Error")
}

func TestLoadLanguages(t *testing.T) {
	defer func(old config.Cfg) { config.Config = old }(config.Config)

	colorMap = make(map[string]Language)
	require.NoError(t, LoadColors(&config.Config), "load colors")

	require.Equal(t, "#701516", Color("Ruby"), "color value for 'Ruby'")
}

func TestLoadLanguagesCustomPath(t *testing.T) {
	defer func(old config.Cfg) { config.Config = old }(config.Config)

	jsonPath, err := filepath.Abs("testdata/fake-languages.json")
	require.NoError(t, err)

	config.Config.Ruby.LinguistLanguagesPath = jsonPath

	colorMap = make(map[string]Language)
	require.NoError(t, LoadColors(&config.Config), "load colors")

	require.Equal(t, "foo color", Color("FooBar"))
}
