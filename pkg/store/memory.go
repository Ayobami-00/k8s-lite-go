package store

import (
	"fmt"
	"sync"

	"github.com/Ayobami-00/k8s-lite-go/pkg/api"
)

// InMemoryStore is an in-memory implementation of the Store interface.
// It is primarily for testing and simplicity, not for production use.
type InMemoryStore struct {
	mu    sync.RWMutex
	pods  map[string]*api.Pod  // Key: "namespace/name"
	nodes map[string]*api.Node // Key: "name"
}

// NewInMemoryStore creates a new InMemoryStore.
func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{
		pods:  make(map[string]*api.Pod),
		nodes: make(map[string]*api.Node),
	}
}

func podKey(namespace, name string) string {
	return fmt.Sprintf("%s/%s", namespace, name)
}

// CreatePod adds a new pod to the store.
func (s *InMemoryStore) CreatePod(pod *api.Pod) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := podKey(pod.Namespace, pod.Name)
	if _, exists := s.pods[key]; exists {
		return fmt.Errorf("pod %s in namespace %s already exists", pod.Name, pod.Namespace)
	}
	s.pods[key] = pod
	return nil
}

// GetPod retrieves a pod from the store.
func (s *InMemoryStore) GetPod(namespace, name string) (*api.Pod, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key := podKey(namespace, name)
	pod, exists := s.pods[key]
	if !exists {
		return nil, fmt.Errorf("pod %s in namespace %s not found", name, namespace)
	}
	return pod, nil
}

// UpdatePod updates an existing pod in the store.
func (s *InMemoryStore) UpdatePod(pod *api.Pod) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := podKey(pod.Namespace, pod.Name)
	if _, exists := s.pods[key]; !exists {
		return fmt.Errorf("pod %s in namespace %s not found for update", pod.Name, pod.Namespace)
	}
	s.pods[key] = pod // Replace the existing pod
	return nil
}

// DeletePod removes a pod from the store.
func (s *InMemoryStore) DeletePod(namespace, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := podKey(namespace, name)
	if _, exists := s.pods[key]; !exists {
		return fmt.Errorf("pod %s in namespace %s not found for deletion", name, namespace)
	}
	delete(s.pods, key)
	return nil
}

// ListPods retrieves all pods in a given namespace.
// If namespace is empty, it could be interpreted as list all pods across all namespaces (not implemented here for simplicity yet).
func (s *InMemoryStore) ListPods(namespace string) ([]*api.Pod, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*api.Pod
	for _, pod := range s.pods {
		if pod.Namespace == namespace {
			result = append(result, pod)
		}
	}
	return result, nil
}

// CreateNode adds a new node to the store.
func (s *InMemoryStore) CreateNode(node *api.Node) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.nodes[node.Name]; exists {
		return fmt.Errorf("node %s already exists", node.Name)
	}
	s.nodes[node.Name] = node
	return nil
}

// GetNode retrieves a node from the store.
func (s *InMemoryStore) GetNode(name string) (*api.Node, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	node, exists := s.nodes[name]
	if !exists {
		return nil, fmt.Errorf("node %s not found", name)
	}
	return node, nil
}

// UpdateNode updates an existing node in the store.
func (s *InMemoryStore) UpdateNode(node *api.Node) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.nodes[node.Name]; !exists {
		return fmt.Errorf("node %s not found for update", node.Name)
	}
	s.nodes[node.Name] = node
	return nil
}

// DeleteNode removes a node from the store.
func (s *InMemoryStore) DeleteNode(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.nodes[name]; !exists {
		return fmt.Errorf("node %s not found for deletion", name)
	}
	delete(s.nodes, name)
	return nil
}

// ListNodes retrieves all nodes.
func (s *InMemoryStore) ListNodes() ([]*api.Node, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*api.Node
	for _, node := range s.nodes {
		result = append(result, node)
	}
	return result, nil
}
