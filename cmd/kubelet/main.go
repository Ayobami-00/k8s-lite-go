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

	// 1. Get all pods in the default namespace (Kubelet typically watches specific pods or all assigned)
	// For simplicity, we fetch all and filter.
	pods, err := k.APIClient.ListPods(DefaultNamespace, "") // Get all pods, any phase
	if err != nil {
		log.Printf("[%s] Error fetching pods: %v", k.NodeName, err)
		return
	}

	for _, pod := range pods {
		// Check if the pod is scheduled to this node
		if pod.NodeName == k.NodeName {
			switch pod.Phase {
			case api.PodScheduled:
				log.Printf("[%s] Found scheduled pod %s. 'Starting' it...", k.NodeName, pod.Name)
				// Simulate starting the pod
				updatedPod := pod
				updatedPod.Phase = api.PodRunning
				// updatedPod.HostIP = k.NodeAddress // Kubelet could set this
				// updatedPod.PodIP = "10.0.1.x" // In a real scenario, CNI would assign this

				if err := k.APIClient.UpdatePod(&updatedPod); err != nil {
					log.Printf("[%s] Error updating pod %s to Running: %v", k.NodeName, pod.Name, err)
				} else {
					log.Printf("[%s] Pod %s with image '%s' is now 'Running'.", k.NodeName, pod.Name, pod.Image)
					// k.knownPods[pod.Name] = api.PodRunning
				}
			case api.PodRunning:
				// log.Printf("[%s] Pod %s is already running.", k.NodeName, pod.Name)
				// Potentially check health here
				break
			case api.PodDeleting: // A more robust Kubelet would handle this
				log.Printf("[%s] Detected pod %s marked for deletion. 'Stopping' it...", k.NodeName, pod.Name)
				// Simulate cleanup, then actual deletion could be done by a controller or here after confirmation.
				// For now, we just acknowledge. The API server directly deletes it in our simplified model.
				// delete(k.knownPods, pod.Name)
			default:
				// Do nothing for other phases like Pending, Succeeded, Failed for now
			}
		}
	}
	// TODO: Implement logic to detect and "stop" pods that were running on this node but are no longer in the API server's list or are marked for deletion.
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
