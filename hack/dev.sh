#!/bin/bash

set -e

# Function to display help message
show_help() {
    echo "Usage: dev.sh [OPTION]"
    echo "Local development helper for templatedsecret-controller."
    echo ""
    echo "Options:"
    echo "  build        Build the controller binary"
    echo "  test         Run tests"
    echo "  run          Run controller locally (requires kubectl access to a cluster)"
    echo "  manifests    Generate CRD manifests"
    echo "  docker       Build Docker image locally"
    echo "  deploy-dev   Apply development configuration to connected Kubernetes cluster"
    echo "  help         Show this help message"
}

# Build the controller binary
cmd_build() {
    echo "Building templatedsecret-controller..."
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
deploy-dev)
    cmd_deploy_dev
    ;;
help | *)
    show_help
    ;;
esac
