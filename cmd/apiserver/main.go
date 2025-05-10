package main

import (
	"fmt"
	"log"

	"github.com/Ayobami-00/k8s-lite-go/pkg/api"
	"github.com/Ayobami-00/k8s-lite-go/pkg/store"
	"github.com/gin-gonic/gin"
)

const DefaultNamespace = "default"

type APIServer struct {
	store store.Store
}

func NewAPIServer(s store.Store) *APIServer {
	return &APIServer{store: s}
}

func (s *APIServer) Serve(port string) {
	router := gin.Default() // Use Gin router

	// Pod routes
	// /api/v1/namespaces/{namespace}/pods
	podsGroup := router.Group("/api/v1/namespaces/:namespace/pods")
	{
		podsGroup.POST("", s.createPodHandlerGin)
		podsGroup.GET("", s.listPodsHandlerGin)
		podsGroup.GET("/:podname", s.getPodHandlerGin)
		podsGroup.PUT("/:podname", s.updatePodHandlerGin) // Added route for updating a pod
		podsGroup.DELETE("/:podname", s.deletePodHandlerGin)
	}

	// Node routes
	// /api/v1/nodes
	nodesGroup := router.Group("/api/v1/nodes")
	{
		nodesGroup.POST("", s.createNodeHandlerGin)
		nodesGroup.GET("", s.listNodesHandlerGin)
		nodesGroup.GET("/:nodename", s.getNodeHandlerGin)
		nodesGroup.PUT("/:nodename", s.updateNodeHandlerGin) // Add PUT route for updating a node
		// DELETE for a node could be added here: nodesGroup.DELETE("/:nodename", s.deleteNodeHandlerGin)
	}

	log.Printf("API Server starting on port %s using Gin", port)
	// if err := http.ListenAndServe(":"+port, mux); err != nil { // Old http way
	if err := router.Run(":" + port); err != nil { // Gin way
		log.Fatalf("Failed to start Gin server: %v", err)
	}
}

// Gin handler for creating a pod
func (s *APIServer) createPodHandlerGin(c *gin.Context) {
	namespace := c.Param("namespace")
	var pod api.Pod
	if err := c.ShouldBindJSON(&pod); err != nil {
		c.JSON(400, gin.H{"error": "Invalid request body: " + err.Error()})
		return
	}

	if pod.Name == "" {
		c.JSON(400, gin.H{"error": "Pod name must be provided"})
		return
	}
	pod.Namespace = namespace // Ensure namespace from URL is used
	if pod.Namespace == "" {
		pod.Namespace = DefaultNamespace
	}
	pod.Phase = api.PodPending // Set initial phase
	pod.NodeName = ""          // Not scheduled yet

	if err := s.store.CreatePod(&pod); err != nil {
		c.JSON(500, gin.H{"error": "Failed to create pod: " + err.Error()})
		return
	}
	log.Printf("Created pod %s/%s", pod.Namespace, pod.Name)
	c.JSON(201, pod)
}

// Gin handler for getting a specific pod
func (s *APIServer) getPodHandlerGin(c *gin.Context) {
	namespace := c.Param("namespace")
	podName := c.Param("podname")
	pod, err := s.store.GetPod(namespace, podName)
	if err != nil {
		c.JSON(404, gin.H{"error": "Pod not found: " + err.Error()})
		return
	}
	c.JSON(200, pod)
}

// Gin handler for listing pods in a namespace
func (s *APIServer) listPodsHandlerGin(c *gin.Context) {
	namespace := c.Param("namespace")
	pods, err := s.store.ListPods(namespace)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to list pods: " + err.Error()})
		return
	}
	c.JSON(200, pods)
}

// Gin handler for deleting a specific pod
func (s *APIServer) deletePodHandlerGin(c *gin.Context) {
	namespace := c.Param("namespace")
	podName := c.Param("podname")
	if err := s.store.DeletePod(namespace, podName); err != nil {
		c.JSON(500, gin.H{"error": "Failed to delete pod: " + err.Error()})
		return
	}
	log.Printf("Deleted pod %s/%s", namespace, podName)
	c.JSON(200, gin.H{"message": fmt.Sprintf("Pod %s/%s deleted", namespace, podName)})
}

// Gin handler for updating a specific pod
func (s *APIServer) updatePodHandlerGin(c *gin.Context) {
	namespace := c.Param("namespace")
	podName := c.Param("podname")

	var pod api.Pod
	if err := c.ShouldBindJSON(&pod); err != nil {
		c.JSON(400, gin.H{"error": "Invalid request body: " + err.Error()})
		return
	}

	if pod.Name != podName {
		c.JSON(400, gin.H{"error": fmt.Sprintf("Pod name in body (%s) does not match name in URL (%s)", pod.Name, podName)})
		return
	}
	if pod.Namespace != namespace {
		c.JSON(400, gin.H{"error": fmt.Sprintf("Pod namespace in body (%s) does not match namespace in URL (%s)", pod.Namespace, namespace)})
		return
	}

	// Ensure the pod exists before updating (optional, store might handle this)
	_, err := s.store.GetPod(namespace, podName)
	if err != nil {
		c.JSON(404, gin.H{"error": fmt.Sprintf("Pod %s/%s not found for update: %s", namespace, podName, err.Error())})
		return
	}

	if err := s.store.UpdatePod(&pod); err != nil {
		log.Printf("Failed to update pod in store: %v", err)
		c.JSON(500, gin.H{"error": "Failed to update pod: " + err.Error()})
		return
	}

	c.JSON(200, pod)
}

// Gin handler for creating a node
func (s *APIServer) createNodeHandlerGin(c *gin.Context) {
	var node api.Node
	if err := c.ShouldBindJSON(&node); err != nil {
		c.JSON(400, gin.H{"error": "Invalid request body: " + err.Error()})
		return
	}

	if node.Name == "" {
		c.JSON(400, gin.H{"error": "Node name must be provided"})
		return
	}
	if node.Status == "" {
		node.Status = api.NodeReady // Default to Ready
	}

	if err := s.store.CreateNode(&node); err != nil {
		c.JSON(500, gin.H{"error": "Failed to create node: " + err.Error()})
		return
	}
	log.Printf("Registered node %s", node.Name)
	c.JSON(201, node)
}

// Gin handler for getting a specific node
func (s *APIServer) getNodeHandlerGin(c *gin.Context) {
	nodeName := c.Param("nodename")
	node, err := s.store.GetNode(nodeName)
	if err != nil {
		c.JSON(404, gin.H{"error": "Node not found: " + err.Error()})
		return
	}
	c.JSON(200, node)
}

// Gin handler for listing all nodes
func (s *APIServer) listNodesHandlerGin(c *gin.Context) {
	nodes, err := s.store.ListNodes()
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to list nodes: " + err.Error()})
		return
	}
	c.JSON(200, nodes)
}

// Gin handler for updating a specific node
func (s *APIServer) updateNodeHandlerGin(c *gin.Context) {
	nodeName := c.Param("nodename")
	var updatedNode api.Node

	if err := c.ShouldBindJSON(&updatedNode); err != nil {
		c.JSON(400, gin.H{"error": "Invalid request body: " + err.Error()})
		return
	}

	// Ensure the name from the path is used and matches the body if provided.
	if updatedNode.Name != "" && updatedNode.Name != nodeName {
		c.JSON(400, gin.H{"error": fmt.Sprintf("Node name in body (%s) does not match path (%s)", updatedNode.Name, nodeName)})
		return
	}
	updatedNode.Name = nodeName // Use name from path

	// Check if node exists before updating - GetNode also serves this purpose
	_, err := s.store.GetNode(nodeName)
	if err != nil {
		c.JSON(404, gin.H{"error": "Node not found for update: " + err.Error()}) // StatusNotFound
		return
	}

	if err := s.store.UpdateNode(&updatedNode); err != nil {
		c.JSON(500, gin.H{"error": "Failed to update node: " + err.Error()})
		return
	}
	log.Printf("Updated node %s", updatedNode.Name)
	c.JSON(200, updatedNode)
}

func main() {
	gin.SetMode(gin.ReleaseMode) // Or gin.DebugMode for development
	dataStore := store.NewInMemoryStore()
	server := NewAPIServer(dataStore)
	server.Serve("8080") // Serve on port 8080
}
