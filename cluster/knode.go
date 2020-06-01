package cluster

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"k8s.io/client-go/tools/clientcmd"
)

type NodeportServiceInfo struct {
	Name    string
	Protocl corev1.Protocol
	Port    uint32
}

func GetAllNodePortService(kubeconfig, namespace, servicename *string) []NodeportServiceInfo {
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		panic(err.Error())
	}

	ns := func() string {
		if namespace == nil {
			return "default"
		}
		return *namespace
	}()

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	services, err := clientset.CoreV1().Services(ns).List(metav1.ListOptions{})
	if err != nil {
		panic(err.Error())
	}

	var endpoints []NodeportServiceInfo
	for _, service := range services.Items {
		if service.Spec.Type != corev1.ServiceTypeNodePort {
			continue
		}

		if servicename != nil &&
			*servicename != "*" &&
			service.GetName() != *servicename {
			continue
		}

		for _, ServicePort := range service.Spec.Ports {
			if ServicePort.Protocol != corev1.ProtocolTCP &&
				ServicePort.Protocol != corev1.ProtocolUDP {
				continue
			}

			endpoints = append(endpoints, NodeportServiceInfo{
				Name:    ServicePort.Name,
				Protocl: ServicePort.Protocol,
				Port:    uint32(ServicePort.NodePort),
			})
		}
	}
	return endpoints
}

func GetAllNodeAddress(kubeconfig *string, addresstype corev1.NodeAddressType) []string {
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		panic(err.Error())
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	nodes, err := clientset.CoreV1().Nodes().List(metav1.ListOptions{})
	if err != nil {
		panic(err.Error())
	}

	var address []string
	for _, node := range nodes.Items {
		nodeinfo, err := clientset.CoreV1().Nodes().Get(node.Name, metav1.GetOptions{})
		if err != nil {
			continue
		}

		for _, nodeaddress := range nodeinfo.Status.Addresses {
			if nodeaddress.Type == addresstype {
				address = append(address, nodeaddress.Address)
			}
		}
	}
	return address
}
