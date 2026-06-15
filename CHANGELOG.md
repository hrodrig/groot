# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- **SFTP post-collect upload (band 2 #36)**: `upload.sftp` pushes `.tar.gz` to a remote Linux host over SSH (public-key only, `known_hosts` verification, `BatchMode`). Env: `GROOT_UPLOAD_SFTP_*`. Same failure semantics as S3/GCS. Relay playbook → [groot-selfhosted](https://github.com/hrodrig/groot-selfhosted) `run/examples/airgapped-relay/`.

## [0.6.1] - 2026-06-15

### Changed

- **Documentation: product vs operator repo split** — Helm chart and flat CronJob manifests moved to **[groot-selfhosted](https://github.com/hrodrig/groot-selfhosted)**; README, SPEC links, and `AGENTS.md` clarify product scope. No CLI behavior change.
- **Homebrew cask (`0.6.x #25`)**: post-install hook clears macOS quarantine (`xattr -dr`) on staged binary.

## [0.6.0] - 2026-06-06

### Added

- **Homebrew cask tap (`0.6.x #25`)**: `homebrew_casks` in GoReleaser; release basenames `groot_vX.Y.Z_*` (`{{ .Tag }}`); cask in [homebrew-groot](https://github.com/hrodrig/homebrew-groot) + `scripts/update-homebrew-cask.sh`; README install block.
- **SBOM (`0.6.x #26`)**: SPDX-JSON (archives) + CycloneDX-JSON (binaries) via GoReleaser `sboms:`.
- **Cosign signing (`0.6.x #27`)**: keyless `signs` for `checksums.txt` + `docker_signs` for GHCR images; `cosign-installer` and `id-token: write` in release workflow.
- **Post-collect upload (`0.6.x #28`)**: `upload:` config (S3 + GCS), `pkg/uploader/`, `--no-upload` / `GROOT_NO_UPLOAD`; runs after notify; upload errors do not fail collect.
- **BSD ports (`0.6.x #29`)**: `contrib/freebsd/` + `contrib/openbsd/port/` (kzero/pgwd repackage pattern); FreeBSD/OpenBSD release tarballs via GoReleaser; `make port-*-sync` + `make dist-*`.
- **`docs/plan-0.6.0.md`**: implementation plan for **v0.6.0** (roadmap **0.6.x** items #25–#29).

### Changed

- **Release artifact basenames** break scripts that assumed `groot_0.5.0_*`; new form is `groot_v0.6.0_*` (README basename note).
- **`HOMEBREW_TAP_TOKEN`** passed to GoReleaser in `release.yml`.
- **Docs hygiene** post-**v0.5.0**: positioning copy and ROADMAP refresh (also noted under **[0.5.0]**).

## [0.5.0] - 2026-06-06

### Added

- **Notify on failure (`0.5.x #19`)**: `notify.on_failure` sends optional alerts when collect **aborts** (`on_abort`, default `true`) or when completed collects have **`failed >= min_failed_jobs`** (default `1`). Respects `--no-notify` / `GROOT_NO_NOTIFY`. Partial-failure alerts are **in addition to** the normal success notify.
- **Rich generic webhooks (`0.5.x #20`)**: `notify.generic.extra_fields`, `body_template` (JSON with `{{summary}}`, `{{total}}`, `{{failed}}`, `{{event}}`, … placeholders), and optional **HMAC-SHA256** signing (`hmac_secret`, `hmac_header`, env `GROOT_NOTIFY_GENERIC_HMAC_SECRET`).
- **Email / SMTP (`0.5.x #21`)**: `notify.email` channel (STARTTLS on port 587 by default; implicit TLS via `use_tls`). Env: `GROOT_NOTIFY_EMAIL_HOST`, `_USERNAME`, `_PASSWORD`, `_FROM`, `_TO`.
- **Notify HTTP retry (`0.5.x #24`)**: `notify.retry` (`max_attempts`, `initial_backoff`, `max_backoff`) retries transient **5xx** and network errors for webhook and PagerDuty HTTP clients.
- **Optional secret redaction (`0.5.x #23`)**: `collection.redact_secrets` (default `false`) scans collected `*.log` files with built-in patterns plus optional `redact_patterns` regex list; replaces matches with `[REDACTED]`.
- **In-cluster deploy (`0.5.x #22`)**: Helm chart at `deploy/helm/groot/` (CronJob, ClusterRole, ServiceAccount, ConfigMap, optional PVC for `/out`) and flat manifests at `deploy/k8s/cronjob.yaml`. Image: `ghcr.io/hrodrig/groot`.
- **`docs/plan-0.5.0.md`**: implementation plan for **v0.5.0** (roadmap **0.5.x** items #19–#24).

### Changed

- **Positioning copy**: README, SPEC, ROADMAP, and CLI `Short`/`Long` describe GROOT as a **read-only log and context collector** (not a diagnostics/diagnosis tool); ROADMAP honest gaps refreshed pre-**0.5.x**; **#25** documents **Homebrew cask** + pgwd/kzero release basename convention.
- **`configs/groot.yml.sample`** and **`SampleYAML()`** document new notify, retry, on_failure, and redaction keys.
- **`docs/SPECIFICATIONS.md`**: §4 notify schema, §7 failure alerts and retry, §9 configuration examples, §8 in-cluster deploy; redaction in §5.
- **`README.md`**: notifications (email, HMAC, on_failure), in-cluster deploy, secret redaction, corrected `file_prefix` / output naming; usage examples for `--list-jobs`, redaction, failure notify.
- **`deploy/`**: expanded READMEs with Helm and flat-manifest examples.
- **`docs/README.md`**: index for plan-0.5.0 and quick links to 0.5.x topics.
- **`docs/ROADMAP.md`**: **0.5.x** band closed; **Current focus** → **0.6.x**.

## [0.4.1] - 2026-06-06

### Fixed

- **`Dockerfile` builder image**: bump `golang` base from **1.26.3** to **1.26.4** to match `go.mod`; restores Security workflow Grype image build (`go mod download` was failing with `requires go >= 1.26.4`).

### Changed

- **README**: fixed-tag download examples and `go install` pin updated to **v0.4.1**.

## [0.4.0] - 2026-06-05

### Added

- **Archive manifest (`0.4.x #15`)**: every successful collect writes **`extras/manifest.json`** inside the archive with `groot_version`, `groot_commit`, `collected_at`, `duration_seconds`, `session_base`, `archive_basename`, `file_prefix`, `cluster` (context/cluster/user/server), `jobs` (total/success/failed), and a sorted `paths[]` listing of the captured files. CLI build metadata is injected from the linker via `SetBuildInfo`.
- **`groot collect --list-jobs` (`0.4.x #18`)**: prints planned collection jobs (name, output file, args, `optional`) and exits without writing the capture tree, without creating the `.tar.gz`, and without firing notify. Useful as an operator preview before a real run.
- **Broader `extra_kubectl` resources (`0.4.x #13`)**: `k8srunner` `get` and `describe` now support `configmap`/`cm`, `pvc`, `service`/`svc`, `ingress`/`ing`, and the apps workloads `deployment`/`deploy`, `replicaset`/`rs`, `statefulset`/`sts`, `daemonset`/`ds`. `get --raw <path>` remains the escape hatch for CRDs and generic reads. `explain` and `wait` are still rejected.
- **Job / CronJob log targets (`0.4.x #14`)**: `collection.targets.<ns>` accepts `jobs` and `cronjobs` lists in addition to `deployments` / `statefulsets` / `daemonsets` / `helm_releases`. Pod matching uses the same label keys as Deployments (`app.kubernetes.io/name`, `app.kubernetes.io/instance`, `app`) plus `job-name` for Job pods.
- **Kind E2E in CI (`0.4.x #17`)**: new optional GitHub Actions job `test-e2e-kind` runs `make test-e2e-kind` with `continue-on-error: true` so flakes and Docker variance don't block merges while the budget stabilizes.
- **`docs/plan-0.4.0.md`**: implementation plan for **v0.4.0** (roadmap **0.4.x** items #12–#18), merge order, and release checklist.

### Changed

- **`file_prefix` is now used in naming (`0.4.x #12`)**: `config.file_prefix` (default `groot-capture`) drives both the capture directory and the `.tar.gz` archive basename. Capture folder becomes `<file_prefix>-<timestamp>[-since-<slug>]`; archive becomes `<sessionBase>-<cluster>[-<message>].tar.gz`. Empty value falls back to the default.
- **Docs hygiene (`0.4.x #16`)**: `pkg/config/sample.go` (`SampleYAML()`) and `configs/groot.yml.sample` are now in sync; comments no longer say "kubectl" — wording reflects the **client-go** runtime. `docs/SPECIFICATIONS.md` updated for `--list-jobs`, `file_prefix`, Job/CronJob targets, archive manifest, and the broader `extra_kubectl` resource set.

### Fixed

- `pkg/cmd.ResetPersistentCLI` now also resets `collectCmd` flags, preventing `--list-jobs` (and other collect-local flags) from leaking across tests in the same package.

## [0.3.2] - 2026-06-05

### Security

- Address **GO-2026-5026** / **CVE-2026-39821** (`golang.org/x/net` IDNA Punycode) by upgrading `golang.org/x/net` to v0.55.0 and related `golang.org/x/*` modules via `go mod tidy`.

### Changed

- Go toolchain directive **1.26.3 → 1.26.4** in `go.mod`.

## [0.3.1] - 2026-05-14

**Release note:** **`v0.3.0` was never published** (no git tag and no GitHub Release). **`v0.3.1`** is the first tagged release on that line of work: it includes **everything** listed under **[0.3.0]** below, plus the **0.3.1** items in this section. When upgrading or auditing, treat the **[0.3.0]** section as shipped starting with **`v0.3.1`**.

**`develop` commits (audit):** after **`9c3a6c2`** (*feat(collector): pod RCA table…*), **`65b6df5`** (*fix(config): do not auto-load /etc/groot/groot.yml.sample*), then **`c9788e2`** (*Release v0.3.0 — client-go diagnostics, security, changelog* — sets `VERSION` / docs for **0.3.0**, still **without** a `v0.3.0` tag). Further commits on **`develop`** finish **0.3.1** (kind E2E, README, VHS demo, `docs/badges.md`, `VERSION` bump). Only **`v0.3.1`** was tagged from **`main`** at the merge tip.

### Added

- `testing/`: kind-based E2E (`make test-e2e-kind`, alias `e2e-kind`), `testing/scripts/test-e2e-kind.sh`, `testing/k8s/e2e-workload.yaml`, and `testing/README.md` (Docker probe with optional `GROOT_DOCKER_WAIT_SECS`, optional archive copy via `GROOT_E2E_ARCHIVE`, empty `nodes/` note for the slim config).
- `docs/e2e-kind.md`; `scripts/e2e-kind.sh` forwards to `testing/scripts/test-e2e-kind.sh` (maps `KIND_CLUSTER_NAME` to `GROOT_E2E_CLUSTER` when set).

## [0.3.0] - 2026-05-14

**Not released:** this version appears in the changelog for traceability, but there was **no `v0.3.0` tag**—see the note at **[0.3.1]**.

### Added

- `CHANGELOG.md` for release notes.
- `pkg/k8srunner`: allowlisted read-only diagnostics executed via **client-go** (argv shaped like familiar kubectl verbs, without the kubectl binary).
- `pkg/kubetest`: minimal fake Kubernetes API HTTP server for tests.
- Unit tests for `k8srunner`, `kubeloader`, and `kubetest`; merged coverage gate remains at 80% with headroom.

### Changed

- Kubernetes collection uses **client-go** / **metrics** APIs end-to-end; **no `kubectl` binary** is required at runtime.
- Per-namespace `resources.txt` captures namespace-scoped workload objects as **JSON sections** (pods, services, Deployments, ReplicaSets, StatefulSets, DaemonSets) instead of a single `kubectl get all` text dump.
- `collection.extra_kubectl` validation: allowlist matches implementation (`get`, `describe`, `top`, `logs`, `api-resources`, `api-versions`, `version`, `cluster-info`, `config view …`, `auth can-i …` only).
- `Runner.Metrics` is `metricsversioned.Interface` so tests can inject a metrics fake client.
- Dependency bumps aligned with **govulncheck** / **grype**: `golang.org/x/net` v0.53.0, `golang.org/x/oauth2` v0.27.0, and related `golang.org/x/*` modules via `go mod tidy`.

### Fixed

- `config view` handling: accept `config view` with no extra tokens; apply `-o` / `--output` flags from argv after `view` (previously used the wrong slice and required an extra argument).

### Removed

- `pkg/kubemock` (unused kubectl shim for tests).

### Security

- Address **GO-2026-4918** (`golang.org/x/net` HTTP/2) and **GHSA-6v2p-p543-phr9** (`golang.org/x/oauth2`) by upgrading the affected modules.
