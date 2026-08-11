#!/usr/bin/env bash
# E2E: disposable kind cluster + log workload + groot collect + archive checks.
# Same layout as pgwd: testing/scripts + testing/k8s (see testing/README.md).
# Requires on the host: Docker, kind. kubectl is used only by this script to apply the manifest
# and wait for rollout — groot collect uses client-go, not the kubectl binary.
set -euo pipefail

# Elapsed seconds since this assignment (bash built-in).
SECONDS=0
# Progress: prefer /dev/tty so lines appear immediately under `make` and IDE runners
# (merged subprocess stdout/stderr can still be fully block-buffered).
# Probe must be grouped: bare `: >/dev/tty 2>/dev/null` still prints
# "No such device or address" when the node exists but there is no controlling TTY (CI).
step() {
	local line="==> [${SECONDS}s] $*"
	if { : >/dev/tty; } 2>/dev/null; then
		printf '%s\n' "$line" >/dev/tty
	else
		printf '%s\n' "$line" >&2
	fi
}

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
cd "$REPO_ROOT"

GROOT_BIN="${GROOT_BIN:-$REPO_ROOT/bin/groot}"
if [[ ! -x "$GROOT_BIN" ]]; then
	echo "Building groot into bin/groot (set GROOT_BIN to use an existing binary)" >&2
	go build -o "$REPO_ROOT/bin/groot" ./cmd/groot
	GROOT_BIN="$REPO_ROOT/bin/groot"
fi

need() {
	command -v "$1" >/dev/null 2>&1 || {
		echo "error: '$1' not found in PATH" >&2
		exit 1
	}
}

# Expand leading ~ in a path (e.g. GROOT_E2E_ARCHIVE=~/tmp/foo.tar.gz).
e2e_expand_path() {
	local raw="${1?}"
	case "$raw" in
		\~ | \~/*) printf '%s\n' "${raw/#\~/$HOME}" ;;
		*) printf '%s\n' "$raw" ;;
	esac
}

# Bounded wait for docker info — avoids hanging forever when the engine/socket is wedged
# (same symptom as `docker ps` never returning; restarting Docker Desktop usually fixes it).
need_docker_daemon() {
	local max="${GROOT_DOCKER_WAIT_SECS:-25}"
	step "Verifying Docker daemon responds (timeout ${max}s; set GROOT_DOCKER_WAIT_SECS to override) ..."
	local st=0
	if command -v python3 >/dev/null 2>&1; then
		python3 -c '
import subprocess, sys
t = int(sys.argv[1])
try:
    subprocess.run(
        ["docker", "info"],
        stdout=subprocess.DEVNULL,
        stderr=subprocess.DEVNULL,
        check=True,
        timeout=t,
    )
except subprocess.TimeoutExpired:
    sys.exit(124)
except (subprocess.CalledProcessError, FileNotFoundError):
    sys.exit(1)
' "$max" || st=$?
		case "$st" in
			0) ;;
			124)
				echo "error: Docker did not respond within ${max}s (docker info). The engine or CLI may be wedged — quit and restart Docker Desktop (or the Docker daemon), then retry." >&2
				exit 1
				;;
			*)
				echo "error: Docker daemon is not running or not reachable (docker info failed). Start Docker and retry." >&2
				exit 1
				;;
		esac
	elif command -v timeout >/dev/null 2>&1; then
		st=0
		timeout "${max}" docker info >/dev/null 2>&1 || st=$?
		if [[ "$st" -eq 124 ]]; then
			echo "error: Docker did not respond within ${max}s (docker info). The engine or CLI may be wedged — quit and restart Docker Desktop (or the Docker daemon), then retry." >&2
			exit 1
		fi
		if [[ "$st" -ne 0 ]]; then
			echo "error: Docker daemon is not running or not reachable (docker info failed). Start Docker and retry." >&2
			exit 1
		fi
	else
		echo "warning: install python3 (or GNU timeout) for a bounded Docker check; running docker info without a time limit." >&2
		if ! docker info >/dev/null 2>&1; then
			echo "error: Docker daemon is not running or not reachable (docker info failed). Start Docker and retry." >&2
			exit 1
		fi
	fi
}

need docker
need kind
need kubectl
need_docker_daemon

CLUSTER="${GROOT_E2E_CLUSTER:-groot-e2e}"
NS="e2e-groot"
step "E2E script started (cluster=${CLUSTER}; progress lines on stderr)"
KCFG="$(mktemp "${TMPDIR:-/tmp}/groot-e2e-kubeconfig.XXXXXX")"
OUTDIR="$(mktemp -d "${TMPDIR:-/tmp}/groot-e2e-out.XXXXXX")"
GCFG="$(mktemp "${TMPDIR:-/tmp}/groot-e2e-groot.yml.XXXXXX")"
K8S_MANIFEST="$REPO_ROOT/testing/k8s/e2e-workload.yaml"

cleanup() {
	rm -f "$KCFG" "$GCFG" || true
	rm -rf "$OUTDIR" || true
	if kind get clusters 2>/dev/null | grep -qx "$CLUSTER"; then
		step "Deleting kind cluster ${CLUSTER} (trap cleanup) ..."
		kind delete cluster --name "$CLUSTER" || true
	fi
}
trap cleanup EXIT

step "Checking whether kind cluster '${CLUSTER}' already exists ..."
if kind get clusters 2>/dev/null | grep -qx "$CLUSTER"; then
	echo "error: kind cluster '$CLUSTER' already exists (delete it or set GROOT_E2E_CLUSTER)" >&2
	exit 1
fi

step "Creating kind cluster '${CLUSTER}' (Docker + control plane; often the slowest step) ..."
kind create cluster --name "$CLUSTER" --wait 5m

step "Writing kubeconfig for groot collect ..."
kind get kubeconfig --name "$CLUSTER" >"$KCFG"

step "Applying workload manifest (testing/k8s/e2e-workload.yaml) ..."
kubectl --kubeconfig="$KCFG" apply -f "$K8S_MANIFEST"

step "Waiting for deployment rollout (timeout 180s) ..."
kubectl --kubeconfig="$KCFG" rollout status deployment/log-generator -n "$NS" --timeout=180s
step "Sleeping 15s so pod logs contain marker lines ..."
sleep 15

cat >"$GCFG" <<EOF
# Ephemeral config for testing/scripts/test-e2e-kind.sh.
kubeconfig: $KCFG
output_dir: $OUTDIR
file_prefix: groot-e2e
collection:
  timeout: 10m
  worker_concurrency: 3
  namespaces:
    - $NS
  include_pod_logs: true
  include_previous_logs: false
  pod_log_tail_lines: 200
  include_node_details: false
  include_node_logs: false
  include_pod_metrics: false
  extra_kubectl: []
notify:
  slack: { enabled: false, webhook_url: "" }
  discord: { enabled: false, webhook_url: "" }
  teams: { enabled: false, webhook_url: "" }
  pagerduty: { enabled: false, routing_key: "", severity: "warning", source: "groot" }
  telegram: { enabled: false, token: "", chat_id: "" }
  generic: { enabled: false, webhook_url: "", json_key: "text", headers: {} }
EOF

step "Running groot collect (API + archive) ..."
"$GROOT_BIN" collect --config "$GCFG" --kubeconfig "$KCFG" --no-notify

TAR="$(find "$OUTDIR" -maxdepth 1 -name '*.tar.gz' -type f | sort | tail -n 1)"
if [[ -z "$TAR" || ! -f "$TAR" ]]; then
	echo "error: no .tar.gz produced under $OUTDIR" >&2
	ls -la "$OUTDIR" >&2 || true
	exit 1
fi
step "Verifying archive: $(basename "$TAR")"

# Materialize the member list once. Do not pipe tar into grep -q under pipefail:
# early grep exit → tar SIGPIPE → false "missing member" (seen on GitHub Actions).
MEMBERS="$(tar -tzf "$TAR")" || {
	echo "error: cannot list archive $(basename "$TAR")" >&2
	exit 1
}

if ! grep -q 'e2e-groot/resources.txt' <<<"$MEMBERS"; then
	echo "error: archive missing e2e-groot/resources.txt" >&2
	head -40 <<<"$MEMBERS" >&2
	exit 1
fi

LOG_MEMBER="$(grep -E 'e2e-groot/.*\.log$' <<<"$MEMBERS" | head -1 || true)"
if [[ -z "$LOG_MEMBER" ]]; then
	echo "error: archive missing pod log under e2e-groot/" >&2
	grep e2e-groot <<<"$MEMBERS" >&2 || true
	exit 1
fi

step "Checking log content in: $LOG_MEMBER"
LOG_CONTENT="$(tar -xOzf "$TAR" "$LOG_MEMBER")" || {
	echo "error: cannot extract $LOG_MEMBER from archive" >&2
	exit 1
}
if ! grep -q 'e2e-groot-marker' <<<"$LOG_CONTENT"; then
	echo "error: log file does not contain expected marker lines" >&2
	tail -20 <<<"$LOG_CONTENT" >&2
	exit 1
fi

step "OK: test-e2e-kind passed (cluster=$CLUSTER archive=$(basename "$TAR"), total ~${SECONDS}s)"
if [[ -n "${GROOT_E2E_ARCHIVE:-}" ]]; then
	ARCHIVE_COPY="$(e2e_expand_path "${GROOT_E2E_ARCHIVE}")"
	step "Copying archive to GROOT_E2E_ARCHIVE: ${ARCHIVE_COPY}"
	mkdir -p "$(dirname "$ARCHIVE_COPY")"
	cp -f "$TAR" "$ARCHIVE_COPY"
	step "Saved (not deleted on exit): ${ARCHIVE_COPY}"
	step "List members: tar -tzf $(printf '%q' "$ARCHIVE_COPY") | less"
	step "Print one member: tar -xOzf $(printf '%q' "$ARCHIVE_COPY") 'MEMBER_PATH' | less"
	step "Extract all: mkdir -p e2e-inspect && tar -xzf $(printf '%q' "$ARCHIVE_COPY") -C e2e-inspect"
else
	step "Temporary tree ${OUTDIR} and its .tar.gz are deleted on exit. Set GROOT_E2E_ARCHIVE=~/tmp/test-e2e-groot.tar.gz (or any path) to keep a copy."
fi
