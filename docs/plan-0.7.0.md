# Plan 0.7.0 — airgapped upload and supply-chain follow-up

**Status:** **shipped** — **`v0.7.0`** tagged on `main` (2026-06-15)  \
**Target release:** `v0.7.0`  \
**Roadmap band:** [ROADMAP.md](ROADMAP.md) **0.7.x** items **#36–#38**  \
**Delivery model:** one PR per item on `develop`. No tag until release checklist passes.

---

## Why 0.7.0

After **0.6.x** (S3/GCS upload, SBOM on archives, Cosign, BSD ports), operators in **restricted networks** still copy `.tar.gz` files by hand. A common pattern:

| Hop | Role | Network |
|-----|------|---------|
| **AKS** | Workload cluster | No outbound internet |
| **Bastion** | `groot collect` + kubeconfig | Internal API only; **one** allowed SSH egress to relay |
| **ipA (relay)** | Linux inbox | SSH from bastion; **has internet** |
| **OneDrive** | Long-term storage | Sync from ipA via **rclone** (not groot) |

**0.7.x** closes the bastion → relay hop inside groot (**#36**), documents the relay → OneDrive hop (**#37**), and finishes **container image SBOM** deferred from **0.6.x #26** (**#38**).

The product stays a **single command** (`groot collect`); no new top-level command.

## Agreed implementation order

`#36 → #37 → #38`

- **#36** is the operator-facing feature (SCP/SFTP upload).
- **#37** ships the playbook for the AKS+bastion+ipA+OneDrive topology (contrib only).
- **#38** is release-pipeline work; can land in parallel after #36 if CI time is a concern, but merge after #36 so upload docs stay the release headline.

---

## Item breakdown

### #36 — SCP/SFTP post-collect upload

**Roadmap:** push `.tar.gz` to a remote Linux host over SSH after collect (bastion runtime).

**Use case (reference):** bastion runs groot; `upload.scp` copies the archive to `groot-inbox@ipA:~/inbox/`; ipA watcher (see **#37**) moves files to OneDrive.

**Dependencies:**

- `golang.org/x/crypto/ssh` — SSH client
- `github.com/pkg/sftp` — SFTP (optional path; SCP via `ssh` session is enough for v1 — pick one implementation, prefer **SFTP** for resume-friendly large tars and explicit remote paths)

**Config shape (`pkg/config/config.go`):**

```yaml
upload:
  enabled: true
  continue_on_error: true
  timeout: 10m
  scp:   # or sftp: — one block; name TBD in SPEC (see below)
    enabled: true
    host: ipA.example.com
    port: 22
    user: groot-inbox
    remote_dir: inbox          # relative to remote home or absolute path
    # identity_file: set via env only
    known_hosts_file: /etc/groot/known_hosts
```

**Env overrides (no secrets in YAML):**

| Env | Purpose |
|-----|---------|
| `GROOT_UPLOAD_SCP_HOST` | Relay hostname or IP |
| `GROOT_UPLOAD_SCP_USER` | SSH user |
| `GROOT_UPLOAD_SCP_REMOTE_DIR` | Remote directory |
| `GROOT_UPLOAD_SCP_IDENTITY_FILE` | Path to private key (bastion `~/.ssh/id_ed25519`) |
| `GROOT_UPLOAD_SCP_KNOWN_HOSTS` | `known_hosts` path (required in production) |
| `GROOT_UPLOAD_SCP_PORT` | Default `22` |

**Implementation (`pkg/uploader/`):**

- `scp.go` or `sftp.go` — `SCPUploader` / `SFTPUploader` implementing existing `Uploader` interface
- `NewFanOut` — register when `upload.scp.enabled` (or `upload.sftp.enabled`)
- `ShouldUpload` — true when SCP/SFTP enabled alongside S3/GCS
- Remote path: `remote_dir` + `/` + `filepath.Base(archivePath)` (same key logic as S3 `objectKey`)
- **BatchMode:** refuse password / keyboard-interactive auth; only public-key
- **Host key:** verify against `known_hosts_file`; fail closed if missing or mismatch
- **Permissions:** create remote file `0600` where the protocol allows

**Failure semantics (match 0.6.x #28):**

- Runs after archive write and success notify
- Upload failure **does not fail** collect
- `--no-upload` / `GROOT_NO_UPLOAD=1` skips all providers
- `continue_on_error: true` tries remaining providers

**Files (modified):**

- `pkg/config/config.go`, `pkg/config/sample.go`, `configs/groot.yml.sample`
- `pkg/uploader/uploader.go`, `pkg/uploader/uploader_test.go`
- `pkg/cmd/root.go` — no change if FanOut already wired
- `docs/SPECIFICATIONS.md` — extend §10 with SCP/SFTP table + bastion example
- `README.md` — **Airgapped upload** subsection linking to **#37** playbook
- `CHANGELOG.md` — `(0.7.x #36)` under `[Unreleased]`

**PR title:** `feat(upload): SCP/SFTP post-collect upload (0.7.x #36)`

**Verification:**

- Unit tests: mock SSH server or `httptest` + `x/crypto/ssh` test keys; missing `known_hosts`, wrong host key, success path, timeout
- `make ci` + `make cover` ≥ 80%
- Manual: bastion → test VM, `groot collect` with `upload.scp.enabled: true`, file appears in `~/inbox/`

**Naming note:** Prefer a single YAML block `upload.sftp` (SFTP protocol) with alias doc "SCP-compatible relay" — operators often say "scp to ipA" but SFTP over SSH is the maintainable Go API. SPEC documents both user-facing terms.

---

### #37 — Airgapped relay playbook (bastion → ipA → OneDrive)

**Roadmap:** operational docs + sample units; **no groot code**.

**Deliverables (`contrib/relay/`):**

| File | Purpose |
|------|---------|
| `README.md` | Topology diagram, prerequisites, security checklist |
| `groot-bastion.yml.example` | `output_dir`, `upload.sftp` pointing at ipA |
| `systemd/groot-inbox.path` | Trigger on `~/inbox/*.tar.gz` |
| `systemd/groot-inbox-upload.service` | `rclone move` to OneDrive remote |
| `ssh/hardening.md` | Dedicated user, `authorized_keys`, `ForceCommand` / chroot optional |

**One-time ipA setup (documented, not automated by groot):**

1. `rclone config` → remote `onedrive` (OAuth in browser on ipA)
2. Create user `groot-inbox`, `~/inbox/`, bastion public key in `authorized_hosts`
3. Enable `groot-inbox.path` + `groot-inbox-upload.service`

**PR title:** `docs(0.7.x #37): airgapped relay playbook (bastion → SSH → OneDrive)`

**Verification:**

- Markdown review; example YAML validates against config loader after **#36** lands
- Optional: dry-run `rclone lsd onedrive:` documented as smoke step

---

### #38 — Container image SBOM attestation

**Roadmap:** OCI SBOM on `ghcr.io/hrodrig/groot` images (deferred from **0.6.x #26**).

**Files:**

- `.goreleaser.yaml` — `dockers_v2[].sbom: true` when CI driver supports it
- `.github/workflows/release.yml` — ensure buildx uses `docker-container` driver (QEMU step already present); document fallback if driver cannot attest
- `README.md` — verify SBOM attestation with `cosign verify-attestation` example

**PR title:** `release(0.7.x #38): container image SBOM attestation`

**Verification:**

- Snapshot or tag build attaches SBOM attestation to multi-arch manifest
- Archive SBOM unchanged (still SPDX + CycloneDX from **#26**)

---

## Cross-cutting changes

- `docs/ROADMAP.md` — mark items **Done (v0.7.0)** at tag time
- `CHANGELOG.md` — `[0.7.0]` section with `(0.7.x #N)` refs
- `VERSION` → `0.7.0` at release

## Release checklist (at tag time)

1. **#36–#38** merged; `make release-check` green
2. ROADMAP — 0.7.x items **Done (v0.7.0)**; focus → **1.0.0**
3. Merge `develop` → `main`; tag `v0.7.0`

## Success criteria for v0.7.0

| # | Criterion | Roadmap |
|---|-----------|---------|
| 1 | `groot collect` on bastion with `upload.sftp` pushes `.tar.gz` to relay `~/inbox/` | #36 |
| 2 | Upload fails closed on unknown SSH host key | #36 |
| 3 | `contrib/relay/README.md` documents bastion → ipA → rclone OneDrive end-to-end | #37 |
| 4 | GHCR image has OCI SBOM attestation (or documented CI fallback) | #38 |
| 5 | `make release-check` green; merged coverage ≥ 80% | — |
