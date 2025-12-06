#!/usr/bin/env bash

# This script builds and syncs Bazel-generated files to the source tree.
# It leverages Bazel's caching to speed up subsequent runs when source
# files haven't changed.
#
# Usage:
#   ./hack/sync-generated-from-bazel.sh [--verify]
#
# Options:
#   --verify    Only verify that generated files are up-to-date, don't update them

set -e

source $(dirname "$0")/common.sh
source $(dirname "$0")/config.sh

VERIFY_MODE=false
if [[ "${1:-}" == "--verify" ]]; then
    VERIFY_MODE=true
fi

echo "Building generated files with Bazel (cached)..."

# Build all generation targets
bazel build \
    --config=${ARCHITECTURE} \
    //:all-generated

echo "Syncing generated files to source tree..."

# Get bazel-bin directory
BAZEL_BIN=$(bazel info bazel-bin --config=${ARCHITECTURE})

# Function to sync a single file
sync_file() {
    local src="$1"
    local dst="$2"
    
    if [[ ! -f "$src" ]]; then
        echo "ERROR: Source file not found: $src"
        return 1
    fi
    
    # Create destination directory if it doesn't exist
    mkdir -p "$(dirname "$dst")"
    
    if [[ "$VERIFY_MODE" == "true" ]]; then
        if [[ -f "$dst" ]]; then
            if ! diff -q "$src" "$dst" > /dev/null 2>&1; then
                echo "MISMATCH: $dst is out of date"
                return 1
            fi
        else
            echo "MISSING: $dst does not exist"
            return 1
        fi
    else
        cp -f "$src" "$dst"
        echo "  Synced: $dst"
    fi
}

SYNC_FAILED=false

# Sync manifest files
sync_file "${BAZEL_BIN}/generated-manifests/kubevirt-priority-class.yaml" \
          "${KUBEVIRT_DIR}/manifests/generated/kubevirt-priority-class.yaml" || SYNC_FAILED=true

sync_file "${BAZEL_BIN}/generated-manifests/kv-resource.yaml" \
          "${KUBEVIRT_DIR}/manifests/generated/kv-resource.yaml" || SYNC_FAILED=true

sync_file "${BAZEL_BIN}/generated-manifests/kubevirt-network-policies.yaml.in" \
          "${KUBEVIRT_DIR}/manifests/generated/kubevirt-network-policies.yaml.in" || SYNC_FAILED=true

sync_file "${BAZEL_BIN}/generated-manifests/kubevirt-cr.yaml.in" \
          "${KUBEVIRT_DIR}/manifests/generated/kubevirt-cr.yaml.in" || SYNC_FAILED=true

sync_file "${BAZEL_BIN}/generated-manifests/rbac-operator.authorization.k8s.yaml.in" \
          "${KUBEVIRT_DIR}/manifests/generated/rbac-operator.authorization.k8s.yaml.in" || SYNC_FAILED=true

sync_file "${BAZEL_BIN}/generated-manifests/operator-csv.yaml.in" \
          "${KUBEVIRT_DIR}/manifests/generated/operator-csv.yaml.in" || SYNC_FAILED=true

# Sync OpenAPI spec
sync_file "${BAZEL_BIN}/generated-api/swagger.json" \
          "${KUBEVIRT_DIR}/api/openapi-spec/swagger.json" || SYNC_FAILED=true

# Sync metrics documentation
sync_file "${BAZEL_BIN}/generated-docs/metrics.md" \
          "${KUBEVIRT_DIR}/docs/observability/metrics.md" || SYNC_FAILED=true

if [[ "$VERIFY_MODE" == "true" ]]; then
    if [[ "$SYNC_FAILED" == "true" ]]; then
        echo ""
        echo "ERROR: Generated files are out of date!"
        echo "Run 'make generate' to update them."
        exit 1
    else
        echo "All generated files are up-to-date."
    fi
else
    if [[ "$SYNC_FAILED" == "true" ]]; then
        echo ""
        echo "WARNING: Some files failed to sync."
        exit 1
    else
        echo ""
        echo "Generated files synced successfully."
    fi
fi
