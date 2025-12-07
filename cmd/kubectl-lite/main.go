package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/Ayobami-00/k8s-lite-go/pkg/api"
)

const DefaultNamespace = "default"

func main() {
	apiServerURL := flag.String("apiserver", "http://localhost:8080", "URL of the API server")
	flag.Parse() // Parse global flags first

	if len(flag.Args()) < 1 {
		fmt.Println("Error: No command specified.")
		printUsage()
		os.Exit(1)
	}

	// Initialize client AFTER parsing global flags, so it uses the correct URL
	client, err := api.NewClient(*apiServerURL)
	if err != nil {
		log.Fatalf("Error creating API client: %v", err)
	}

	command := flag.Arg(0)  // Get the command (e.g., "create", "get")
	args := flag.Args()[1:] // Get the arguments for the command

	switch command {
	case "create":
		handleCreateCommand(client, args)
	case "get":
		handleGetCommand(client, args)
	case "delete":
		handleDeleteCommand(client, args)
	case "register": // Special command for nodes, could be merged into 'create node'
		handleRegisterNodeCommand(client, args)
	default:
		fmt.Printf("Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Usage: kubectl-lite --apiserver <url> <command> <subcommand> [flags]")
	fmt.Println("Commands:")
	fmt.Println("  create pod --name <name> --image <image> [--namespace <ns>]")
	fmt.Println("  get pods [--namespace <ns>]")
	fmt.Println("  get pod <name> [--namespace <ns>]")
	fmt.Println("  get nodes")
	fmt.Println("  get node <name>")
	fmt.Println("  delete pod <name> [--namespace <ns>]")
	fmt.Println("  register node --name <name> --address <addr>")
	fmt.Println("Global flags:")
	fmt.Println("  --apiserver <url>  URL of the API server (default: http://localhost:8080)")
}

func handleCreateCommand(client *api.Client, args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: kubectl-lite create <resource_type> [flags]")
		fmt.Println("Example: kubectl-lite create pod --name mypod --image nginx")
		os.Exit(1)
	}

	resourceType := args[0]
	commandArgs := args[1:] // Arguments for the specific resource type's flags

	switch resourceType {
	case "pod":
		createPodCmd := flag.NewFlagSet("create pod", flag.ExitOnError)
		podName := createPodCmd.String("name", "", "Name of the pod")
		podImage := createPodCmd.String("image", "", "Image for the pod")
		podNamespace := createPodCmd.String("namespace", DefaultNamespace, "Namespace for the pod")

		if err := createPodCmd.Parse(commandArgs); err != nil {
			fmt.Printf("Error parsing 'create pod' flags: %v\n", err)
			os.Exit(1)
		}

		if *podName == "" || *podImage == "" {
			fmt.Println("Error: --name and --image are required for creating a pod")
			createPodCmd.Usage()
			os.Exit(1)
		}

		pod := &api.Pod{Name: *podName, Image: *podImage, Namespace: *podNamespace}
		createdPod, err := client.CreatePod(*podNamespace, pod)
		if err != nil {
			log.Fatalf("Error creating pod: %v", err)
		}
		fmt.Printf("Pod %s/%s created\n", createdPod.Namespace, createdPod.Name)
	default:
		fmt.Printf("Error: Unknown resource type for create: %s\n", resourceType)
		fmt.Println("Supported resource types for create: pod")
		os.Exit(1)
	}
}

func handleGetCommand(client *api.Client, args []string) {
	getCmd := flag.NewFlagSet("get", flag.ExitOnError)
	podNamespace := getCmd.String("namespace", DefaultNamespace, "Namespace for pods")

	if len(args) < 1 {
		fmt.Println("Usage: kubectl-lite get <resource_type> [resource_name] [flags]")
		os.Exit(1)
	}
	resourceType := args[0]
	var resourceName string
	if len(args) > 1 && !strings.HasPrefix(args[1], "-") {
		resourceName = args[1]
		_ = getCmd.Parse(args[2:])
	} else {
		_ = getCmd.Parse(args[1:])
	}

	switch resourceType {
	case "pods", "pod":
		if resourceName == "" { // List all pods in namespace
			pods, err := client.ListPods(*podNamespace, "") // No phase filter
			if err != nil {
				log.Fatalf("Error getting pods: %v", err)
			}
			prettyPrint(pods)
		} else { // Get specific pod
			pod, err := client.GetPod(*podNamespace, resourceName)
			if err != nil {
				log.Fatalf("Error getting pod %s/%s: %v", *podNamespace, resourceName, err)
			}
			prettyPrint(pod)
		}
	case "nodes", "node":
		if resourceName == "" { // List all nodes
			nodes, err := client.ListNodes("") // No status filter
			if err != nil {
				log.Fatalf("Error getting nodes: %v", err)
			}
			prettyPrint(nodes)
		} else { // Get specific node
			node, err := client.GetNode(resourceName)
			if err != nil {
				log.Fatalf("Error getting node %s: %v", resourceName, err)
			}
			prettyPrint(node)
		}
	default:
		fmt.Printf("Unknown resource type for get: %s\n", resourceType)
		os.Exit(1)
	}
}

func handleDeleteCommand(client *api.Client, args []string) {
	deleteCmd := flag.NewFlagSet("delete", flag.ExitOnError)
	podNamespace := deleteCmd.String("namespace", DefaultNamespace, "Namespace for the pod")

	if len(args) < 2 {
		fmt.Println("Usage: kubectl-lite delete <resource_type> <resource_name> [flags]")
		os.Exit(1)
	}
	resourceType := args[0]
	resourceName := args[1]
	_ = deleteCmd.Parse(args[2:])

	switch resourceType {
	case "pod":
		if resourceName == "" {
			fmt.Println("Error: pod name is required for delete pod")
			os.Exit(1)
		}
		err := client.DeletePod(*podNamespace, resourceName)
		if err != nil {
			log.Fatalf("Error deleting pod %s/%s: %v", *podNamespace, resourceName, err)
		}
		fmt.Printf("Pod %s/%s deleted\n", *podNamespace, resourceName)
	default:
		fmt.Printf("Unknown resource type for delete: %s\n", resourceType)
		os.Exit(1)
	}
}

func handleRegisterNodeCommand(client *api.Client, args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: kubectl-lite register node --name <nodename> --address <nodeaddress>")
		os.Exit(1)
	}

	resourceType := args[0]
	commandArgs := args[1:]

	if resourceType != "node" {
		fmt.Printf("Error: 'register' command only supports 'node' resource type, got: %s\n", resourceType)
		fmt.Println("Usage: kubectl-lite register node --name <nodename> --address <nodeaddress>")
		os.Exit(1)
	}

	registerNodeCmd := flag.NewFlagSet("register node", flag.ExitOnError)
	nodeName := registerNodeCmd.String("name", "", "Name of the node")
	nodeAddress := registerNodeCmd.String("address", "", "Address of the node (e.g. IP)")

	if err := registerNodeCmd.Parse(commandArgs); err != nil {
		fmt.Printf("Error parsing 'register node' flags: %v\n", err)
		os.Exit(1)
	}

	if *nodeName == "" || *nodeAddress == "" {
		fmt.Println("Error: --name and --address are required for registering a node")
		registerNodeCmd.Usage()
		os.Exit(1)
	}

	node := &api.Node{Name: *nodeName, Address: *nodeAddress, Status: "Ready"} // Assuming Address field exists in api.Node
	createdNode, err := client.CreateNode(node)
	if err != nil {
		log.Fatalf("Error registering node: %v", err)
	}
	fmt.Printf("Node %s registered with address %s\n", createdNode.Name, createdNode.Address)
}

func prettyPrint(data interface{}) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(data); err != nil {
		log.Fatalf("Error pretty printing JSON: %v", err)
	}
}
