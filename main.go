package main

import (
	"flag"
	"fmt"
	"log"
	"time"

	consul "github.com/hashicorp/consul/api"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"net"
)

type consulSideKick struct {
	namespace    string
	podName      string
	consulClient *consul.Client
	k8sClient    *kubernetes.Clientset
}

func (c consulSideKick) getPodInfo() (ip string, ownerSelector labels.Selector, err error) {
	pod, err := c.k8sClient.CoreV1().Pods(c.namespace).Get(c.podName, v1.GetOptions{})
	if err != nil {
		return "", nil, fmt.Errorf("cannot find consul pod (%s/%s): %v", c.namespace, c.podName, err)
	}
	if len(pod.OwnerReferences) != 1 {
		return "", nil, fmt.Errorf("cannot determine owner of consul pod (%s/%s)", c.namespace, c.podName)
	}
	ownerReference := pod.OwnerReferences[0]
	if !*ownerReference.Controller || ownerReference.Kind != "ReplicaSet" {
		return "", nil, fmt.Errorf("consul pod (%s/%s) is not owned by a ReplicaSet", c.namespace, c.podName)
	}
	replicaSetName := ownerReference.Name
	replicaSet, err := c.k8sClient.ReplicaSets(c.namespace).Get(replicaSetName, v1.GetOptions{})
	if err != nil {
		return "", nil, fmt.Errorf("Cannot access ReplicaSet (%s/%s)", c.namespace, replicaSetName)
	}
	podSelector := replicaSet.Spec.Selector
	selector, err := v1.LabelSelectorAsSelector(podSelector)
	if err != nil {
		return "", nil, fmt.Errorf("cannot prettyprint selector (selector: %v): %v", podSelector, err)
	}
	return pod.Status.PodIP, selector, nil
}

func (c consulSideKick) getPodIPs(selector labels.Selector) (map[string]struct{}, error) {
	podIPs := map[string]struct{}{}
	listOptions := v1.ListOptions{LabelSelector: selector.String()}
	podList, err := c.k8sClient.Pods(c.namespace).List(listOptions)
	if err != nil {
		return nil, fmt.Errorf("cannot obtain peer pods (selector: %s): %v", selector, err)
	}
	for _, pod := range podList.Items {
		podIPs[pod.Status.PodIP] = struct{}{}
	}
	return podIPs, nil
}

func (c consulSideKick) consolidatePeers() error {
	podIP, ownerSelector, err := c.getPodInfo()
	if err != nil {
		return err
	}
	consulPodIPs, err := c.getPodIPs(ownerSelector)
	if err != nil {
		return fmt.Errorf("cannot obtain pod IPs: %v", err)
	}

	currentConsulPeers, err := c.consulClient.Status().Peers()
	if err != nil {
		return fmt.Errorf("cannot obtain consul peers: %v", err)
	}

	for _, peer := range currentConsulPeers {
		peerIP, _, err := net.SplitHostPort(peer)
		if err != nil {
			log.Printf("Cannot parse peer %q: %v", peer, err)
		}
		if _, peerExists := consulPodIPs[peerIP]; !peerExists {
			log.Printf("Deleting peer %s", peer)
			if err := c.consulClient.Agent().ForceLeave(peer); err != nil {
				log.Printf("Cannot force peer %q to leave: %v", peer, err)
			}
		}
		// The remaining ips are the ones which should be added
		delete(consulPodIPs, peerIP)
	}

	for peerToAdd, _ := range consulPodIPs {
		// Don't ask pod to join itself
		if peerToAdd == podIP {
			continue
		}
		log.Printf("Adding peer %s", peerToAdd)
		if err := c.consulClient.Agent().Join(peerToAdd, false); err != nil {
			log.Printf("Cannot join peer  %q: %v", peerToAdd, err)
		}
	}

	return nil
}

func main() {
	consulApiHost := flag.String("consul-api-host", "localhost:8500", "Consul HTTP API host and port")
	podName := flag.String("pod-name", "", "Pod name where consul is running")
	namespace := flag.String("namespace", "default", "Namespace where consul is running")
	pollPeriod := flag.Duration("poll-period", time.Second*5, "Polling period")
	kubeConfig := flag.String("kubeconfig", "", "Path to the kubeconfig file")
	flag.Parse()

	consulConfig := consul.DefaultConfig()
	consulConfig.Address = *consulApiHost
	consulClient, err := consul.NewClient(consulConfig)
	if err != nil {
		log.Fatal(err)
	}

	var k8sConfig *rest.Config
	if *kubeConfig != "" {
		var err error
		k8sConfig, err = clientcmd.BuildConfigFromFlags("", *kubeConfig)
		if err != nil {
			log.Fatalf("cannot create cluster configuration from %s: %v", *kubeConfig, err)
		}
	} else {
		var err error
		k8sConfig, err = rest.InClusterConfig()
		if err != nil {
			log.Fatalf("cannot create kubernetes in-cluster configuration, are you running in a pod?: %v", err)
		}
	}
	k8sClient, err := kubernetes.NewForConfig(k8sConfig)
	if err != nil {
		log.Fatalf("cannot create kubernetes client: %v", err)
	}

	consulSideKick := consulSideKick{
		namespace:    *namespace,
		podName:      *podName,
		consulClient: consulClient,
		k8sClient:    k8sClient,
	}

	for ; ; time.Sleep(*pollPeriod) {
		if err := consulSideKick.consolidatePeers(); err != nil {
			log.Print(err)
		}
	}

}
