#!/bin/bash

# This script validates that all Kubernetes resources in a YAML file have the required fields
# Specifically checking for apiVersion and kind fields

set -euo pipefail

FILE=${1:-"dist/templatedsecret-controller-base.yaml"}

echo "Validating Kubernetes manifests in $FILE..."

# Split the YAML file into individual resources
resources=$(grep -n "^---" "$FILE" | cut -d: -f1)

# Add line 1 and last line to the array
resources="1 $resources $(wc -l <"$FILE")"
read -ra resource_lines <<<"$resources"

errors=0

# Loop through each resource and check for apiVersion and kind
for ((i = 0; i < ${#resource_lines[@]} - 1; i++)); do
    start=${resource_lines[$i]}
    end=${resource_lines[$i + 1]}

    # Skip the separator line
    if [[ $start == *"---"* ]]; then
        start=$((start + 1))
    fi

    # Extract this resource
    resource=$(sed -n "${start},${end}p" "$FILE")

    # Check if this resource has apiVersion and kind
    if ! echo "$resource" | grep -q "apiVersion:"; then
        echo "ERROR: Resource at lines $start-$end is missing apiVersion"
        echo "$resource" | head -n 5
        echo "..."
        errors=$((errors + 1))
    fi

    if ! echo "$resource" | grep -q "kind:"; then
        echo "ERROR: Resource at lines $start-$end is missing kind"
        echo "$resource" | head -n 5
        echo "..."
        errors=$((errors + 1))
    fi
done

if [ $errors -eq 0 ]; then
    echo "All resources in $FILE have required fields."
    exit 0
else
    echo "Found $errors errors in $FILE"
    exit 1
fi
