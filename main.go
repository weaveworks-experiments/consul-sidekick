package main

import (
	"flag"
	"fmt"
	"log"
	"time"

	consul "github.com/hashicorp/consul/api"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"net"
)

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
	consulAgent := consulClient.Agent()

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

	pod, err := k8sClient.CoreV1().Pods(*namespace).Get(*podName, v1.GetOptions{})
	if err != nil {
		log.Fatalf("cannot find consul pod (%s/%s): %v", *namespace, *podName, err)
	}
	if len(pod.OwnerReferences) != 1 {
		log.Fatalf("cannot determine owner of consul pod (%s/%s)", *namespace, *podName)
	}
	ownerReference := pod.OwnerReferences[0]
	if !*ownerReference.Controller || ownerReference.Kind != "ReplicaSet" {
		log.Fatalf("consul pod (%s/%s) is not owned by a ReplicaSet (other controllers are not supported)", *namespace, *podName)
	}
	replicaSetName := ownerReference.Name
	replicaSet, err := k8sClient.ReplicaSets(*namespace).Get(replicaSetName, v1.GetOptions{})
	if err != nil {
		log.Fatalf("Cannot access ReplicaSet (%s/%s)", *namespace, replicaSetName)
	}
	podSelector := replicaSet.Spec.Selector

	for ; ; time.Sleep(*pollPeriod) {
		consulPodIPs := map[string]struct{}{}
		selector, err := v1.LabelSelectorAsSelector(podSelector)
		if err != nil {
			log.Printf("cannot prettyprint selector (selector: %v): %v", podSelector, err)
			continue
		}
		listOptions := v1.ListOptions{LabelSelector: selector.String()}
		podList, err := k8sClient.Pods(*namespace).List(listOptions)
		if err != nil {
			fmt.Printf("cannot obtain peer pods (selector: %s): %v", podSelector, err)
			continue
		}
		for _, pod := range podList.Items {
			// TODO: refactor
			consulPodIPs[pod.Status.PodIP] = struct{}{}
		}

		currentConsulPeers, err := consulClient.Status().Peers()
		if err != nil {
			log.Printf("cannot obtain consul currentConsulPeers: %v", err)
			continue
		}

		// TODO: refactor
		pod, err := k8sClient.CoreV1().Pods(*namespace).Get(*podName, v1.GetOptions{})
		if err != nil {
			log.Printf("cannot find consul pod (%s/%s): %v", *namespace, *podName, err)
			continue
		}
		selfPodIP := pod.Status.PodIP

		for _, peer := range currentConsulPeers {
			peerIP, _, err := net.SplitHostPort(peer)
			if err != nil {
				log.Printf("Cannot parse peer %q: %v", peer, err)
			}
			if _, peerExists := consulPodIPs[peerIP]; !peerExists {
				log.Printf("Deleting peer %s", peer)
				if err := consulAgent.ForceLeave(peer); err != nil {
					log.Printf("Cannot force peer %q to leave: %v", peer, err)
				}
			}
			// The remaining ips are the ones which should be added
			delete(consulPodIPs, peerIP)
		}

		for peerToAdd, _ := range consulPodIPs {
			// Don't try to join ourself
			if peerToAdd == selfPodIP {
				continue
			}
			log.Printf("Adding peer %s", peerToAdd)
			if err := consulAgent.Join(peerToAdd, false); err != nil {
				log.Printf("Cannot join peer  %q: %v", peerToAdd, err)
			}
		}

	}

}
