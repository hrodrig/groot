# groot roadmap

This file is the **in-repo** source of truth for **planned** work and known gaps. It complements:

- **[SPECIFICATIONS.md](SPECIFICATIONS.md)** — behavior contract, config shape, and what **`groot collect`** does **today**
- **[CHANGELOG.md](../CHANGELOG.md)** — what shipped in each release

User-facing overview: **[README.md](../README.md)** and **[configs/groot.yml.sample](../configs/groot.yml.sample)**.

When a roadmap item ships, update **CHANGELOG** (reference **`(band #N)`** in bullets) and mark the item **Done** here—or move highlights into the **Shipped** table.

**Last reviewed:** 2026-06-06 (**0.5.x** closed in **v0.5.0**; focus **0.6.x**)

### Versioning note

Public releases start at **v0.1.3** (early CLI and packaging). Section headings below (**0.3.x**, **0.4.x**, …) are **planned semver bands** for grouping work—not always a single tag per band. **`v0.3.0` was never published**; treat **[0.3.0]** changelog material as shipped from **`v0.3.1`** onward.

### Strategic direction

GROOT is a **read-only log and context collector**: one **`groot collect`** produces a **timestamped `.tar.gz`** for incident response and RCA. The runtime path is **client-go** end-to-end (no `kubectl` binary). Configuration is **YAML + env**; optional **notify** fan-out fires after a **successful** collect and, when configured, on **abort** or **partial job failure** (**0.5.x #19**).

**Target architecture:** deepen **collector fidelity** (more resources, clearer archive layout) while keeping the **single-command** UX. **In-cluster scheduling** is available via Helm/CronJob (**0.5.x #22**). **Notifications** cover webhooks (with templates and HMAC), email, retry, and failure alerts (**0.5.x #19–#24**). **Next band (0.6.x):** supply chain and wider distribution—**Homebrew** cask, SBOM, Cosign, optional object-store upload—without turning GROOT into a long-running controller.

**Honest gaps today:**

- **`k8srunner`** `extra_kubectl` is intentionally narrow (`get`/`describe`/`top` subsets; no `explain`/`wait`).
- **Release artifact basenames** use `groot_0.4.1_*` (GoReleaser `{{ .Version }}`); **pgwd** / **kzero** use `*_v0.x.y_*` (`{{ .Tag }}`). Align before **0.6.x #25** so Homebrew cask URLs match the family convention.
- **No post-collect object-store upload** (S3/GCS) — credentials via env only in **0.6.x #28**.
- **No Homebrew cask tap** yet (**0.6.x #25**).

**Completed bands:** **0.1.x–0.2.x** (initial CLI, packaging, notifications, container). **0.3.x** (client-go collector, RCA tables, kind E2E harness, security hardening through **v0.3.2**). **0.4.x** (archive manifest, `file_prefix` naming, `--list-jobs`, broader `extra_kubectl`, Job/CronJob targets, kind E2E in CI) — see [plan-0.4.0.md](plan-0.4.0.md). **0.5.x** (notify on failure, rich webhooks, email, retry, secret redaction, in-cluster Helm/CronJob) — see [plan-0.5.0.md](plan-0.5.0.md).

**Current focus (planned work):**

| Band | Open items |
|------|------------|
| **0.3.x** | **Closed** (last item **#11** security patch in **v0.3.2**) |
| **0.4.x** | **Closed** in **v0.4.0** (items **#12–#18**); see [plan-0.4.0.md](plan-0.4.0.md) |
| **0.5.x** | **Closed** in **v0.5.0** (items **#19–#24**); see [plan-0.5.0.md](plan-0.5.0.md) |
| **0.6.x** | **Homebrew**, SBOM/Cosign, optional object-store upload |
| **1.0.0** | Config schema stability, multi-cluster / inspect commands, K8s version matrix in CI |

---

## Shipped

| Release | Highlights |
|---------|------------|
| **0.1.3 – 0.2.1** | Early **groot collect**, YAML config, parallel collection, archive output, Slack/Teams/webhook notify, rootless container, GoReleaser **.deb/.rpm/.tar.gz**, CI and security scans. |
| **0.3.1** | **Client-go** collector (no `kubectl` binary); **`k8srunner`** allowlist; **`kubetest`** fake API; per-namespace **JSON `resources.txt`**; **RCA TSV** extras; **`extra_kubectl`** validation; kind **E2E** harness (`make test-e2e-kind`); README/VHS demo; config sample load fix. Ships unpublished **0.3.0** work—see [CHANGELOG](../CHANGELOG.md). |
| **0.3.2** | **GO-2026-5026** / **CVE-2026-39821** (`golang.org/x/net` → v0.55.0); Go **1.26.4**. |
| **0.4.0** | **Archive manifest** `extras/manifest.json` with version, cluster, jobs, paths. **`file_prefix`** now drives capture dir and archive basename. **`groot collect --list-jobs`** prints planned jobs without writing output. **`extra_kubectl` `get`/`describe`** supports `configmap`/`cm`, `pvc`, `service`/`svc`, `ingress`/`ing`, `deployment`/`rs`/`sts`/`ds` and aliases. **`collection.targets`** accepts `jobs` / `cronjobs` lists matched by `job-name` and standard labels. **Kind E2E in CI** (`make test-e2e-kind`) running with `continue-on-error: true`. Docs hygiene: `pkg/config/sample.go` and `configs/groot.yml.sample` in sync, references to `kubectl` removed. |
| **0.4.1** | **`Dockerfile`** builder **Go 1.26.4** (Security Grype image build); README download examples for **v0.4.1**. |
| **0.5.0** | **Notify on failure** (`notify.on_failure`: abort + partial-failure threshold). **Rich generic webhooks** (`extra_fields`, `body_template`, HMAC). **Email/SMTP** channel. **HTTP notify retry/backoff**. **Optional log redaction** (`collection.redact_secrets`). **In-cluster deploy**: Helm chart (`deploy/helm/groot/`) and flat CronJob manifests (`deploy/k8s/`). |

---

## 0.3.x — client-go collector and RCA (complete in v0.3.2)

Replace fork/exec **kubectl** with **client-go** / metrics APIs; strengthen tests and release hygiene.

| # | Item | Status |
|---|------|--------|
| 1 | **Client-go collector**: nodes, events, pod logs, control-plane logs, metrics, namespace resources—no `kubectl` at runtime. | **Done** (0.3.1) |
| 2 | **`pkg/k8srunner`**: allowlisted read-only argv (`get`, `describe`, `top`, `logs`, `api-resources`, `cluster-info`, `config view`, `auth can-i`, …). | **Done** (0.3.1) |
| 3 | **`pkg/kubetest`**: minimal fake Kubernetes API HTTP server for unit tests. | **Done** (0.3.1) |
| 4 | Per-namespace **`resources.txt`** as JSON sections (pods, services, Deployments, RS, STS, DS). | **Done** (0.3.1) |
| 5 | **RCA extras**: `all-pod-node-placement.tsv`, `all-pods-rca.tsv` (placement + metrics + log path). | **Done** (0.3.1) |
| 6 | **`extra_kubectl` validation** aligned with implementation allowlist. | **Done** (0.3.1) |
| 7 | **Kind E2E** harness (`testing/`, `docs/e2e-kind.md`, `make test-e2e-kind`). | **Done** (0.3.1) |
| 8 | Security: **GO-2026-4918** / **GHSA-6v2p-p543-phr9** dependency bumps. | **Done** (0.3.1) |
| 9 | Fix **`config view`** argv handling in `k8srunner`. | **Done** (0.3.1) |
| 10 | Remove unused **`pkg/kubemock`**. | **Done** (0.3.1) |
| 11 | Security: **GO-2026-5026** / **CVE-2026-39821** (`golang.org/x/net` → v0.55.0); Go **1.26.4**. | **Done** (0.3.2) |

**Out of scope for 0.3.x:** Helm chart, email notify, arbitrary webhook templates, multi-cluster bundles.

---

## 0.4.x — collector depth, docs, and CI E2E

**Implementation plan:** [plan-0.4.0.md](plan-0.4.0.md) (target **`v0.4.0`**). Merge order: **#15 → #16 → #12 → #18 → #13 → #17 → #14**.

Improve diagnostic coverage and operator trust without new top-level commands.

| # | Item | Status |
|---|------|--------|
| 12 | **`file_prefix`**: use config value in capture directory and `.tar.gz` naming (today only `<timestamp>-<cluster>` + `--message` / `since-*`). | **Done (v0.4.0)** |
| 13 | **Extend `k8srunner` `extra_kubectl`**: broader `get`/`describe` kinds (e.g. Ingress, PVC, ConfigMap, CRD instances where safe); document unsupported verbs (`explain`, `wait`) in README. | **Done (v0.4.0)** |
| 14 | **Log targets**: optional **Job** / **CronJob** selectors in `collection.targets` (label-based, same as Deployments/STS/DS). | **Done (v0.4.0)** |
| 15 | **Archive manifest**: `extras/manifest.json` (or `README.txt` inside tar) listing paths, groot version, cluster context, collect duration—speeds ticket handoff. | **Done (v0.4.0)** |
| 16 | **Docs hygiene**: sync `configs/groot.yml.sample` and comments that still say “kubectl” with client-go reality; link **ROADMAP** from README. | **Done (v0.4.0)** |
| 17 | **Kind E2E in CI**: optional GitHub Actions job (`make test-e2e-kind`), flake policy and runtime budget (pattern: [pgwd](https://github.com/hrodrig/pgwd) `test-e2e-kube`). | **Done (v0.4.0)** |
| 18 | **`groot collect --dry-run` or `--list-jobs`**: print planned API calls / output paths without mutating disk (operator preview). | **Done (v0.4.0)** |

**Out of scope for 0.4.x:** mutating cluster operations, storing archives in cloud by default.

---

## 0.5.x — notifications and Kubernetes-native operations

Make GROOT easier to run on a schedule inside the cluster and more honest when collects fail.

| # | Item | Status |
|---|------|--------|
| 19 | **Notify on failure**: optional alert when collect aborts or partial failures exceed a threshold (config + `--no-notify` respect). | **Done (v0.5.0)** |
| 20 | **Rich generic webhooks**: optional JSON body template, extra fixed fields, HMAC signing header. | **Done (v0.5.0)** |
| 21 | **Email / SMTP** channel (or document recommended proxy pattern if descoped). | **Done (v0.5.0)** |
| 22 | **In-cluster deploy**: Helm chart or documented **CronJob** + RBAC Role/ServiceAccount + ConfigMap sample (GHCR image, volume for `out/`). | **Done (v0.5.0)** |
| 23 | **Optional secret redaction** pass on collected log files (regex / known key names; off by default). | **Done (v0.5.0)** |
| 24 | **Retry/backoff** for notify HTTP clients on transient 5xx / network errors. | **Done (v0.5.0)** |

**Out of scope for 0.5.x:** write access to cluster resources, continuous monitoring agent.

---

## 0.6.x — distribution, supply chain, and upload

| # | Item | Status |
|---|------|--------|
| 25 | **Homebrew cask tap** ([homebrew-groot](https://github.com/hrodrig/homebrew-groot)): `homebrew_casks` in GoReleaser (same pattern as [pgwd](https://github.com/hrodrig/homebrew-pgwd)), `HOMEBREW_TAP_TOKEN` in CI, release **`.tar.gz`** basenames aligned to `groot_vX.Y.Z_*` (`{{ .Tag }}` like pgwd/kzero). | Pending |
| 26 | **SBOM** generation in GoReleaser (Syft or equivalent). | Pending |
| 27 | **Cosign** image/binary signing in release pipeline. | Pending |
| 28 | **Optional post-collect upload**: S3/GCS-compatible push of `.tar.gz` (credentials via env; no long-lived keys in config). | Pending |
| 29 | **FreeBSD port** or documented community packaging (if demand). | Pending |

---

## 1.0.0 (future) — stable contract and platform depth

Major when config shape, archive layout, and CLI surface are stable enough for long-term compatibility promises.

| # | Item | Status |
|---|------|--------|
| 30 | **`config_version`** (or equivalent) in YAML with documented migration notes. | Pending |
| 31 | **Second command** family: e.g. **`groot validate`** (API + RBAC preflight) or **`groot inspect <archive>`** (summarize existing bundle without cluster). | Pending |
| 32 | **Multi-cluster collect** into one archive (multiple kubecontexts, prefixed paths). | Pending |
| 33 | **CI matrix**: kind E2E against more than one Kubernetes minor; documented minimum supported cluster version in README. | Pending |
| 34 | **Stable archive layout version** field for downstream RCA tooling. | Pending |

---

## Maintenance notes

- **Release cadence:** security patches (e.g. **v0.3.2**) ship promptly; feature bands (**0.4.x+**) should earn changelog entries, README/demo refresh, and `make release-check` before tag.
- **Triad sync:** on each release—**ROADMAP** (Done + Shipped), **CHANGELOG** (`(band #N)` references), **VERSION** / README badges.
- **Large bands:** when a band spans multiple PRs, add **`docs/plan-X.Y.Z.md`** (merge order, success criteria, release checklist)—see [kzero plan-0.6.0](https://github.com/hrodrig/kzero/blob/develop/docs/plan-0.6.0.md) as reference.
- **Security:** run **`make security`** before tag; keep `govulncheck` / **grype** green; triage CodeQL on the Security tab.
- **Coverage:** merged statement gate **80%** (`COVER_MIN`); `coverage.out` is gitignored—do not commit.
- **client-go / k8s.io pin:** bump `k8s.io/*` modules together; document tested cluster minor in README when matrix lands (**1.0.0 #33**).
- **E2E:** local kind test remains optional for contributors; CI job (**0.4.x #17**) should use `continue-on-error` or a nightly workflow until flake budget is agreed.
