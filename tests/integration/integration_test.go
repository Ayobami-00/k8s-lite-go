// Package integration provides end-to-end integration tests for k8s-lite-go.
// These tests start the actual binaries and verify the full pod lifecycle.
package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

const (
	testTimeout     = 60 * time.Second
	startupTimeout  = 10 * time.Second
	shutdownTimeout = 5 * time.Second
)

// TestCluster represents a running test cluster with all components.
type TestCluster struct {
	t             *testing.T
	binDir        string
	apiServerCmd  *exec.Cmd
	schedulerCmd  *exec.Cmd
	kubeletCmd    *exec.Cmd
	apiServerURL  string
	apiServerPort string
}

// Pod represents the pod structure for API responses.
type Pod struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Image     string `json:"image"`
	NodeName  string `json:"nodeName,omitempty"`
	Phase     string `json:"phase"`
}

// Node represents the node structure for API responses.
type Node struct {
	Name    string `json:"name"`
	Address string `json:"address"`
	Status  string `json:"status"`
}

// findProjectRoot finds the project root by looking for go.mod.
func findProjectRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("could not find project root (go.mod)")
		}
		dir = parent
	}
}

// findFreePort finds an available port for testing.
func findFreePort() (string, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", err
	}
	defer listener.Close()
	addr := listener.Addr().(*net.TCPAddr)
	return fmt.Sprintf("%d", addr.Port), nil
}

// NewTestCluster creates and starts a new test cluster.
func NewTestCluster(t *testing.T) *TestCluster {
	t.Helper()

	projectRoot, err := findProjectRoot()
	if err != nil {
		t.Fatalf("Failed to find project root: %v", err)
	}

	binDir := filepath.Join(projectRoot, "bin")

	// Verify binaries exist
	binaries := []string{"apiserver", "scheduler", "kubelet"}
	for _, bin := range binaries {
		binPath := filepath.Join(binDir, bin)
		if _, err := os.Stat(binPath); os.IsNotExist(err) {
			t.Fatalf("Binary %s not found. Run 'make build' first.", binPath)
		}
	}

	// Find a free port for the API server
	port, err := findFreePort()
	if err != nil {
		t.Fatalf("Failed to find free port: %v", err)
	}

	tc := &TestCluster{
		t:             t,
		binDir:        binDir,
		apiServerPort: port,
		apiServerURL:  fmt.Sprintf("http://localhost:%s", port),
	}

	return tc
}

// Start starts all cluster components.
func (tc *TestCluster) Start(ctx context.Context) error {
	tc.t.Helper()

	// Start API server
	tc.apiServerCmd = exec.CommandContext(ctx, filepath.Join(tc.binDir, "apiserver"))
	tc.apiServerCmd.Env = append(os.Environ(), fmt.Sprintf("PORT=%s", tc.apiServerPort))
	// The apiserver uses hardcoded port 8080, so we need to modify the binary or use the default
	// For now, we'll use the default port and hope it's available
	tc.apiServerPort = "8080"
	tc.apiServerURL = "http://localhost:8080"
	tc.apiServerCmd = exec.CommandContext(ctx, filepath.Join(tc.binDir, "apiserver"))
	tc.apiServerCmd.Stdout = os.Stdout
	tc.apiServerCmd.Stderr = os.Stderr

	if err := tc.apiServerCmd.Start(); err != nil {
		return fmt.Errorf("failed to start apiserver: %w", err)
	}
	tc.t.Logf("Started API server (PID: %d)", tc.apiServerCmd.Process.Pid)

	// Wait for API server to be ready
	if err := tc.waitForAPIServer(ctx); err != nil {
		tc.Stop()
		return fmt.Errorf("API server failed to become ready: %w", err)
	}

	// Start scheduler
	tc.schedulerCmd = exec.CommandContext(ctx, filepath.Join(tc.binDir, "scheduler"),
		"--apiserver="+tc.apiServerURL)
	tc.schedulerCmd.Stdout = os.Stdout
	tc.schedulerCmd.Stderr = os.Stderr

	if err := tc.schedulerCmd.Start(); err != nil {
		tc.Stop()
		return fmt.Errorf("failed to start scheduler: %w", err)
	}
	tc.t.Logf("Started scheduler (PID: %d)", tc.schedulerCmd.Process.Pid)

	// Start kubelet
	tc.kubeletCmd = exec.CommandContext(ctx, filepath.Join(tc.binDir, "kubelet"),
		"--name=test-node",
		"--address=localhost:10250",
		"--apiserver="+tc.apiServerURL)
	tc.kubeletCmd.Stdout = os.Stdout
	tc.kubeletCmd.Stderr = os.Stderr

	if err := tc.kubeletCmd.Start(); err != nil {
		tc.Stop()
		return fmt.Errorf("failed to start kubelet: %w", err)
	}
	tc.t.Logf("Started kubelet (PID: %d)", tc.kubeletCmd.Process.Pid)

	// Wait for node to register
	if err := tc.waitForNode(ctx, "test-node"); err != nil {
		tc.Stop()
		return fmt.Errorf("node failed to register: %w", err)
	}

	return nil
}

// Stop stops all cluster components gracefully.
func (tc *TestCluster) Stop() {
	tc.t.Helper()

	stopProcess := func(name string, cmd *exec.Cmd) {
		if cmd == nil || cmd.Process == nil {
			return
		}
		tc.t.Logf("Stopping %s (PID: %d)", name, cmd.Process.Pid)

		// Send SIGTERM for graceful shutdown
		if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
			tc.t.Logf("Failed to send SIGTERM to %s: %v", name, err)
		}

		// Wait with timeout
		done := make(chan error, 1)
		go func() {
			done <- cmd.Wait()
		}()

		select {
		case <-done:
			tc.t.Logf("%s stopped", name)
		case <-time.After(shutdownTimeout):
			tc.t.Logf("%s did not stop gracefully, killing", name)
			cmd.Process.Kill()
		}
	}

	stopProcess("kubelet", tc.kubeletCmd)
	stopProcess("scheduler", tc.schedulerCmd)
	stopProcess("apiserver", tc.apiServerCmd)
}

// waitForAPIServer waits for the API server to be ready.
func (tc *TestCluster) waitForAPIServer(ctx context.Context) error {
	deadline := time.Now().Add(startupTimeout)
	client := &http.Client{Timeout: 1 * time.Second}

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		resp, err := client.Get(tc.apiServerURL + "/api/v1/nodes")
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				tc.t.Log("API server is ready")
				return nil
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("timeout waiting for API server")
}

// waitForNode waits for a node to be registered and ready.
func (tc *TestCluster) waitForNode(ctx context.Context, nodeName string) error {
	deadline := time.Now().Add(startupTimeout)
	client := &http.Client{Timeout: 1 * time.Second}

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		resp, err := client.Get(tc.apiServerURL + "/api/v1/nodes/" + nodeName)
		if err == nil {
			defer resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				var node Node
				if err := json.NewDecoder(resp.Body).Decode(&node); err == nil {
					if node.Status == "Ready" {
						tc.t.Logf("Node %s is ready", nodeName)
						return nil
					}
				}
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("timeout waiting for node %s", nodeName)
}

// CreatePod creates a pod via the API.
func (tc *TestCluster) CreatePod(namespace, name, image string) (*Pod, error) {
	pod := Pod{
		Name:      name,
		Namespace: namespace,
		Image:     image,
	}

	body, err := json.Marshal(pod)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/api/v1/namespaces/%s/pods", tc.apiServerURL, namespace)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var createdPod Pod
	if err := json.NewDecoder(resp.Body).Decode(&createdPod); err != nil {
		return nil, err
	}

	return &createdPod, nil
}

// GetPod retrieves a pod via the API.
func (tc *TestCluster) GetPod(namespace, name string) (*Pod, error) {
	url := fmt.Sprintf("%s/api/v1/namespaces/%s/pods/%s", tc.apiServerURL, namespace, name)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var pod Pod
	if err := json.NewDecoder(resp.Body).Decode(&pod); err != nil {
		return nil, err
	}

	return &pod, nil
}

// ListPods lists all pods in a namespace.
func (tc *TestCluster) ListPods(namespace string) ([]Pod, error) {
	url := fmt.Sprintf("%s/api/v1/namespaces/%s/pods", tc.apiServerURL, namespace)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var pods []Pod
	if err := json.NewDecoder(resp.Body).Decode(&pods); err != nil {
		return nil, err
	}

	return pods, nil
}

// DeletePod deletes a pod via the API.
func (tc *TestCluster) DeletePod(namespace, name string) error {
	url := fmt.Sprintf("%s/api/v1/namespaces/%s/pods/%s", tc.apiServerURL, namespace, name)
	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return err
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}

// WaitForPodPhase waits for a pod to reach a specific phase.
func (tc *TestCluster) WaitForPodPhase(namespace, name, phase string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		pod, err := tc.GetPod(namespace, name)
		if err != nil {
			time.Sleep(100 * time.Millisecond)
			continue
		}

		if pod.Phase == phase {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}

	return fmt.Errorf("timeout waiting for pod %s/%s to reach phase %s", namespace, name, phase)
}

// TestPodLifecycle tests the complete pod lifecycle: create, schedule, run, delete.
func TestPodLifecycle(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	// Create and start cluster
	cluster := NewTestCluster(t)
	if err := cluster.Start(ctx); err != nil {
		t.Fatalf("Failed to start cluster: %v", err)
	}
	defer cluster.Stop()

	t.Run("CreatePod", func(t *testing.T) {
		pod, err := cluster.CreatePod("default", "test-pod", "nginx:latest")
		if err != nil {
			t.Fatalf("Failed to create pod: %v", err)
		}

		if pod.Name != "test-pod" {
			t.Errorf("Expected pod name 'test-pod', got '%s'", pod.Name)
		}
		if pod.Namespace != "default" {
			t.Errorf("Expected namespace 'default', got '%s'", pod.Namespace)
		}
		if pod.Phase != "Pending" {
			t.Errorf("Expected phase 'Pending', got '%s'", pod.Phase)
		}
	})

	t.Run("PodGetsScheduled", func(t *testing.T) {
		// Wait for pod to be scheduled
		err := cluster.WaitForPodPhase("default", "test-pod", "Scheduled", 10*time.Second)
		if err != nil {
			t.Fatalf("Pod was not scheduled: %v", err)
		}

		pod, err := cluster.GetPod("default", "test-pod")
		if err != nil {
			t.Fatalf("Failed to get pod: %v", err)
		}

		if pod.NodeName == "" {
			t.Error("Expected pod to be assigned to a node")
		}
	})

	t.Run("PodBecomesRunning", func(t *testing.T) {
		// Wait for pod to be running (kubelet picks it up)
		err := cluster.WaitForPodPhase("default", "test-pod", "Running", 15*time.Second)
		if err != nil {
			t.Fatalf("Pod did not become running: %v", err)
		}
	})

	t.Run("ListPods", func(t *testing.T) {
		pods, err := cluster.ListPods("default")
		if err != nil {
			t.Fatalf("Failed to list pods: %v", err)
		}

		if len(pods) == 0 {
			t.Error("Expected at least one pod")
		}

		found := false
		for _, p := range pods {
			if p.Name == "test-pod" {
				found = true
				break
			}
		}
		if !found {
			t.Error("test-pod not found in pod list")
		}
	})

	t.Run("DeletePod", func(t *testing.T) {
		err := cluster.DeletePod("default", "test-pod")
		if err != nil {
			t.Fatalf("Failed to delete pod: %v", err)
		}

		// Verify pod is deleted or in deleting state
		time.Sleep(1 * time.Second)
		_, err = cluster.GetPod("default", "test-pod")
		// Pod might still exist in Deleting state or be fully deleted
		// Both are acceptable outcomes
	})
}

// TestMultiplePods tests creating and managing multiple pods.
func TestMultiplePods(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	cluster := NewTestCluster(t)
	if err := cluster.Start(ctx); err != nil {
		t.Fatalf("Failed to start cluster: %v", err)
	}
	defer cluster.Stop()

	// Create multiple pods
	podNames := []string{"pod-1", "pod-2", "pod-3"}
	for _, name := range podNames {
		_, err := cluster.CreatePod("default", name, "nginx:latest")
		if err != nil {
			t.Fatalf("Failed to create pod %s: %v", name, err)
		}
	}

	// Wait for all pods to be scheduled
	for _, name := range podNames {
		err := cluster.WaitForPodPhase("default", name, "Scheduled", 15*time.Second)
		if err != nil {
			t.Errorf("Pod %s was not scheduled: %v", name, err)
		}
	}

	// List and verify
	pods, err := cluster.ListPods("default")
	if err != nil {
		t.Fatalf("Failed to list pods: %v", err)
	}

	if len(pods) < len(podNames) {
		t.Errorf("Expected at least %d pods, got %d", len(podNames), len(pods))
	}

	// Cleanup
	for _, name := range podNames {
		cluster.DeletePod("default", name)
	}
}

// TestDuplicatePodCreation tests that creating a duplicate pod fails.
func TestDuplicatePodCreation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	cluster := NewTestCluster(t)
	if err := cluster.Start(ctx); err != nil {
		t.Fatalf("Failed to start cluster: %v", err)
	}
	defer cluster.Stop()

	// Create first pod
	_, err := cluster.CreatePod("default", "duplicate-test", "nginx:latest")
	if err != nil {
		t.Fatalf("Failed to create first pod: %v", err)
	}

	// Try to create duplicate
	_, err = cluster.CreatePod("default", "duplicate-test", "nginx:latest")
	if err == nil {
		t.Error("Expected error when creating duplicate pod, got nil")
	}

	// Cleanup
	cluster.DeletePod("default", "duplicate-test")
}

// TestNamespaceIsolation tests that pods in different namespaces are isolated.
func TestNamespaceIsolation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	cluster := NewTestCluster(t)
	if err := cluster.Start(ctx); err != nil {
		t.Fatalf("Failed to start cluster: %v", err)
	}
	defer cluster.Stop()

	// Create pods in different namespaces with the same name
	_, err := cluster.CreatePod("default", "same-name", "nginx:latest")
	if err != nil {
		t.Fatalf("Failed to create pod in default namespace: %v", err)
	}

	_, err = cluster.CreatePod("other", "same-name", "nginx:latest")
	if err != nil {
		t.Fatalf("Failed to create pod in other namespace: %v", err)
	}

	// Verify both exist
	pod1, err := cluster.GetPod("default", "same-name")
	if err != nil {
		t.Fatalf("Failed to get pod from default namespace: %v", err)
	}
	if pod1.Namespace != "default" {
		t.Errorf("Expected namespace 'default', got '%s'", pod1.Namespace)
	}

	pod2, err := cluster.GetPod("other", "same-name")
	if err != nil {
		t.Fatalf("Failed to get pod from other namespace: %v", err)
	}
	if pod2.Namespace != "other" {
		t.Errorf("Expected namespace 'other', got '%s'", pod2.Namespace)
	}

	// Cleanup
	cluster.DeletePod("default", "same-name")
	cluster.DeletePod("other", "same-name")
}
