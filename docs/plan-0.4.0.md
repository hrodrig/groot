# Plan 0.4.0 — collector depth, docs, and CI E2E

**Status:** complete on `develop` — pending **`v0.4.0`** tag on `main`  
**Target release:** `v0.4.0` on `main` after `make release-check`  
**Roadmap band:** [ROADMAP.md](ROADMAP.md) **0.4.x** items **#12–#18**  
**Agreed implementation order:** **#15 → #16 → #12 → #18 → #13 → #17 → #14**

## Why 0.4.0

After **0.3.x** (client-go collector, RCA tables, kind E2E harness, security through **v0.3.2**), the next operator wins are:

1. **Archive handoff** — `extras/manifest.json` inside every `.tar.gz`  
2. **Honest naming** — activate `file_prefix` in capture and archive basenames  
3. **Preview** — `groot collect --list-jobs` without writing disk  
4. **Broader diagnostics** — more `extra_kubectl` resources and Job/CronJob log targets  
5. **CI confidence** — optional kind E2E job (non-blocking)

Helm chart, notify-on-failure, and Homebrew stay in **0.5.x+** (see [ROADMAP.md](ROADMAP.md)).

## Success criteria (0.4.0)

| # | Criterion | Roadmap |
|---|-----------|---------|
| 1 | Every successful collect writes **`extras/manifest.json`** (version, cluster, jobs, paths) inside the archive | #15 |
| 2 | **`file_prefix`** appears in capture dir and archive basename (`<prefix>-<timestamp>…`) | #12 |
| 3 | **`groot collect --list-jobs`** prints planned jobs and exits without creating output | #18 |
| 4 | **`extra_kubectl`** supports additional **get**/**describe** kinds (ingress, pvc, configmap, service, apps workloads) | #13 |
| 5 | **`collection.targets`** accepts **`jobs`** and **`cronjobs`** lists for pod log filtering | #14 |
| 6 | Sample config and **`SampleYAML()`** use client-go wording (no stale “kubectl” comments) | #16 |
| 7 | GitHub Actions runs **`make test-e2e-kind`** as **`continue-on-error: true`** | #17 |
| 8 | `make ci` and `make release-check` green; merged coverage ≥ 80% | — |

**Out of scope for 0.4.0:**

- Generic CRD **get** (use **`get --raw`** only)
- **`--dry-run`** alias ( **`--list-jobs`** is the shipped preview flag)
- Mutating cluster operations; cloud upload; email notify

---

## Work status (snapshot)

| Item | Roadmap | Code WIP | Docs/tests | Notes |
|------|---------|----------|------------|-------|
| Manifest | #15 | `pkg/collector/manifest.go` | tests added | Wire `SetBuildInfo` from CLI (done in `root.go`) |
| Docs hygiene | #16 | `configs/groot.yml.sample` updated | `pkg/config/sample.go` **pending** | README output naming § pending |
| `file_prefix` | #12 | `captureSessionBase`, `archiveBasename` | `collector_test.go` updated | Verify E2E script expectations |
| `--list-jobs` | #18 | `jobs_plan.go`, `root.go` | `root_test.go` added | Document in SPEC §3 |
| `k8srunner` extend | #13 | `get_extended.go`, `runner.go` | k8srunner tests **pending** | Update SPEC §6 resource table |
| Kind E2E CI | #17 | `.github/workflows/ci.yml` job | `testing/README.md` **pending** | `continue-on-error: true` |
| Job/CronJob targets | #14 | `config.go`, `collector.go` matching | partial tests | Sample YAML examples in WIP |

**Local tree:** changes exist on `develop` but are **not committed** as of plan authoring. Run `git status` before continuing.

---

## Implementation order (PR-sized slices)

Merge to **`develop`** in this order. Each slice should pass `make ci`.

| Step | Item | Roadmap | Primary files |
|------|------|---------|---------------|
| 1 | Archive manifest | #15 | `pkg/collector/manifest.go`, `collector.go`, `pkg/cmd/root.go` |
| 2 | Sample + README hygiene | #16 | `configs/groot.yml.sample`, `pkg/config/sample.go`, `README.md` |
| 3 | `file_prefix` naming | #12 | `pkg/collector/collector.go`, tests, README, SPEC §5 |
| 4 | `--list-jobs` | #18 | `pkg/collector/jobs_plan.go`, `pkg/cmd/root.go`, SPEC §3 |
| 5 | `k8srunner` resources | #13 | `pkg/k8srunner/get_extended.go`, `runner.go`, tests, SPEC §6 |
| 6 | Kind E2E CI | #17 | `.github/workflows/ci.yml`, `testing/README.md`, `docs/e2e-kind.md` |
| 7 | Job/CronJob targets | #14 | `pkg/config/config.go`, `collector.go`, tests, SPEC §4 |

Steps 1–5 may ship as one or two PRs; step 6 can land independently; step 7 can follow 5.

---

### #15 — Archive manifest

**Deliverable:** `extras/manifest.json` written after jobs complete, before `DirToTarGz`.

**JSON shape (minimum):**

```json
{
  "groot_version": "0.4.0",
  "groot_commit": "…",
  "collected_at": "2026-06-05T12:00:00Z",
  "duration_seconds": 42.5,
  "session_base": "groot-capture-20260605-120000",
  "archive_basename": "groot-capture-20260605-120000-my-cluster",
  "file_prefix": "groot-capture",
  "cluster": { "context": "…", "cluster": "…", "user": "…", "server": "…" },
  "jobs": { "total": 10, "success": 9, "failed": 1 },
  "paths": ["extras/kubeconfig.txt", "…"]
}
```

**Acceptance:** `tar -xOzf … extras/manifest.json` parses; paths list is sorted relative paths.

---

### #16 — Docs hygiene

- Replace “kubectl” comments with **client-go** / **API collection** wording in sample config and `SampleYAML()`.
- README **Output naming** must describe `<file_prefix>-<timestamp>…` (after #12).
- ROADMAP link already in README header/TOC.

---

### #12 — `file_prefix`

- **Capture folder:** `<sanitize(file_prefix)>-<timestamp>[-since-<slug>]`
- **Archive:** `<sessionBase>-<cluster>[-<message-suffix>].tar.gz`
- Default prefix when empty: `groot-capture` (unchanged behavior for default config).

---

### #18 — `--list-jobs`

```bash
groot collect --list-jobs --config groot.yml
```

- Requires API reachability (dynamic jobs: nodes, pod logs).
- Prints: `name -> file [optional] args=[…]`
- Does not create `output_dir` capture tree or `.tar.gz`.
- Does not send notifications.

---

### #13 — Extend `extra_kubectl`

**`get` (new):** configmap/cm, pvc, service/svc, ingress/ing, deployment, replicaset/rs, statefulset/sts, daemonset/ds.

**`describe` (new):** configmap, pvc, service, ingress (short summaries).

**Still rejected:** `explain`, `wait`; unsupported kinds return clear errors.

**Example extra lines in config:**

```yaml
extra_kubectl:
  - "get ingress -A"
  - "get pvc -A"
```

---

### #17 — Kind E2E in CI

Pattern: [pgwd `test-e2e-kube`](https://github.com/hrodrig/pgwd/blob/develop/.github/workflows/ci.yml).

- Job name: `test-e2e-kind`
- Runs: `make test-e2e-kind`
- **`continue-on-error: true`** until flake budget is agreed
- Does **not** block merge on first flakes; monitor in Actions tab

---

### #14 — Job / CronJob log targets

```yaml
collection:
  targets:
    default:
      jobs:
        - batch-import
      cronjobs:
        - nightly-sync
```

**Matching:** same label keys as Deployments (`app.kubernetes.io/name`, `instance`, `app`) plus **`job-name`** for Job pods.

---

## Release checklist

1. **Code complete** on `develop`; `make ci` and `make release-check` green.  
2. **ROADMAP** — mark **0.4.x #12–#18** **Done (v0.4.0)**; add **Shipped** row; set **Current focus** to **0.5.x**.  
3. **SPECIFICATIONS** — §3 (`--list-jobs`), §4 (targets, `file_prefix`), §5 (manifest, naming), §6 (resources).  
4. **CHANGELOG** — move `[Unreleased]` bullets under `## [0.4.0] - YYYY-MM-DD` with `(0.4.x #N)` references.  
5. **VERSION** — `0.4.0`; README version badge if needed.  
6. **Sample** — `configs/groot.yml.sample` and `SampleYAML()` in sync.  
7. Merge **`develop` → `main`**.  
8. **Tag** — `git tag -a v0.4.0 -m "Release 0.4.0"` on `main`; `git push origin v0.4.0`.

---

## Test strategy

| Area | Tests |
|------|-------|
| Manifest | `manifest_integration_test.go`, `manifest_test.go` (paths walk) |
| Naming | `TestCaptureSessionBase`, `TestArchiveBasename` |
| `--list-jobs` | `pkg/cmd/root_test.go` with `kubetest` API server |
| Targets | `TestMatchesTargetsJobs`, `TestHasTargets` for jobs/cronjobs |
| k8srunner | Table tests per new **get**/**describe** resource (fake clientset) |
| E2E | `make test-e2e-kind` locally; CI job optional/non-blocking |

---

## Handoff notes (Hermes)

- **`.cursor/`** is local-only — planning conventions live in **SPEC / ROADMAP / CHANGELOG / CONTRIBUTING**, not in committed Cursor rules.
- If E2E CI is red, fix flakes or Docker/kind setup before making the job required.
- Prefer one focused PR for #15–#18 and a second for #13–#14 + CI, or a single **0.4.0** PR if review load is acceptable.
