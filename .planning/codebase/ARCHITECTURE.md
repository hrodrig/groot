<!-- refreshed: 2026-08-10 -->
<!-- last_mapped_commit: 805eba0 -->
# Architecture

**Analysis Date:** 2026-08-10

## System Overview

```text
┌─────────────────────────────────────────────────────────────┐
│  CLI boundary (thin main + Cobra)                           │
│  `cmd/groot/main.go`  →  `internal/cmd`                     │
│  Commands: collect | validate | inspect | notify | version  │
│            completion | (root: --print-sample-config,       │
│                          --test-connection)                 │
├──────────────────┬──────────────────┬───────────────────────┤
│  Config          │  Collector       │  Side effects         │
│  `internal/      │  engine          │  after collect        │
│   config`        │  `internal/      │  `internal/notifier`  │
│  (Viper/YAML)    │   collector`     │  `internal/uploader`  │
└────────┬─────────┴────────┬─────────┴──────────┬────────────┘
         │                  │                     │
         ▼                  ▼                     ▼
┌─────────────────────────────────────────────────────────────┐
│  Kubernetes access (no kubectl binary)                       │
│  `internal/kubeloader` → rest.Config                         │
│  `internal/k8srunner`  → allowlisted get/logs/describe/...   │
│  client-go + metrics client                                  │
└─────────────────────────────────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────────────────────────────┐
│  Local artifacts                                             │
│  Capture dir → `internal/archive` (.tar.gz)                  │
│  `extras/manifest.json` (ArchiveLayoutVersion = 1)           │
│  Optional: S3 / GCS / SFTP upload; webhook/SMTP/PD notify    │
└─────────────────────────────────────────────────────────────┘
```

**Out of this repo:** operator deploy (Docker/Podman, Helm CronJob, systemd/cron) lives in companion **[groot-selfhosted](https://github.com/hrodrig/groot-selfhosted)**. Do not add charts or bastion runbooks here (`AGENTS.md`).

**Not implemented:** `groot analyze` (ROADMAP **#69**) — no Go command or package. Offline heuristics are Band 4; today use `groot inspect` for archive listing + manifest only.

## Component Responsibilities

| Component | Responsibility | File |
|-----------|----------------|------|
| Process entry | Signals, `os.Exit`, build ldflags → `cmd.SetBuildInfo` | `cmd/groot/main.go` |
| Cobra root / collect | Flags, orchestration of load → collect → notify → upload → exit codes | `internal/cmd/root.go` |
| Validate / inspect CLI | Preflight without writing; offline archive inspect | `internal/cmd/validate.go` |
| Exit taxonomy | Stable codes 0–5 via `*ExitError` | `internal/cmd/exitcode.go` |
| kubectl plugin detect | Basename `kubectl-groot` / env force | `internal/cmd/plugin.go` |
| Config load | YAML + env (`GROOT_*`), `config_version: 1` | `internal/config/config.go` |
| Collector `Service` | Job plan, workers, redact, manifest, archive finalize | `internal/collector/collector.go` |
| Preflight | Config/cluster/RBAC/disk checks (validate) | `internal/collector/preflight.go` |
| Inspect | Read `.tar.gz`, list files, extract manifest | `internal/collector/inspect.go` |
| k8srunner | Allowlisted kubectl-shaped argv via client-go | `internal/k8srunner/runner.go` |
| kubeloader | Kubeconfig → `rest.Config` | `internal/kubeloader/loader.go` |
| Archive | Directory → `.tar.gz` with session prefix | `internal/archive/targz.go` |
| Notifier FanOut | Slack/Discord/Teams/Telegram/generic/email/PagerDuty | `internal/notifier/notifier.go` |
| Uploader FanOut | S3 / GCS / SFTP | `internal/uploader/uploader.go` |
| Console log | Colored INFO/WARN/CMD/OK/Error | `internal/logx/logx.go` |
| Fake K8s API | httptest for unit/integration tests | `internal/kubetest/server.go` |

## Pattern Overview

**Overall:** Layered CLI — thin `main` + Cobra adapter over a collector domain service; Kubernetes accessed only through allowlisted runner; post-run fan-out for notify/upload.

**Key Characteristics:**
- **Evidence archive first** — collect packs read-only cluster context; no online diagnosis (`SPECIFICATIONS.md`).
- **No kubectl binary** — `extra_kubectl` and built-in jobs are argv slices executed by `k8srunner.Runner.Run`.
- **Worker pool** — `Collection.WorkerConcurrency` goroutines drain a job channel (`runJobs` in `internal/collector/collector.go`).
- **Stable exit categories** — scripts branch on codes from `internal/cmd/exitcode.go`, not stderr scraping.
- **Contract docs at repo root** — `SPECIFICATIONS.md` (behavior), `ROADMAP.md` (planned), `CHANGELOG.md` (shipped).

## Layers

**CLI / adapter (`internal/cmd`):**
- Purpose: Parse flags, load config, map domain results to stdout/stderr and exit codes.
- Location: `internal/cmd/`
- Contains: Cobra commands, render helpers (`renderValidate`, `renderInspect`), plugin/completion/version.
- Depends on: `config`, `collector`, `notifier`, `uploader`, `logx`, `kubeloader`.
- Used by: `cmd/groot/main.go` only.

**Configuration (`internal/config`):**
- Purpose: Typed `Config` from YAML (Viper) + env overrides; sample YAML.
- Location: `internal/config/`
- Contains: `Load`, `SampleYAML`, pod_logs_since normalization, upload/notify structs.
- Depends on: Viper, stdlib.
- Used by: All commands that need runtime settings.

**Collector domain (`internal/collector`):**
- Purpose: Plan jobs, execute to capture dir, redact, write tables/manifest, finalize archive; preflight; inspect.
- Location: `internal/collector/`
- Contains: `Service`, jobs, summary, manifest, inspect, preflight, redaction, unhealthy counts.
- Depends on: `config`, `k8srunner`, `kubeloader`, `archive`, client-go.
- Used by: `internal/cmd` (collect/validate/inspect paths).

**Kubernetes execution (`internal/k8srunner` + `internal/kubeloader`):**
- Purpose: Build clients; execute allowlisted read-only operations.
- Location: `internal/k8srunner/`, `internal/kubeloader/`
- Contains: `Runner.Run` switch on get/describe/logs/top/auth/cluster-info/…
- Depends on: client-go, metrics clientset.
- Used by: `collector.Service` (and connection tests in `root.go`).

**Side-effect adapters (`internal/notifier`, `internal/uploader`):**
- Purpose: Fan-out after successful (or abort-failure) collect.
- Location: `internal/notifier/`, `internal/uploader/`
- Contains: `Sender` / `Uploader` interfaces; HTTP retry; S3/GCS/SFTP implementations.
- Depends on: `config`, `collector.Summary`.
- Used by: `collect` RunE in `internal/cmd/root.go`.

**Cross-cutting utilities:**
- `internal/archive` — packing only.
- `internal/logx` — human console (not structured observability).
- `internal/kubetest` — test doubles for API server.

## Data Flow

### Primary Request Path (`groot collect`)

1. `main` installs signal context and calls `cmd.ExecuteContext` (`cmd/groot/main.go`).
2. `collectCmd.RunE` loads config, applies `--since` / `--kubeconfig` overrides (`internal/cmd/root.go`).
3. `collector.New(cfg)` → `Service.Run(ctx)` creates session dir, `initK8s`, metadata, `buildJobs`, `runJobs` (`internal/collector/collector.go`).
4. Optional redact pass; workload/RCA tables; `writeManifest`; `archive.DirToTarGz`; remove capture dir.
5. On success: `notifier.NewFanOut(cfg).Notify`; optional failure notify; `uploader.NewFanOut(cfg).Upload`.
6. Optional `--summary` / `--output json`; `--strict` may return exit 5.
7. `main` maps error through `cmd.ExitCodeOf` → `os.Exit`.

### Preflight Path (`groot validate`)

1. Load config + disk flag overrides (`internal/cmd/validate.go`).
2. `collector.Service.Preflight(ctx)` — namespaces warn, kube client, cluster-info, RBAC matrix, disk free (`internal/collector/preflight.go`).
3. Render text/JSON; exit 1 (config/disk/RBAC) or 2 (kubernetes.*) on errors.

### Offline Inspect Path (`groot inspect <archive.tar.gz>`)

1. `collector.InspectArchive(path)` opens gzip/tar, lists files, reads `extras/manifest.json` (`internal/collector/inspect.go`).
2. Render text/JSON — **no cluster access**.
3. `groot analyze` is **not** wired (ROADMAP #69).

### Connection Test Path

1. Root or collect with `--test-connection`: `kubeloader.RESTConfig` → list namespaces → `ReadKubeMetadata` (`internal/cmd/root.go` `runConnectionTest`).

**State Management:**
- Per-run state on `collector.Service` (`RunID`, hooks, clientset, archive SHA).
- Package-level Cobra flag vars in `internal/cmd` (reset via `ResetPersistentCLI` for tests).
- No long-lived daemon or in-process cache across invocations.

## Key Abstractions

**`collector.Service`:**
- Purpose: Own one collect/preflight lifecycle against a `config.Config`.
- Examples: `internal/collector/collector.go`, `preflight.go`
- Pattern: Constructor + setters (`SetHooks`, `SetBuildInfo`, `SetMessage`) then `Run` / `Preflight`.

**`job` (internal to collector):**
- Purpose: Named argv + output relative path + optional flag.
- Examples: `buildJobs`, `baseKubectlJobs` in `internal/collector/collector.go`
- Pattern: Plan then execute; optional jobs do not abort the whole run the same way as mandatory failures (see SPEC for abort rules).

**`k8srunner.Runner`:**
- Purpose: Single allowlist for “kubectl-shaped” reads without spawning kubectl.
- Examples: `internal/k8srunner/runner.go`
- Pattern: `Run(ctx, argv []string) ([]byte, error)` dispatch.

**Fan-out (`notifier.FanOut`, `uploader.FanOut`):**
- Purpose: N destinations from config; continue-on-error for upload when configured.
- Examples: `internal/notifier/notifier.go`, `internal/uploader/uploader.go`
- Pattern: Interface slice (`Sender`, `Uploader`) built in `NewFanOut`.

**`cmd.ExitError`:**
- Purpose: Carry numeric exit code through Cobra `RunE` error chain.
- Examples: `internal/cmd/exitcode.go`
- Pattern: Return `NewExitError(code, err)` from handlers; `ExitCodeOf` in `main`.

**Archive layout contract:**
- Purpose: Versioned capture layout (`ArchiveLayoutVersion = 1`) + `extras/manifest.json`.
- Examples: `internal/collector/manifest.go`, `archive/targz.go`
- Pattern: Session folder prefix inside tar; inspect/golden tests lock the shape.

## Entry Points

**Binary `groot` / `kubectl-groot`:**
- Location: `cmd/groot/main.go`
- Triggers: User shell, cron/CI, kubectl plugin discovery (same binary basename).
- Responsibilities: Signal handling, execute Cobra, map exit codes.

**Cobra command tree:**
- Location: registered in `internal/cmd/root.go` `init()`
- Triggers: `groot <subcommand>`; plugin strips `groot` token so `kubectl groot collect` → argv `kubectl-groot collect …`.
- Responsibilities: User-facing surface — extend here, keep domain in packages.

**Product E2E (kind):**
- Location: `testing/scripts/test-e2e-kind.sh`, `testing/k8s/e2e-workload.yaml`
- Triggers: Manual / CI helpers documented in `docs/e2e-kind.md`
- Responsibilities: Cluster-level smoke; not the unit-test fake API.

## Architectural Constraints

- **Threading:** Concurrent job workers (`WorkerConcurrency`); hook callbacks serialized with `hooksMu`. Do not assume single-threaded filesystem writes without job-level path uniqueness.
- **Global state:** Cobra package vars and `rootCmd` in `internal/cmd`; test helpers must call `ResetPersistentCLI`. Telegram API base override var in notifier for tests.
- **Circular imports:** Avoid `collector` importing `internal/cmd`. Notifier/uploader may import `collector` for `Summary` only — keep CLI rendering in `cmd`.
- **No kubectl binary dependency:** New collection features must extend `k8srunner` allowlist or dedicated client-go helpers, not `exec.Command("kubectl", …)`.
- **Deploy scope:** Helm/cron/bastion → **groot-selfhosted**, not this tree.
- **Analyze gap:** Do not invent `internal/analyzer` or `analyze` Cobra command until ROADMAP #69 is planned/executed; extend `inspect` only for listing/manifest needs.

## Anti-Patterns

### Spawning kubectl for collection

**What happens:** Calling out to a `kubectl` binary for jobs or `extra_kubectl`.
**Why it's wrong:** Product contract is client-go only; breaks air-gapped/minimal images.
**Do this instead:** Add a case in `k8srunner.Runner.Run` (`internal/k8srunner/runner.go`) or a focused helper used by the collector.

### Putting domain logic in `cmd/groot` or thick Cobra handlers

**What happens:** Large business rules inside `RunE` or a new file under `cmd/`.
**Why it's wrong:** Breaks testability and the thin-main layout; duplicates paths for plugin/standalone.
**Do this instead:** Implement in `internal/collector` (or a new `internal/<pkg>`), call from `internal/cmd`.

### Shipping deploy manifests in this repo

**What happens:** Adding Helm charts or CronJob YAML under `groot`.
**Why it's wrong:** Violates repo split in `AGENTS.md`.
**Do this instead:** Propose changes in **groot-selfhosted**; keep product image/binary release here (GoReleaser / `Dockerfile`).

### Implementing “analyze” as ad-hoc inspect flags

**What happens:** Encoding OOM/CrashLoop heuristics into `InspectArchive` without a #69 plan.
**Why it's wrong:** Blurs inspect (inventory) vs analyze (heuristics); ROADMAP defers #69.
**Do this instead:** Keep inspect inventory-only; plan #69 as a separate command that reuses golden fixtures (`internal/collector/inspect_golden_test.go` pattern).

## Error Handling

**Strategy:** Wrap with `%w`; classify at CLI boundary with `NewExitError` / `NewExitErrorf`.

**Patterns:**
- Config / flag / YAML → exit **1** (`ExitConfigError`).
- Kubernetes client/API → exit **2** (`ExitKubernetesError`).
- Collect abort / inspect open failure → exit **3** (`ExitCollectAborted`).
- Notify failure after successful collect → exit **4** (`ExitNotifyFailed`).
- `--strict` partial failures → exit **5** (`ExitPartialFailed`).
- Preflight: map finding check prefix `kubernetes.` to exit 2, else 1 (`internal/cmd/validate.go`).

## Cross-Cutting Concerns

**Logging:** `internal/logx` for human console; `--quiet` suppresses INFO/WARN/CMD/OK (not notify). Verbose enables CMD lines via collect hooks.

**Validation:** Config load + `Preflight` for operator readiness; `config_version` must be 1 (`SupportedConfigVersion`).

**Authentication:** Kubernetes via kubeconfig / in-cluster rules in `kubeloader`; notify/upload use webhook URLs, SMTP, AWS/GCP/SFTP credentials from config/env — never commit secrets (`.env` forbidden for agents).

**Redaction:** Optional post-collect pass when `collection.redact_secrets` (`internal/collector/redact.go`).

---

*Architecture analysis: 2026-08-10*
