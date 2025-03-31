# Makefile for templatedsecret-controller

# Image settings
IMG ?= ghcr.io/drae/templatedsecret-controller
TAG ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
PLATFORMS ?= linux/amd64,linux/arm64

# Get the currently used golang install path (in GOPATH/bin)
GOBIN=$(shell go env GOPATH)/bin

# Build settings
LDFLAGS := -ldflags="-X 'main.Version=$(TAG)' -buildid="
BUILD_FLAGS := -trimpath -mod=vendor $(LDFLAGS)
ENVTEST_K8S_VERSION = 1.27.1

.PHONY: all
all: build

# Run tests
.PHONY: test
test:
	go test ./... -coverprofile cover.out

# Build the binary
.PHONY: build
build: fmt vet
	go build $(BUILD_FLAGS) -o bin/controller ./cmd/controller/...

# Run code generation
.PHONY: generate
generate:
	$(CONTROLLER_GEN) object:headerFile="code-header-template.txt" paths="./pkg/apis/..."

# Run manifests generation
.PHONY: manifests
manifests:
	$(CONTROLLER_GEN) crd paths="./pkg/apis/..." output:crd:artifacts:config=config/crds

# Run fmt against code
.PHONY: fmt
fmt:
	go fmt ./...

# Run vet against code
.PHONY: vet
vet:
	go vet ./...

# Build the docker image
.PHONY: docker-build
docker-build:
	docker buildx build --platform=$(PLATFORMS) --build-arg SGCTRL_VER=$(TAG) -t ${IMG}:${TAG} .

# Push the docker image
.PHONY: docker-push
docker-push:
	docker buildx build --platform=$(PLATFORMS) --build-arg SGCTRL_VER=$(TAG) -t ${IMG}:${TAG} --push .

# Find or download controller-gen
.PHONY: controller-gen
controller-gen:
	@{ \
	set -e; \
	CONTROLLER_GEN_TMP_DIR=$$(mktemp -d); \
	cd $$CONTROLLER_GEN_TMP_DIR; \
	go mod init tmp; \
	go get sigs.k8s.io/controller-tools/cmd/controller-gen@v0.11.3; \
	GOBIN=$(GOBIN) go install sigs.k8s.io/controller-tools/cmd/controller-gen@v0.11.3; \
	rm -rf $$CONTROLLER_GEN_TMP_DIR; \
	}
	CONTROLLER_GEN=$(GOBIN)/controller-gen

# Ensure vendor directory is up-to-date
.PHONY: vendor
vendor:
	go mod vendor
	go mod tidy