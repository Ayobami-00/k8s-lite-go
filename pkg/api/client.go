package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// Client is a client for the k8s-lite-go API server.
type Client struct {
	baseURL    *url.URL
	httpClient *http.Client
}

// NewClient creates a new API client.
func NewClient(baseURLStr string) (*Client, error) {
	baseURL, err := url.Parse(baseURLStr)
	if err != nil {
		return nil, fmt.Errorf("parsing base URL: %w", err)
	}
	return &Client{
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}, nil
}

func (c *Client) buildURL(pathSegments ...string) string {
	finalPath := c.baseURL.Path
	for _, segment := range pathSegments {
		finalPath = fmt.Sprintf("%s/%s", finalPath, segment)
	}
	// Create a copy of baseURL to avoid modifying the original
	u := *c.baseURL
	u.Path = finalPath
	return u.String()
}

// CreateNode sends a POST request to create/register a node.
func (c *Client) CreateNode(node *Node) (*Node, error) {
	urlStr := c.buildURL("api", "v1", "nodes")

	body, err := json.Marshal(node)
	if err != nil {
		return nil, fmt.Errorf("marshalling node: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, urlStr, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		// TODO: Read body for more detailed error message from server
		return nil, fmt.Errorf("server returned non-Created status for create node: %d", resp.StatusCode)
	}

	var createdNode Node
	if err := json.NewDecoder(resp.Body).Decode(&createdNode); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}
	return &createdNode, nil
}

// UpdateNode sends a PUT request to update a node.
func (c *Client) UpdateNode(node *Node) error {
	if node.Name == "" {
		return fmt.Errorf("node name must be specified for update")
	}
	urlStr := c.buildURL("api", "v1", "nodes", node.Name)

	body, err := json.Marshal(node)
	if err != nil {
		return fmt.Errorf("marshalling node: %w", err)
	}

	req, err := http.NewRequest(http.MethodPut, urlStr, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// TODO: Read body for more detailed error message from server
		return fmt.Errorf("server returned non-OK status for update node: %d", resp.StatusCode)
	}
	return nil
}

// ListPods fetches pods, optionally filtering by phase.
// For now, it gets all pods for the namespace and filters client-side if phase is specified.
// A more efficient API would support server-side filtering by phase.
func (c *Client) ListPods(namespace string, phase PodPhase) ([]Pod, error) {
	urlStr := c.buildURL("api", "v1", "namespaces", namespace, "pods")
	req, err := http.NewRequest(http.MethodGet, urlStr, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned non-OK status: %d", resp.StatusCode)
	}

	var allPods []Pod
	if err := json.NewDecoder(resp.Body).Decode(&allPods); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	if phase == "" { // No phase filter, return all
		return allPods, nil
	}

	var filteredPods []Pod
	for _, pod := range allPods {
		if pod.Phase == phase {
			filteredPods = append(filteredPods, pod)
		}
	}
	return filteredPods, nil
}

// ListNodes fetches nodes, optionally filtering by status.
// Similar to ListPods, filters client-side for simplicity.
func (c *Client) ListNodes(status NodeStatus) ([]Node, error) {
	urlStr := c.buildURL("api", "v1", "nodes")
	req, err := http.NewRequest(http.MethodGet, urlStr, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned non-OK status: %d", resp.StatusCode)
	}

	var allNodes []Node
	if err := json.NewDecoder(resp.Body).Decode(&allNodes); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	if status == "" { // No status filter, return all
		return allNodes, nil
	}

	var filteredNodes []Node
	for _, node := range allNodes {
		if node.Status == status {
			filteredNodes = append(filteredNodes, node)
		}
	}
	return filteredNodes, nil
}

// UpdatePod sends a PUT request to update a pod.
func (c *Client) UpdatePod(pod *Pod) error {
	urlStr := c.buildURL("api", "v1", "namespaces", pod.Namespace, "pods", pod.Name)

	body, err := json.Marshal(pod)
	if err != nil {
		return fmt.Errorf("marshalling pod: %w", err)
	}

	req, err := http.NewRequest(http.MethodPut, urlStr, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// TODO: Read body for more detailed error message from server
		return fmt.Errorf("server returned non-OK status for update: %d", resp.StatusCode)
	}
	// Optionally decode the response body if the updated pod is returned
	return nil
}
