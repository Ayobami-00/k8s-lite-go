package api

import "time"

// NodeStatus represents the status of a node.
// +enum
type NodeStatus string

const (
	NodeReady    NodeStatus = "Ready"
	NodeNotReady NodeStatus = "NotReady"
)

// Node represents a worker machine in the cluster.
type Node struct {
	Name    string     `json:"name"`
	Address string     `json:"address"` // e.g., "localhost:8081"
	Status  NodeStatus `json:"status"`
}

// PodPhase represents the phase of a pod.
// +enum
type PodPhase string

const (
	PodPending     PodPhase = "Pending"   // The pod has been accepted by the system, but one or more of the container images has not been created. This includes time before being scheduled as well as time spent downloading images over the network.
	PodScheduled   PodPhase = "Scheduled" // The pod has been scheduled to a node, but is not yet running.
	PodRunning     PodPhase = "Running"   // The pod has been bound to a node, and all of the containers have been created. At least one container is still running, or is in the process of starting or restarting.
	PodDeleted     PodPhase = "Deleted"   // The pod's resources have been reclaimed by the Kubelet. This is a final state.
	PodSucceeded   PodPhase = "Succeeded" // All containers in the pod have terminated in success, and will not be restarted.
	PodFailed      PodPhase = "Failed"    // All containers in the pod have terminated, and at least one container has terminated in failure. The container either exited with non-zero status or was terminated by the system.
	PodDeleting    PodPhase = "Deleting"  // The pod is marked for deletion.
	PodTerminating PodPhase = "Terminating"
)

// Pod represents the smallest deployable units of computing that you can create and manage.
type Pod struct {
	Name              string     `json:"name"`
	Namespace         string     `json:"namespace"`
	Image             string     `json:"image"`                       // Image name (e.g., "nginx:latest")
	NodeName          string     `json:"nodeName,omitempty"`          // Name of the node the pod is assigned to, omitempty because it's not set initially
	Phase             PodPhase   `json:"phase"`                       // Current phase of the pod
	HostIP            string     `json:"hostIP,omitempty"`            // IP address of the host to which the pod is assigned
	PodIP             string     `json:"podIP,omitempty"`             // IP address of the pod
	DeletionTimestamp *time.Time `json:"deletionTimestamp,omitempty"` // Added for soft delete
}
