#!/bin/bash

set -e -u

if [[ $# -ne 1 ]]; then
    echo "Usage: ./hack/prepare-release.sh <version>"
    echo "Example: ./hack/prepare-release.sh v0.1.0"
    exit 1
fi

VERSION="$1"
VERSION_NO_V="${VERSION#v}"

echo "===== Preparing release for templated-secret-controller $VERSION ====="

# Check that we're on a clean working directory
if [[ -n "$(git status --porcelain)" ]]; then
    echo "Error: Working directory is not clean. Please commit or stash your changes first."
    exit 1
fi

echo "==> Building binary..."
make build

echo "==> Running tests..."
./hack/test.sh

echo "==> Updating version references..."
# Update version in README or other places if needed
sed -i "s/TAG ?= .*/TAG ?= $VERSION/" Makefile

echo "==> Generating Kubernetes manifests..."
mkdir -p dist

# Build the manifests using kustomize
kustomize build config/kustomize/base >dist/templated-secret-controller-base.yaml

# Update the prod overlay with the new version and build it
cd config/kustomize/overlays/prod
kustomize edit set image controller=ghcr.io/drae/templated-secret-controller:$VERSION
cd ../../../..
kustomize build config/kustomize/overlays/prod >dist/templated-secret-controller-$VERSION.yaml

echo "==> Generating release notes draft..."
PREV_TAG=$(git describe --tags --abbrev=0 2>/dev/null || echo "")

if [ -n "$PREV_TAG" ]; then
    echo "## Changes since ${PREV_TAG}" >dist/release-notes.md
    echo "" >>dist/release-notes.md
    git log --pretty=format:"* %s" ${PREV_TAG}..HEAD >>dist/release-notes.md
else
    echo "## Initial release" >dist/release-notes.md
    echo "" >>dist/release-notes.md
    git log --pretty=format:"* %s" >>dist/release-notes.md
fi

# Add installation instructions
echo "" >>dist/release-notes.md
echo "## Installation" >>dist/release-notes.md
echo "" >>dist/release-notes.md
echo "Install the controller with:" >>dist/release-notes.md
echo '```shell' >>dist/release-notes.md
echo "kubectl apply -f https://github.com/drae/templated-secret-controller/releases/download/$VERSION/templated-secret-controller-$VERSION.yaml" >>dist/release-notes.md
echo '```' >>dist/release-notes.md

echo "==> Done!"
echo ""
echo "Release assets prepared in the 'dist' directory:"
ls -la dist/

echo ""
echo "Next steps:"
echo "1. Review the changes and release notes in dist/"
echo "2. Commit any version changes: git add . && git commit -m \"Prepare release $VERSION\""
echo "3. Tag the release: git tag $VERSION"
echo "4. Push changes and tags: git push && git push origin $VERSION"
echo "5. The GitHub Actions workflow will build, sign, and publish the release"
