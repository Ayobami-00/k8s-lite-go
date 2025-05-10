package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/Ayobami-00/k8s-lite-go/pkg/api"
)

const DefaultNamespace = "default"

// Kubelet represents a node agent.
type Kubelet struct {
	NodeName    string
	NodeAddress string // Mock address for this Kubelet/Node
	APIClient   *api.Client
	// knownPods map[string]api.PodPhase // To track pods it's "running"
}

func NewKubelet(nodeName, nodeAddress, apiServerURL string) (*Kubelet, error) {
	client, err := api.NewClient(apiServerURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create API client: %w", err)
	}
	return &Kubelet{
		NodeName:    nodeName,
		NodeAddress: nodeAddress,
		APIClient:   client,
		// knownPods:  make(map[string]api.PodPhase),
	}, nil
}

// registerNode registers this Kubelet's node with the API server.
func (k *Kubelet) registerNode() error {
	node := &api.Node{
		Name:    k.NodeName,
		Address: k.NodeAddress,
		Status:  api.NodeReady, // Assume ready on startup
	}
	createdNode, err := k.APIClient.CreateNode(node)
	if err != nil {
		// It might already exist if Kubelet restarted, try to update (get and then put if needed)
		// For simplicity, we'll just log an error. A real Kubelet would handle this more gracefully.
		log.Printf("Failed to register node %s, attempting to update: %v", k.NodeName, err)
		// Attempt to update if creation failed (e.g. node already exists)
		if errUpdate := k.APIClient.UpdateNode(node); errUpdate != nil {
			return fmt.Errorf("failed to register or update node %s: %w (update error: %v)", k.NodeName, err, errUpdate)
		}
		log.Printf("Node %s updated successfully after initial registration failure.", k.NodeName)
		return nil
	}
	log.Printf("Node %s registered successfully with address %s and status %s", createdNode.Name, createdNode.Address, createdNode.Status)
	return nil
}

// syncPods is the main loop for the Kubelet to manage pods on its node.
func (k *Kubelet) syncPods() {
	log.Printf("[%s] Syncing pods...", k.NodeName)

	// 1. Get all pods in the default namespace
	pods, err := k.APIClient.ListPods(DefaultNamespace, "") // Get all pods, any phase
	if err != nil {
		log.Printf("[%s] Error fetching pods: %v", k.NodeName, err)
		return
	}

	for _, pod := range pods {
		// Check if the pod is scheduled to this node
		if pod.NodeName == k.NodeName {

			// **NEW SECTION: Handle terminating pods first**
			if pod.DeletionTimestamp != nil {
				// If the pod is marked for deletion, process its termination.
				if pod.Phase != api.PodSucceeded && pod.Phase != api.PodFailed && pod.Phase != api.PodDeleted { // Also check against PodDeleted
					log.Printf("[%s] Detected terminating pod %s. Simulating cleanup and marking as Deleted.", k.NodeName, pod.Name)
					updatedPod := pod                 // Make a copy
					updatedPod.Phase = api.PodDeleted // CHANGE THIS LINE
					// updatedPod.Phase = api.PodSucceeded (OLD LINE)

					if err := k.APIClient.UpdatePod(&updatedPod); err != nil {
						log.Printf("[%s] Error updating pod %s to Deleted after termination: %v", k.NodeName, pod.Name, err)
					} else {
						log.Printf("[%s] Pod %s marked as Deleted after termination processing.", k.NodeName, pod.Name)
					}
				} else {
					// Pod is terminating but already in a final state (Succeeded, Failed, or Deleted).
					log.Printf("[%s] Pod %s is terminating and already in state %s. No Kubelet action needed.", k.NodeName, pod.Name, pod.Phase)
				}
				continue
			}
			// **END OF NEW SECTION**

			// Original switch statement, now effectively for non-terminating pods
			switch pod.Phase {
			case api.PodScheduled:
				log.Printf("[%s] Found scheduled pod %s. 'Starting' it...", k.NodeName, pod.Name)
				updatedPod := pod
				updatedPod.Phase = api.PodRunning
				if err := k.APIClient.UpdatePod(&updatedPod); err != nil {
					log.Printf("[%s] Error updating pod %s to Running: %v", k.NodeName, pod.Name, err)
				} else {
					log.Printf("[%s] Pod %s with image '%s' is now 'Running'.", k.NodeName, pod.Name, pod.Image)
				}
			case api.PodRunning:
				// log.Printf("[%s] Pod %s is already running.", k.NodeName, pod.Name)
				// Potentially check health here
				break

			case api.PodTerminating:
				log.Printf("[%s] Pod %s found in Terminating phase. Processing termination.", k.NodeName, pod.Name)
				if pod.Phase != api.PodSucceeded && pod.Phase != api.PodFailed && pod.Phase != api.PodDeleted { // Also check against PodDeleted
					updatedPod := pod
					updatedPod.Phase = api.PodDeleted // CHANGE THIS
					if err := k.APIClient.UpdatePod(&updatedPod); err != nil {
						log.Printf("[%s] Error updating pod %s from Terminating to Deleted: %v", k.NodeName, pod.Name, err)
					} else {
						log.Printf("[%s] Pod %s (in Terminating phase) marked as Deleted.", k.NodeName, pod.Name)
					}
				}

			case api.PodDeleting: // This was an older phase name you had.
				log.Printf("[%s] Detected pod %s in PodDeleting phase. Handling as terminating.", k.NodeName, pod.Name)
				// Similar logic to PodTerminating or rely on DeletionTimestamp check
				if pod.DeletionTimestamp == nil { // If timestamp wasn't set, but phase is Deleting
					log.Printf("[%s] Warning: Pod %s in PodDeleting phase but DeletionTimestamp is nil. This should be synchronized.", k.NodeName, pod.Name)
				}
				// The DeletionTimestamp check at the top should handle most cases.
				// If we reach here and it's not Succeeded/Failed, update it.
				if pod.Phase != api.PodSucceeded && pod.Phase != api.PodFailed {
					updatedPod := pod
					updatedPod.Phase = api.PodSucceeded
					if err := k.APIClient.UpdatePod(&updatedPod); err != nil {
						log.Printf("[%s] Error updating pod %s from PodDeleting to Succeeded: %v", k.NodeName, pod.Name, err)
					} else {
						log.Printf("[%s] Pod %s (in PodDeleting phase) marked as Succeeded.", k.NodeName, pod.Name)
					}
				}

			default:
				// Do nothing for other phases like Pending (handled by scheduler), Succeeded, Failed (final states)
				if pod.Phase != api.PodPending && pod.Phase != api.PodSucceeded && pod.Phase != api.PodFailed {
					log.Printf("[%s] Pod %s found in unhandled phase: %s", k.NodeName, pod.Name, pod.Phase)
				}
			}
		}
	}
	// TODO: Implement logic to detect and "stop" pods that were running on this node but are no longer in the API server's list
}

func main() {
	nodeName := flag.String("name", "", "Name of this node (kubelet)")
	nodeAddress := flag.String("address", "localhost:10250", "Address of this node (e.g. IP or hostname, port is informational for mock)")
	apiServerURL := flag.String("apiserver", "http://localhost:8080", "URL of the API server")
	syncInterval := flag.Duration("sync-interval", 10*time.Second, "Pod synchronization interval")
	flag.Parse()

	if *nodeName == "" {
		log.Fatalf("Node name must be specified using -name flag")
	}

	log.Printf("Kubelet for node '%s' starting. Node address: %s. API Server: %s", *nodeName, *nodeAddress, *apiServerURL)

	k, err := NewKubelet(*nodeName, *nodeAddress, *apiServerURL)
	if err != nil {
		log.Fatalf("Failed to create Kubelet: %v", err)
	}

	if err := k.registerNode(); err != nil {
		log.Fatalf("Failed to register node with API server: %v. Ensure API server is running.", err)
	}

	log.Printf("Kubelet for node '%s' registered. Starting pod sync loop with interval %v.", *nodeName, *syncInterval)

	for {
		k.syncPods()
		time.Sleep(*syncInterval)
	}
}

// Helper function to get hostname, useful for default node name
func getHostname() string {
	hostname, err := os.Hostname()
	if err != nil {
		return "unknown-host"
	}
	return hostname
}
