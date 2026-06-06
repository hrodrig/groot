# groot specifications (behavior contract)

## 1. Purpose

`groot` is a Go CLI that collects **read-only** Kubernetes diagnostics into a single **`.tar.gz`** archive for incident response, troubleshooting, and root cause analysis (RCA).

This document is the source of truth for **observable behavior** and test expectations. Planned work and gaps live in **[ROADMAP.md](ROADMAP.md)**; shipped releases in **[CHANGELOG.md](../CHANGELOG.md)**.

## 2. Scope

### In scope

- One primary command: **`groot collect`** (plus global flags and **`--test-connection`**).
- YAML configuration (`groot.yml` or `--config`) with **`GROOT_*`** environment overrides (Viper).
- Parallel collection jobs against the Kubernetes API via **client-go** and **metrics** clients—**no `kubectl` binary** at runtime.
- Timestamped capture directory, then **`.tar.gz`** archive beside `output_dir`; ephemeral capture folder removed after archiving.
- Optional outbound **notify** channels after a **completed** collect (HTTP webhooks, Telegram, PagerDuty Events v2).
- Rootless container image (distroless nonroot) for manual or cron-style runs.

### Out of scope (v1)

- Mutating cluster resources (scale, delete, patch, apply).
- Continuous monitoring or in-cluster controller (no Helm chart / Operator in this contract—see ROADMAP **0.5.x #22**).
- Built-in email/SMTP notify (see ROADMAP **0.5.x #21**).
- Arbitrary generic webhook body templates, HMAC signing, or non-JSON webhook bodies.
- Secret redaction inside collected log files (archives may contain sensitive data—operator responsibility).
- Multi-cluster capture in one archive (see ROADMAP **1.0.0 #32**).

### Design principles

- **Read-only diagnostics only**—`collection.extra_kubectl` is allowlisted at config load and implemented via **`pkg/k8srunner`** (argv slices, no shell).
- **Fail the CLI** on config errors, Kubernetes client init failure, archive failure, or **notify delivery failure** after collect.
- **Partial job failures** inside collect are counted in `Summary` but do **not** fail the command; failures are logged (and included in notify summary text).
- **Honest config**: fields reserved for future use (e.g. **`file_prefix`**) are documented; behavior matches code today.

## 3. CLI contract

### Commands

| Command | Behavior |
|---------|----------|
| `groot` (no subcommand) | Prints help unless a global action flag is set. |
| `groot collect` | Runs the full collection workflow (see §5). |
| `groot --version` | Build metadata (`version`, `commit`, `branch`, `buildDate`). |
| `groot --print-sample-config` | Writes sample YAML to **stdout** and exits (root or `collect`). |
| `groot --test-connection` | Loads config, lists one namespace via API, prints connection OK (root or `collect`). |

### Persistent flags (root + collect)

| Flag | Effect |
|------|--------|
| `--config <path>` | Explicit YAML path; else search order in §4. |
| `--kubeconfig <path>` | Overrides `kubeconfig` in config for this run. |
| `--verbose` | Per-job `CMD` / `OK` / `ERR` console lines. |
| `--quiet` | Suppresses normal console output; **does not** disable notify. |
| `--no-notify` | Skips all notify channels. Env: `GROOT_NO_NOTIFY=1` (also `true`/`yes`). |
| `--no-color` | Disables ANSI colors in console output. |
| `--message <text>` | Sanitized suffix appended to **archive** basename (not capture dir). |
| `--test-connection` | Connectivity probe only (see above). |
| `--print-sample-config` | Sample config only. |

### `collect` flags

| Flag | Effect |
|------|--------|
| `--since <duration>` | Pod logs only: sets `collection.pod_logs_since` for this run (overrides config). Bare number = hours; or Go duration (`24h`, `45m`). |

### Exit semantics

- **0**: collect completed; notify succeeded (if enabled and not skipped).
- **Non-zero**: config load/validation, client init, context deadline (`collection.timeout`), archive error, or notify error.

## 4. Configuration contract

### Load order (when `--config` is empty)

1. `./groot.yml`
2. `~/.groot/groot.yml`
3. `/etc/groot/groot.yml`

**Not auto-loaded:** `/etc/groot/groot.yml.sample` (packaged documentation only).

`KUBECONFIG` environment variable overrides `kubeconfig` after unmarshal.

### Top-level keys

| Key | Type | Default (if omitted) | Notes |
|-----|------|----------------------|-------|
| `kubeconfig` | string | `""` | Empty → client-go default rules + in-cluster when applicable. |
| `output_dir` | string | `./out` | Supports `~` and `${VAR}` expansion. |
| `file_prefix` | string | `groot-capture` | **Reserved**—not used in capture or archive names today (ROADMAP **0.4.x #12**). |
| `collection` | object | see below | |
| `notify` | object | channels disabled | |

### `collection`

| Key | Default | Behavior |
|-----|---------|----------|
| `timeout` | `20m` | Whole-run context deadline. |
| `worker_concurrency` | `6` (min effective `4` if `<1`) | Parallel job workers. |
| `namespaces` | `[kube-system]` | Creates per-NS dirs; writes `resources.txt` JSON sections. |
| `targets` | `{}` | Per-NS pod log filters (Deployments, StatefulSets, DaemonSets, `helm_releases` by label). Empty/missing entry → broad pod logs for that NS. |
| `extra_kubectl` | `[]` | Allowlisted argv strings (§6); no shell. |
| `include_pod_logs` | `true` | Pod + control-plane log streams. |
| `include_previous_logs` | `true` | Optional `*.previous.log` jobs. |
| `pod_log_tail_lines` | `1500` | `0` = no tail limit. |
| `pod_logs_since` | `""` | Optional `--since` for pod logs; affects session slug (§5). |
| `include_node_details` | `true` | Per-node describe + top under `nodes/`. |
| `include_node_logs` | `true` | Kubelet log query + optional `/var/log/messages` proxy. |
| `node_log_tail_lines` | `5000` | `0` = API default for kubelet query. |
| `include_pod_metrics` | `true` | Cluster-wide `top pods` when metrics-server available. |

### `notify`

Each channel: `enabled` + credentials. Multiple destinations: **`;`-separated** URLs, chat IDs, or routing keys.

| Channel | Config keys | Env overrides |
|---------|-------------|---------------|
| Slack | `notify.slack.webhook_url` | `GROOT_NOTIFY_SLACK_WEBHOOK_URL` |
| Discord | `notify.discord.webhook_url` | `GROOT_NOTIFY_DISCORD_WEBHOOK_URL` |
| Teams | `notify.teams.webhook_url` | `GROOT_NOTIFY_TEAMS_WEBHOOK_URL` |
| Telegram | `token`, `chat_id` | `GROOT_NOTIFY_TELEGRAM_TOKEN`, `GROOT_NOTIFY_TELEGRAM_CHAT_ID` |
| Generic JSON | `webhook_url`, `json_key` (default `text`), `headers` | `GROOT_NOTIFY_GENERIC_WEBHOOK_URL` |
| PagerDuty v2 | `routing_key`, `severity` (default `warning`), `source` (default `groot`) | `GROOT_NOTIFY_PAGERDUTY_ROUTING_KEY` |

**Generic webhook contract:** single JSON object `{"<json_key>":"<summary line>"}` only.

**PagerDuty:** HTTP **202** expected; `custom_details` includes `total`, `success`, `failed`, `duration`, `output_dir`, `archive_path`.

Validation: enabled channels must have non-empty credentials after env merge.

## 5. Collection workflow

### Session and archive naming

1. **Capture folder** under `output_dir`: `<sessionBase>/` where `sessionBase` is:
   - `YYYYMMDD-HHMMSS`, or
   - `YYYYMMDD-HHMMSS-since-<slug>` when `pod_logs_since` / `--since` is set (`slug` = sanitized since value).
2. **Archive file**: `<sessionBase>-<cluster>[-<message-suffix>].tar.gz` in `output_dir`.
   - `<cluster>` from kubeconfig context metadata (sanitized), else `unknown-cluster`.
   - `<message-suffix>` from `--message` when non-empty after sanitization.
3. After successful tar, **capture folder is deleted**; only `.tar.gz` remains.

Tar paths are prefixed with the capture folder name (`<session>/…` inside the archive).

### Job execution

- Jobs built from: base diagnostics, `extra_kubectl`, node details/logs, namespace resources, pod logs (filtered by `targets`), metrics, RCA writers.
- Workers run jobs concurrently up to `worker_concurrency`.
- Optional jobs may fail without aborting the whole run (`Optional: true` on internal jobs).
- `Summary` reports `Total`, `Success`, `Failed`, `Failures[]`, `Duration`, `OutputDir`, `ArchivePath`.

### Typical artifacts (non-exhaustive)

| Path | Content |
|------|---------|
| `extras/cluster-info.txt` | API discovery summary |
| `extras/nodes-wide.txt` | All nodes wide |
| `extras/all-pods-wide.txt` | All pods cluster-wide |
| `extras/all-cluster-events.log` | All events |
| `extras/all-pods-top.txt` | Pod metrics (if enabled) |
| `extras/kubeconfig.txt` | Context/cluster/user/server |
| `extras/all-pod-node-placement.tsv` | Pod → node placement |
| `extras/all-pods-rca.tsv` | RCA-oriented table |
| `<ns>/resources.txt` | JSON sections for workloads in namespace |
| `<ns>/<pod>__<node>.log` | Pod logs (`unknown-node` if unscheduled) |
| `nodes/` | Per-node describe, metrics, kubelet logs when enabled |

## 6. `extra_kubectl` and `k8srunner`

Config validation (`ValidateExtraKubectl`) allows only:

| Verb | Notes |
|------|-------|
| `get`, `describe`, `top`, `logs` | Subset of resources/kinds in runner (see below) |
| `api-resources`, `api-versions`, `version`, `cluster-info` | Discovery |
| `config view` | Optional `-o` / `--output` after `view` |
| `auth can-i` | Authorization check |

**Rejected at config load:** `explain`, `wait`, and any other verb.

**Rejected at runtime (`k8srunner.Run`):**

- `explain`, `wait`
- `get` unsupported resources (supported: **pods, nodes, namespaces, events, `--raw`**)
- `describe` unsupported kinds (supported: **pod, node** summaries)
- `top` unsupported targets

Argv is split on whitespace in config—**no shell quoting** for pipelines or redirects.

## 7. Notifications

- Fire **once** after collect returns a `Summary` (success path through archive creation).
- Message format (all channels):  
  `GROOT finished. total=… success=… failed=… duration=… output=… archive=…`
- **Discord** content truncated to 2000 runes.
- Notify errors **fail the command** (`send notifications: …`).
- **`--no-notify`** / `GROOT_NO_NOTIFY` skips all channels.
- **No notify on collect abort** before summary (ROADMAP **0.5.x #19**).

## 8. Kubernetes access

- **`pkg/kubeloader`**: kubeconfig path or in-cluster config → `rest.Config`.
- **RBAC**: read/list/get/watch logs as required by selected jobs; metrics API when `include_pod_metrics` or RCA metrics columns used.
- **Tested client modules:** `k8s.io/*` v0.32.5 (see `go.mod`).

## 9. Testing baseline

| Layer | Expectation |
|-------|-------------|
| Unit | `go test -race ./...`; fake API via **`pkg/kubetest`**; table-driven **`k8srunner`** tests. |
| Coverage | Merged statement coverage ≥ **80%** (`COVER_MIN`, `make cover`). |
| E2E | **`make test-e2e-kind`** (kind + Docker); **not** part of default `make ci` (ROADMAP **0.4.x #17**). |
| Security | `make security` (govulncheck, gocyclo, grype) before release; CodeQL on GitHub. |

When behavior in this document changes, update **SPEC**, **ROADMAP** item status, and **CHANGELOG** (`(band #N)` references) in the same change set or release.
