#!/bin/bash

set -euo pipefail

if [ -z "$GITHUB_ACTION" ]; then
  go clean -testcache
fi

set -u

go test ./pkg/... -test.v $@

echo UNIT SUCCESS
