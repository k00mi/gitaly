package praefect

import (
	"io/ioutil"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/gitaly/internal/log"
	"gitlab.com/gitlab-org/gitaly/internal/praefect/config"
	"gitlab.com/gitlab-org/gitaly/internal/praefect/conn"
	"gitlab.com/gitlab-org/gitaly/internal/praefect/models"
	"gitlab.com/gitlab-org/gitaly/internal/praefect/protoregistry"
	"gitlab.com/gitlab-org/gitaly/internal/testhelper"
	"gitlab.com/gitlab-org/gitaly/proto/go/gitalypb"
)

var testLogger = logrus.New()

func init() {
	testLogger.SetOutput(ioutil.Discard)
}

func TestSecondaryRotation(t *testing.T) {
	t.Skip("secondary rotation will change with the new data model")
}

func TestStreamDirector(t *testing.T) {
	datastore := NewMemoryDatastore(config.Config{
		Nodes: []*models.Node{
			&models.Node{
				Address:        "tcp://gitaly-primary.example.com",
				Storage:        "praefect-internal-1",
				DefaultPrimary: true,
			},
			&models.Node{
				Address: "tcp://gitaly-backup1.example.com",
				Storage: "praefect-internal-2",
			}},
	})

	targetRepo := gitalypb.Repository{
		StorageName:  "praefect",
		RelativePath: "/path/to/hashed/storage",
	}

	ctx, cancel := testhelper.Context()
	defer cancel()

	clientConnections := conn.NewClientConnections()
	clientConnections.RegisterNode("praefect-internal-1", "tcp://gitaly-primary.example.com")

	coordinator := NewCoordinator(log.Default(), datastore, clientConnections)
	require.NoError(t, coordinator.RegisterProtos(protoregistry.GitalyProtoFileDescriptors...))

	frame, err := proto.Marshal(&gitalypb.GarbageCollectRequest{
		Repository: &targetRepo,
	})
	require.NoError(t, err)

	_, conn, jobUpdateFunc, err := coordinator.streamDirector(ctx, "/gitaly.RepositoryService/GarbageCollect", &mockPeeker{frame})
	require.NoError(t, err)
	t.Logf("CONN %+v", conn)

	jobs, err := datastore.GetJobs(JobStatePending, 1, 10)
	require.NoError(t, err)
	require.Len(t, jobs, 1)

	targetNode, err := datastore.GetStorageNode(1)
	require.NoError(t, err)
	sourceNode, err := datastore.GetStorageNode(0)
	require.NoError(t, err)

	expectedJob := ReplJob{
		ID:         1,
		TargetNode: targetNode,
		SourceNode: sourceNode,
		State:      JobStatePending,
		Repository: models.Repository{RelativePath: targetRepo.RelativePath, Primary: sourceNode, Replicas: []models.Node{targetNode}},
	}

	require.Equal(t, expectedJob, jobs[0], "ensure replication job created by stream director is correct")

	jobUpdateFunc()

	jobs, err = coordinator.datastore.GetJobs(JobStateReady, 1, 10)
	require.NoError(t, err)
	require.Len(t, jobs, 1)

	expectedJob.State = JobStateReady
	require.Equal(t, expectedJob, jobs[0], "ensure replication job's status has been updatd to JobStateReady")
}

type mockPeeker struct {
	frame []byte
}

func (m *mockPeeker) Peek() ([]byte, error) {
	return m.frame, nil
}

func (m *mockPeeker) Modify(payload []byte) error {
	return nil
}
