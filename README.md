# templated-secret-controller

A Kubernetes controller for generating secrets from existing resources.

## Overview

templated-secret-controller provides a custom resource for generating and managing Kubernetes secrets by combining data from other resources:

- **SecretTemplate**: Generate secrets using data from existing Kubernetes resources, including other Secrets, ConfigMaps, Services, and more.

> **Note on naming:** While the controller is named "templated-secret-controller" (with a hyphen), the API group remains "templatedsecret.starstreak.dev" (without a hyphen) for compatibility with code generation tools.

## Key Features

- Generate secrets by combining data from multiple existing Kubernetes resources
- Template data using JSONPath expressions to extract specific values
- Continuously reconcile secrets when source resources change
- Support for various Kubernetes resource types as input sources
- Role-based access control for reading input resources

## Installation

### Using Helm

Deploy the controller using Helm:

```shell
# Clone the repository
git clone https://github.com/drae/templated-secret-controller.git
cd templated-secret-controller

# Install with default settings
helm install templated-secret-controller ./charts/templated-secret-controller

# Install with metrics disabled
helm install templated-secret-controller ./charts/templated-secret-controller --set metrics.enabled=false

# Install without CRDs (if you've already installed them)
helm install templated-secret-controller ./charts/templated-secret-controller --set crds.create=false
```

For more information on configuration options, see the [Helm chart README](./charts/templated-secret-controller/README.md).

### Using Kustomize

Deploy the controller directly with kustomize:

```shell
# Production deployment
kubectl apply -k https://github.com/drae/templated-secret-controller/config/kustomize/overlays/prod

# Development deployment
kubectl apply -k https://github.com/drae/templated-secret-controller/config/kustomize/overlays/dev
```

### Using pre-built manifests

Download and apply the latest release manifests:

```shell
kubectl apply -f https://github.com/drae/templated-secret-controller/releases/latest/download/templated-secret-controller.yaml
```

## Example

```yaml
apiVersion: templatedsecret.starstreak.dev/v1alpha1
kind: SecretTemplate
metadata:
  name: combined-secret
spec:
  inputResources:
    - name: secret1
      ref:
        apiVersion: v1
        kind: Secret
        name: secret1
    - name: secret2
      ref:
        apiVersion: v1
        kind: Secret
        name: secret2
  template:
    type: mysecrettype
    data:
      key1: $(.secret1.data.key1)
      key2: $(.secret1.data.key2)
      key3: $(.secret2.data.key3)
      key4: $(.secret2.data.key4)
```

See [the SecretTemplate documentation](docs/secret-template.md) for more detailed examples and explanations.

## Local Development

This project uses standard Go tools and Kubernetes controller patterns:

```shell
# Build
make build

# Run tests
make test

# Build container image
make docker-build
```

## CI/CD

The project uses GitHub Actions for continuous integration and deployment:

- CI workflow runs on PRs and pushes to main
- Release workflow triggers on tags formatted as 'v*'
- Images are published to GitHub Container Registry
