# groot roadmap

This file is the **in-repo** source of truth for **planned** work and known gaps. It complements:

- **[SPECIFICATIONS.md](SPECIFICATIONS.md)** — behavior contract, config shape, and what **`groot collect`** does **today**
- **[CHANGELOG.md](CHANGELOG.md)** — what shipped in each release

User-facing overview: **[README.md](README.md)** and **[configs/groot.yml.sample](configs/groot.yml.sample)**.

When a roadmap item ships, update **CHANGELOG** (reference **`(band #N)`** in bullets) and mark the item **Done** here—or move highlights into the **Shipped** table.

**Last reviewed:** 2026-06-29 (Band **0.9.x** closed at **v0.9.2**; **Band 3** active — [plan-1.0.0.md](docs/plan-1.0.0.md))

### Versioning note

Public releases start at **v0.1.3** (early CLI and packaging). **`v0.3.0` was never published**; treat **[0.3.0]** changelog material as shipped from **`v0.3.1`** onward.

Bands group semver minors into planning horizons — **Band 1** (shipped, 0.1.x–0.6.x), **Band 2** (shipped, 0.7.x), **Band 0.8.x** (RCA depth, shipped), **Band 0.9.x** (shipped, path to 1.0), **Band 3** (1.0.0 contract freeze, **active**), **Band 4** (1.1.x+ backlog). Individual items carry **global IDs** (#1–#N) stable across bands.

### Strategic direction

GROOT is a **read-only log and context collector**: one **`groot collect`** produces a **timestamped `.tar.gz`** for incident response and RCA. The runtime path is **client-go** end-to-end (no `kubectl` binary). Configuration is **YAML + env**; optional **notify** fan-out fires after a **successful** collect and, when configured, on **abort** or **partial job failure** (#19).

**Product positioning:** groot = **ticket-ready bundle** (`.tar.gz` + manifest + RCA TSVs + notify/upload). Tools like [kubectl-gather](https://github.com/nirs/kubectl-gather) optimize for **multi-cluster YAML trees and manual diff** — complementary, not identical (#60).

**Target architecture:** **0.9.x** ships operator wins (validate, inspect, kubectl plugin, summary, collector fixes). **1.0.0** freezes **config + archive layout** and moves code to **`internal/`**. **1.1.x+** adds multi-cluster, analyze, streaming, and addons — only after the contract is stable.

**Honest gaps today:**

- **No `config_version`** or formal migration path (#30 — blocks **1.0.0** only).
- **No structured output** (`--output json`) (#40 — **1.0.0**).

**Current focus (planned work):**

| Band | Status | Items |
|------|--------|-------|
| **Band 1** (0.1.x–0.6.x) | **Shipped** (#1–#29) | Initial CLI through BSD ports; see [plan-0.4.0](docs/plan-0.4.0.md), [plan-0.5.0](docs/plan-0.5.0.md), [plan-0.6.0](docs/plan-0.6.0.md) |
| **Band 2** (0.7.x) | **Shipped** (#36–#38) | SFTP upload, airgapped relay, container image SBOM; see [plan-0.7.0.md](docs/plan-0.7.0.md) |
| **Band 0.8.x** | **Shipped** (#39) | Workload requests/limits in RCA extras — **`v0.8.0`** |
| **Band 0.9.x** | **Shipped (v0.9.2)** | Operator wins — see [plan-0.9.0.md](docs/plan-0.9.0.md) (#31, #42, #60, #64, #79–#86) |
| **Band 3** (1.0.0) | **Active** | Contract freeze — see [plan-1.0.0.md](docs/plan-1.0.0.md) (#30, #34, #35, #40, #48, #87) |
| **Band 4** (1.1.x+) | **Backlog** (#32–#33, #41–#78, #50–#76) | Multi-cluster, analyze, stream, addons, distribution — post-1.0 |

---

## Shipped

| Release | Band | Highlights |
|---------|------|------------|
| **0.1.3 – 0.2.1** | 1 | Early **groot collect**, YAML config, parallel collection, archive output, Slack/Teams/webhook notify, rootless container, GoReleaser **.deb/.rpm/.tar.gz**, CI and security scans. |
| **0.3.1** | 1 | **Client-go** collector (no `kubectl` binary); **`k8srunner`** allowlist; **`kubetest`** fake API; per-namespace **JSON `resources.txt`**; **RCA TSV** extras; **`extra_kubectl`** validation; kind **E2E** harness (`make test-e2e-kind`); README/VHS demo; config sample load fix. Ships unpublished **0.3.0** work—see [CHANGELOG](CHANGELOG.md). |
| **0.3.2** | 1 | **GO-2026-5026** / **CVE-2026-39821** (`golang.org/x/net` → v0.55.0); Go **1.26.4**. |
| **0.4.0** | 1 | **Archive manifest** `extras/manifest.json` with version, cluster, jobs, paths. **`file_prefix`** now drives capture dir and archive basename. **`groot collect --list-jobs`** prints planned jobs without writing output. **`extra_kubectl` `get`/`describe`** supports `configmap`/`cm`, `pvc`, `service`/`svc`, `ingress`/`ing`, `deployment`/`rs`/`sts`/`ds` and aliases. **`collection.targets`** accepts `jobs` / `cronjobs` lists matched by `job-name` and standard labels. **Kind E2E in CI** (`make test-e2e-kind`) running with `continue-on-error: true`. Docs hygiene: `pkg/config/sample.go` and `configs/groot.yml.sample` in sync, references to `kubectl` removed. |
| **0.4.1** | 1 | **`Dockerfile`** builder **Go 1.26.4** (Security Grype image build); README download examples for **v0.4.1**. |
| **0.5.0** | 1 | **Notify on failure** (`notify.on_failure`: abort + partial-failure threshold). **Rich generic webhooks** (`extra_fields`, `body_template`, HMAC). **Email/SMTP** channel. **HTTP notify retry/backoff**. **Optional log redaction** (`collection.redact_secrets`). **In-cluster deploy**: Helm chart (`deploy/helm/groot/`) and flat CronJob manifests (`deploy/k8s/`). |
| **0.6.0** | 1 | **Homebrew cask** tap + `groot_vX.Y.Z_*` release basenames. **SBOM** (SPDX + CycloneDX). **Cosign** keyless signing (checksums + images). **Post-collect S3/GCS upload**. **FreeBSD + OpenBSD** ports with release CI. |
| **0.6.1** | 1 | **Product vs operator split**: Helm/CronJob manifests and deploy runbooks moved to **[groot-selfhosted](https://github.com/hrodrig/groot-selfhosted)**; README/SPEC/AGENTS clarify product scope (docs only). |
| **0.7.0** | 2 | **SFTP post-collect upload** (`upload.sftp`): bastion → SSH relay via public-key SFTP (`--no-upload` honored). **Airgapped relay playbook** (groot-selfhosted): bastion → SFTP → rclone → OneDrive topology with systemd watcher. **Container image SBOM**: OCI attestation enabled on `dockers_v2` (`sbom: true` via buildx docker-container driver). |
| **0.7.1** | 2 | **`cluster_name` config**: optional archive basename cluster label; resolution chain when empty (kubeconfig → cluster-info → API server host). |
| **0.7.2** | 2 | **Node log capture**: host logs via node proxy → `nodes/<node>.log`; kubelet log query optional; fewer spurious failures on managed clouds (e.g. AKS). |
| **0.8.0** | 0.8 | **Workload resource RCA**: `extras/workload-resources.tsv` (per-container requests/limits + owner); **`all-pods-rca.tsv`** adds declared resource columns for OOM/capacity RCA (#39). |
| **0.9.0 – 0.9.2** | 0.9 | **Operator wins**: Helm fix, kubectl plugin, validate/inspect, `--summary`, `run_id`, exit codes, completion, profiles, README comparison (#79–#86, #64, #60, #31). **v0.9.1**/**v0.9.2**: Windows/OpenBSD GoReleaser build fixes. See [plan-0.9.0.md](docs/plan-0.9.0.md). |

---

## Band 0.8.x — RCA depth (collector fidelity)

| # | Band | Item | Status |
|---|------|------|--------|
| 39 | 0.8 | **Workload requests/limits in RCA extras**: `extras/workload-resources.tsv` (per-container CPU/memory requests and limits, owner kind/name); **`all-pods-rca.tsv`** merges pod-level declared resources with metrics and log paths. | **Done (v0.8.0)** |

**Out of scope for 0.8.x:** continuous metrics agent (time-series CSV belongs outside groot); mutating cluster operations.

---

## Band 0.9.x (shipped) — path to 1.0

**Implementation plan:** [plan-0.9.0.md](docs/plan-0.9.0.md) (closed at **`v0.9.2`**). Merge order: **#79 → #64+#85 → #80 → #81 → #82 → #31+#83 → #42+#84 → #86 → #60**.

Ship **operator value** before the **1.0.0** compatibility promise. No `config_version` or `internal/` in this band.

| # | Band | Item | Status |
|---|------|------|--------|
| 79 | 0.9 | **Fix Helm `helm_releases` target matching** + unit tests for `matchesTargetsByLabels` / `resolvePodsForLogs`. | **Done (v0.9.0)** |
| 64 | 0.9 | **kubectl plugin**: ship **`kubectl-groot`** binary; `kubectl groot collect` via plugin discovery. | **Done (v0.9.0)** |
| 85 | 0.9 | **Krew + Makefile**: `make install-kubectl-plugin`; document Krew index submission. | **Done (v0.9.0)** |
| 80 | 0.9 | **Shell completion**: `groot completion bash\|zsh\|fish\|powershell` (and `kubectl groot completion`). | **Done (v0.9.0)** |
| 81 | 0.9 | **`run_id` + `archive_sha256`**: in `manifest.json`, notify text, upload metadata. | **Done (v0.9.0)** |
| 82 | 0.9 | **Exit code taxonomy**: documented codes for config, API, abort, notify (see plan-0.9.0). | **Done (v0.9.0)** |
| 31 | 0.9 | **`groot validate`** (config, API, RBAC, disk) and **`groot inspect <archive>`** (manifest + file tree minimum). | **Done (v0.9.0)** |
| 83 | 0.9 | **Disk space preflight** (part of validate): fail/warn when `output_dir` lacks space. | **Done (v0.9.0)** |
| 42 | 0.9 | **`groot collect --summary`**: one-screen result for operators. | **Done (v0.9.0)** |
| 84 | 0.9 | **Signal-first job ordering**: events Warning+ and unhealthy pods before bulk logs. | **Done (v0.9.0)** |
| 86 | 0.9 | **Config profiles** in `examples/profiles/` (incident-quick, bastion-airgap, eks-managed, compliance-full). | **Done (v0.9.0)** |
| 60 | 0.9 | **README comparison** vs kubectl-gather / alternatives; incident workflow example. | **Done (v0.9.0)** |

**Out of scope for 0.9.x:** `config_version`, `internal/` refactor, multi-cluster, stream, analyze, progress bar, smart rules, web dashboard.

---

## Band 3 (1.0.0) — stable contract only

**Target:** **v1.0.0**. Major when config shape, archive layout, and module layout are stable.

**Implementation plan:** [plan-1.0.0.md](docs/plan-1.0.0.md). Merge order: **#30 → #34 → #35 → #40 → #87 → #48**.

**Prerequisite met:** **v0.9.2** shipped (2026-06-29).

| # | Band | Item | Status |
|---|------|------|--------|
| 30 | 3 | **`config_version`** in YAML with documented migration notes. | Pending |
| 34 | 3 | **`archive_layout_version`** in `extras/manifest.json` for downstream RCA tooling. | Pending |
| 35 | 3 | **`pkg/` → `internal/` layout** (kzero/pgwd parity); no user-facing CLI change. | Pending |
| 40 | 3 | **`--output json` / `yaml`**: Summary for collect; checks for validate; manifest tree for inspect. | Pending |
| 87 | 3 | **Golden archive fixtures** for inspect (and future analyze) regression tests. | Pending |
| 48 | 3 | **CODEOWNERS** and GitHub issue/PR templates. | Pending |

**Explicitly deferred past 1.0.0:** multi-cluster (#32), CI matrix (#33), stream (#41), analyze (#69), progress bar (#44), smart rules (#71), addon system (#65), embedded dashboard (#74). See **Band 4**.

---

## Band 4 (1.1.x+) — backlog (post-1.0)

Items from community review **not** blocking **1.0.0**. Prioritize after [plan-1.0.0.md](docs/plan-1.0.0.md) ships.

### Platform depth (high value post-1.0)

| # | Band | Item | Status |
|---|------|------|--------|
| 32 | 4 | **Multi-cluster collect** into one archive (multiple contexts, prefixed paths). | Pending |
| 33 | 4 | **CI matrix**: kind E2E against more than one Kubernetes minor; documented minimum cluster version. | Pending |
| 41 | 4 | **`groot stream`**: live pod log tailing with `collection.targets` filters. | Pending |
| 43 | 4 | **Auto-detect kubeconfig / `--context`**: multi-path `KUBECONFIG`, context selection. | Pending |
| 69 | 4 | **`groot analyze <archive>`**: heuristics (OOM, CrashLoop, trends) + executive `.md` summary. | Pending |
| 56 | 4 | **`groot diff`**: compare two archives from the same cluster. | Pending |
| 70 | 4 | **Dry-run size estimation**: extend `--list-jobs` with storage footprint estimate. | Pending |
| 71 | 4 | **Smart collect rules**: conditional `collection.rules` in YAML. | Pending |
| 72 | 4 | **Incremental collection**: `--incremental` with state in `~/.groot/state/`. | Pending |

### Collector and UX polish

| # | Band | Item | Status |
|---|------|------|--------|
| 44 | 4 | **Progress bar / spinner** during collect. | Pending |
| 45 | 4 | **Secret redaction enhancements**: JWT/AWS patterns, `--dry-run-redact`, extend to `.txt`/`.tsv`. | Pending |
| 66 | 4 | **Label selector flag (`-l` / `--selector`)**. | Pending |
| 67 | 4 | **Dynamic API rate limiting** (429 backoff). | Pending |
| 68 | 4 | **Per-log `LimitBytes`**. | Pending |
| 77 | 4 | **`--target` CLI flag** for ad-hoc filtering. | Pending |
| 78 | 4 | **`collection.events_min_type`** (Warning+ filter). | Pending |
| 54 | 4 | **TUI mode** (`groot tui`). | Pending |
| 55 | 4 | **Continuous watcher** (`groot watch`). | Pending |
| 57 | 4 | **`--compress-level`**, `--quick` lite mode. | Pending |
| 58 | 4 | **Edge/K3s/MicroK8s** adjustments. | Pending |

### Distribution, extensibility, and community

| # | Band | Item | Status |
|---|------|------|--------|
| 50 | 4 | **Additional package managers**: Scoop, Nix, Chocolatey, Snap. | Pending |
| 51 | 4 | **Grafana / self-hosted dashboard** (groot-selfhosted). | Pending |
| 52 | 4 | **Post-collect YAML hooks**. | Pending |
| 53 | 4 | **Observability export** (Prometheus, Loki/Promtail detection). | Pending |
| 65 | 4 | **Addon system** (kubectl-gather-style; after multi-cluster). | Pending |
| 46 | 4 | **Generic `examples/`** beyond profiles (#86). | Pending |
| 47 | 4 | **CONTRIBUTING.md** collector guide. | Pending |
| 49 | 4 | **Promote kind E2E** from `continue-on-error` to required/nightly. | Pending |
| 59 | 4 | **E2E matrix expansion** (minikube, OpenShift local). | Pending |
| 61 | 4 | **Ticketing integration** (Jira/GitHub Issues drafts). | Pending |
| 62 | 4 | **Post-collect analysis hooks** (Popeye, kubectl-debug). | Pending |
| 63 | 4 | **Community growth** (Reddit, CNCF Slack, articles). | Pending |
| 73 | 4 | **Archive encryption/signing** (`--encrypt`, `--sign`). | Pending |
| 74 | 4 | **Embedded web dashboard** (`groot serve`) — prefer groot-selfhosted + Grafana (#51). | Pending |
| 75 | 4 | **Alternate formats** (SQLite, Parquet export). | Pending |
| 76 | 4 | **`groot cleanup`** retention policy for `output_dir`. | Pending |

**Out of scope (long-term):** mutating cluster operations; full OpenTelemetry agent; managed SaaS; native Windows GUI.

---

## Maintenance notes

- **Release cadence:** security patches ship promptly; feature bands should earn changelog entries, README/demo refresh, and `make release-check` before tag.
- **Triad sync:** on each release—**ROADMAP** (Done + Shipped), **CHANGELOG** (`(band #N)` references), **VERSION** / README badges.
- **Large bands:** when a band spans multiple PRs, add **`docs/plan-X.Y.Z.md`** (merge order, success criteria, release checklist)—see [kzero plan-0.6.0](https://github.com/hrodrig/kzero/blob/develop/docs/plan-0.6.0.md) as reference.
- **Security:** run **`make security`** before tag; keep `govulncheck` / **grype** green; triage CodeQL on the Security tab.
- **Coverage:** merged statement gate **80%** (`COVER_MIN`); `coverage.out` is gitignored—do not commit.
- **client-go / k8s.io pin:** bump `k8s.io/*` modules together; document tested cluster minor in README when matrix lands (#33).
- **E2E:** local kind test remains optional for contributors; CI job (#17) uses `continue-on-error` until flake budget stabilizes (#49 in Band 4 backlog).
