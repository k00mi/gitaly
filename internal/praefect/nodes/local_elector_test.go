package nodes

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/gitaly/internal/praefect/models"
	"gitlab.com/gitlab-org/gitaly/internal/testhelper"
	"gitlab.com/gitlab-org/gitaly/internal/testhelper/promtest"
	"google.golang.org/grpc"
)

func TestPrimaryAndSecondaries(t *testing.T) {
	socket := testhelper.GetTemporaryGitalySocketFileName()
	svr, _ := testhelper.NewServerWithHealth(t, socket)
	defer svr.Stop()

	cc, err := grpc.Dial(
		"unix://"+socket,
		grpc.WithInsecure(),
	)

	require.NoError(t, err)

	storageName := "default"
	mockHistogramVec := promtest.NewMockHistogramVec()

	cs := newConnectionStatus(models.Node{Storage: storageName}, cc, testhelper.DiscardTestEntry(t), mockHistogramVec)
	strategy := newLocalElector(storageName, true)

	strategy.addNode(cs, true)
	strategy.bootstrap(time.Second)

	primary, err := strategy.GetPrimary()

	require.NoError(t, err)
	require.Equal(t, primary, cs)

	secondaries, err := strategy.GetSecondaries()

	require.NoError(t, err)
	require.Equal(t, 0, len(secondaries))

	secondary := newConnectionStatus(models.Node{Storage: storageName}, cc, testhelper.DiscardTestEntry(t), nil)
	strategy.addNode(secondary, false)

	secondaries, err = strategy.GetSecondaries()

	require.NoError(t, err)
	require.Equal(t, 1, len(secondaries))
	require.Equal(t, secondary, secondaries[0])
}
