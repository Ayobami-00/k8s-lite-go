package main

import (
	"flag"
	"log"
	"time"

	"github.com/Ayobami-00/k8s-lite-go/pkg/api"
)

const DefaultNamespace = "default" // Should match apiserver's default if not specified

var nextNodeIndex = 0 // For simple round-robin scheduling

func schedulePods(client *api.Client) {
	// 1. Get pending pods
	pendingPods, err := client.ListPods(DefaultNamespace, api.PodPending)
	if err != nil {
		log.Printf("Error fetching pending pods: %v", err)
		return
	}

	if len(pendingPods) == 0 {
		log.Println("No pending pods to schedule.")
		return
	}
	log.Printf("Found %d pending pods.", len(pendingPods))

	// 2. Get ready nodes
	readyNodes, err := client.ListNodes(api.NodeReady)
	if err != nil {
		log.Printf("Error fetching ready nodes: %v", err)
		return
	}

	if len(readyNodes) == 0 {
		log.Println("No ready nodes available to schedule pods.")
		return
	}
	log.Printf("Found %d ready nodes.", len(readyNodes))

	// 3. Assign pods to nodes (simple round-robin)
	for _, pod := range pendingPods {
		// Explicitly check if the pod is marked for deletion, even if filtered by ListPods
		// This handles potential race conditions or changes in ListPods behavior.
		if pod.DeletionTimestamp != nil {
			log.Printf("Scheduler: Skipping pod %s/%s as it is marked for deletion.", pod.Namespace, pod.Name)
			continue
		}

		// Select node
		if len(readyNodes) == 0 { // Should not happen if check above is done, but defensive
			log.Printf("No ready nodes left to schedule pod %s/%s", pod.Namespace, pod.Name)
			continue
		}
		selectedNode := readyNodes[nextNodeIndex%len(readyNodes)]
		nextNodeIndex++

		// Update pod object
		podToUpdate := pod // Make a copy to avoid modifying the one in the list directly
		podToUpdate.NodeName = selectedNode.Name
		podToUpdate.Phase = api.PodScheduled
		// podToUpdate.HostIP = selectedNode.Address // Or some IP from the node if available

		log.Printf("Attempting to schedule pod %s/%s to node %s", podToUpdate.Namespace, podToUpdate.Name, selectedNode.Name)

		// 4. Update pod on API server
		if err := client.UpdatePod(&podToUpdate); err != nil {
			log.Printf("Error updating pod %s/%s: %v", podToUpdate.Namespace, podToUpdate.Name, err)
			// Consider if we should retry or skip this pod for now
		} else {
			log.Printf("Successfully scheduled pod %s/%s to node %s", podToUpdate.Namespace, podToUpdate.Name, selectedNode.Name)
		}
	}
}

func main() {
	apiServerURL := flag.String("apiserver", "http://localhost:8080", "URL of the API server")
	scheduleInterval := flag.Duration("interval", 5*time.Second, "Scheduling interval")
	flag.Parse()

	log.Printf("Scheduler starting. Connecting to API server at %s", *apiServerURL)

	client, err := api.NewClient(*apiServerURL)
	if err != nil {
		log.Fatalf("Failed to create API client: %v", err)
	}

	log.Printf("Scheduler connected. Starting scheduling loop with interval %v.", *scheduleInterval)

	// Main scheduling loop
	for {
		schedulePods(client)
		time.Sleep(*scheduleInterval)
	}
}
