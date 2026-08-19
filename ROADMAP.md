# groot roadmap

This file is the **in-repo** source of truth for **planned** work and known gaps.

> **ROADMAP** = *what* we build and *when*. **[SPECIFICATIONS.md](SPECIFICATIONS.md)** = *how* each feature behaves today. Any roadmap item that changes observable behavior must land a SPEC update (or clear non-behavior note) **before** the release that ships it. **[CHANGELOG.md](CHANGELOG.md)** records what shipped.

User-facing overview: **[README.md](README.md)** and **[configs/groot.yml.sample](configs/groot.yml.sample)**.

When a roadmap item ships, update **CHANGELOG** (reference **`(band #N)`** in bullets) and mark the item **Done** here—or move highlights into the **Shipped** table.

**Last reviewed:** 2026-08-18 (**v1.1.2** security patch — Go 1.26.6 stdlib CVEs, golangci-lint v2.12.2; Band 4 continues with `#32` / `#56`; backlog **`#97`** WebDAV/Nextcloud)

### Versioning note

Public releases start at **v0.1.3** (early CLI and packaging). **`v0.3.0` was never published**; treat **[0.3.0]** changelog material as shipped from **`v0.3.1`** onward.

Bands group semver minors into planning horizons. Individual items carry **global IDs** (#1–#N) stable across bands.

### Band status (scan first)

| Band | Version | Focus | Status | Closed |
|------|---------|-------|--------|--------|
| **1** | 0.1.x–0.6.x | Core CLI, notify, packaging, S3/GCS, BSD | ✅ Shipped | **2026-06-06** (v0.6.1) |
| **2** | 0.7.x | SFTP, airgap relay, node logs, cluster_name | ✅ Shipped | **2026-06-17** (v0.7.2) |
| **0.8** | 0.8.x | RCA workload resources | ✅ Shipped | **2026-06-17** (v0.8.0) |
| **0.9** | 0.9.x | Operator wins (validate/inspect/plugin) | ✅ Shipped | **2026-06-29** (v0.9.2) |
| **3** | 1.0.0 | Contract freeze | ✅ Shipped | **2026-07-03** (v1.0.0) |
| **3 maint.** | 1.0.1–1.0.6 | Security + audit hygiene + docs/packaging | ✅ Shipped | **2026-08-04** (v1.0.6) |
| **4** | 1.1.x+ | Multi-cluster, analyze, triggered watch, addons | 📋 Active backlog | **2026-08-10** (`v1.1.0` = `#69` analyze) |

### Current focus (in flight)

| Priority | Item | Theme | Notes |
|----------|------|-------|-------|
| **Shipped** | **#69** `groot analyze <archive>` | Analysis | **Done (v1.1.0)** — offline heuristics + executive/LLM Markdown + goldens. |
| **Next** | **#32** multi-cluster collect | Multi-cluster | [plan-1.0.0.md](docs/plan-1.0.0.md) post-1.0 order; first open Band 4 theme after analyze. |
| **Next** | **#56** `groot diff` | Analysis | Shared `arcread` reader; natural follow-on to `#69`. |
| Then | **#43** kubeconfig / `--context` | Multi-cluster | Operator ergonomics. |
| Then | **#33** CI kind matrix | Platform | Documented minimum cluster version. |
| Later | **#65** addon system | Ecosystem | After multi-cluster + plugin maturity. |
| Later | **#97** WebDAV / Nextcloud upload | Ecosystem | Post-collect `upload.webdav` for Nextcloud (and generic WebDAV); complements S3/GCS/SFTP. FileZilla/SFTP VPS remains supported today without this. |
| Later | **#55** event-driven watch | Triggered collect | Critical services → collect → S3/GCS/SFTP + notify (not live log tail). |

Ship lock for analyze: **[`docs/plan-1.1.0.md`](docs/plan-1.1.0.md)**.

### Strategic direction

GROOT is a **read-only log and context collector**: one **`groot collect`** produces a **timestamped `.tar.gz`** for incident response and RCA. The runtime path is **client-go** end-to-end (no `kubectl` binary). Configuration is **YAML + env**; optional **notify** fan-out fires after a **successful** collect and, when configured, on **abort** or **partial job failure** (#19).

**Product positioning:** groot = **ticket-ready bundle** (`.tar.gz` + manifest + RCA TSVs + notify/upload). Scheduled / bastion runbooks: **[groot-selfhosted](https://github.com/hrodrig/groot-selfhosted)**. On-demand in-cluster collect: **[groot-trigger](https://github.com/hrodrig/groot-trigger)**. VPS archive catalog: **[groot-share](https://github.com/hrodrig/groot-share)** (**gfs**); deploy: **[groot-share-selfhosted](https://github.com/hrodrig/groot-share-selfhosted)**. Tools like [kubectl-gather](https://github.com/nirs/kubectl-gather) optimize for **multi-cluster YAML trees and manual diff** — complementary, not identical (#60).

**Target architecture:** **0.9.x** shipped operator wins. **1.0.0** froze **config + archive layout** and moved code to **`internal/`**. **1.1.x+** (Band 4) adds multi-cluster, analyze, triggered collect-on-signal, and addons — only after that contract.

### Known gaps / Non-goals

**Gaps (honest — still pending):**

- **`--output yaml`** for collect not implemented (#40 partial; JSON shipped).
- **Multi-cluster**, **analyze**, **triggered watch**, **progress bar** — Band 4 only (`stream` live tail is out of philosophy; see Non-goals).
- ~~**No `man(1)` / nfpm man install**~~ — shipped **v1.0.6** (#96 / GH #2).

**Non-goals (do not expect these as product core):**

- Not an **automatic cluster diagnosis** product — evidence archive first; **`groot analyze`** (#69) may add heuristics later, not a full SRE co-pilot.
- Not a **kubectl-gather** replacement — complementary workflows (#60).
- Not **live log streaming / tail** (`groot stream`) — outside product philosophy (use `kubectl logs -f`, stern, or a log shipper). Continuous value for groot is **triggered collect → archive → upload → notify**, not a log agent.
- Not **multi-cluster native** until Band 4 `#32`.
- Not **mutating** cluster operations, full OpenTelemetry agent, managed SaaS, or native Windows GUI (long-term out of scope).
- Not a long-lived **HTTP server** or **archive catalog** inside this CLI — on-demand collect is **[groot-trigger](https://github.com/hrodrig/groot-trigger)**; VPS door is **[groot-share](https://github.com/hrodrig/groot-share)** (**gfs**), deployed via **[groot-share-selfhosted](https://github.com/hrodrig/groot-share-selfhosted)**.

### Community signals (Band 4)

| Badge | Meaning |
|-------|---------|
| 🚀 | Maintainer priority |
| 🤝 | Help wanted (good first / docs / tests contribution) |
| 💡 | Design / RFC needed before code |
| ⏳ | Blocked on another item or external dependency |

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
| **1.0.0** | 3 | **Stable contract**: `config_version`, `archive_layout_version`, `internal/` layout, `collect --output json`, golden inspect test, governance templates (#30, #34, #35, #40, #48, #87). Pre-1.0 hygiene (notifier, SFTP fail-closed, SIGTERM). See [plan-1.0.0.md](docs/plan-1.0.0.md). |
| **1.0.1** | 3 | **Security patch**: Go **1.26.5** — **CVE-2026-39822**, **CVE-2026-42505**; `Dockerfile` builder image aligned with `go.mod`. |
| **1.0.2** | 3 | **Maintenance patch**: distroless **`static-debian13:nonroot`** runtime base (`Dockerfile`, `Dockerfile.release`). |
| **1.0.3** | 3 | **Post-audit hygiene**: Docker CMD `--help`, email/GCS test coverage, `groot notify test`, `x/crypto` v0.54.0, QPS docs (#88–#95). See [plan-1.0.3.md](docs/plan-1.0.3.md). |
| **1.0.4** | 3 | **Security patch**: `grpc` **v1.82.1** (GHSA-hrxh-6v49-42gf); OpenTelemetry **v1.44.0** (GO-2026-5158); `-v` Usage dump fix; kubectl-groot Gatekeeper docs. |
| **1.0.5** | 3 | **Docs**: Homebrew 6+ tap trust (`brew install --cask hrodrig/groot/groot`); README/BSD pins. |
| **1.0.6** | 3 | **Packaging/docs**: man(1)+nfpm+BSD (#96), CONTRIBUTING collector guide (#47), examples beyond profiles (#46). |
| **1.1.0** | 4 | **offline `groot analyze`** (#69): heuristics + executive/LLM Markdown + golden fixtures. |
| **1.1.1** | 4 | Patch: kubeconfig `~` expansion, unique `sessionBase` short, S3 credential trim. |
| **1.1.2** | 4 | **Security:** Go **1.26.6** stdlib CVEs (govulncheck); golangci-lint **v2.12.2**; family companion repo docs. |

---

## Band 0.8.x — RCA depth (collector fidelity)

**Closed:** **2026-06-17** at **`v0.8.0`**. Criterion: workload requests/limits in RCA extras (#39) in production archives.

| # | Band | Item | Status |
|---|------|------|--------|
| 39 | 0.8 | **Workload requests/limits in RCA extras**: `extras/workload-resources.tsv` (per-container CPU/memory requests and limits, owner kind/name); **`all-pods-rca.tsv`** merges pod-level declared resources with metrics and log paths. | **Done (v0.8.0)** |

**Out of scope for 0.8.x:** continuous metrics agent (time-series CSV belongs outside groot); mutating cluster operations.

---

## Band 0.9.x (shipped) — path to 1.0

**Closed:** **2026-06-29** at **`v0.9.2`**. Criterion: kubectl plugin, exit codes, high-signal-first, operator wins (#31, #42, #60, #64, #79–#86) shipped.

**Implementation plan:** [plan-0.9.0.md](docs/plan-0.9.0.md). Merge order: **#79 → #64+#85 → #80 → #81 → #82 → #31+#83 → #42+#84 → #86 → #60**.

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

**Closed:** **2026-07-03** at **`v1.0.0`**. Criterion: config + archive layout frozen; `internal/` layout; governance templates; see success criteria in [plan-1.0.0.md](docs/plan-1.0.0.md).

**Target (fulfilled):** Major when config shape, archive layout, and module layout are stable.

**Implementation plan:** [plan-1.0.0.md](docs/plan-1.0.0.md). Merge order: **#30 → #34 → #35 → #40 → #87 → #48**.

**Prerequisite met:** **v0.9.2** shipped (2026-06-29).

### 1.0.0 contract surface (measurable)

What “stable contract” means for GROOT — changes here need a **major** or clearly documented migration (see SPEC):

| Surface | Commitment |
|---------|------------|
| **CLI** | Subcommand shape for `collect` / `validate` / `inspect` / `notify`; documented **exit codes** (SPEC §12); `--output json` schemas for those commands. |
| **Config** | `groot.yml` keys with **`config_version: 1`**; legacy absent/`0` still loads; breaking key renames require bump or migration notes. |
| **Archive** | Layout versioned via **`archive_layout_version`** in `extras/manifest.json`; stable paths for logs / extras / RCA TSVs for downstream tools. |
| **Plugin** | **`kubectl-groot`** basename discovery (`kubectl groot …`); same CLI surface as `groot`. |

Authoritative detail: **[SPECIFICATIONS.md](SPECIFICATIONS.md)** (behavior today). New observable behavior → update SPEC before ship.

| # | Band | Item | Status |
|---|------|------|--------|
| 30 | 3 | **`config_version`** in YAML with documented migration notes. | **Done (v1.0.0)** |
| 34 | 3 | **`archive_layout_version`** in `extras/manifest.json` for downstream RCA tooling. | **Done (v1.0.0)** |
| 35 | 3 | **`pkg/` → `internal/` layout** (kzero/pgwd parity); no user-facing CLI change. | **Done (v1.0.0)** |
| 40 | 3 | **`--output json`**: Summary for collect; checks for validate; manifest tree for inspect (`yaml` deferred). | **Done (v1.0.0)** |
| 87 | 3 | **Golden archive fixtures** for inspect (and future analyze) regression tests. | **Done (v1.0.0)** |
| 48 | 3 | **CODEOWNERS** and GitHub issue/PR templates. | **Done (v1.0.0)** |

**Explicitly deferred past 1.0.0:** multi-cluster (#32), CI matrix (#33), stream (#41), analyze (#69), progress bar (#44), smart rules (#71), addon system (#65), embedded dashboard (#74). See **Band 4**.

---

## Band 3 maintenance (1.0.1–1.0.6) — security + post-audit hygiene

**Closed:** **2026-08-04** at **`v1.0.6`**. Criterion: Dependabot #1 (`grpc` ≥ v1.82.1); prior 1.0.3 hygiene still required. See [plan-1.0.3.md](docs/plan-1.0.3.md) for #88–#95. **v1.0.6** also ships early Band 4 packaging/docs items (#46, #47, #96).

**1.0.1** — Go **1.26.5** (stdlib CVEs). **1.0.2** — distroless Debian 13. **1.0.3** — audit items below. **1.0.4** — `grpc` **v1.82.1** + OpenTelemetry **v1.44.0**. **1.0.5** — Homebrew 6+ tap trust docs. **1.0.6** — man/nfpm/BSD, CONTRIBUTING collector guide, expanded `examples/`.

Source for 1.0.3: Hermes audit **2026-07-12** (validated). **#95 adds one CLI subcommand** (`groot notify test`); no config schema change.

| # | Band | Item | Status |
|---|------|------|--------|
| 88 | 3 | **Docker default CMD `--help`** — stop auto-running collect against `groot.yml.sample`. | **Done (v1.0.3)** |
| 89 | 3 | **Email notifier tests** — fake SMTP in CI; optional Mailgun integration (`-tags=integration`). | **Done (v1.0.3)** |
| 90 | 3 | **GCS upload tests** — emulator/fake server in CI; optional real bucket integration. | **Done (v1.0.3)** |
| 91 | 3 | **Cleanup `internal/cmd/out/`** capture artifacts in cmd tests. | **Done (v1.0.3)** |
| 92 | 3 | **`--version` exit path** — remove `os.Exit(0)` from `versionPreRun`; use `ErrVersionPrinted`. | **Done (v1.0.3)** |
| 93 | 3 | **Document QPS/Burst** (50/100) vs `worker_concurrency`; configurable rate limit → **#67**. | **Done (v1.0.3)** |
| 94 | 3 | **Bump `golang.org/x/crypto`** to v0.54.0+ (grype GO-2026-5932 hygiene; govulncheck clean). | **Done (v1.0.3)** |
| 95 | 3 | **`groot notify test`** — synthetic summary to all enabled channels; no cluster/collect. | **Done (v1.0.3)** |

---

## Band 4 (1.1.x+) — backlog by theme

Items from community review **not** blocking **1.0.0**. Prioritize after contract freeze (done). Draft **`docs/plan-1.1.0.md`** when the first theme ships.

Themes (labels for contribute/vote — item IDs unchanged):

| Theme | Intent | Sample IDs |
|-------|--------|------------|
| **Multi-cluster** | Federated collect, context selection | #32, #43 |
| **Triggered collect** | Watch critical signals → collect → upload → notify | #55 (reframed; not live tail) |
| **Analysis** | Offline heuristics on archives | #69, #56, #62 |
| **Platform** | Managed clouds, CI matrix, edge | #33, #58, #49, #59 |
| **Collector / UX** | Progress, flags, redaction, TUI | #44–#45, #54, #66–#68, #70–#72, #77–#78 |
| **Ecosystem** | Packages, hooks, dashboards, community, upload sinks | #46–#53, #61, #63, #65, #73–#76, **#96**, **#97** |

**Explicitly out of philosophy:** live **`groot stream`** / continuous log tail (#41) — see Known gaps / Non-goals. Prefer shippers or `kubectl` for that job.

### Theme: Multi-cluster

| # | Band | Item | Status |
|---|------|------|--------|
| 32 | 4 | 🚀 **Multi-cluster collect** into one archive (multiple contexts, prefixed paths). | Pending |
| 43 | 4 | 🚀 **Auto-detect kubeconfig / `--context`**: multi-path `KUBECONFIG`, context selection. | Pending |

### Theme: Triggered collect (watch → archive)

Event-driven **full collect** (not log streaming): watch critical workloads/services; on signal (CrashLoop, OOM, Warning threshold, probe fail), run the same collect pipeline, then **upload** (S3/GCS/SFTP; WebDAV/Nextcloud → **#97**) and **notify**. Complements CronJob schedule in groot-selfhosted with incident-triggered captures.

| # | Band | Item | Status |
|---|------|------|--------|
| 55 | 4 | 🚀💡 **`groot watch`** (reframed): monitor critical targets; on signal → **collect → upload → notify**. Not a live log tailer. Design: signal sources, debounce, overlap with cron, exit/daemon model. | Pending |

### Theme: Streaming (deferred / likely won't ship)

| # | Band | Item | Status |
|---|------|------|--------|
| 41 | 4 | **`groot stream`**: live pod log tailing — **out of product philosophy** (log agent / stern territory). Kept for ID stability; do not prioritize. Prefer remove-from-contract later if unused. | Pending (deprioritized) |

### Theme: Analysis

| # | Band | Item | Status |
|---|------|------|--------|
| 69 | 4 | 🚀 **`groot analyze <archive>`**: offline heuristics (CrashLoop, OOM, ImagePull, NotReady, Evicted) + executive/LLM Markdown; golden fixtures under `testing/fixtures/archives/`. | **Done (v1.1.0)** |
| 56 | 4 | **`groot diff`**: compare two archives from the same cluster. | Pending |
| 62 | 4 | **Post-collect analysis hooks** (Popeye, kubectl-debug). | Pending |

### Theme: Platform

| # | Band | Item | Status |
|---|------|------|--------|
| 33 | 4 | **CI matrix**: kind E2E against more than one Kubernetes minor; documented minimum cluster version. | Pending |
| 49 | 4 | **Promote kind E2E** from `continue-on-error` to required/nightly. | Pending |
| 58 | 4 | **Edge/K3s/MicroK8s** adjustments. | Pending |
| 59 | 4 | **E2E matrix expansion** (minikube, OpenShift local). | Pending |

### Theme: Collector / UX

| # | Band | Item | Status |
|---|------|------|--------|
| 44 | 4 | 🤝 **Progress bar / spinner** during collect. | Pending |
| 45 | 4 | **Secret redaction enhancements**: JWT/AWS patterns, `--dry-run-redact`, extend to `.txt`/`.tsv`. | Pending |
| 66 | 4 | 🤝 **Label selector flag (`-l` / `--selector`)**. | Pending |
| 67 | 4 | **Dynamic API rate limiting** (429 backoff). | Pending |
| 68 | 4 | **Per-log `LimitBytes`**. | Pending |
| 70 | 4 | **Dry-run size estimation**: extend `--list-jobs` with storage footprint estimate. | Pending |
| 71 | 4 | 💡 **Smart collect rules**: conditional `collection.rules` in YAML. | Pending |
| 72 | 4 | 💡 **Incremental collection**: `--incremental` with state in `~/.groot/state/`. | Pending |
| 77 | 4 | 🤝 **`--target` CLI flag** for ad-hoc filtering. | Pending |
| 78 | 4 | **`collection.events_min_type`** (Warning+ filter). | Pending |
| 54 | 4 | 💡 **TUI mode** (`groot tui`). | Pending |
| 57 | 4 | **`--compress-level`**, `--quick` lite mode. | Pending |

### Theme: Ecosystem

| # | Band | Item | Status |
|---|------|------|--------|
| 96 | 4 | 🚀 **man page + nfpm + BSD** ([GH #2](https://github.com/hrodrig/groot/issues/2)): `contrib/man/man1/groot.1` + `kubectl-groot.1`; GoReleaser nfpm + archives; Homebrew `manpages:`; FreeBSD/OpenBSD ports; `make man-sync`. | Done (v1.0.6) |
| 50 | 4 | **Additional package managers**: Scoop, Nix, Chocolatey, Snap. | Pending |
| 51 | 4 | **Grafana / self-hosted dashboard** (groot-selfhosted). | Pending |
| 52 | 4 | **Post-collect YAML hooks**. | Pending |
| 53 | 4 | **Observability export** (Prometheus, Loki/Promtail detection). | Pending |
| 65 | 4 | ⏳ **Addon system** (kubectl-gather-style; after multi-cluster). | Pending |
| 46 | 4 | 🤝 **Generic `examples/`** beyond profiles (#86): notify/upload/collection skeletons + GKE/AKS profiles; index in `examples/README.md`. | Done (v1.0.6) |
| 47 | 4 | 🤝 **CONTRIBUTING.md** collector guide. | Done (v1.0.6) |
| 61 | 4 | **Ticketing integration** — draft issues from collect summary / `run_id` / archive link: **GitHub Issues**, **GitLab Issues** (API), **Jira**. Same payload shape; provider adapters. | Pending |
| 63 | 4 | **Community growth** (Reddit, CNCF Slack, articles). | Pending |
| 73 | 4 | 💡 **Archive encryption/signing** (`--encrypt`, `--sign`). | Pending |
| 74 | 4 | **Embedded web dashboard** (`groot serve`) — prefer groot-selfhosted + Grafana (#51). | Pending |
| 75 | 4 | **Alternate formats** (SQLite, Parquet export). | Pending |
| 76 | 4 | **`groot cleanup`** retention policy for `output_dir`. | Pending |
| 97 | 4 | 🤝 **WebDAV / Nextcloud post-collect upload** (`upload.webdav`): PUT archive to Nextcloud (or any WebDAV endpoint) after collect; basic auth / app-password via env; path prefix; honor `--no-upload` / `upload.continue_on_error`. Complements S3/GCS/SFTP. Operators who only need FileZilla can keep using **SFTP** to a storage VPS without this item. | Pending |

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
