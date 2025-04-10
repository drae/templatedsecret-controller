# Makefile for templated-secret-controller

# Image settings
IMG ?= ghcr.io/drae/templated-secret-controller
TAG ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
PLATFORMS ?= linux/amd64,linux/arm64
# Allow skipping platform flags in environments that don't support it
DOCKER_BUILD_PLATFORM_FLAGS ?= --platform=$(PLATFORMS)

# Get the currently used golang install path (in GOPATH/bin)
GOBIN=$(shell go env GOPATH)/bin
CONTROLLER_GEN=$(GOBIN)/controller-gen

# Build settings
LDFLAGS := -ldflags="-X 'main.Version=$(TAG)' -buildid="
BUILD_FLAGS := -trimpath -mod=vendor $(LDFLAGS)
ENVTEST_K8S_VERSION = 1.27.1

.PHONY: all
all: build

# Run tests
.PHONY: test
test:
	NAMESPACE=templated-secret-dev go test ./... -coverprofile cover.out

# Show test coverage details
.PHONY: coverage
coverage: test
	go tool cover -func=cover.out

# Show test coverage in browser
.PHONY: coverage-html
coverage-html: test
	go tool cover -html=cover.out

# Run tests with specific package path
.PHONY: test-pkg
test-pkg:
	NAMESPACE=templated-secret-dev go test $(PKG) -coverprofile cover.out -v

# Test coverage for each package
.PHONY: coverage-by-pkg
coverage-by-pkg:
	@echo "Running tests and generating coverage by package..."
	@for pkg in $$(go list ./... | grep -v "/vendor/" | grep -v "/test/ci"); do \
		echo "Testing $$pkg"; \
		NAMESPACE=templated-secret-dev go test -coverprofile=coverage.tmp $$pkg || exit 1; \
		if [ -f coverage.tmp ]; then \
			go tool cover -func=coverage.tmp | tail -n 1; \
			rm coverage.tmp; \
		fi; \
	done

# Run tests only for uncovered areas (less than 50% coverage)
.PHONY: test-low-coverage
test-low-coverage: coverage
	@echo "Packages with coverage below 50%:"
	@go tool cover -func=cover.out | grep -v "100.0%" | awk '$$3 < 50.0 {print $$1}'

# Build the binary
.PHONY: build
build: fmt vet
	go build $(BUILD_FLAGS) -o bin/controller ./cmd/controller/...

# Run code generation
.PHONY: generate
generate: controller-gen
	$(CONTROLLER_GEN) object:headerFile="code-header-template.txt" paths="./pkg/apis/..."

# Run manifests generation
.PHONY: manifests
manifests: controller-gen
	$(CONTROLLER_GEN) crd paths="./pkg/apis/templatedsecret/v1alpha1" output:crd:artifacts:config=config/kustomize/base/crds

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
	docker buildx build $(DOCKER_BUILD_PLATFORM_FLAGS) --build-arg SGCTRL_VER=$(TAG) -t ${IMG}:${TAG} .

# Push the docker image
.PHONY: docker-push
docker-push:
	docker buildx build $(DOCKER_BUILD_PLATFORM_FLAGS) --build-arg SGCTRL_VER=$(TAG) -t ${IMG}:${TAG} --push .

# Find or download controller-gen
.PHONY: controller-gen
controller-gen:
ifeq (, $(shell which controller-gen))
	@{ \
	set -e; \
	CONTROLLER_GEN_TMP_DIR=$$(mktemp -d); \
	cd $$CONTROLLER_GEN_TMP_DIR; \
	go mod init tmp; \
	go get sigs.k8s.io/controller-tools/cmd/controller-gen@v0.17.2; \
	GOBIN=$(GOBIN) go install sigs.k8s.io/controller-tools/cmd/controller-gen@v0.17.2; \
	rm -rf $$CONTROLLER_GEN_TMP_DIR; \
	}
endif

# Ensure vendor directory is up-to-date
.PHONY: vendor
vendor:
	go mod vendor
	go mod tidy