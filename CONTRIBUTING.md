# Contributing to k8s-lite-go

Thank you for your interest in contributing to k8s-lite-go! This project is designed to be educational and beginner-friendly, so we welcome contributions from developers of all skill levels.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [How to Contribute](#how-to-contribute)
- [Development Setup](#development-setup)
- [Coding Guidelines](#coding-guidelines)
- [Testing](#testing)
- [Pull Request Process](#pull-request-process)
- [Reporting Issues](#reporting-issues)

## Code of Conduct

This project adheres to a [Code of Conduct](CODE_OF_CONDUCT.md). By participating, you are expected to uphold this code. Please report unacceptable behavior to the project maintainers.

## Getting Started

1. **Fork the repository** on GitHub
2. **Clone your fork** locally:
   ```sh
   git clone https://github.com/YOUR_USERNAME/k8s-lite-go.git
   cd k8s-lite-go
   ```
3. **Add the upstream remote**:
   ```sh
   git remote add upstream https://github.com/Ayobami-00/k8s-lite-go.git
   ```
4. **Create a branch** for your changes:
   ```sh
   git checkout -b feature/your-feature-name
   ```

## How to Contribute

### Types of Contributions We Welcome

- **Bug fixes**: Found a bug? We'd love a fix!
- **Documentation**: Improvements to README, code comments, or new docs
- **New features**: Enhancements that align with the project's educational goals
- **Tests**: Additional test coverage is always appreciated
- **Refactoring**: Code quality improvements
- **Examples**: New usage examples or tutorials

### Good First Issues

Look for issues labeled `good first issue` - these are specifically chosen to be approachable for newcomers.

## Development Setup

### Prerequisites

- Go 1.18 or higher
- GNU Make
- Git

### Building

```sh
# Build all binaries
make build

# Build specific components
make bin/apiserver
make bin/scheduler
make bin/kubelet
make bin/kubectl-lite
```

### Running Locally

```sh
# Terminal 1: Start API server
make run-apiserver

# Terminal 2: Start scheduler
make run-scheduler

# Terminal 3: Start kubelet
make run-kubelet NODE_NAME=node1 NODE_ADDRESS=localhost:10250

# Terminal 4: Interact with the cluster
make kubectl CMD="create pod --name=test --image=nginx"
make kubectl CMD="get pods"
```

## Coding Guidelines

### Go Style

- Follow the [Effective Go](https://golang.org/doc/effective_go) guidelines
- Use `gofmt` to format your code (or `go fmt ./...`)
- Run `go vet ./...` to catch common issues
- Keep functions focused and reasonably sized
- Write descriptive variable and function names

### Project Structure

```
k8s-lite-go/
├── cmd/           # Binary entry points
│   ├── apiserver/
│   ├── scheduler/
│   ├── kubelet/
│   └── kubectl-lite/
├── pkg/           # Shared packages
│   ├── api/       # API types and client
│   └── store/     # Data storage
└── tests/         # Integration tests
    └── integration/
```

### Commit Messages

- Use clear, descriptive commit messages
- Start with a verb in the imperative mood (e.g., "Add", "Fix", "Update")
- Keep the first line under 72 characters
- Reference issues when applicable (e.g., "Fixes #123")

Example:
```
Add pod status endpoint to API server

- Implement GET /api/v1/namespaces/{ns}/pods/{name}/status
- Add corresponding client method
- Include unit tests

Fixes #42
```

## Testing

### Running Tests

```sh
# Run all tests
make test

# Run unit tests only
make test-unit

# Run integration tests only (requires binaries)
make test-integration
```

### Writing Tests

- **Unit tests**: Place in the same package as the code being tested (e.g., `memory_test.go`)
- **Integration tests**: Place in `tests/integration/`
- Aim for meaningful test coverage, not just high percentages
- Test edge cases and error conditions

### Test Guidelines

- Use table-driven tests where appropriate
- Keep tests independent and idempotent
- Clean up any resources created during tests
- Use `t.Helper()` in test helper functions

## Pull Request Process

1. **Update your branch** with the latest upstream changes:
   ```sh
   git fetch upstream
   git rebase upstream/main
   ```

2. **Run tests** to ensure everything passes:
   ```sh
   make test
   ```

3. **Push your branch** to your fork:
   ```sh
   git push origin feature/your-feature-name
   ```

4. **Open a Pull Request** on GitHub with:
   - A clear title describing the change
   - A description explaining what and why
   - Reference to any related issues

5. **Address review feedback** promptly and respectfully

### PR Checklist

- [ ] Code follows the project's style guidelines
- [ ] Tests pass locally (`make test`)
- [ ] New code includes appropriate tests
- [ ] Documentation is updated if needed
- [ ] Commit messages are clear and descriptive

## Reporting Issues

### Bug Reports

When reporting a bug, please include:

- **Description**: Clear description of the issue
- **Steps to Reproduce**: Minimal steps to reproduce the behavior
- **Expected Behavior**: What you expected to happen
- **Actual Behavior**: What actually happened
- **Environment**: Go version, OS, etc.
- **Logs**: Any relevant error messages or logs

### Feature Requests

For feature requests, please describe:

- **Problem**: What problem does this solve?
- **Proposed Solution**: How would you like it to work?
- **Alternatives**: Any alternative solutions you've considered
- **Educational Value**: How does this help people learn about Kubernetes?

## Questions?

If you have questions about contributing, feel free to:

- Open an issue with the `question` label
- Start a discussion in GitHub Discussions (if enabled)

Thank you for contributing to k8s-lite-go! 🎉
