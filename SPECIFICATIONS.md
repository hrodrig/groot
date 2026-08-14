# groot specifications (behavior contract)

## 1. Purpose

`groot` is a Go CLI that **collects read-only Kubernetes logs and cluster context** into a single **`.tar.gz`** archive for incident response, troubleshooting, and root cause analysis (RCA). Offline **`groot analyze`** emits evidence-backed **hints** from an existing archive (not a live-cluster diagnosis and not a definitive root-cause claim).

This document is the source of truth for **observable behavior** and test expectations. Planned work and gaps live in **[ROADMAP.md](ROADMAP.md)**; shipped releases in **[CHANGELOG.md](../CHANGELOG.md)**.

## 2. Scope

### In scope

- Primary command: **`groot collect`** (plus global flags and **`--test-connection`**).
- Offline archive commands: **`groot inspect`**, **`groot analyze`** (no kubeconfig / API).
- YAML configuration (`groot.yml` or `--config`) with **`GROOT_*`** environment overrides (Viper).
- Parallel collection jobs against the Kubernetes API via **client-go** and **metrics** clients—**no `kubectl` binary** at runtime.
- Timestamped capture directory, then **`.tar.gz`** archive beside `output_dir`; ephemeral capture folder removed after archiving.
- Optional outbound **notify** channels after a **completed** collect (HTTP webhooks, Telegram, PagerDuty Events v2, email/SMTP).
- Rootless container image (`gcr.io/distroless/static-debian13:nonroot`) for manual or cron-style runs.
- Optional **Helm chart** and flat **CronJob** manifests for scheduled in-cluster collection (maintained in **[groot-selfhosted](https://github.com/hrodrig/groot-selfhosted)** `run/deploy/`).

### Out of scope (v1)

- Mutating cluster resources (scale, delete, patch, apply).
- Continuous monitoring or in-cluster controller beyond a **CronJob** schedule (no Operator).
- Long-lived HTTP inside this CLI (on-demand collect API: **[groot-trigger](https://github.com/hrodrig/groot-trigger)**).
- Archive catalog / VPS door (**[groot-share](https://github.com/hrodrig/groot-share)** / **gfs**); gfs deploy playbooks live in **[groot-share-selfhosted](https://github.com/hrodrig/groot-share-selfhosted)**.
- Arbitrary non-JSON webhook bodies.
- Multi-cluster capture in one archive (see ROADMAP **1.0.0 #32**).

### Packaging / man pages (ROADMAP #96 / [GH #2](https://github.com/hrodrig/groot/issues/2))

- Authoritative man sources: **`contrib/man/man1/groot.1`** and **`contrib/man/man1/kubectl-groot.1`**.
- Each release bumps the man **`.TH`** lines to match **`VERSION`** (date + `groot vX.Y.Z`) via **`make man-sync`**.
- GoReleaser **nfpm** (`.deb` / `.rpm`) installs gzipped pages under **`/usr/share/man/man1/`**.
- Release **archives** ship uncompressed pages under **`share/man/man1/`** (Homebrew cask `manpages:`; FreeBSD/OpenBSD ports install from the same layout).
- **FreeBSD** and **OpenBSD** ports install those pages so **`man groot`** / **`man kubectl-groot`** work on BSD — ports are first-class, not Linux-only afterthoughts.
- Release hygiene: **`make man-sync`**, VHS (`docs/demo.gif`), README/CHANGELOG, and BSD **`PORTVERSION`** stay in the VERSION bump checklist.

### Design principles

- **Read-only collection only**—`collection.extra_kubectl` is allowlisted at config load and implemented via **`internal/k8srunner`** (argv slices, no shell).
- **Fail the CLI** on config errors, Kubernetes client init failure, archive failure, or **notify delivery failure** after collect.
- **Partial job failures** inside collect are counted in `Summary` but do **not** fail the command; failures are logged (and included in notify summary text).
- **Honest config**: reserved or partial fields are documented; behavior matches code and this spec.

## 3. CLI contract

### Commands

| Command | Behavior |
|---------|----------|
| `groot` (no subcommand) | Prints help unless a global action flag is set. |
| `groot collect` | Runs the full collection workflow (see §5). |
| `groot --version` / `groot -v` | `groot vX.Y.Z (commit=… branch=… built=…)`. |
| `groot version` | Same output as `--version`. |
| `groot --print-sample-config` | Writes sample YAML to **stdout** and exits (root or `collect`). |
| `groot --test-connection` | Loads config, lists one namespace via API, prints connection OK (root or `collect`). |
| `groot notify test` | Sends a synthetic summary to all enabled notify channels (see §15). Does **not** run collect or contact the cluster. |
| `groot validate` | Preflight checks without writing an archive (see §12). |
| `groot inspect <archive>` | Offline inventory of a collect `.tar.gz` (see §13). |
| `groot analyze <archive>` | Offline heuristic hints from a collect `.tar.gz` (see §16). |

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
| `--list-jobs` | Print the planned collection jobs (name, output file, args, `optional`) and exit **without** writing the capture tree or `.tar.gz`, and without firing notify. Requires API reachability for dynamic jobs (nodes, pod logs). |
| `--output <text\|json>` | After a successful collect, emit `Summary` as JSON to stdout (`json`) or suppress structured output (`text`, default). |
| `--summary` | After success, print a one-screen human-readable footer (jobs, duration, archive, unhealthy pod counts). |

### Exit semantics (0.9.x #82)

| Code | Meaning |
|------|---------|
| 0 | Success (collect completed; notify succeeded when enabled and not skipped) |
| 1 | Config validation (YAML load, `--since`, missing config file) |
| 2 | Kubernetes client / API error (auth, list, handshake) |
| 3 | Collect aborted (timeout, archive failure); also **archive open/read failure** for `inspect` / `analyze` |
| 4 | Notify delivery failed |
| 5 | Reserved — partial failures ≥ threshold in opt-in `--strict` mode (`--strict` flag, default threshold 1; see [plan-0.9.0.md](docs/plan-0.9.0.md)) |

\`\`\`
Partial job failures inside collect remain **non-fatal** by default; they are counted in `Summary` and logged.

## 4. Configuration contract

### Load order (when `--config` is empty)

1. `./groot.yml`
2. `~/.groot/groot.yml`
3. `/etc/groot/groot.yml`

**Not auto-loaded:** `/etc/groot/groot.yml.sample` (packaged documentation only).

`KUBECONFIG` environment variable overrides `kubeconfig` after unmarshal. Both the YAML value and `KUBECONFIG` are expanded (`~` / `${VAR}`; multipath lists per entry).

### Top-level keys

| Key | Type | Default (if omitted) | Notes |
|-----|------|----------------------|-------|
| `config_version` | int | `0` (legacy) | **1.0.0+:** use `1` for new configs. Absent or `0` = legacy pre-1.0 behavior. Loader rejects unknown future versions. |
| `kubeconfig` | string | `""` | Empty → client-go default rules + in-cluster when applicable. Supports `~` / `${VAR}` expansion (same as `output_dir`). Multipath values use `os.PathListSeparator` (same as `KUBECONFIG`). |
| `cluster_name` | string | `""` | Optional archive basename cluster label. When empty, resolved from kubeconfig → `kube-public/cluster-info` → API server host. |
| `output_dir` | string | `./out` | Supports `~` and `${VAR}` expansion. |
| `file_prefix` | string | `groot-capture` | **Used** in capture directory and archive basename (ROADMAP **0.4.x #12**). |
| `collection` | object | see below | |
| `notify` | object | channels disabled | |

### `collection`

| Key | Default | Behavior |
|-----|---------|----------|
| `timeout` | `20m` | Whole-run context deadline. |
| `worker_concurrency` | `6` (min effective `4` if `<1`) | Parallel job workers. |
| `namespaces` | `[kube-system]` | Creates per-NS dirs; writes `resources.txt` JSON sections. |
| `targets` | `{}` | Per-NS pod log filters (`deployments`, `statefulsets`, `daemonsets`, `jobs`, `cronjobs`, `helm_releases`) matched by labels. Empty/missing entry → broad pod logs for that NS. |
| `extra_kubectl` | `[]` | Allowlisted argv strings (§6); no shell. |
| `include_pod_logs` | `true` | Pod + control-plane log streams. |
| `include_previous_logs` | `true` | Optional `*.previous.log` jobs. |
| `pod_log_tail_lines` | `1500` | `0` = no tail limit. |
| `pod_logs_since` | `""` | Optional `--since` for pod logs; affects session slug (§5). |
| `include_node_details` | `true` | Per-node describe + top under `nodes/`. |
| `include_node_logs` | `true` | Per-node host logs via `…/proxy/logs/messages` → `nodes/<node>.log`; optional kubelet log query → `nodes/<node>-kubelet.log`. Both job types are optional (failure does not abort the run). |
| `node_log_tail_lines` | `5000` | When kubelet log query runs: `0` = API default; `>0` adds `tailLines`. Ignored for the messages proxy. |
| `include_pod_metrics` | `true` | Cluster-wide `top pods` when metrics-server available. |
| `redact_secrets` | `false` | When `true`, scan collected `*.log` files and replace likely secret values before archiving. |
| `redact_patterns` | `[]` | Optional extra regex patterns (RE2 syntax); invalid patterns fail at collect time. |

### `notify`

Each channel: `enabled` + credentials. Multiple destinations: **`;`-separated** URLs, chat IDs, or routing keys.

| Channel | Config keys | Env overrides |
|---------|-------------|---------------|
| Slack | `notify.slack.webhook_url` | `GROOT_NOTIFY_SLACK_WEBHOOK_URL` |
| Discord | `notify.discord.webhook_url` | `GROOT_NOTIFY_DISCORD_WEBHOOK_URL` |
| Teams | `notify.teams.webhook_url` | `GROOT_NOTIFY_TEAMS_WEBHOOK_URL` |
| Telegram | `token`, `chat_id` | `GROOT_NOTIFY_TELEGRAM_TOKEN`, `GROOT_NOTIFY_TELEGRAM_CHAT_ID` |
| Generic JSON | `webhook_url`, `json_key` (default `text`), `headers`, `extra_fields`, `body_template`, `hmac_secret`, `hmac_header` (default `X-Groot-Signature`) | `GROOT_NOTIFY_GENERIC_WEBHOOK_URL`, `GROOT_NOTIFY_GENERIC_HMAC_SECRET` |
| Email | `host`, `port` (default `587`), `username`, `password`, `from`, `to`, `use_tls`, `skip_verify` | `GROOT_NOTIFY_EMAIL_HOST`, `_USERNAME`, `_PASSWORD`, `_FROM`, `_TO` |
| PagerDuty v2 | `routing_key`, `severity` (default `warning`), `source` (default `groot`) | `GROOT_NOTIFY_PAGERDUTY_ROUTING_KEY |

**On failure (`notify.on_failure`):**

| Key | Default | Behavior |
|-----|---------|----------|
| `enabled` | `false` | Master switch for failure alerts. |
| `on_abort` | `true` | When `enabled`, notify if collect aborts before completion (archive error, client init, timeout, …). |
| `min_failed_jobs` | `1` | When `enabled` and collect completes, also notify if `summary.Failed >= min_failed_jobs` (in addition to success notify). |

**HTTP retry (`notify.retry`):** `max_attempts` (default `3`), `initial_backoff` (default `1s`), `max_backoff` (default `10s`). Retries transient **5xx** and network errors only; **4xx** fails immediately.

**Generic webhook contract:**

- **Default:** single JSON object `{"<json_key>":"<summary line>"}` plus optional `extra_fields` (values support `{{summary}}`, `{{total}}`, `{{failed}}`, `{{event}}`, … placeholders).
- **`body_template`:** when set, POST the rendered JSON template instead of the default shape. Must be valid JSON after substitution.
- **HMAC:** when `hmac_secret` is set, header `hmac_header` (default `X-Groot-Signature`) is `sha256=<hex>` over the raw POST body (HMAC-SHA256).

**PagerDuty:** HTTP **202** expected; `custom_details` includes `total`, `success`, `failed`, `duration`, `output_dir`, `archive_path`.

Validation: enabled channels must have non-empty credentials after env merge.

## 5. Collection workflow

### Session and archive naming

1. **Capture folder** under `output_dir`: `<sessionBase>/` where `sessionBase` is:
   - `<sanitize(file_prefix)>-<short>-<timestamp>`, or
   - `<sanitize(file_prefix)>-<short>-<timestamp>-since-<slug>` when `pod_logs_since` / `--since` is set (`slug` = sanitized since value).
   - `<short>` is the lowercase random suffix from this run’s `run_id` (same entropy as `#81`), so concurrent collects in the same second do not collide on a shared `output_dir`.
   - Default `file_prefix` is `groot-capture`; empty value falls back to the same default.
2. **Archive file**: `<sessionBase>-<cluster>[-<message-suffix>].tar.gz` in `output_dir`.
   - `<cluster>` resolution order: `cluster_name` config (if set) → kubeconfig cluster metadata → `kube-public/cluster-info` ConfigMap → API server host (sanitized) → `unknown-cluster`.
   - `<message-suffix>` from `--message` when non-empty after sanitization.
3. After successful tar, **capture folder is deleted**; only `.tar.gz` remains.

Tar paths are prefixed with the capture folder name (`<session>/…` inside the archive).

### Archive manifest

After all jobs complete (and before the capture folder is removed), `extras/manifest.json` is written inside the archive with the structure below. This speeds ticket handoff and lets downstream tools verify what was collected.

```json
{
  "groot_version": "0.5.0",
  "groot_commit": "…",
  "config_version": 1,
  "archive_layout_version": 1,
  "collected_at": "2026-06-05T12:00:00Z",
  "duration_seconds": 42.5,
  "session_base": "groot-capture-7kqv2xy-20260605-120000",
  "archive_basename": "groot-capture-20260605-120000-my-cluster",
  "file_prefix": "groot-capture",
  "cluster": { "context": "…", "cluster": "…", "user": "…", "server": "…" },
  "jobs": { "total": 10, "success": 9, "failed": 1 },
  "paths": ["extras/kubeconfig.txt", "…"]
}
```

`paths` is the sorted list of files under the capture tree (slash-separated, relative to the capture folder). Empty fields fall back to `"unknown"` or `"dev"`.

### Job execution

- Jobs built from: base diagnostics, `extra_kubectl`, node details/logs, namespace resources, pod logs (filtered by `targets`), metrics, RCA writers.
- When `redact_secrets` is `true`, collected `*.log` files are scanned and likely secret values replaced with `[REDACTED]` after jobs complete and before `extras/manifest.json` and archiving.
- Workers run jobs concurrently up to `worker_concurrency`.
- **Kubernetes API rate limit:** Groot sets client-go **`QPS=50`** and **`Burst=100`** on the shared REST config (`internal/kubeloader`). All collection jobs share one token bucket; workers compete for those tokens. **`worker_concurrency`** controls how many jobs run in parallel, not the QPS cap. On large clusters or slow apiserver, reduce **`worker_concurrency`** if you see long stalls or HTTP **429** responses. QPS/Burst are **not** YAML-configurable today (roadmap **#67**).
- Optional jobs may fail without aborting the whole run (`Optional: true` on internal jobs).
- `Summary` reports `Total`, `Success`, `Failed`, `Failures[]`, `Duration`, `OutputDir`, `ArchivePath`.

### Typical artifacts (non-exhaustive)

| Path | Content |
|------|---------|
| `extras/manifest.json` | Archive manifest (groot version, cluster, jobs, paths) — see §5. |
| `extras/cluster-info.txt` | API discovery summary |
| `extras/nodes-wide.txt` | All nodes wide |
| `extras/all-pods-wide.txt` | All pods cluster-wide |
| `extras/all-cluster-events.log` | All events |
| `extras/all-pods-top.txt` | Pod metrics (if enabled) |
| `extras/kubeconfig.txt` | Context/cluster/user/server |
| `extras/all-pod-node-placement.tsv` | Pod → node placement |
| `extras/workload-resources.tsv` | Per-container CPU/memory requests and limits + owner reference |
| `extras/all-pods-rca.tsv` | RCA-oriented table (usage + declared resources + log path) |
| `<ns>/resources.txt` | JSON sections for workloads in namespace |
| `<ns>/<pod>__<node>.log` | Pod logs (`unknown-node` if unscheduled) |
| `nodes/<node>.log` | Host `/var/log/messages` via node proxy when enabled |
| `nodes/<node>-kubelet.log` | Kubelet via Node Log Query API when cluster supports it (optional) |
| `nodes/<node>-describe.txt`, `-top.txt` | Per-node describe and metrics when `include_node_details` |

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
- `get` unsupported resources (supported: **pods, nodes, namespaces, events, configmap, pvc, service, ingress, deployment, replicaset, statefulset, daemonset, `--raw`**)
- `describe` unsupported kinds (supported: **pod, node, configmap, pvc, service, ingress** summaries)
- `top` unsupported targets

`get` aliases: `cm` (configmap), `svc` (service), `ing` (ingress), `deploy`/`rs`/`sts`/`ds` (apps workloads). `describe` aliases: same set as `get`. `get --raw <path>` passes the path straight to the API server for CRD or generic reads.

Argv is split on whitespace in config—**no shell quoting** for pipelines or redirects.

## 7. Notifications

- Fire **once** after collect returns a `Summary` on the success path (archive created).
- Optional **failure alerts** when `notify.on_failure.enabled` (see §4): on collect abort and/or when partial job failures exceed `min_failed_jobs`. Respects **`--no-notify`** / `GROOT_NO_NOTIFY`.
- **Success** message format (all channels):  
  `GROOT finished. total=… success=… failed=… duration=… output=… archive=…`
- **Failure** message format:  
  `GROOT FAILED. reason=…` (abort) or `GROOT finished with failures. …` (partial threshold).
- **Discord** content truncated to 2000 runes.
- Notify errors **fail the command** (`send notifications: …` / `send failure notifications: …`).
- HTTP notify clients **retry** transient 5xx and network errors per `notify.retry` (§4).

## 8. Kubernetes access

- **`internal/kubeloader`**: kubeconfig path or in-cluster config → `rest.Config`.
- **RBAC**: read/list/get/watch logs as required by selected jobs; metrics API when `include_pod_metrics` or RCA metrics columns used.
- **In-cluster scheduling**: CronJob + ClusterRole + ConfigMap + optional PVC for `/out`; image `ghcr.io/hrodrig/groot`. Helm chart and flat manifests: **[groot-selfhosted](https://github.com/hrodrig/groot-selfhosted)** `run/deploy/`.
- **Tested client modules:** `k8s.io/*` v0.32.5 (see `go.mod`).

## 9. Configuration examples

Full annotated sample: **`configs/groot.yml.sample`** (same as `groot --print-sample-config`).

### Notify on failure + Slack

```yaml
notify:
  on_failure:
    enabled: true
    on_abort: true
    min_failed_jobs: 1
  slack:
    enabled: true
    webhook_url: "https://hooks.slack.com/services/…"
```

### Generic webhook with template and HMAC

```yaml
notify:
  generic:
    enabled: true
    webhook_url: "https://hooks.example/groot"
    body_template: '{"text":"{{summary}}","failed":{{failed}},"event":"{{event}}"}'
    hmac_secret: "signing-key"
    hmac_header: "X-Groot-Signature"
```

Rendered POST (success): `{"text":"GROOT finished. total=…","failed":0,"event":"success"}`. Header: `X-Groot-Signature: sha256=<hex>`.

### Email

```yaml
notify:
  email:
    enabled: true
    host: smtp.example.com
    port: 587
    from: groot@example.com
    to: "ops@example.com"
```

### Secret redaction

```yaml
collection:
  redact_secrets: true
  redact_patterns:
    - '(?i)internal-api-key\s*=\s*\S+'
```

### In-cluster CronJob (Helm values excerpt)

```yaml
schedule: "0 */6 * * *"
image:
  repository: ghcr.io/hrodrig/groot
  tag: "0.5.0"
config:
  grootYml: |
    output_dir: /out
    collection:
      namespaces: [kube-system, default]
    notify:
      on_failure:
        enabled: true
```

Install: see **[groot-selfhosted](https://github.com/hrodrig/groot-selfhosted/blob/main/run/deploy/README.md)** (`helm upgrade --install groot ./run/deploy/helm/groot …`).

## 10. Post-collect upload

Optional upload of the finished **`.tar.gz`** after notify on the success path.

| Key | Type | Default | Behavior |
|-----|------|---------|----------|
| `upload.enabled` | bool | `false` | Master switch. |
| `upload.continue_on_error` | bool | `true` | Try remaining providers when one fails. |
| `upload.timeout` | duration | `5m` | Per-provider upload deadline. |
| `upload.s3.enabled` | bool | `false` | Enable S3 (or S3-compatible) upload. |
| `upload.s3.bucket` | string | — | Required when S3 enabled; env `GROOT_UPLOAD_S3_BUCKET`. |
| `upload.s3.region` | string | — | AWS region; env `GROOT_UPLOAD_S3_REGION` or `AWS_REGION`. |
| `upload.s3.key_prefix` | string | — | Object key prefix; env `GROOT_UPLOAD_S3_KEY_PREFIX`. |
| `upload.s3.endpoint` | string | — | S3-compatible endpoint URL; env `GROOT_UPLOAD_S3_ENDPOINT`. |
| `upload.gcs.enabled` | bool | `false` | Enable GCS upload. |
| `upload.gcs.bucket` | string | — | Required when GCS enabled; env `GROOT_UPLOAD_GCS_BUCKET`. |
| `upload.gcs.key_prefix` | string | — | Object key prefix; env `GROOT_UPLOAD_GCS_KEY_PREFIX`. |
| `upload.sftp.enabled` | bool | `false` | Enable SFTP upload (bastion → relay). |
| `upload.sftp.host` | string | — | Required when SFTP enabled; env `GROOT_UPLOAD_SFTP_HOST`. |
| `upload.sftp.port` | int | `22` | SSH port; env `GROOT_UPLOAD_SFTP_PORT`. |
| `upload.sftp.user` | string | — | SSH user; env `GROOT_UPLOAD_SFTP_USER`. |
| `upload.sftp.remote_dir` | string | — | Remote target directory; env `GROOT_UPLOAD_SFTP_REMOTE_DIR`. |
| `upload.sftp.known_hosts_file` | string | — | Path to `known_hosts`; env `GROOT_UPLOAD_SFTP_KNOWN_HOSTS`. Supports `~` / `${VAR}`. **Required** when SFTP is enabled unless `allow_insecure_host_key: true`. |
| `upload.sftp.allow_insecure_host_key` | bool | `false` | Testing only — skips host key verification (MITM risk). |
| `upload.sftp.identity_file` | string | — | **Env only** (`GROOT_UPLOAD_SFTP_IDENTITY_FILE`); never in YAML. Supports `~` / `${VAR}`. |

- Credentials: **AWS** via standard `AWS_*` env vars (`AWS_ACCESS_KEY_ID` / `AWS_SECRET_ACCESS_KEY` / optional `AWS_SESSION_TOKEN` are **trimmed** of surrounding whitespace before use); **GCS** via `GOOGLE_APPLICATION_CREDENTIALS` (or workload identity in-cluster); **SFTP** via SSH key (`GROOT_UPLOAD_SFTP_IDENTITY_FILE`). When **`STORAGE_EMULATOR_HOST`** is set (local emulator / fake GCS), Groot uses that endpoint with no auth (standard client-go emulator path).
- SFTP auth: **public-key only** (BatchMode — password/keyboard-interactive rejected). Host key verified against `known_hosts_file`; **required** unless `allow_insecure_host_key: true` (testing only).
- Object key: `<key_prefix>/<archive-basename>` (prefix optional). SFTP remote path: `<remote_dir>/<archive-basename>`.
- Runs **after** archive write and success notify; **upload errors do not fail** the collect command (logged at ERROR).
- **`--no-upload`** / `GROOT_NO_UPLOAD=1` skips upload entirely.

### S3 example

```yaml
upload:
  enabled: true
  s3:
    enabled: true
    bucket: my-archives
    region: us-east-1
    key_prefix: groot/prod
```

### SFTP example (bastion → relay)

```yaml
upload:
  enabled: true
  sftp:
    enabled: true
    host: ipA.example.com
    port: 22
    user: groot-inbox
    remote_dir: inbox
    known_hosts_file: /etc/groot/known_hosts
```

Identity file from env: `GROOT_UPLOAD_SFTP_IDENTITY_FILE=/home/groot/.ssh/id_ed25519`

## 11. Testing baseline

| Layer | Expectation |
|-------|-------------|
| Unit | `go test -race ./...`; fake API via **`internal/kubetest`**; table-driven **`k8srunner`** tests. |
| Coverage | Merged statement coverage ≥ **80%** (`COVER_MIN`, `make cover`). |
| E2E | **`make test-e2e-kind`** (kind + Docker); **not** part of default `make ci` (ROADMAP **0.4.x #17**). |
| Security | `make security` (govulncheck, gocyclo, grype) before release; CodeQL on GitHub. |

When behavior in this document changes, update **SPEC**, **ROADMAP** item status, and **CHANGELOG** (`(band #N)` references) in the same change set or release.

## 12. `groot validate` — preflight checks (0.9.x #31, #83)

```
groot validate [--output text|json] [--config <path>] [--min-disk <bytes>] [--warn-disk <bytes>]
```

Validates the environment before a collect run. All checks are independent; failures are aggregated.

**Check order:** config load → Kubernetes client init → cluster-info → RBAC matrix (`auth can-i` for `list namespaces`, `list pods`, `get pods/log`, `list events`, `list nodes`) → disk space on `output_dir`.

**Disk defaults:** hard fail at <256 MiB (`collection.min_free_bytes`), warn at <1 GiB (`collection.warn_free_bytes`). Conservative overhead estimate = 8 MiB × number of namespaces.

**Exit codes (see §3):** 0 on pass (warnings allowed), 1 on config/disk/RBAC failure, 2 on Kubernetes-API failure.

## 13. `groot inspect <archive>` — archive inventory (0.9.x #31)

```
groot inspect <archive.tar.gz> [--output text|json] [--max-decompressed <bytes>]
```

Inventories an existing `.tar.gz` produced by `groot collect`. Does **not** require cluster access. Reads `extras/manifest.json` when present, lists all files with sizes, and reports the archive size and file count.

| Flag | Default | Effect |
|------|---------|--------|
| `--output <text\|json>` | `text` | Human inventory or JSON `InspectInfo`. |
| `--max-decompressed <bytes>` | `0` (= default cap) | Override total decompressed-byte cap for Pass-1 index (see §13.1). |

**Exit codes (see §3):** 0 success, 3 archive open/read failure (including safety-cap rejection). Missing/unparseable manifest is non-fatal inventory degrade (reported in output; does not force exit 1 in current CLI).

### 13.1 Offline archive reader caps (`arcread`)

`inspect` and `analyze` open archives through the shared offline reader. Pass-1 indexes the `.tar.gz` **without extracting to disk** and fails closed on hostile or oversized input.

| Cap | Default | Notes |
|-----|---------|-------|
| Max member size | **64 MiB** | Single regular-file member. |
| Max regular files | **100_000** | Count of regular-file members. |
| Max decompressed total | **16 GiB** | Sum of member bodies drained during Pass-1 index. |

Exceeding a cap → open error → CLI exit **3**. Operators may raise only the decompressed total via `--max-decompressed` on `inspect` / `analyze` (raw byte count; `0` keeps the default). Path traversal, absolute member names, and symlink/hardlink members are rejected.

## 14. Config profiles and examples (0.9.x #86, 1.1.x #46)

Ready-to-use YAML files for common scenarios live in [`examples/`](examples/) (index: [`examples/README.md`](examples/README.md)).

**Profiles** ([`examples/profiles/`](examples/profiles/)):

- **incident-quick.yml** — narrow namespaces, `--since 1h`, events + failing pods, no notify/upload.
- **compliance-full.yml** — all namespaces, full logs, redaction enabled, metrics on.
- **bastion-airgap.yml** — SFTP upload via SSH relay, no external webhooks.
- **eks-managed.yml** — skip node logs (not supported on managed control planes), metrics enabled.
- **gke-managed.yml** / **aks-managed.yml** — same managed-node posture as EKS (no host node logs; metrics + redact).

**Beyond profiles (#46):**

- [`examples/notify/`](examples/notify/) — channel smokes for `groot notify test` (Slack, Teams, generic webhook, PagerDuty, Mailgun SMTP).
- [`examples/upload/`](examples/upload/) — S3 / GCS / SFTP post-collect skeletons (credentials via env).
- [`examples/collection/`](examples/collection/) — `targets` + `extra_kubectl`, redaction patterns.

Copy a profile as a starting point (`cp examples/profiles/incident-quick.yml groot.yml`) and edit to match your cluster. Secrets stay in env — never commit webhook URLs or keys.

## 15. `groot notify test` — channel smoke test (1.0.3 #95)

```
groot notify test [--config <path>] [--event notify.test|success|failure]
```

Loads `notify.*` from config and posts a **synthetic** `collector.Summary` to every enabled channel. Does **not** run collect, write an archive, or contact the Kubernetes API. Ignores `--no-notify` / `GROOT_NO_NOTIFY` (the command exists solely to exercise notify).

| Flag | Default | Effect |
|------|---------|--------|
| `--config <path>` | (search order §4) | YAML with at least one enabled notify destination. |
| `--event <name>` | `notify.test` | Payload style: `notify.test` (dedicated test line), `success` (post-collect success text), `failure` (failure alert with simulated reason). |

**Exit codes (see §3):** 0 on delivery success; **1** when config is invalid, no channel is enabled, or `--event` is unknown; **4** when any enabled channel returns a delivery error.

**Operator use:** verify Slack/Teams/webhooks, Mailgun SMTP (`GROOT_NOTIFY_EMAIL_*`), PagerDuty routing keys, etc., before scheduling cron/Helm collects. Step-by-step Mailgun/SMTP runbook: [docs/notify-smoke-test.md](docs/notify-smoke-test.md); sample config: [examples/notify/mailgun-smoke.yml](examples/notify/mailgun-smoke.yml).

## 16. `groot analyze <archive>` — offline heuristic hints (1.1.x #69)

```
groot analyze <archive.tar.gz> [--output text|json|llm] [--max-decompressed <bytes>]
```

Opens a local `groot collect` archive and emits evidence-backed heuristic **hints** (CrashLoopBackOff, OOMKilled, ImagePullBackOff, NotReady, Evicted when supported by archive evidence). Does **not** require kubeconfig or cluster API access. Findings are hypotheses from offline evidence — not a definitive root-cause diagnosis.

| Flag | Default | Effect |
|------|---------|--------|
| `--output <text\|json\|llm>` | `text` | Executive Markdown, structured Report JSON, or budgeted LLM-ready paste Markdown. |
| `--max-decompressed <bytes>` | `0` (= default cap) | Override Pass-1 decompressed total (see §13.1). |

**Output notes:**

- `text` — executive Markdown; includes `run_id` / `archive_sha256` when present in manifest.
- `json` — same findings model (`Report`) as Markdown renderers.
- `llm` — paste pack with assistant instructions, ranked findings, head/tail omit markers under a **32 KiB** byte budget, and an explicit secrets warning (collect-time redaction may not cover all cited member types).

**Analyze-local member read caps** (after open; optional members degrade with Notes, not exit 3):

| Member class | Cap |
|--------------|-----|
| Cluster/warning events logs | **2 MiB** |
| TSV / `*/resources.txt` / pods-wide text | **32 MiB** |

**Golden fixture corpus:** The committed source-tree corpus under `testing/fixtures/archives/` covers healthy, CrashLoopBackOff, OOMKilled, ImagePullBackOff, and missing-manifest degrade scenarios. CI golden tests lock executive and LLM Markdown output for these fixtures so heuristic or rendering regressions fail the build.

**Exit codes (see §3):** 0 success (including zero hints / healthy empty summary); 3 archive open/read failure (including §13.1 cap rejection). Invalid `--output` is an ordinary error (not exit 3).

