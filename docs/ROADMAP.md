# groot roadmap

This file is the **in-repo** source of truth for **planned** work and known gaps. It complements:

- **[SPECIFICATIONS.md](SPECIFICATIONS.md)** — behavior contract, config shape, and what **`groot collect`** does **today**
- **[CHANGELOG.md](../CHANGELOG.md)** — what shipped in each release

User-facing overview: **[README.md](../README.md)** and **[configs/groot.yml.sample](../configs/groot.yml.sample)**.

When a roadmap item ships, update **CHANGELOG** (reference **`(band #N)`** in bullets) and mark the item **Done** here—or move highlights into the **Shipped** table.

**Last reviewed:** 2026-06-17 (Band 2 patch **0.7.2** — node log capture; Band 3 next)

### Versioning note

Public releases start at **v0.1.3** (early CLI and packaging). **`v0.3.0` was never published**; treat **[0.3.0]** changelog material as shipped from **`v0.3.1`** onward.

Bands group semver minors into three planning horizons — **Band 1** (shipped, 0.1.x–0.6.x), **Band 2** (current, 0.7.x), **Band 3** (future, 1.0.0). Individual items carry **global IDs** (#1–#38) stable across bands.

### Strategic direction

GROOT is a **read-only log and context collector**: one **`groot collect`** produces a **timestamped `.tar.gz`** for incident response and RCA. The runtime path is **client-go** end-to-end (no `kubectl` binary). Configuration is **YAML + env**; optional **notify** fan-out fires after a **successful** collect and, when configured, on **abort** or **partial job failure** (#19).

**Target architecture:** deepen **collector fidelity** (more resources, clearer archive layout) while keeping the **single-command** UX. **In-cluster scheduling** via Helm/CronJob (#22). **Notifications** and optional **S3/GCS upload** (#19–#24, #28). **Distribution** spans Linux packages, container, **Homebrew** cask, BSD ports, SBOM, and Cosign (#25–#29). **Band 2** adds **SFTP post-collect upload** (SCP-compatible relay pattern) for bastion / airgapped clusters (#36–#37) and **container image SBOM** (#38). **Band 3 (1.0.0):** stable config contract, multi-cluster, inspect/validate commands.

**Honest gaps today:**

- **`k8srunner`** `extra_kubectl` is intentionally narrow (`get`/`describe`/`top` subsets; no `explain`/`wait`).
- **No `config_version`** or formal migration path (#30).
- **No second command** (`validate` / `inspect`) (#31).

**Current focus (planned work):**

| Band | Status | Items |
|------|--------|-------|
| **Band 1** (0.1.x–0.6.x) | **Shipped** (#1–#29) | Initial CLI through BSD ports; see [plan-0.4.0](plan-0.4.0.md), [plan-0.5.0](plan-0.5.0.md), [plan-0.6.0](plan-0.6.0.md) |
| **Band 2** (0.7.x) | **Shipped** (#36–#38) | SFTP upload (SCP-compatible relay), airgapped relay playbook, container image SBOM; see [plan-0.7.0.md](plan-0.7.0.md) |
| **Band 3** (1.0.0) | **Future** (#30–#35) | Config schema stability, multi-cluster, inspect/validate, CI matrix, `pkg/` → `internal/` |

---

## Shipped

| Release | Band | Highlights |
|---------|------|------------|
| **0.1.3 – 0.2.1** | 1 | Early **groot collect**, YAML config, parallel collection, archive output, Slack/Teams/webhook notify, rootless container, GoReleaser **.deb/.rpm/.tar.gz**, CI and security scans. |
| **0.3.1** | 1 | **Client-go** collector (no `kubectl` binary); **`k8srunner`** allowlist; **`kubetest`** fake API; per-namespace **JSON `resources.txt`**; **RCA TSV** extras; **`extra_kubectl`** validation; kind **E2E** harness (`make test-e2e-kind`); README/VHS demo; config sample load fix. Ships unpublished **0.3.0** work—see [CHANGELOG](../CHANGELOG.md). |
| **0.3.2** | 1 | **GO-2026-5026** / **CVE-2026-39821** (`golang.org/x/net` → v0.55.0); Go **1.26.4**. |
| **0.4.0** | 1 | **Archive manifest** `extras/manifest.json` with version, cluster, jobs, paths. **`file_prefix`** now drives capture dir and archive basename. **`groot collect --list-jobs`** prints planned jobs without writing output. **`extra_kubectl` `get`/`describe`** supports `configmap`/`cm`, `pvc`, `service`/`svc`, `ingress`/`ing`, `deployment`/`rs`/`sts`/`ds` and aliases. **`collection.targets`** accepts `jobs` / `cronjobs` lists matched by `job-name` and standard labels. **Kind E2E in CI** (`make test-e2e-kind`) running with `continue-on-error: true`. Docs hygiene: `pkg/config/sample.go` and `configs/groot.yml.sample` in sync, references to `kubectl` removed. |
| **0.4.1** | 1 | **`Dockerfile`** builder **Go 1.26.4** (Security Grype image build); README download examples for **v0.4.1**. |
| **0.5.0** | 1 | **Notify on failure** (`notify.on_failure`: abort + partial-failure threshold). **Rich generic webhooks** (`extra_fields`, `body_template`, HMAC). **Email/SMTP** channel. **HTTP notify retry/backoff**. **Optional log redaction** (`collection.redact_secrets`). **In-cluster deploy**: Helm chart (`deploy/helm/groot/`) and flat CronJob manifests (`deploy/k8s/`). |
| **0.6.0** | 1 | **Homebrew cask** tap + `groot_vX.Y.Z_*` release basenames. **SBOM** (SPDX + CycloneDX). **Cosign** keyless signing (checksums + images). **Post-collect S3/GCS upload**. **FreeBSD + OpenBSD** ports with release CI. |
| **0.6.1** | 1 | **Product vs operator split**: Helm/CronJob manifests and deploy runbooks moved to **[groot-selfhosted](https://github.com/hrodrig/groot-selfhosted)**; README/SPEC/AGENTS clarify product scope (docs only). |
| **0.7.0** | 2 | **SFTP post-collect upload** (`upload.sftp`): bastion → SSH relay via public-key SFTP (`--no-upload` honored). **Airgapped relay playbook** (groot-selfhosted): bastion → SFTP → rclone → OneDrive topology with systemd watcher. **Container image SBOM**: OCI attestation enabled on `dockers_v2` (`sbom: true` via buildx docker-container driver). |
| **0.7.1** | 2 | **`cluster_name` config**: optional archive basename cluster label; resolution chain when empty (kubeconfig → cluster-info → API server host). |
| **0.7.2** | 2 | **Node log capture**: host logs via node proxy → `nodes/<node>.log`; kubelet log query optional; fewer spurious failures on managed clouds (e.g. AKS). |

---

## Band 1 (shipped) — 0.1.x through 0.6.x

### 0.3.x — client-go collector and RCA (complete in v0.3.2)

Replace fork/exec **kubectl** with **client-go** / metrics APIs; strengthen tests and release hygiene.

| # | Band | Item | Status |
|---|------|------|--------|
| 1 | 1 | **Client-go collector**: nodes, events, pod logs, control-plane logs, metrics, namespace resources—no `kubectl` at runtime. | **Done** (0.3.1) |
| 2 | 1 | **`pkg/k8srunner`**: allowlisted read-only argv (`get`, `describe`, `top`, `logs`, `api-resources`, `cluster-info`, `config view`, `auth can-i`, …). | **Done** (0.3.1) |
| 3 | 1 | **`pkg/kubetest`**: minimal fake Kubernetes API HTTP server for unit tests. | **Done** (0.3.1) |
| 4 | 1 | Per-namespace **`resources.txt`** as JSON sections (pods, services, Deployments, RS, STS, DS). | **Done** (0.3.1) |
| 5 | 1 | **RCA extras**: `all-pod-node-placement.tsv`, `all-pods-rca.tsv` (placement + metrics + log path). | **Done** (0.3.1) |
| 6 | 1 | **`extra_kubectl` validation** aligned with implementation allowlist. | **Done** (0.3.1) |
| 7 | 1 | **Kind E2E** harness (`testing/`, `docs/e2e-kind.md`, `make test-e2e-kind`). | **Done** (0.3.1) |
| 8 | 1 | Security: **GO-2026-4918** / **GHSA-6v2p-p543-phr9** dependency bumps. | **Done** (0.3.1) |
| 9 | 1 | Fix **`config view`** argv handling in `k8srunner`. | **Done** (0.3.1) |
| 10 | 1 | Remove unused **`pkg/kubemock`**. | **Done** (0.3.1) |
| 11 | 1 | Security: **GO-2026-5026** / **CVE-2026-39821** (`golang.org/x/net` → v0.55.0); Go **1.26.4**. | **Done** (0.3.2) |

**Out of scope for 0.3.x:** Helm chart, email notify, arbitrary webhook templates, multi-cluster bundles.

---

### 0.4.x — collector depth, docs, and CI E2E

**Implementation plan:** [plan-0.4.0.md](plan-0.4.0.md) (target **`v0.4.0`**). Merge order: **#15 → #16 → #12 → #18 → #13 → #17 → #14**.

Improve diagnostic coverage and operator trust without new top-level commands.

| # | Band | Item | Status |
|---|------|------|--------|
| 12 | 1 | **`file_prefix`**: use config value in capture directory and `.tar.gz` naming (today only `<timestamp>-<cluster>` + `--message` / `since-*`). | **Done (v0.4.0)** |
| 13 | 1 | **Extend `k8srunner` `extra_kubectl`**: broader `get`/`describe` kinds (e.g. Ingress, PVC, ConfigMap, CRD instances where safe); document unsupported verbs (`explain`, `wait`) in README. | **Done (v0.4.0)** |
| 14 | 1 | **Log targets**: optional **Job** / **CronJob** selectors in `collection.targets` (label-based, same as Deployments/STS/DS). | **Done (v0.4.0)** |
| 15 | 1 | **Archive manifest**: `extras/manifest.json` (or `README.txt` inside tar) listing paths, groot version, cluster context, collect duration—speeds ticket handoff. | **Done (v0.4.0)** |
| 16 | 1 | **Docs hygiene**: sync `configs/groot.yml.sample` and comments that still say "kubectl" with client-go reality; link **ROADMAP** from README. | **Done (v0.4.0)** |
| 17 | 1 | **Kind E2E in CI**: optional GitHub Actions job (`make test-e2e-kind`), flake policy and runtime budget (pattern: [pgwd](https://github.com/hrodrig/pgwd) `test-e2e-kube`). | **Done (v0.4.0)** |
| 18 | 1 | **`groot collect --dry-run` or `--list-jobs`**: print planned API calls / output paths without mutating disk (operator preview). | **Done (v0.4.0)** |

**Out of scope for 0.4.x:** mutating cluster operations, storing archives in cloud by default.

---

### 0.5.x — notifications and Kubernetes-native operations

Make GROOT easier to run on a schedule inside the cluster and more honest when collects fail.

| # | Band | Item | Status |
|---|------|------|--------|
| 19 | 1 | **Notify on failure**: optional alert when collect aborts or partial failures exceed a threshold (config + `--no-notify` respect). | **Done (v0.5.0)** |
| 20 | 1 | **Rich generic webhooks**: optional JSON body template, extra fixed fields, HMAC signing header. | **Done (v0.5.0)** |
| 21 | 1 | **Email / SMTP** channel (or document recommended proxy pattern if descoped). | **Done (v0.5.0)** |
| 22 | 1 | **In-cluster deploy**: Helm chart or documented **CronJob** + RBAC Role/ServiceAccount + ConfigMap sample (GHCR image, volume for `out/`). | **Done (v0.5.0)** |
| 23 | 1 | **Optional secret redaction** pass on collected log files (regex / known key names; off by default). | **Done (v0.5.0)** |
| 24 | 1 | **Retry/backoff** for notify HTTP clients on transient 5xx / network errors. | **Done (v0.5.0)** |

**Out of scope for 0.5.x:** write access to cluster resources, continuous monitoring agent.

---

### 0.6.x — distribution, supply chain, and upload

| # | Band | Item | Status |
|---|------|------|--------|
| 25 | 1 | **Homebrew cask tap** ([homebrew-groot](https://github.com/hrodrig/homebrew-groot)): `homebrew_casks` in GoReleaser (same pattern as [pgwd](https://github.com/hrodrig/homebrew-pgwd)), `HOMEBREW_TAP_TOKEN` in CI, release **`.tar.gz`** basenames aligned to `groot_vX.Y.Z_*` (`{{ .Tag }}` like pgwd/kzero). | **Done (v0.6.0)** |
| 26 | 1 | **SBOM** generation in GoReleaser (Syft or equivalent). | **Done (v0.6.0)** |
| 27 | 1 | **Cosign** image/binary signing in release pipeline. | **Done (v0.6.0)** |
| 28 | 1 | **Optional post-collect upload**: S3/GCS-compatible push of `.tar.gz` (credentials via env; no long-lived keys in config). | **Done (v0.6.0)** |
| 29 | 1 | **FreeBSD port** or documented community packaging (if demand). | **Done (v0.6.0)** |

---

## Band 2 (active) — 0.7.x airgapped upload and supply-chain follow-up

**Implementation plan:** [plan-0.7.0.md](plan-0.7.0.md) (target **`v0.7.0`**). Merge order: **#36 → #37 → #38**.

Post-collect delivery for clusters **without outbound internet**: groot runs on a **bastion** with kubeconfig to the API; the bastion is the only hop allowed to SSH a **relay host** (e.g. one external IP); the relay syncs archives to cloud storage (OneDrive via **rclone** on Linux — **out of groot scope**, documented in **#37**).

| # | Band | Item | Status |
|---|------|------|--------|
| 36 | 2 | **SFTP post-collect upload** (`upload.sftp`): push `.tar.gz` to a remote Linux host over SSH (public-key auth via env `GROOT_UPLOAD_SFTP_IDENTITY_FILE`; `known_hosts` file; `BatchMode` — no password prompts). Same failure semantics as S3/GCS (`continue_on_error`, `--no-upload`). Operators say "SCP to relay" — this is the SFTP implementation of that pattern. | **Done (v0.7.0)** |
| 37 | 2 | **Airgapped relay playbook**: **[groot-selfhosted → airgapped-relay](https://github.com/hrodrig/groot-selfhosted/blob/main/run/examples/airgapped-relay/README.md)** — topology bastion → SSH relay → **rclone** OneDrive. `groot-bastion.yml`, `systemd.path`+`.service` watcher, SSH hardening. No groot code. | **Done (v0.7.0)** |
| 38 | 2 | **Container image SBOM**: OCI SBOM attestation enabled on `dockers_v2` (`sbom: true`) — release workflow already uses buildx `docker-container` driver via `docker/setup-buildx-action@v3`. | **Done (v0.7.0)** |

**Out of scope for Band 2:** native OneDrive / Microsoft Graph upload inside groot; password-based SSH; groot running inside AKS pods with egress to relay (bastion is the supported runtime for this topology).

---

## Band 3 (future) — 1.0.0 stable contract and platform depth

Major when config shape, archive layout, and CLI surface are stable enough for long-term compatibility promises.

| # | Band | Item | Status |
|---|------|------|--------|
| 30 | 3 | **`config_version`** (or equivalent) in YAML with documented migration notes. | Pending |
| 31 | 3 | **Second command** family: e.g. **`groot validate`** (API + RBAC preflight) or **`groot inspect <archive>`** (summarize existing bundle without cluster). | Pending |
| 32 | 3 | **Multi-cluster collect** into one archive (multiple kubecontexts, prefixed paths). | Pending |
| 33 | 3 | **CI matrix**: kind E2E against more than one Kubernetes minor; documented minimum supported cluster version in README. | Pending |
| 34 | 3 | **Stable archive layout version** field for downstream RCA tooling. | Pending |
| 35 | 3 | **`pkg/` → `internal/` layout** (kzero/pgwd parity): `pkg/cmd`→`internal/cli`, `pkg/config`→`internal/config`, `pkg/collector`→`internal/collector`, `pkg/notifier`→`internal/notifier`, `pkg/uploader`→`internal/uploader`, `pkg/k8srunner`→`internal/k8srunner`, `pkg/kubeloader`→`internal/kubeloader`, `pkg/kubetest`→`internal/kubetest`, `pkg/archive`→`internal/archive`, `pkg/logx`→`internal/log`; update imports + SPEC/docs; no user-facing CLI change. Do **before** promising stable library surface—groot is a CLI, not a public SDK. | Pending (1.0.x) |

---

## Maintenance notes

- **Release cadence:** security patches (e.g. **v0.3.2**) ship promptly; feature bands should earn changelog entries, README/demo refresh, and `make release-check` before tag.
- **Triad sync:** on each release—**ROADMAP** (Done + Shipped), **CHANGELOG** (`(band #N)` references), **VERSION** / README badges.
- **Large bands:** when a band spans multiple PRs, add **`docs/plan-X.Y.Z.md`** (merge order, success criteria, release checklist)—see [kzero plan-0.6.0](https://github.com/hrodrig/kzero/blob/develop/docs/plan-0.6.0.md) as reference.
- **Security:** run **`make security`** before tag; keep `govulncheck` / **grype** green; triage CodeQL on the Security tab.
- **Coverage:** merged statement gate **80%** (`COVER_MIN`); `coverage.out` is gitignored—do not commit.
- **client-go / k8s.io pin:** bump `k8s.io/*` modules together; document tested cluster minor in README when matrix lands (#33).
- **E2E:** local kind test remains optional for contributors; CI job (#17) should use `continue-on-error` or a nightly workflow until flake budget is agreed.
