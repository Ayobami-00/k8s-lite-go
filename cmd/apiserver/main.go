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
		podsGroup.DELETE("/:podname", s.deletePodHandlerGin)
	}

	// Node routes
	// /api/v1/nodes
	nodesGroup := router.Group("/api/v1/nodes")
	{
		nodesGroup.POST("", s.createNodeHandlerGin)
		nodesGroup.GET("", s.listNodesHandlerGin)
		nodesGroup.GET("/:nodename", s.getNodeHandlerGin)
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

func main() {
	gin.SetMode(gin.ReleaseMode) // Or gin.DebugMode for development
	dataStore := store.NewInMemoryStore()
	server := NewAPIServer(dataStore)
	server.Serve("8080") // Serve on port 8080
}
