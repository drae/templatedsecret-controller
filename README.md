# templatedsecret-controller

A Kubernetes controller for generating secrets.

## Overview

templatedsecretsecret-controller provides a custom resource for generating and managing Kubernetes secrets:

- **SecretTemplate**: Generate secrets with various types of data (certificates, passwords, SSH and RSA keys)

## Key Features

- Generate secrets with various types of content
- Easy integration with Kubernetes applications
- Follows standard Kubernetes controller patterns

## Installation

### Using Kustomize (recommended)

Deploy the controller directly with kustomize:

```shell
# Production deployment
kubectl apply -k https://github.com/drae/templatedsecretsecret-controller/config/kustomize/overlays/prod

# Development deployment
kubectl apply -k https://github.com/drae/templatedsecretsecret-controller/config/kustomize/overlays/dev
```

### Using pre-built manifests

Download and apply the latest release manifests:

```shell
kubectl apply -f https://github.com/drae/templatedsecretsecret-controller/releases/latest/dowtemplatedsecretplatedsecret-controller.yaml
```

## Usage Examples

### SecretTemplate Example

Create a SecretTemplate to generate a secret with a password:

```yaml
apiVersion: templatedsecret.starstreak.dev/v1alpha1
kind: SecretTemplate
metadata:
  name: app-password
  namespace: default
spec:
  serviceAccountName: default
  inputResources: []
  template:
    type: Opaque
    stringData:
      password: $(password:gen)
```

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
