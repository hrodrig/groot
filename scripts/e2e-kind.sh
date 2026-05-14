#!/usr/bin/env bash
# Deprecated path: use testing/scripts/test-e2e-kind.sh or `make test-e2e-kind`.
# Forwards KIND_CLUSTER_NAME to GROOT_E2E_CLUSTER when the latter is unset.
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
if [[ -n "${KIND_CLUSTER_NAME:-}" && -z "${GROOT_E2E_CLUSTER:-}" ]]; then
	export GROOT_E2E_CLUSTER="$KIND_CLUSTER_NAME"
fi
exec "$ROOT/testing/scripts/test-e2e-kind.sh"
