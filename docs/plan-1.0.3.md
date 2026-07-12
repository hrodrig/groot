# Plan 1.0.3 — post-audit hygiene (review 2026-07-12)

**Status:** **planned** — work on `develop`; tag **`v1.0.3`** when checklist passes  
**Target release:** **`v1.0.3`** (maintenance / test-hygiene patch; **no CLI or config contract change**)  
**Source reviews:** [.no-va-al-repo/20260712-hermes-deepseek/deepseek-review-20260712.md](../.no-va-al-repo/20260712-hermes-deepseek/deepseek-review-20260712.md) (validated 2026-07-12); cross-ref [review 2026-07-02](../.no-va-al-repo/deep-seek-review-20260702.md) (notifier races **already fixed** in v1.0.0)  
**Roadmap band:** **Band 3 maintenance** items **#88–#94** — see [ROADMAP.md](../ROADMAP.md)  
**Delivery model:** one PR per item (or pair small items); **`make release-check`** before tag.

---

## Why 1.0.3

**v1.0.2** shipped distroless Debian 13. Gates are green (81.1% coverage, race, govulncheck). The 2026-07-12 audit found **no release blockers** but **real gaps** in peripheral features (email notify, GCS upload) and one **container UX** footgun (Docker default CMD runs collect against the **sample** config).

This band closes hygiene debt **without** Band 4 features (analyze, multi-cluster, etc.).

---

## Dependency pins (clarification)

| Module | `go.mod` today | Notes |
|--------|----------------|-------|
| `golang.org/x/net` | **v0.55.0** (indirect) | Bumped for CVE in **v0.3.2** — this is likely the “0.55” you remember |
| `golang.org/x/crypto` | **v0.53.0** (**direct**) | Used by SFTP (`ssh`, `knownhosts`); **not** at 0.55 |
| Latest published `x/crypto` | **v0.54.0** (Jul 2026) | Item **#94** bumps direct pin; re-run `govulncheck` + grype |

**M4 (GO-2026-5932):** grype flags `x/crypto` v0.53.0; **govulncheck reports 0 vulnerable call paths**. Bump to v0.54.0 for scanner hygiene, not an emergency patch.

---

## Agreed implementation order

`#88 → #94 → #91 → #89 → #90 → #92 → #93`

Optional local-only (not required for tag): integration tests documented below under **Operator test infra**.

---

## Item breakdown

### #88 — Docker image default CMD (review H1)

**Problem:** `docker run ghcr.io/hrodrig/groot` (no args) runs `collect --config /app/groot.yml.sample` against placeholder targets (`api`, `redis`, `node-agent`, …). Confusing for newcomers; **groot-selfhosted** docs always mount a real config.

**Resolves:** Safe default for bare `docker run`; sample remains in image for `--print-sample-config` / copy-out, not auto-executed.

**Deliverables:**

- `Dockerfile` + `Dockerfile.release`: `CMD ["--help"]` (keep `ENTRYPOINT ["/app/groot"]`).
- README one-liner: image default prints help; operators pass `collect --config …` (link groot-selfhosted `run/docker/`).
- No SPEC change (runtime contract unchanged).

**PR title:** `fix(docker): default CMD --help instead of sample collect (1.0.3 #88)`

**Effort:** S

---

### #94 — Bump `golang.org/x/crypto` (review M4)

**Problem:** Direct dep at v0.53.0; grype GO-2026-5932 (not called per govulncheck).

**Resolves:** Scanner green on direct dep; stays current for SFTP/SSH paths.

**Deliverables:**

- `go get golang.org/x/crypto@v0.54.0 && go mod tidy`
- `make security` + `make cover` green
- CHANGELOG bullet under `[Unreleased]` / `1.0.3`

**PR title:** `chore(deps): bump golang.org/x/crypto to v0.54.0 (1.0.3 #94)`

**Effort:** S

---

### #91 — Test artifact cleanup `internal/cmd/out/` (review M1)

**Problem:** 19 stale `groot-capture-*` dirs from local/CI runs; gitignored but noisy.

**Resolves:** Repeatable clean tree; tests don’t accumulate captures.

**Deliverables:**

- `t.Cleanup` / shared helper in cmd integration tests that write under `out/`
- Optional `TestMain` in `internal/cmd` package to remove `out/groot-capture-*` after package tests
- Document in CONTRIBUTING: default output dir for cmd tests

**PR title:** `test(cmd): cleanup capture dirs under internal/cmd/out (1.0.3 #91)`

**Effort:** S

---

### #89 — Email notifier tests (review H2)

**Problem:** `internal/notifier/email.go` — **0%** coverage; feature in SPEC/README since v0.5.0.

**Resolves:** CI catches SMTP/TLS/auth regressions; parity with Slack/Teams httptest coverage.

**Deliverables (CI — always on):**

- `email_test.go` with **local fake SMTP** (`net.Listen("tcp", "127.0.0.1:0")` + minimal RFC8211-ish server or `github.com/mailhog/MailHog` not required — keep stdlib-only if possible)
- Cases: (1) plain send OK, (2) STARTTLS port 587 path, (3) implicit TLS (`useTLS`), (4) auth failure, (5) multiple recipients `;`, (6) empty host error
- Target **≥80%** on `email.go` functions

**Optional integration (local / nightly — `//go:build integration`):**

- Env-driven Mailgun SMTP (no secrets in repo):

  | Env | Example |
  |-----|---------|
  | `GROOT_TEST_SMTP_HOST` | `smtp.mailgun.org` |
  | `GROOT_TEST_SMTP_PORT` | `587` |
  | `GROOT_TEST_SMTP_USER` | `postmaster@…` |
  | `GROOT_TEST_SMTP_PASSWORD` | Mailgun SMTP password |
  | `GROOT_TEST_SMTP_TO` | sandbox recipient |

- `go test -tags=integration ./internal/notifier/…` — skip if env unset (`t.Skip`)
- Document in plan + `testing/README.md` or CONTRIBUTING snippet

**PR title:** `test(notifier): email SMTP unit tests (1.0.3 #89)`

**Effort:** M (unit); +S for integration scaffold

---

### #90 — GCS upload tests (review H3)

**Problem:** GCS `Upload()` ~18%; `gcsClientOptions()` 0%; S3 already ~80% via httptest.

**Resolves:** Parity with S3; emulator path exercised.

**Deliverables (CI — always on):**

- Extend `gcs_test.go` / `uploader_test.go`:
  - success upload (fake GCS — **fake-gcs-server** in testcontainer **or** httptest shim if too heavy → prefer [`STORAGE_EMULATOR_HOST`](https://cloud.google.com/storage/docs/emulator) against ephemeral fake server)
  - `gcsClientOptions()` with `t.Setenv("STORAGE_EMULATOR_HOST", …)`
  - bucket empty → error (existing, keep)
  - context cancel mid-upload
  - metadata / key prefix / `run_id` in object name if applicable

**Optional integration (local — `//go:build integration`):**

- Real GCS bucket (your service):

  | Env | Purpose |
  |-----|---------|
  | `GROOT_TEST_GCS_BUCKET` | dedicated test bucket |
  | `GROOT_TEST_GCS_PREFIX` | e.g. `groot-ci/` (auto-delete prefix) |
  | `GOOGLE_APPLICATION_CREDENTIALS` or ADC | auth |

- Optional: same for **S3-compatible** endpoint (your MinIO/S3):

  | Env | Purpose |
  |-----|---------|
  | `GROOT_TEST_S3_BUCKET` | test bucket |
  | `GROOT_TEST_S3_ENDPOINT` | custom endpoint URL |
  | `GROOT_TEST_S3_REGION` | region |
  | `AWS_ACCESS_KEY_ID` / `AWS_SECRET_ACCESS_KEY` | credentials |

- S3 httptest already covers CI; integration validates **your** endpoint quirks (path-style, TLS, etc.)

**PR title:** `test(uploader): GCS upload and emulator coverage (1.0.3 #90)`

**Effort:** M

---

### #92 — `versionPreRun` exit taxonomy (review M2)

**Problem:** `os.Exit(0)` in `versionPreRun` bypasses `main.go` exit plumbing (except in `testing.Testing()`).

**Resolves:** Single exit path; `--version` still exits 0.

**Deliverables:**

- Always return `ErrVersionPrinted`; handle in `ExecuteContext` / `run()` → exit 0
- Remove `os.Exit(0)` from library code
- Test: `groot --version` exit code via `cmd` test helper

**PR title:** `refactor(cmd): route --version through ErrVersionPrinted (1.0.3 #92)`

**Effort:** S

---

### #93 — Document client-go QPS/Burst vs worker pool (review M3)

**Problem:** Operators on large clusters may not know `kubeloader` fixes QPS=50/Burst=100; review asked for configurability.

**Resolves:** Documented behavior; optional follow-up → Band 4 **#67** (dynamic rate limit).

**Deliverables (1.0.3 scope — docs only):**

- SPEC §5 (collect): note default QPS/Burst, client-go token bucket, tune `collection.worker_concurrency` for large clusters
- README troubleshooting bullet
- ROADMAP **#67** remains the YAML-configurable rate-limit item

**PR title:** `docs: document kube client QPS/Burst and worker concurrency (1.0.3 #93)`

**Effort:** S

---

## Review M5 — no separate item (already partially covered)

| Finding | Action |
|---------|--------|
| `runRedactPass()` 0% | **Defer** — 3-line wrapper; `RedactCaptureLogs` has 3 unit tests |
| `CountUnhealthyPods()` 0% | Optional small test with `kubetest` fake API in same band if time; else Band 4 |

---

## Deferred — review 2026-07-02 (not 1.0.3)

Tracked for **Band 4** or later maintenance, not blocking 1.0.3:

| Topic | Status v1.0.2 |
|-------|----------------|
| Notifier global races | **Fixed** v1.0.0 |
| `collector.go` god file (~1006 L) | Open — refactor Band 4 |
| `DirToTarGz(ctx)` | Open |
| `resolvePodsForLogs` single-pass | Open |
| Redact atomic rename | Open — related **#45** |

---

## Operator test infra (Hermes — use in integration runs)

**Do not commit credentials.** Use env + `-tags=integration`.

```bash
# Mailgun (optional integration)
export GROOT_TEST_SMTP_HOST=smtp.mailgun.org
export GROOT_TEST_SMTP_PORT=587
export GROOT_TEST_SMTP_USER=...
export GROOT_TEST_SMTP_PASSWORD=...
export GROOT_TEST_SMTP_TO=...

# GCS (optional integration)
export GROOT_TEST_GCS_BUCKET=...
export GROOT_TEST_GCS_PREFIX=groot-test/
# GOOGLE_APPLICATION_CREDENTIALS=... or gcloud ADC

# S3-compatible (optional integration)
export GROOT_TEST_S3_BUCKET=...
export GROOT_TEST_S3_ENDPOINT=https://...
export AWS_ACCESS_KEY_ID=...
export AWS_SECRET_ACCESS_KEY=...

go test -tags=integration ./internal/notifier/... ./internal/uploader/...
```

CI default: **unit tests only** (httptest / fake SMTP / fake GCS emulator) — no secrets in GitHub Actions.

---

## Release checklist (at **v1.0.3** tag)

1. **#88–#94** merged (minimum **#88**, **#89**, **#90**, **#94** for audit closure).
2. `make release-check` green on `develop`.
3. ROADMAP — items **Done (v1.0.3)**; Shipped table row; **Last reviewed** date.
4. CHANGELOG — `[1.0.3]`; `(band #88)` … `(band #94)` references.
5. VERSION + README badge sync.
6. Merge `develop` → `main`; annotated tag `v1.0.3`.

---

## Success criteria for v1.0.3

| # | Criterion |
|---|-----------|
| 1 | Bare `docker run ghcr.io/hrodrig/groot` prints help, not sample collect |
| 2 | `email.go` ≥80% statement coverage in CI |
| 3 | GCS upload + `gcsClientOptions` covered in CI (emulator or fake server) |
| 4 | `golang.org/x/crypto` ≥ v0.54.0; govulncheck/grype clean on called paths |
| 5 | No stale captures committed; cmd tests cleanup `internal/cmd/out/` |
| 6 | `make release-check` green; no SPEC/config contract change |

---

*Plan created 2026-07-12 from Hermes audit validation session.*
