# Plan 0.9.0 — path to 1.0 (operator wins)

**Status:** **planned** — not started  
**Target release:** **`v0.9.0`** (one or more tags on `develop` before **1.0.0**)  
**Roadmap band:** [ROADMAP.md](../ROADMAP.md) **Band 0.9.x** items **#31**, **#42**, **#60**, **#64**, **#79–#86**  
**Delivery model:** one PR per item (or paired where noted). No **1.0.0** tag until [plan-1.0.0.md](plan-1.0.0.md) completes.

---

## Why 0.9.0

After **0.8.x** (RCA depth), operators need **trust and speed** before a **1.0.0** compatibility promise:

1. **Collector correctness** — Helm target matching must not miss or mis-select pods (#79).
2. **Discoverability** — `kubectl groot` + optional Krew (#64, #85).
3. **Preflight** — validate RBAC and disk before a long collect (#31, #83).
4. **Incident UX** — summary at exit, signal-first jobs (#42, #84).
5. **Ops traceability** — stable `run_id` and archive checksum (#81).
6. **Scripting** — documented exit codes (#82).

**0.9.x does not promise** frozen config schema or `internal/` layout — that is **1.0.0** only.

## Agreed implementation order

`#79 → #64+#85 → #80 → #81 → #82 → #31+#83 → #42+#84 → #86 → #60`

Inspect ships as part of **#31** (manifest + file listing minimum; problem hints optional in 0.9, deepen in 1.1 with #69).

---

## Item breakdown

### #79 — Fix Helm `helm_releases` target matching + tests

**Problem:** `matchesTargetsByLabels` for `helm_releases` is narrow; false positives/negatives possible.

**Deliverables:**

- Align matching with Helm conventions (`app.kubernetes.io/instance`, `app.kubernetes.io/managed-by=Helm`, release name labels).
- Unit tests for `matchesTargetsByLabels` and `resolvePodsForLogs` (table-driven).

**PR title:** `fix(collector): Helm release target matching (0.9.x #79)`

**Verification:** `make ci`; new tests cover helm + deployment collision cases.

---

### #64 + #85 — kubectl plugin + Krew / Makefile install

**#64:** GoReleaser ships **`kubectl-groot`** alongside **`groot`**; root command detects plugin invocation; README documents `kubectl groot collect`.

**#85:** `make install-kubectl-plugin`; optional Krew index manifest (document submission; index may live in separate repo).

**PR title:** `feat(cli): kubectl-groot plugin binary (0.9.x #64, #85)`

**Verification:** `kubectl groot version` works after install; `kubectl plugin list` shows `groot`.

---

### #80 — Shell completion

**Deliverables:** `groot completion bash|zsh|fish` (Cobra generator); README one-liner; same for `kubectl groot completion`.

**PR title:** `feat(cli): shell completion (0.9.x #80)`

---

### #81 — `run_id` and `archive_sha256` in manifest and notify

**Deliverables:**

- `extras/manifest.json`: `run_id`, `archive_sha256` (SHA-256 of final `.tar.gz`).
- Notify summary line and generic webhook `extra_fields` include `run_id` when set.
- Upload object metadata (S3/GCS user metadata) includes `run_id` when supported.

**PR title:** `feat(collector): run_id and archive checksum in manifest (0.9.x #81)`

---

### #82 — Exit code taxonomy

**Deliverables:** Document and implement stable codes:

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Config validation |
| 2 | Kubernetes client / API |
| 3 | Collect aborted (timeout, archive failure) |
| 4 | Notify delivery failed |
| 5 | Partial job failures ≥ configured threshold (when `--strict` or config flag; default remains 0 if partial failures only) |

**Note:** Default behavior today: partial failures still exit **0**. #82 adds **documented** codes; breaking change for `--strict` only if explicitly opted in.

**PR title:** `feat(cli): documented exit codes (0.9.x #82)`

**Verification:** SPEC §3 exit semantics; table-driven tests in `pkg/cmd`.

---

### #31 + #83 — `groot validate` and disk preflight

**#31:** New subcommand `groot validate` — config load, API handshake, RBAC `auth can-i` matrix for jobs the config would run.

**#83:** Validate checks free space on `output_dir` against configurable minimum (default warn at <1 GiB, fail at <256 MiB or `--min-disk`).

**#31 inspect (minimum):** `groot inspect <archive>` — print manifest, sorted file tree, sizes; no cluster required.

**PR title:** `feat(cli): groot validate and inspect (0.9.x #31, #83)`

---

### #42 + #84 — Summary mode and signal-first jobs

**#42:** `groot collect --summary` — human-readable footer (jobs, failures, CrashLoopBackOff count, archive path).

**#84:** Priority queue — schedule Warning+ events and unhealthy pods before bulk namespace logs (same workers; reorder job list).

**PR title:** `feat(collector): --summary and signal-first job order (0.9.x #42, #84)`

---

### #86 — Config profiles (`examples/profiles/`)

**Deliverables:**

| Profile | Use case |
|---------|----------|
| `incident-quick.yml` | Narrow namespaces, `--since 1h`, events + failing pods |
| `bastion-airgap.yml` | SFTP upload, minimal notify |
| `eks-managed.yml` | Skip node logs where unsupported, metrics on |
| `compliance-full.yml` | `redact_secrets: true`, all namespaces |

README links to profiles; no new config keys required.

**PR title:** `docs(0.9.x #86): example config profiles`

---

### #60 — README positioning vs kubectl-gather

**Deliverables:** README section — honest comparison table (bundle vs YAML tree, notify/upload, multi-cluster deferred to 1.1). Three-command incident example.

**PR title:** `docs(0.9.x #60): GROOT vs kubectl-gather comparison`

---

## Release checklist (at **v0.9.0** tag)

1. All items above merged; `make release-check` green.
2. ROADMAP — Band **0.9.x** items **Done (v0.9.0)**; focus → **1.0.0** ([plan-1.0.0.md](plan-1.0.0.md)).
3. CHANGELOG — `[0.9.0]` with `(0.9.x #N)` refs.
4. VERSION → `0.9.0`; README badge; VHS demo refresh if CLI UX changed.

## Success criteria for v0.9.0

| # | Criterion | Roadmap |
|---|-----------|---------|
| 1 | Helm release targets match intended pods; unit tests green | #79 |
| 2 | `kubectl groot collect` works from PATH | #64 |
| 3 | `groot validate` catches RBAC/disk issues before collect | #31, #83 |
| 4 | `groot inspect archive.tar.gz` prints manifest + tree | #31 |
| 5 | `--summary` prints actionable one-screen result | #42 |
| 6 | Manifest includes `run_id` and `archive_sha256` | #81 |
| 7 | Exit codes documented in SPEC | #82 |
| 8 | `make release-check` green; coverage ≥ 80% | — |
