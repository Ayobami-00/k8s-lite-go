BIN_DIR := ./bin
APISERVER_BIN := $(BIN_DIR)/apiserver
SCHEDULER_BIN := $(BIN_DIR)/scheduler
KUBELET_BIN := $(BIN_DIR)/kubelet
KUBECTL_LITE_BIN := $(BIN_DIR)/kubectl-lite

GO_FILES_APISERVER := $(wildcard cmd/apiserver/*.go)
GO_FILES_SCHEDULER := $(wildcard cmd/scheduler/*.go)
GO_FILES_KUBELET := $(wildcard cmd/kubelet/*.go)
GO_FILES_KUBECTL_LITE := $(wildcard cmd/kubectl-lite/*.go)

.PHONY: all build clean run-apiserver run-scheduler run-kubelet kubectl test test-unit test-integration

all: build

build: $(APISERVER_BIN) $(SCHEDULER_BIN) $(KUBELET_BIN) $(KUBECTL_LITE_BIN)

$(BIN_DIR):
	@mkdir -p $(BIN_DIR)

$(APISERVER_BIN): $(GO_FILES_APISERVER) | $(BIN_DIR)
	@echo "Building apiserver..."
	@go build -o $(APISERVER_BIN) ./cmd/apiserver

$(SCHEDULER_BIN): $(GO_FILES_SCHEDULER) | $(BIN_DIR)
	@echo "Building scheduler..."
	@go build -o $(SCHEDULER_BIN) ./cmd/scheduler

$(KUBELET_BIN): $(GO_FILES_KUBELET) | $(BIN_DIR)
	@echo "Building kubelet..."
	@go build -o $(KUBELET_BIN) ./cmd/kubelet

$(KUBECTL_LITE_BIN): $(GO_FILES_KUBECTL_LITE) | $(BIN_DIR)
	@echo "Building kubectl-lite..."
	@go build -o $(KUBECTL_LITE_BIN) ./cmd/kubectl-lite

run-apiserver: $(APISERVER_BIN)
	@echo "Starting API server..."
	@$(APISERVER_BIN)

run-scheduler: $(SCHEDULER_BIN)
	@echo "Starting scheduler..."
	@$(SCHEDULER_BIN)

# Example: make run-kubelet NODE_NAME=node1 NODE_ADDRESS=localhost:10250
run-kubelet: $(KUBELET_BIN)
	@echo "Starting Kubelet (NODE_NAME=$(NODE_NAME), NODE_ADDRESS=$(NODE_ADDRESS))..."
	@$(KUBELET_BIN) --name=$(NODE_NAME) --address=$(NODE_ADDRESS) --apiserver=http://localhost:8080

# Example: make kubectl CMD="get pods"
# Example: make kubectl CMD="create pod --name mypod --image nginx"
kubectl: build
	@echo "Running kubectl-lite $(CMD)..."
	@$(KUBECTL_LITE_BIN) --apiserver=http://localhost:8080 $(CMD)

clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BIN_DIR)

# Test targets
test: test-unit test-integration

test-unit:
	@echo "Running unit tests..."
	@go test -v -short ./pkg/...

test-integration: build
	@echo "Running integration tests..."
	@go test -v -timeout 120s ./tests/integration/...

# Help target to display available commands
help:
	@echo "Available targets:"
	@echo "  all                      - Build all binaries (default)"
	@echo "  build                    - Build all binaries"
	@echo "  $(APISERVER_BIN)       - Build the apiserver"
	@echo "  $(SCHEDULER_BIN)     - Build the scheduler"
	@echo "  $(KUBELET_BIN)       - Build the kubelet"
	@echo "  $(KUBECTL_LITE_BIN) - Build kubectl-lite"
	@echo "  run-apiserver            - Run the API server"
	@echo "  run-scheduler            - Run the scheduler"
	@echo "  run-kubelet NODE_NAME=<name> NODE_ADDRESS=<addr> - Run the Kubelet (e.g., make run-kubelet NODE_NAME=node1 NODE_ADDRESS=localhost:10250)"
	@echo "  kubectl CMD='<command_string>' - Run kubectl-lite with the specified command (e.g., make kubectl CMD='get pods')"
	@echo "  clean                    - Remove build artifacts"
	@echo "  test                     - Run all tests (unit + integration)"
	@echo "  test-unit                - Run unit tests only"
	@echo "  test-integration         - Run integration tests (requires build)"
	@echo "  help                     - Show this help message"