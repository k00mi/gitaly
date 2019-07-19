package praefect

import (
	"io/ioutil"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/gitaly/internal/praefect/config"
	"gitlab.com/gitlab-org/gitaly/internal/praefect/models"
)

var testLogger = logrus.New()

func init() {
	testLogger.SetOutput(ioutil.Discard)
}

func TestSecondaryRotation(t *testing.T) {
	cfg := config.Config{
		PrimaryServer:    &models.GitalyServer{Name: "primary"},
		SecondaryServers: []*models.GitalyServer{&models.GitalyServer{Name: "secondary_1"}, &models.GitalyServer{Name: "secondary_2"}},
		Whitelist:        []string{"/repoA", "/repoB"},
	}
	d := NewMemoryDatastore(cfg)
	c := NewCoordinator(testLogger, d)

	primary, err := d.GetDefaultPrimary()
	require.NoError(t, err)

	require.NoError(t, c.rotateSecondaryToPrimary(primary))

	primary, err = d.GetDefaultPrimary()
	require.NoError(t, err)
	require.Equal(t, *cfg.SecondaryServers[0], primary, "the first secondary should have gotten promoted to be primary")

	repositories, err := d.GetRepositoriesForPrimary(primary)
	require.NoError(t, err)

	for _, repository := range repositories {
		shardSecondaries, err := d.GetShardSecondaries(models.Repository{RelativePath: repository})
		require.NoError(t, err)

		require.Len(t, shardSecondaries, 2)
		require.Equal(t, *cfg.SecondaryServers[1], shardSecondaries[0])
	}
}
