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

	cluster "envoy_nodeport_eds/cluster"
	config "envoy_nodeport_eds/config"
	util "envoy_nodeport_eds/util"

	api "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v2"
	cache "github.com/envoyproxy/go-control-plane/pkg/cache"
	xds "github.com/envoyproxy/go-control-plane/pkg/server"
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
	var addressType *string
	addressType = flag.String("addressType", string(corev1.NodeInternalIP), "node address type")

	var namespace *string
	namespace = flag.String("namespace", "default", "kube namespace")

	var serviceName *string
	serviceName = flag.String("serviceName", "*", "service name (default is all nodeport service)")

	var intervals *int
	intervals = flag.Int("intervals", 30, "update intervals")

	var listenerAddress *string
	listenerAddress = flag.String("listener", "0.0.0.0", "listener address")

	var port *uint
	port = flag.Uint("port", 8000, "listener port")

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

		sleep := func() {
			time.Sleep(time.Duration(*intervals) * time.Second)
		}

		for {
			address, err := cluster.GetAllNodeAddress(kubeconfig, corev1.NodeAddressType(*addressType))
			if err != nil {
				sleep()
				continue
			}

			nodeports, err := cluster.GetAllNodePortService(kubeconfig, namespace, serviceName)
			if err != nil {
				sleep()
				continue
			}

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

			sleep()
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
