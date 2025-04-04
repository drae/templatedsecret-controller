#!/bin/bash

set -eo pipefail

# Function to display help message
show_help() {
    echo "Usage: dev.sh [OPTION]"
    echo "Local development helper for templated-secret-controller."
    echo ""
    echo "Options:"
    echo "  build        Build the controller binary"
    echo "  test         Run tests"
    echo "  run          Run controller locally (requires kubectl access to a cluster)"
    echo "  manifests    Generate CRD manifests"
    echo "  docker       Build Docker image locally"
    echo "  docker-push  Build and push Docker image to registry"
    echo "  deploy-dev   Apply development configuration to connected Kubernetes cluster"
    echo "  help         Show this help message"
}

# Build the controller binary
cmd_build() {
    echo "Building templated-secret-controller..."
    make build
}

# Run tests
cmd_test() {
    echo "Running tests..."
    make test
}

# Run controller locally
cmd_run() {
    echo "Running controller locally..."
    go run ./cmd/controller/main.go
}

# Generate manifests
cmd_manifests() {
    echo "Generating CRD manifests..."
    make manifests
}

# Build Docker image
cmd_docker() {
    echo "Building Docker image..."
    make docker-build
}

# Build and push Docker image
cmd_docker_push() {
    echo "Building and pushing Docker image..."
    make docker-push
}

# Deploy development configuration to cluster
cmd_deploy_dev() {
    echo "Deploying development configuration to cluster..."
    kubectl apply -k config/kustomize/overlays/dev
}

# Choose the right command based on the first argument
case "$1" in
build)
    cmd_build
    ;;
test)
    cmd_test
    ;;
run)
    cmd_run
    ;;
manifests)
    cmd_manifests
    ;;
docker)
    cmd_docker
    ;;
docker-push)
    cmd_docker_push
    ;;
deploy-dev)
    cmd_deploy_dev
    ;;
help | *)
    show_help
    ;;
esac
