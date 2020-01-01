package util

import (
	corev1 "k8s.io/api/core/v1"

	core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
)

func ConvKubePortoclToEnvoyPortocl(protocol corev1.Protocol) core.SocketAddress_Protocol {
	if protocol == corev1.ProtocolTCP {
		return core.SocketAddress_TCP
	}

	if protocol == corev1.ProtocolUDP {
		return core.SocketAddress_UDP
	}

	panic(protocol)
}
