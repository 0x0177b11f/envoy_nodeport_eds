package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"time"

	cluster "github.com/0x0177b11f/envoy_nodeport_eds/cluster"
	config "github.com/0x0177b11f/envoy_nodeport_eds/config"
	util "github.com/0x0177b11f/envoy_nodeport_eds/util"

	api "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v2"
	cache "github.com/envoyproxy/go-control-plane/pkg/cache/v2"
	xds "github.com/envoyproxy/go-control-plane/pkg/server/v2"
	grpc "google.golang.org/grpc"

	corev1 "k8s.io/api/core/v1"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

func main() {
	var kubeconfig *string

	if home := homeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}

	var addressType *string = flag.String("addressType", string(corev1.NodeInternalIP), "node address type")
	var namespace *string = flag.String("namespace", "default", "kube namespace")
	var serviceName *string = flag.String("serviceName", "*", "service name (default is all nodeport service)")
	var intervals *int = flag.Int("intervals", 30, "update intervals")
	var listenerAddress *string = flag.String("listener", "0.0.0.0", "listener address")
	var port *uint = flag.Uint("port", 8000, "listener port")

	flag.Parse()

	if !(*addressType == string(corev1.NodeHostName) ||
		*addressType == string(corev1.NodeExternalIP) ||
		*addressType == string(corev1.NodeInternalIP) ||
		*addressType == string(corev1.NodeExternalDNS) ||
		*addressType == string(corev1.NodeInternalDNS)) {
		panic(errors.New("addressType error"))
	}

	ch := make(chan []config.EndpointAddress)
	go func() {
		for {
			func() {
				defer func() {
					if r := recover(); r != nil {
						fmt.Println("Error: ", r)
					}
				}()

				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()

				address := cluster.GetAllNodeAddress(ctx, kubeconfig, corev1.NodeAddressType(*addressType))
				nodeports := cluster.GetAllNodePortService(ctx, kubeconfig, namespace, serviceName)

				var EndpointList []config.EndpointAddress
				for _, addre := range address {
					for _, nodeport := range nodeports {
						EndpointList = append(EndpointList, config.EndpointAddress{
							Name:     nodeport.Name,
							Address:  addre,
							Port:     nodeport.Port,
							Protocol: util.ConvKubePortoclToEnvoyPortocl(nodeport.Protocl),
						})
					}
				}
				if len(EndpointList) > 0 {
					ch <- EndpointList
				}
			}()

			time.Sleep(time.Duration(*intervals) * time.Second)
		}
	}()

	snapshotCache := cache.NewSnapshotCache(false, cache.IDHash{}, nil)
	server := xds.NewServer(context.Background(), snapshotCache, nil)
	grpcServer := grpc.NewServer()
	listen, _ := net.Listen("tcp", fmt.Sprintf("%s:%d", *listenerAddress, *port))

	discovery.RegisterAggregatedDiscoveryServiceServer(grpcServer, server)
	api.RegisterEndpointDiscoveryServiceServer(grpcServer, server)
	api.RegisterClusterDiscoveryServiceServer(grpcServer, server)
	api.RegisterRouteDiscoveryServiceServer(grpcServer, server)
	api.RegisterListenerDiscoveryServiceServer(grpcServer, server)

	go func() {
		var count uint64 = 1
		for {
			endpoints := <-ch
			cfg := config.NewNodeConfig(*namespace)

			clustersEndpoint := make(map[string][]config.EndpointAddress)
			for _, endpoint := range endpoints {
				name := endpoint.Name
				if name == "" {
					name = "None"
				}
				clustersEndpoint[name] = append(clustersEndpoint[name], endpoint)
			}

			for key, value := range clustersEndpoint {
				cfg.AddClusterWithStaticEndpoint(key, value)
			}

			count++
			config.UpdateSnapshotCache(snapshotCache, cfg, string(count))
		}
	}()

	if err := grpcServer.Serve(listen); err != nil {
		log.Println(err)
	}
}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}
