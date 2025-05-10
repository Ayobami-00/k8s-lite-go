package store

import "github.com/Ayobami-00/k8s-lite-go/pkg/api"

// Store defines the interface for interacting with the backend data store.
// It handles the storage and retrieval of API objects like Pods and Nodes.
type Store interface {
	// Pod operations
	CreatePod(pod *api.Pod) error
	GetPod(namespace, name string) (*api.Pod, error)
	UpdatePod(pod *api.Pod) error
	DeletePod(namespace, name string) error
	ListPods(namespace string) ([]*api.Pod, error)

	// Node operations
	CreateNode(node *api.Node) error
	GetNode(name string) (*api.Node, error)
	UpdateNode(node *api.Node) error
	DeleteNode(name string) error
	ListNodes() ([]*api.Node, error)
}
