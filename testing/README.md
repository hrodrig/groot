# Testing

Local and end-to-end helpers for groot (same idea as [pgwd](https://github.com/hrodrig/pgwd) `testing/`).

## E2E Kubernetes (kind)

Validates **`groot collect`** against a real API server: creates a **kind** cluster, applies a small workload that writes lines to stdout (pod logs), runs collect with a minimal config, and checks the **`.tar.gz`**.

**Why it feels slow:** `make test-e2e-kind` runs **`build` first** (same as `make build`), then the script. Most wall time is usually **`kind create cluster --wait 5m`** (Docker images + control plane), then rollout wait, a fixed **15s** sleep for log lines, **`groot collect`**, and finally **`kind delete cluster`** on exit. The script prints **`==> [Ns] ...`** to **`/dev/tty`** when available (so lines show up immediately under `make` and some IDE terminals); otherwise it falls back to stderr. Stdout alone is often fully block-buffered when the subprocess is not attached to a TTY.

**Requires (this script only):** the **Docker daemon** must respond to **`docker info`** before `kind` runs. The check uses **`python3`** (or GNU **`timeout`**) with a **default 25s** wall limit so a wedged engine (e.g. `docker ps` hanging forever) fails fast with a restart hint instead of stalling the script. Override with **`GROOT_DOCKER_WAIT_SECS`**. If neither `python3` nor `timeout` is available, the script warns and runs **`docker info`** unbounded. Also [kind](https://kind.sigs.k8s.io/docs/user/quick-start/#installation) and `kubectl` on the machine where you run the test (apply manifest + wait for rollout). **Groot itself does not use the `kubectl` binary** — it talks to the API with client-go; `kubectl` here is only a convenience for the disposable cluster setup.

From the repository root:

```bash
make test-e2e-kind
```

Or run the script directly:

```bash
testing/scripts/test-e2e-kind.sh
```

Override cluster name (default `groot-e2e`):

```bash
GROOT_E2E_CLUSTER=my-cluster make test-e2e-kind
```

Keep a copy of the generated **`.tar.gz`** at a path you choose (parent directories are created). Leading **`~`** is expanded. The temp run directory is still removed on exit; this file is **not** deleted by the script.

**Note:** use a destination **outside** the script’s temporary output tree (e.g. `~/tmp/...` or any fixed path). If `GROOT_E2E_ARCHIVE` pointed **inside** that temp directory, `rm -rf "$OUTDIR"` in cleanup could remove your copy as well.

```bash
GROOT_E2E_ARCHIVE=~/tmp/test-e2e-groot.tar.gz make test-e2e-kind
```

Inspect a member (path from `tar -tzf …`):

```bash
tar -xOzf ~/tmp/test-e2e-groot.tar.gz '20260514-190227/e2e-groot/log-generator-....log' | less
```

Legacy: `KIND_CLUSTER_NAME` is still honored if you run **`./scripts/e2e-kind.sh`** (it maps to `GROOT_E2E_CLUSTER`).

**Artifacts:**

| Path | Role |
|------|------|
| `testing/k8s/e2e-workload.yaml` | Namespace `e2e-groot` + Deployment `log-generator` (BusyBox loop). |
| `testing/scripts/test-e2e-kind.sh` | kind lifecycle, apply manifest, `groot collect`, assertions, cleanup (`trap`). |

**E2E capture layout:** the script passes an **ephemeral** groot config (see the `cat >"$GCFG"` block in `testing/scripts/test-e2e-kind.sh`). It sets **`collection.include_node_details: false`** and **`collection.include_node_logs: false`** so the run stays fast and avoids extra node/proxy surface. Groot still **creates** a `nodes/` directory under the capture root, but with those flags off it **does not write** per-node describe/top/kubelet files there — so after extracting the archive, **`nodes/` may be empty**. That is expected for this test, not a bug. For a full capture with node material, run **`groot collect`** with your own config and set **`include_node_details`** / **`include_node_logs`** to **`true`** (see the main README and `configs/groot.yml.sample`).

**Notes:**

- Not part of default **`make ci`** (needs Docker + cluster + time). Optional GitHub Actions job can call **`make test-e2e-kind`** like pgwd’s `test-e2e-kube`.
- Override binary with **`GROOT_BIN=/path/to/groot`**; otherwise the script builds **`./bin/groot`** if missing.

More detail: [docs/e2e-kind.md](../docs/e2e-kind.md).

## Notify smoke test (SMTP / Mailgun)

Validate **`notify.email`** (or any enabled channel) **without** kind or collect:

```bash
make build   # or go build -o ./bin/groot ./cmd/groot
# export GROOT_NOTIFY_EMAIL_* — see docs/notify-smoke-test.md
./bin/groot notify test --config examples/notify/mailgun-smoke.yml
```

Full runbook (Mailgun panel, env vars, troubleshooting, port 465): **[docs/notify-smoke-test.md](../docs/notify-smoke-test.md)**.
