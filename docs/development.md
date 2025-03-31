# Development Guide

This document describes how to develop and contribute to the templatedsecret-controller project.

## Prerequisites

- Go 1.21+
- kubectl
- Docker
- Access to a Kubernetes cluster (for testing)

## Project Structure

```
templatedsecretsecret-controller/
├── cmd/                  # Application entry points
├── config/               # Kubernetes manifests and configuration
│   ├── kustomize/        # Kustomize-based deployment configs
│       ├── base/         # Base resources
│       └── overlays/     # Environment-specific overlays
├── docs/                 # Documentation
├── examples/             # Usage examples
├── hack/                 # Development scripts
└── pkg/                  # Core controller logic and API types
```

## Development Workflow

### Setting up your Environment

1. Clone the repository:

   ```shell
   git clone https://github.com/drae/templatedsecretsecret-controller.git
   cd templatedsecretsecret-controller
   ```

2. Install dependencies:

   ```shell
   go mod download
   ```

### Common Development Tasks

The project includes a development helper script at `hack/dev.sh` that provides shortcuts for common tasks:

```shell
# Build the controller
./hack/dev.sh build

# Run tests
./hack/dev.sh test

# Generate CRD manifests
./hack/dev.sh manifests

# Run the controller locally (against your current kubeconfig cluster)
./hack/dev.sh run

# Build a Docker image
./hack/dev.sh docker

# Deploy development configuration to your cluster
./hack/dev.sh deploy-dev
```

You can also use the Makefile directly for more granular control:

```shell
# Build the controller
make build

# Run tests
make test

# Generate CRD manifests
make manifests

# Build and push a multi-architecture Docker image
make docker-push
```

## Testing

### Unit Tests

Run unit tests with:

```shell
make test
```

### End-to-End Tests

End-to-end tests are located in the `test/e2e` directory and can be executed with:

```shell
go test ./test/e2e -v
```

## Releasing

Releases are handled through GitHub Actions when a git tag is pushed:

1. Tag a new version:

   ```shell
   git tag -a v0.1.0 -m "Release v0.1.0"
   git push origin v0.1.0
   ```

2. The GitHub Actions workflow will:
   - Build and test the code
   - Build multi-architecture container images (amd64, arm64)
   - Push images to GitHub Container Registry
   - Generate and attach Kubernetes manifests to the GitHub release

## Code Conventions

This project follows standard Go conventions:

- Code should be formatted with `gofmt`
- Follow standard Go naming conventions
- Write tests for new functionality
- Update documentation for user-facing changes
