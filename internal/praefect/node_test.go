package praefect

import (
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/gitaly/internal/praefect/config"
	"gitlab.com/gitlab-org/gitaly/internal/testhelper"
	"google.golang.org/grpc"
)

func TestDialNodes(t *testing.T) {
	ctx, cancel := testhelper.Context()
	defer cancel()

	tmp, err := ioutil.TempDir("", "praefect")
	require.NoError(t, err)
	defer os.RemoveAll(tmp)

	type nodeAssertion struct {
		storage string
		token   string
	}

	expectedNodes := []nodeAssertion{
		{
			storage: "healthy",
			token:   "healthy-token",
		},
		{
			storage: "unhealthy",
			token:   "unhealthy-token",
		},
	}

	var cfgNodes []*config.Node
	for _, n := range expectedNodes {
		socket := filepath.Join(tmp, n.storage)
		ln, err := net.Listen("unix", socket)
		require.NoError(t, err)
		srv := grpc.NewServer()
		defer srv.Stop()
		go srv.Serve(ln)

		cfgNodes = append(cfgNodes, &config.Node{
			Storage: n.storage,
			Token:   n.token,
			Address: fmt.Sprintf("%s://%s", ln.Addr().Network(), ln.Addr().String()),
		})
	}

	nodeSet, err := DialNodes(ctx,
		[]*config.VirtualStorage{{Name: "virtual-storage", Nodes: cfgNodes}}, nil, nil,
	)
	require.NoError(t, err)
	defer nodeSet.Close()

	var actualNodes []nodeAssertion
	for _, nodes := range nodeSet {
		for _, node := range nodes {
			actualNodes = append(actualNodes, nodeAssertion{
				storage: node.Storage,
				token:   node.Token,
			})
		}
	}

	require.ElementsMatch(t, expectedNodes, actualNodes)
}
