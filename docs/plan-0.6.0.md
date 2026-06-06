# Plan 0.6.0 — distribution, supply chain, and upload

**Status:** draft  \
**Target release:** `v0.6.0`  \
**Roadmap band:** [ROADMAP.md](ROADMAP.md) **0.6.x** items **#25–#29**  \
**Delivery model:** one PR per item on `develop`. No tag until you decide.

---

## Why 0.6.0

After **0.5.x** (notify, in-cluster, redaction, retry), operators and CI pipelines need:

1. **Wider distribution** — Homebrew tap for macOS/Linux users that don't `go install`
2. **Supply chain transparency** — SBOM attached to every release
3. **Binary + image authenticity** — Cosign signatures
4. **Post-collect storage** — auto-upload `.tar.gz` to S3/GCS (env creds, no long-lived keys)
5. **BSD packaging** — **FreeBSD** port + **OpenBSD** port (full, not skeleton) so `pkg install groot` / `pkg_add groot` work on both

The product stays a **single command** (`groot collect`); 0.6.x adds **packaging and delivery** around it. No new top-level command, no controller pattern.

## Agreed implementation order

`#25 → #26 → #27 → #28 → #29` (pipeline → real code → BSD ports). The BSD port item (#29) is **full quality**, not skeleton — it lands in the same release band. Merge order is chosen so the riskier config changes (basename rename, keyless signing) land first and the longest-tail real code (#28) gets reviewed against an already-stable pipeline.

Each item is a **separate PR** on `develop`. Merge order matches above so #28 (real code) gets the most review and #25–#27 (config) can land fast.

---

## Item breakdown

### #25 — Homebrew cask + release basename alignment

**Roadmap:** `groot` cask tap + `{{ .Tag }}` naming matching pgwd/kzero family.

**Files in `groot` repo (this PR):**
- `.goreleaser.yaml`:
  - `archives[0].name_template`: `groot_{{ .Tag }}_{{ .Os }}_{{ .Arch }}` (currently `{{ .Version }}`)
  - `nfpms[0].file_name_template`: `groot_{{ .Tag }}_{{ .Arch }}`
  - new top-level `brews:` stanza pointing at `homebrew-groot` tap (repo: `hrodrig/homebrew-groot`, folder: `Casks/groot.rb`, **Pull Request** mode, **skip upload** for the cask — same as pgwd pattern)
- `README.md`: install block `brew install hrodrig/groot/groot`
- `contrib/homebrew/Casks/groot.rb.template` — placeholder source-of-truth cask file. Goreleaser will copy this into the tap repo on release and substitute `version` / `sha256` / `url`. The PR reviewer (you) opens the new tap repo with this template as a starting point; goreleaser keeps the tap in sync on every tag.
- `contrib/homebrew/README.md` — explains: (1) the tap repo `hrodrig/homebrew-groot` is bootstrapped from `groot.rb.template`; (2) the GoReleaser `brews:` stanza auto-updates the tap on each release; (3) the manual fallback `make update-cask` command (a tiny `scripts/update-homebrew-cask.sh`) for situations where automation is disabled
- `scripts/update-homebrew-cask.sh` — read `contrib/homebrew/Casks/groot.rb.template`, substitute `{{VERSION}}` and `{{SHA256}}` from latest release artifacts, write to tap repo (clone or git pull if `TAP_REPO` env set), commit, push. Idempotent.
- `.github/workflows/release.yml`: add `HOMEBREW_TAP_TOKEN` (PAT with `repo` scope on the tap repo) to release job env

**Bootstrap of the tap repo `hrodrig/homebrew-groot` (you do this manually, I prepare the source):**
1. You create the repo `hrodrig/homebrew-groot` (public, no GitHub Pages needed)
2. Copy `contrib/homebrew/Casks/groot.rb.template` to that repo at `Casks/groot.rb`, commit `chore: bootstrap from hrodrig/groot contrib/homebrew`
3. From that point on, goreleaser owns it via the `brews:` stanza in `.goreleaser.yaml`

**PR title:** `release(0.6.x #25): Homebrew cask + release basename alignment`

**Why this comes first:** the basename change is a **breaking artifact rename** for anyone scripting off `groot_0.4.1_*`. Pairing it with the Homebrew cask makes the migration visible in the same release.

**Verification:**
- `goreleaser check`
- snapshot build: `goreleaser build --snapshot --clean` produces `groot_v0.6.0-next_linux_amd64.tar.gz`
- cask file syntactically valid: `brew audit --strict --online hrodrig/groot/groot` (only after first real tag)

---

### #26 — SBOM generation

**Roadmap:** Syft SBOM in release pipeline.

**Files:**
- `.goreleaser.yaml`:
  - new top-level `sboms:` block — `formats: [spdx-json, cyclonedx-json]`, `artifacts: archive`
  - **NB:** current `dockers_v2.sbom: false` due to docker-driver attestation limit. Switch `dockers_v2` to use the `docker-container` driver by relying on the existing buildx setup (QEMU step already in workflow) — the constraint is build-driver, not buildx itself. Keep `sbom: true` for images; rely on `--attestation=type=sbom` for OCI attestations. If it still blocks in CI, fall back to `sbom: false` and rely on `sboms:` for archive SBOMs only (mirrors pgwd).
- `README.md`: link to SBOM artifact per release (`out: groot_<ver>_sbom.spdx.json`)

**PR title:** `release(0.6.x #26): SBOM generation (Syft, SPDX + CycloneDX)`

**Verification:**
- snapshot build: SBOM file present in `dist/`
- SPDX schema validates with `python -c "import json,sys; json.load(open(sys.argv[1]))"` and `spdx-tools validate`

---

### #27 — Cosign signing

**Roadmap:** sign release binaries and container image.

**Files:**
- `.goreleaser.yaml`:
  - new top-level `signs:` — `cmd: cosign`, `artifacts: checksum`, `output_signature: true`, `keyless: false`, env `COSIGN_KEY` / `COSIGN_PASSWORD` from CI secrets
  - `dockers_v2` `signs:` — `cmd: cosign sign -key=…`, **keyless** mode with OIDC (uses `id-token: write` permission); fallback to key-based if keyless OIDC flakes
- `.github/workflows/release.yml`:
  - add `id-token: write` permission
  - new secrets: `COSIGN_KEY` (base64), `COSIGN_PASSWORD` (keyless path drops these)
- `README.md`: verify block `cosign verify --certificate-identity-regexp … --certificate-oidc-issuer … ghcr.io/hrodrig/groot:vX.Y.Z`

**PR title:** `release(0.6.x #27): Cosign signatures (binaries + container image)`

**Verification:**
- `cosign verify-blob … checksums.txt` succeeds against the public key / OIDC
- `cosign verify --certificate-identity-regexp 'https://github.com/hrodrig/groot' …` succeeds for image

---

### #28 — Post-collect upload (S3 / GCS) — the real code

**Roadmap:** optional `upload:` block, env-only credentials.

**Dependencies (per user choice):**
- `github.com/aws/aws-sdk-go-v2` — `config`, `credentials`, `service/s3` (modular, tree-shaken via `aws-sdk-go-v2/service/s3` direct path)
- `github.com/aws/aws-sdk-go-v2/feature/s3/manager` — Uploader (multipart for big `.tar.gz`)
- `cloud.google.com/go/storage` — GCS writer

**Files (new):**
- `pkg/uploader/uploader.go` — `Uploader` interface: `Upload(ctx, archivePath, summary) (*Result, error)` returning object key + ETag/VersionID
- `pkg/uploader/s3.go` — `S3Uploader` using aws-sdk-go-v2 manager.Uploader, presign disabled (server-side), honors `endpoint` for S3-compatible (MinIO, RustFS, R2, etc.)
- `pkg/uploader/gcs.go` — `GCSUploader` using cloud.google.com/go/storage writer
- `pkg/uploader/noop.go` — `NoopUploader` for `--no-upload` and tests
- `pkg/uploader/fanout.go` — same shape as `pkg/notifier`'s `FanOut` (if multiple providers configured, run all; first error propagates; continue if `continue_on_error: true`)
- `pkg/uploader/uploader_test.go` — table tests, httptest server mocks for S3 + GCS (use aws-sdk-go-v2 `s3` test fixtures, or handcrafted mock for simplicity — decide during impl)
- `pkg/uploader/redact_meta.go` — strip creds from `summary` before logging the upload result

**Files (modified):**
- `pkg/config/config.go`:
  - new `UploadConfig` struct: `Enabled bool`, `Provider string` (`s3`|`gcs`), `Bucket string`, `Region string`, `KeyPrefix string`, `Endpoint string` (S3-compatible), `StorageClass string` (S3), `ContentType string`, `CacheControl string`, `Metadata map[string]string`, `Timeout time.Duration`, `ContinueOnError bool`
  - S3-only: `SSE string` (e.g. `AES256`, `aws:kms`), `SSEKMSKeyID string`, `ACL string`
  - GCS-only: `KMSKey string`, `PredefinedACL string`
  - env map: `GROOT_UPLOAD_*` (top-level) + `AWS_*` / `GOOGLE_*` (provider SDK defaults)
- `pkg/cmd/root.go`:
  - new persistent flag `--no-upload`
  - new `var noUpload bool`
  - new helper `skipUploads()`
  - after the notify block (line ~181), if archive exists and `!skipUploads()`: call `uploader.NewFanOut(cfg).Upload(ctx, summary.ArchivePath, summary)`
  - log result: `logger.OK("uploaded: s3://bucket/key (size=%d etag=%s) …")` then any failure
- `pkg/config/sample.go` (`SampleYAML()`) + `configs/groot.yml.sample` — add `upload:` example with both providers commented
- `docs/SPECIFICATIONS.md` — new §10 **Upload** mirroring §4 (notify) layout: schema, env override, retry behavior, failure semantics, examples
- `README.md` — new section **Upload**, `GROOT_UPLOAD_*` env block, IAM/RBAC hint (min S3: `s3:PutObject` on bucket/prefix; GCS: `roles/storage.objectCreator`)
- `CHANGELOG.md` — bullet under `[Unreleased]` with `(0.6.x #28)` reference
- `docs/plan-0.6.0.md` — close checklist when shipped

**PR title:** `feat(upload): post-collect S3/GCS upload (0.6.x #28)`

**Failure semantics (must match notify):**
- Upload runs **after** the archive is written and **after** success notifications
- Upload failure **does not fail the collect** (the `.tar.gz` is already on disk; notifier failure already covered by 0.5.x `#19`)
- Errors are logged at `ERROR` level and surfaced in the summary
- `--no-upload` / `GROOT_NO_UPLOAD=1` skips entirely
- `continue_on_error: true` (default) tries all configured providers even if one fails

**Verification:**
- `make ci` + `make cover` (gate ≥ 80% merged, per existing `COVER_MIN`)
- new tests cover: skip path, success path, S3 + GCS, continue_on_error, env override, S3-compatible endpoint (point at minio local)
- integration test using a recorded request (e.g. `httptest.NewServer` returning 200 + ETag header)
- manual smoke: set `GROOT_UPLOAD_*` against a real bucket / GCS test bucket, run `make run`, confirm object appears with correct `Content-Type` and metadata

---

### #29 — BSD ports (FreeBSD + OpenBSD, full quality)

**Roadmap:** real, building, installable ports on both BSDs — not a skeleton.

#### FreeBSD — `contrib/freebsd/`

- `contrib/freebsd/Makefile` — Go port skeleton (Go 1.26.4, `go install` from upstream tag, `DISTNAME=groot-${PORTVERSION}`, `GO_MODULE=github.com/hrodrig/groot`, fetches source tarball from `MASTER_SITES=GH`, `WRKSRC` set so `go build ./cmd/groot` lands the binary)
- `contrib/freebsd/distinfo` — placeholder SHA256 hashes; CI step in `release.yml` runs `makesum` against the real source tarball and commits
- `contrib/freebsd/pkg-descr` — short description + `WWW` URL
- `contrib/freebsd/pkg-plist` — `/usr/local/bin/groot`, `/usr/local/etc/groot/groot.yml.sample`, `share/licenses/groot-*/LICENSE`, `share/doc/groot/README.md` (mirrors the `nfpms` layout; sample config is `sample|noreplace`)
- `contrib/freebsd/files/groot.in` — `rc.d` daemon template (NEWS+`-f` noop until 0.7.x adds watch mode; for now the script just `enable`s cleanly and `status` reports `groot not running` — same pattern as kzero `contrib/freebsd/`)
- `contrib/freebsd/README.md` — testing in a FreeBSD 14 jail, step-by-step: `git clone …/groot`, `cp -r contrib/freebsd /usr/ports/sysutils/groot`, `cd /usr/ports/sysutils/groot && make package`
- `deploy/README.md` — link to `contrib/freebsd/`

#### OpenBSD — `contrib/openbsd/`

OpenBSD port is **not** in the standard ports tree, but lives next to the project. Structure mirrors FreeBSD but uses OpenBSD conventions:

- `contrib/openbsd/Makefile` — OpenBSD port Makefile (`MODGO_MOD_NAME`, `MODGO_VERSION`, `GH_ACCOUNT=hrodrig GH_PROJECT=groot`, `WRKDIST=${WRKSRC}`, `MODGO_LDFLAGS="-X main.version=${GH_TAGNAME}"`); honors `MODGO_IMPORT_PATH` per OpenBSD Go ports policy
- `contrib/openbsd/distinfo` — placeholder SHA256 hashes; CI step regenerates
- `contrib/openbsd/pkg/DESCR` — short description (`D`irectory layout follows OpenBSD)
- `contrib/openbsd/pkg/PLIST` — `/usr/local/bin/groot`, `/usr/local/share/examples/groot/groot.yml.sample`, `/usr/local/share/doc/groot/README.md`, `/usr/local/share/doc/groot/LICENSE`, `/usr/local/share/doc/groot/CHANGELOG.md`
- `contrib/openbsd/README.md` — testing in an OpenBSD 7.x VM: `doas pkg_add go`, then `cd /usr/ports/sysutils/groot && make package` (once added) or `cd contrib/openbsd && make package` against the local copy
- `deploy/README.md` — link to `contrib/openbsd/`

#### CI for BSD ports

`.github/workflows/release.yml`:
- new `release-ports` job (BSD-friendly) using `vmactions/freebsd-vm@v1` and `vmactions/openbsd-vm@v1` (QEMU-backed, multi-arch)
- both jobs run only on `tags: ['v*']`
- FreeBSD: `cd contrib/freebsd && make package` produces `pkg` artifact, uploads to release
- OpenBSD: `cd contrib/openbsd && make package` produces `.tgz`, uploads to release
- on hash mismatch, CI runs `makesum` for the touched port and pushes a follow-up commit (so `distinfo` stays in sync)

#### ROADMAP

#29 flips to **Done (v0.6.0)** when the release tag publishes the first built port artifacts. Until then, the **PR** marks items **Done (PR#, develop)** with a "first build pending tag" note.

**PR title:** `chore(packaging): FreeBSD + OpenBSD ports (0.6.x #29)`

**Verification:**
- FreeBSD: `cd contrib/freebsd && make package` succeeds in a FreeBSD 14 jail, produces `Work/pkg/groot-0.6.0.pkg`
- OpenBSD: `cd contrib/openbsd && make package` succeeds in OpenBSD 7.5, produces `groot-0.6.0.tgz`
- `pkg install` / `pkg_add` of the artifact installs `groot` and runs `groot --version` successfully
- RC.d script (`groot.in`) parses with `service groot onestart` exit-0 (no-op until watch mode lands)

---

## Cross-cutting changes

- `README.md` — version badge, install section (brew + go install + deb/rpm + FreeBSD), upload section
- `docs/SPECIFICATIONS.md` — version bump in header, new §10, reference ROADMAP items still Pending
- `docs/ROADMAP.md` — after each PR lands, mark item **Done (PR#, develop)**; full band closure happens at tag time
- `CHANGELOG.md` — `[Unreleased]` gets one bullet per merged PR with `(0.6.x #N)` reference

## Dependency / binary size budget (for #28)

aws-sdk-go-v2 (s3 + config + credentials + manager) + cloud.google.com/go/storage adds ~20–30 MB to the static binary. This is acceptable for a one-shot CLI but worth flagging in `README`. If size becomes a problem, split into `groot-collect` (no upload) and `groot-upload` subcommands — punt to 0.7.x.

## Release checklist (at tag time, not now)

1. All 5 PRs merged to `develop`; `make ci` and `make release-check` green
2. `homebrew-groot` tap repo created and has write access for `HOMEBREW_TAP_TOKEN`
3. COSIGN secrets configured (or OIDC ready)
4. ROADMAP — all 0.6.x items **Done (v0.6.0)**; add **Shipped** row; set focus to **1.0.0**
5. CHANGELOG — `[0.6.0]` section with all `(0.6.x #N)` refs
6. VERSION → `0.6.0`; README version badge → `0.6.0`
7. Sample YAML and SPEC in sync
8. Merge `develop` → `main`
9. `git tag -a v0.6.0 -m "Release 0.6.0 — distribution, supply chain, and upload"` and push

## Success criteria for v0.6.0

| # | Criterion | Roadmap |
|---|-----------|---------|
| 1 | `brew install hrodrig/groot/groot` works on macOS arm64 | #25 |
| 2 | Release artifacts named `groot_vX.Y.Z_*` (matches pgwd/kzero) | #25 |
| 3 | `dist/*.spdx.json` and `dist/*.cyclonedx.json` attached to GitHub release | #26 |
| 4 | `cosign verify-blob --signature dist/checksums.txt.sig …` succeeds | #27 |
| 5 | `groot collect` with `upload.enabled: true` + S3 config uploads `.tar.gz` to bucket | #28 |
| 6 | Same with `provider: gcs` and GOOGLE_APPLICATION_CREDENTIALS env | #28 |
| 7 | `contrib/freebsd/Makefile` parses; `bmake package` succeeds in a FreeBSD 14 jail | #29 |
| 8 | `make release-check` green; merged coverage ≥ 80% | — |
