#!/bin/bash

set -e -x -u

go clean -testcache

echo "Note: if you have not deployed a recent version of TemplatedSecret Controller your e2e tests may fail! Make sure the controller is deployed in your cluster."

# create ns if not exists because the `apply -f -` won't complain on a no-op if the ns already exists.
kubectl create ns $NAMESPACE --dry-run=client -o yaml | kubectl apply -f -
go test ./test/e2e/ -timeout 60m -test.v $@

echo E2E SUCCESS
