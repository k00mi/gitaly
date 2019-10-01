package conn

import (
	"errors"
	"sync"

	gitalyauth "gitlab.com/gitlab-org/gitaly/auth"
	"gitlab.com/gitlab-org/gitaly/client"
	gitalyconfig "gitlab.com/gitlab-org/gitaly/internal/config"
	"gitlab.com/gitlab-org/gitaly/internal/praefect/grpc-proxy/proxy"
	"google.golang.org/grpc"
)

// ClientConnections contains ready to use grpc client connections
type ClientConnections struct {
	connMutex sync.RWMutex
	nodes     map[string]*grpc.ClientConn
}

// NewClientConnections creates a new ClientConnections struct
func NewClientConnections() *ClientConnections {
	return &ClientConnections{
		nodes: make(map[string]*grpc.ClientConn),
	}
}

// RegisterNode will direct traffic to the supplied downstream connection when the storage location
// is encountered.
func (c *ClientConnections) RegisterNode(storageName, listenAddr string) error {
	conn, err := client.Dial(listenAddr,
		[]grpc.DialOption{
			grpc.WithDefaultCallOptions(grpc.CallCustomCodec(proxy.Codec())),
			grpc.WithPerRPCCredentials(gitalyauth.RPCCredentials(gitalyconfig.Config.Auth.Token)),
		},
	)
	if err != nil {
		return err
	}

	c.setConn(storageName, conn)

	return nil
}

func (c *ClientConnections) setConn(storageName string, conn *grpc.ClientConn) {
	c.connMutex.Lock()
	c.nodes[storageName] = conn
	c.connMutex.Unlock()
}

// GetConnection gets the grpc client connection based on an address
func (c *ClientConnections) GetConnection(storageName string) (*grpc.ClientConn, error) {
	c.connMutex.RLock()
	cc, ok := c.nodes[storageName]
	c.connMutex.RUnlock()
	if !ok {
		return nil, errors.New("client connection not found")
	}

	return cc, nil

}
