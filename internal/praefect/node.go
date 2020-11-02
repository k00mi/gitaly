package praefect

import (
	"gitlab.com/gitlab-org/gitaly/internal/praefect/nodes"
	"google.golang.org/grpc"
)

// Node is a storage node in a virtual storage.
type Node struct {
	// Storage is the name of the storage node.
	Storage string
	// Address is the address of the node.
	Address string
	// Token is the authentication token of the node.
	Token string
	// Connection is a gRPC connection to the storage node.
	Connection *grpc.ClientConn
}

// NodeSet contains nodes by their virtual storage and storage names.
type NodeSet map[string]map[string]Node

// NodeSetFromNodeManager converts connections set up by the node manager
// in to a NodeSet. This is a temporary adapter required due to cyclic
// imports between the praefect and nodes packages.
func NodeSetFromNodeManager(mgr nodes.Manager) NodeSet {
	nodes := mgr.Nodes()

	set := make(NodeSet, len(nodes))
	for virtualStorage, nodes := range nodes {
		set[virtualStorage] = make(map[string]Node, len(nodes))
		for _, node := range nodes {
			set[virtualStorage][node.GetStorage()] = toNode(node)
		}
	}

	return set
}

func toNode(node nodes.Node) Node {
	return Node{
		Storage:    node.GetStorage(),
		Address:    node.GetAddress(),
		Token:      node.GetToken(),
		Connection: node.GetConnection(),
	}
}
