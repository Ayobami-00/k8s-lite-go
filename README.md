# k8s-lite-go

A minimal, educational re-implementation of Kubernetes core control plane components in Go. This project is designed to demystify how Kubernetes works by letting you build, run, and experiment with a simplified version—from scratch.

---

## What is "Lite"?

"Lite" means this project implements the **core ideas** of Kubernetes (API server, scheduler, Kubelet, pods, nodes) in a way that's easy to understand and hack on:
- **In-memory state** (no etcd)
- **No real containers** (Kubelet just logs actions)
- **No networking, RBAC, or authentication**
- **Polling, not event streams**
- **Only pods and nodes supported**

Perfect for learning, teaching, or experimenting!

---

## Folder Structure

```
k8s-lite-go/
├── cmd/
│   ├── apiserver/      # The API server binary (main.go)
│   ├── scheduler/      # The scheduler binary (main.go)
│   └── kubelet/        # The Kubelet binary (main.go)
├── pkg/
│   ├── api/            # Shared API types and client (types.go, client.go)
│   └── store/          # In-memory store implementation (memory.go, store.go)
├── Makefile            # Build and CLI automation commands
├── article.md          # In-depth article explaining the project
├── README.md           # This file
```

**Key files:**
- `cmd/apiserver/main.go`: REST API server, CRUD for pods/nodes, business logic
- `cmd/scheduler/main.go`: Scheduler loop, assigns pods to nodes
- `cmd/kubelet/main.go`: Kubelet (node agent), simulates pod execution and cleanup
- `pkg/api/types.go`: Pod, Node, PodPhase definitions
- `pkg/api/client.go`: Go client for API server
- `pkg/store/memory.go`: In-memory state management
- `Makefile`: Build and CLI automation

---

## Getting Started

### Prerequisites
- Go 1.18+
- GNU Make

### Setup
1. **Clone the repo:**

   ```sh
   git clone https://github.com/Ayobami-00/k8s-lite-go
   cd k8s-lite-go
   ```
2. **Build all binaries:**

   ```sh
   make build
   ```
   This will build the API server, scheduler, and kubelet binaries.

---

## Running the Cluster

You can run each component in its own terminal (or background process):

### 1. Start the API Server
```sh
make apiserver
```

### 2. Start the Scheduler
```sh
make run-scheduler
```

### 3. Start a Kubelet (simulates a node, e.g. "node1")
```sh
make kubelet NODE=node1
```
You can run multiple kubelets (with different NODE names) to simulate a multi-node cluster.

---

## Interacting with the Cluster (Testing the Pod Lifecycle)

A simple CLI is provided via the Makefile. Example flows:

### 1. Create a Pod
```sh
make kubectl CMD="create pod --name=mypod1 --image=nginx:latest"
```

### 2. List Pods
```sh
make kubectl CMD="get pods"
```

### 3. Delete a Pod (soft deletion)
```sh
make kubectl CMD="delete pod mypod1"
```
---

## Credits
This project is for educational purposes and is inspired by the core ideas of Kubernetes.
