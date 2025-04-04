#!/bin/bash

set -euo pipefail

kubectl create ns $NAMESPACE --dry-run=client -o yaml | kubectl apply -f -

go clean -testcache
go test ./test/ci/ -timeout 60m -test.v $@

echo "✅ CI tests passed"
