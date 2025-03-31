#!/bin/bash

set -e -x -u

# Get the correct Go module paths
MODULE_PATH="github.com/drae/templatedsecret-controller"
API_PACKAGE="${MODULE_PATH}/pkg/apis/templatedsecret/v1alpha1"
HEADER_FILE="./code-header-template.txt"

# Install tools
rm -rf vendor

go get k8s.io/code-generator@latest
go mod download k8s.io/code-generator

# Clean up any existing generated code
rm -rf pkg/client
mkdir -p pkg/client

# Generate deepcopy methods
echo "Generating deepcopy methods..."
go run k8s.io/code-generator/cmd/deepcopy-gen \
  --go-header-file ${HEADER_FILE} \
  --output-file zz_generated.deepcopy.go \
  --bounding-dirs ${API_PACKAGE} \
  ${API_PACKAGE}

# Generate client code - adjusted to point directly to v1alpha1
echo "Generating client code..."
go run k8s.io/code-generator/cmd/client-gen \
  --go-header-file ${HEADER_FILE} \
  --clientset-name versioned \
  --input-base "" \
  --input ${MODULE_PATH}/pkg/apis/templatedsecret/v1alpha1 \
  --output-pkg ${MODULE_PATH}/pkg/client/clientset/ \
  --output-dir ./pkg/client/clientset

# Generate lister code
echo "Generating lister code..."
go run k8s.io/code-generator/cmd/lister-gen \
  --go-header-file ${HEADER_FILE} \
  --output-pkg ${MODULE_PATH}/pkg/client/listers/ \
  --output-dir ./pkg/client/listers \
  ${API_PACKAGE}

# Generate informer code
echo "Generating informer code..."
go run k8s.io/code-generator/cmd/informer-gen \
  --go-header-file ${HEADER_FILE} \
  --versioned-clientset-package ${MODULE_PATH}/pkg/client/clientset/versioned \
  --listers-package ${MODULE_PATH}/pkg/client/listers \
  --output-pkg ${MODULE_PATH}/pkg/client/informers/ \
  --output-dir ./pkg/client/informers/ \
  ${API_PACKAGE}

# Install vendor dependencies and cleanup
go mod vendor
go mod tidy

echo "=== Code generation complete ==="
echo "Generated files:"
find pkg/client -type f | head -n 5
