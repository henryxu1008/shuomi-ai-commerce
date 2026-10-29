#!/usr/bin/env bash

set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
workflow="$repo_root/.github/workflows/listingkit-deploy.yml"
runner_manifest="$repo_root/deployments/kubernetes/listingkit-workbench/release-authority/listingkit-release-gate-runners.yaml"
worker_v2_manifest="$repo_root/deployments/kubernetes/listingkit-workbench/base/image-agent-temporal-worker-deployment.yaml"
worker_v3_manifest="$repo_root/deployments/kubernetes/listingkit-workbench/base/image-agent-temporal-worker-v3-deployment.yaml"

[[ -f "$workflow" && -f "$runner_manifest" ]] || {
  printf 'image-agent canary order guard inputs are missing\n' >&2
  exit 1
}

expected_workflow="$(cat <<'YAML'
name: ListingKit API Deploy

"on":
  workflow_dispatch: {}

permissions: {}

jobs:
  retired:
    runs-on: ubuntu-latest
    steps:
      - run: |
          echo "::error::Legacy ListingKit API release and rollback are retired; current-application production release is NOT_READY."
          exit 1
YAML
)"
# Require the actual fixed failure source; a marker cannot bypass this check.
[[ "$(tr -d '\r' < "$workflow")" == "$expected_workflow" ]] || {
  printf 'retired API workflow must fail before any canary or worker mutation\n' >&2
  exit 1
}
grep -q '"image-agent-manual-v3-canary"' "$runner_manifest" || {
  printf 'v3 canary runner must use the isolated image-agent-manual-v3-canary queue\n' >&2
  exit 1
}
if grep -q -- '-canary-task-queue", "image-agent-manual-v3"' "$runner_manifest"; then
  printf 'v3 canary runner must not share the production worker queue\n' >&2
  exit 1
fi
[[ ! -e "$worker_v2_manifest" && ! -e "$worker_v3_manifest" ]] || {
  printf 'retired raw image-agent worker manifests must be absent\n' >&2
  exit 1
}

printf '%s\n' 'retired API entry and retained image-agent canary queue isolation passed'
