package cluster

import (
	"time"

	"github.com/golang/protobuf/ptypes"

	api "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	endpoint "github.com/envoyproxy/go-control-plane/envoy/api/v2/endpoint"
)

func CreateEndpoint(Address string, Port uint32, Protocol core.SocketAddress_Protocol) *endpoint.Endpoint {
	var PortSpecifier core.SocketAddress_PortValue
	PortSpecifier.PortValue = Port

	var SocketAddress core.SocketAddress
	SocketAddress.Address = Address
	SocketAddress.PortSpecifier = &PortSpecifier
	SocketAddress.Protocol = Protocol

	var EndpointAddress core.Address
	EndpointAddress.Address = &core.Address_SocketAddress{
		SocketAddress: &SocketAddress,
	}

	return &endpoint.Endpoint{
		Address: &EndpointAddress,
	}
}

func CreateLoadAssignment(ClusterName string, Endpoints []*endpoint.Endpoint) *api.ClusterLoadAssignment {
	var LbEndpoints []*endpoint.LbEndpoint

	for _, Endpoint := range Endpoints {
		LbEndpoints = append(LbEndpoints, &endpoint.LbEndpoint{
			HostIdentifier: &endpoint.LbEndpoint_Endpoint{
				Endpoint: Endpoint,
			},
		})
	}

	return &api.ClusterLoadAssignment{
		ClusterName: ClusterName,
		Endpoints: []*endpoint.LocalityLbEndpoints{{
			LbEndpoints: LbEndpoints,
		}},
	}
}

func CreateCluster(Name string, LoadAssignment *api.ClusterLoadAssignment) *api.Cluster {
	return &api.Cluster{
		Name:                 Name,
		ConnectTimeout:       ptypes.DurationProto(1 * time.Second),
		ClusterDiscoveryType: &api.Cluster_Type{Type: api.Cluster_EDS},
		DnsLookupFamily:      api.Cluster_V4_ONLY,
		LbPolicy:             api.Cluster_ROUND_ROBIN,
		LoadAssignment:       LoadAssignment,
	}
}
