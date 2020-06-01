package config

import (
	"log"

	cluster "github.com/0x0177b11f/envoy_nodeport_eds/cluster"

	core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	endpoint "github.com/envoyproxy/go-control-plane/envoy/api/v2/endpoint"
	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	cache "github.com/envoyproxy/go-control-plane/pkg/cache/v2"
)

type EndpointAddress struct {
	Name     string
	Address  string
	Port     uint32
	Protocol core.SocketAddress_Protocol
}

type NodeConfig struct {
	node      string
	endpoints []types.Resource
	clusters  []types.Resource
	routes    []types.Resource
	listeners []types.Resource
	runtimes  []types.Resource
}

func NewNodeConfig(NodeName string) *NodeConfig {
	return &NodeConfig{
		node:      NodeName,
		endpoints: []types.Resource{},
		clusters:  []types.Resource{},
		routes:    []types.Resource{},
		listeners: []types.Resource{},
	}
}

func (n *NodeConfig) AddClusterWithStaticEndpoint(ClusterName string, AddressList []EndpointAddress) {
	var endpoints []*endpoint.Endpoint
	for _, address := range AddressList {
		newEndpoint := cluster.CreateEndpoint(address.Address, address.Port, address.Protocol)
		endpoints = append(endpoints, newEndpoint)
	}
	loadAssignmentgo := cluster.CreateLoadAssignment(ClusterName, endpoints)
	cluster := cluster.CreateCluster(ClusterName, loadAssignmentgo)
	n.clusters = append(n.clusters, cluster)
	n.endpoints = append(n.endpoints, loadAssignmentgo)
}

func UpdateSnapshotCache(s cache.SnapshotCache, n *NodeConfig, version string) {
	err := s.SetSnapshot(
		n.node,
		cache.NewSnapshot(
			version,
			n.endpoints,
			n.clusters,
			n.routes,
			n.listeners,
			n.runtimes))

	if err != nil {
		log.Println(err)
	}
}
