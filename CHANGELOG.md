# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- **man(1) + packaging (1.1.x #96 / GH #2):** ship `contrib/man/man1/groot.1` and `kubectl-groot.1`; GoReleaser archives + nfpm (`.deb`/`.rpm`) install man pages; Homebrew cask `manpages:`; FreeBSD/OpenBSD ports install the same pages; `make man-sync` bumps `.TH` from `VERSION`.
- **CONTRIBUTING collector guide (1.1.x #47):** how jobs are wired (`buildJobs` / `k8s_exec` / `k8srunner`), when to prefer `extra_kubectl`, test layers, and a PR checklist.
- **Examples beyond profiles (1.1.x #46):** `examples/README.md` index; notify smokes (Slack/Teams/generic/PagerDuty); upload skeletons (S3/GCS/SFTP); collection (`targets`/`extra_kubectl`, redact); GKE/AKS managed profiles.

## [1.0.5] - 2026-08-01

Docs patch — Homebrew 6+ tap trust guidance; no CLI or config contract changes.

### Changed

- **Docs (Homebrew 6+):** README install block documents [tap trust](https://docs.brew.sh/Tap-Trust) — prefer fully qualified `brew install --cask hrodrig/groot/groot`; explain unrelated “Skipping … not trusted” warnings during auto-update.
- **README / BSD ports:** version badge and package pins synced to **1.0.5**.
- **VHS demo** (`docs/demo.gif`): regenerated at **v1.0.5** so `groot --version` matches the release.

## [1.0.4] - 2026-07-29

Security patch — `google.golang.org/grpc` **v1.82.1** (Dependabot #1) and OpenTelemetry **v1.44.0**; no CLI or config contract changes.

### Security

- **Dependencies:** bump `google.golang.org/grpc` to **v1.82.1** (Dependabot #1 / [GHSA-hrxh-6v49-42gf](https://github.com/advisories/GHSA-hrxh-6v49-42gf) — xDS RBAC + HTTP/2 Rapid Reset). Transitive via GCS client; groot does not expose a gRPC server.
- **Dependencies:** bump `go.opentelemetry.io/otel` (and `metric`/`trace`/`sdk`) to **v1.44.0** — closes grype **GO-2026-5158** / [GHSA-5wrp-cwcj-q835](https://github.com/advisories/GHSA-5wrp-cwcj-q835) (baggage header length). Transitive via GCS; govulncheck reports no vulnerable call path in groot.

### Fixed

- **`-v` / `--version`**: no longer prints empty `Error:` plus full Usage after the version line (cobra treated `ErrVersionPrinted` as a PreRun failure). Align with kzero: root `SilenceUsage`/`SilenceErrors`; `main` prints real errors to stderr.
- **macOS Gatekeeper**: Homebrew cask post-install clears quarantine on **`kubectl-groot`** as well as **`groot`**; README documents both `xattr` one-liners.

### Changed

- **ROADMAP**: Kimi review — band status table + close dates, **Current focus** (in flight), explicit **1.0.0 contract surface**, Band 4 **themes** + community signals, expanded **Known gaps / Non-goals**, stronger SPEC triad note. Product stance: **`stream` out of philosophy**; **`watch`** reframed as triggered collect → upload → notify (#55).
- **README**: version badge and container pull pin updated to **v1.0.4**.
- **BSD ports**: `PORTVERSION` / `DISTNAME` synced to **1.0.4**.

## [1.0.3] - 2026-07-12

Maintenance patch — post-audit hygiene (#88–#95); **`groot notify test`** adds one CLI subcommand (no config schema change).

### Fixed

- **Container image default CMD** prints `--help` instead of running `collect` against the bundled sample config (`groot.yml.sample` remains in the image for copy/reference). `(1.0.3 #88)`

### Security

- **Dependencies:** bump `golang.org/x/crypto` to **v0.54.0** (and transitive `golang.org/x/net`, `golang.org/x/sys`, `golang.org/x/text`) — grype GO-2026-5932 hygiene; govulncheck reports no vulnerable call paths. `(1.0.3 #94)`

### Changed

- **Tests:** `internal/cmd` `TestMain` removes stale `out/groot-capture-*` dirs after package tests (gitignored artifacts from local cmd runs). `(1.0.3 #91)`
- **`--version` / `-v`:** always returns through `ErrVersionPrinted` and `ExecuteContext` (no `os.Exit(0)` in library code). `(1.0.3 #92)`
- **Docs:** Kubernetes client **QPS 50 / burst 100** vs `collection.worker_concurrency`; YAML-tunable rate limit remains roadmap **#67**. `(1.0.3 #93)`
- **README**: version badge and container pull pin updated to **v1.0.3**.
- **BSD ports**: `PORTVERSION` / `DISTNAME` synced to **1.0.3**.
- **VHS demo** (`docs/demo.gif`): refreshed at **v1.0.3** (`groot notify test` in tape).

### Added

- **Tests:** fake SMTP server coverage for email notifier (plain, STARTTLS, implicit TLS, AUTH PLAIN, FanOut integration). `(1.0.3 #89)`
- **Tests:** GCS upload via `fake-gcs-server` + `STORAGE_EMULATOR_HOST`; covers `gcsClientOptions`, success path, cancel, missing archive. `(1.0.3 #90)`
- **`groot notify test`** — send a synthetic summary to all enabled notify channels without running collect or contacting the cluster; `--event notify.test|success|failure`; exit **4** on delivery failure. `(1.0.3 #95)`
- **Docs:** Mailgun/SMTP notify smoke runbook ([`docs/notify-smoke-test.md`](docs/notify-smoke-test.md)) and [`examples/notify/mailgun-smoke.yml`](examples/notify/mailgun-smoke.yml). `(1.0.3 #95)`

## [1.0.2] - 2026-07-11

Maintenance patch — distroless **Debian 13** runtime base (no CLI or config contract changes).

### Changed

- **Container runtime base**: `gcr.io/distroless/static-debian12:nonroot` → **`static-debian13:nonroot`** in `Dockerfile` and `Dockerfile.release` (Debian 12 EOL; distroless defaults to trixie).
- **README**: version badge and container pull pin updated to **v1.0.2**.
- **BSD ports**: `PORTVERSION` / `DISTNAME` synced to **1.0.2**.

## [1.0.1] - 2026-07-10

Security patch — Go toolchain **1.26.5** (no CLI or config contract changes).

### Security

- **Go 1.26.4 → 1.26.5** in `go.mod` and **`Dockerfile`** builder image: **CVE-2026-39822** (`os` root escape via symlink + trailing slash) and **CVE-2026-42505** (`crypto/tls` ECH privacy leak).

### Changed

- **README**: version badge and container pull pin updated to **v1.0.1**.
- **VHS demo** (`docs/demo.gif`): refreshed at **v1.0.1**.
- **BSD ports**: `PORTVERSION` / `DISTNAME` synced to **1.0.1**.

## [1.0.0] - 2026-07-03

Stable contract release (Band 3 — [plan-1.0.0.md](docs/plan-1.0.0.md)).

### Added

- **`config_version`** in YAML with loader validation; `1` is supported, absent/`0` = legacy pre-1.0 configs (1.0.0 #30).
- **`archive_layout_version`** in `extras/manifest.json` on every new collect (1.0.0 #34).
- **`groot collect --output json`** emits `Summary` JSON on success; `validate` / `inspect` already supported `--output json` from 0.9.x (1.0.0 #40).
- **Golden inspect fixture test** for archive regression without a cluster (1.0.0 #87).
- **Governance**: `.github/CODEOWNERS`, issue templates, PR template (1.0.0 #48).

### Changed

- **`pkg/` → `internal/`** module layout; groot is a CLI, not a public Go SDK — no CLI flag changes (1.0.0 #35).
- **SFTP upload**: `known_hosts_file` is **required** when SFTP is enabled unless `allow_insecure_host_key: true` (testing only). Previously defaulted to insecure host key verification.
- **VHS demo** (`docs/demo.gif`): regenerated at **v1.0.0** (`groot --version`, validate, completion, sample config).
- **README hero** (`docs/assets/groot-readme-hero.png`): refreshed for **v1.0** (stable contract, structured output, inspect).

### Fixed

- **Notifier HTTP client**: per-instance retry config (no global mutable state).
- **Archive tar walk**: close file descriptors per file (no FD leak on large trees).
- **Control-plane pod list**: log API failures instead of silent omission.
- **Manifest paths**: cache directory walk between pre/post-archive manifest writes.
- **SIGINT/SIGTERM**: graceful context cancellation from `main`.
- **SFTP config**: validate `user` at load time.

### Migration

- New configs: set `config_version: 1` at the top of `groot.yml`.
- Legacy configs without `config_version` continue to load unchanged.
- New archives include `archive_layout_version: 1` in `extras/manifest.json`.
- SFTP operators: set `upload.sftp.known_hosts_file` (or `GROOT_UPLOAD_SFTP_KNOWN_HOSTS`) before upgrading if you relied on implicit insecure host keys.

## [0.9.2] - 2026-06-29

### Fixed

- **OpenBSD release build**: `diskfree_openbsd.go` uses `F_bsize` / `F_blocks` / `F_bavail` field names; GoReleaser `GOOS=openbsd` targets compile after **v0.9.1** release failure.

### Changed

- **VHS demo** (`docs/demo.gif`): tape runs `make install`; regenerate with `PATH="$(go env GOPATH)/bin:$PATH" vhs docs/demo.tape` so Homebrew/system installs do not shadow the repo binary; shows **0.9.x** subcommands (`validate`, `completion`).

## [0.9.1] - 2026-06-29

### Fixed

- **Windows release build**: split `diskFree` into `diskfree_unix.go` (`syscall.Statfs`) and `diskfree_windows.go` (`GetDiskFreeSpaceEx`) so GoReleaser cross-compiles for `GOOS=windows` after `groot validate` disk preflight landed in **v0.9.0**.

## [0.9.0] - 2026-06-28

### Fixed

- **Helm `helm_releases` target matching (0.9.x #79)**: `matchesTargetsByLabels` now honours Helm-canonical labels (`app.kubernetes.io/instance` and `app.kubernetes.io/name`) and rejects pods whose `app.kubernetes.io/managed-by` declares a non-Helm owner (Kustomize, Operators). Legacy Helm 2 (`heritage=Tiller` + `release=<name>`) is also recognised. New `helmMatches` helper with 12 case-driven unit tests covering modern, legacy, non-Helm rejection, and edge cases; `matchesTargetsByLabels` now routes Helm targets through it.

### Added

- **kubectl-groot plugin (0.9.x #64, #85)**: same `cmd/groot/main.go` now ships as both `groot` and `kubectl-groot` from GoReleaser (`builds:` second entry, `binary: kubectl-groot`; archives, nfpms, container images, and Homebrew cask all include both binaries in the same artefacts). New `pkg/cmd/plugin.go` exposes `IsPluginInvocation()` (basename match on `kubectl-groot` or explicit `GROOT_FORCE_KUBECTL_PLUGIN=1`); `FormatVersion` swaps the banner between `groot` and `kubectl-groot` so logs tell which entry point fired (greeting stays `I am Groot` for both). `make install-kubectl-plugin` lays the binary down as `kubectl-groot` on a ``$PATH``-resident `PREFIX` and refuses otherwise. Krew plugin manifest lives at `contrib/krew/groot.yaml` with a manual submission runbook at `contrib/krew/SUBMISSION.md`. README has a full plugin-install section with Homebrew, local build, and Krew examples. 12 new unit tests in `pkg/cmd/plugin_test.go` + 2 version-banner tests in `pkg/cmd/version_test.go`.
- **Shell completion (0.9.x #80)**: new `groot completion <bash|zsh|fish|powershell>` subcommand wraps Cobra's generators with strict arg validation. Defaults to writing the script to **stdout** so it installs with a redirect (e.g. `groot completion zsh > "${fpath[1]}/_groot"`, `groot completion bash | sudo install -m 0644 /dev/stdin /etc/bash_completion.d/groot`); errors and unsupported-shell messages go to **stderr**. Listed in root `--help`. 8 unit tests in `pkg/cmd/completion_test.go`.
- **Exit code taxonomy (0.9.x #82)**: stable codes for scripting — `0` success, `1` config validation (YAML / `--since` / missing config file), `2` Kubernetes client or API error, `3` collect aborted (timeout / archive failure), `4` notify delivery failed; `5` reserved for partial-job-failure `--strict` opt-in. New `pkg/cmd/exitcode.go` defines `ExitError`, `ExitCodeOf`, and the five `Exit*` constants; `pkg/cmd/root.go` RunE funnels every error through `NewExitError` so the code is preserved across `%w` wrappers. `cmd/groot/main.go:exitCode` now delegates to `cmd.ExitCodeOf` (was binary 0/1). SPEC §3 documents the contract. 9 unit tests in `pkg/cmd/exitcode_test.go` + 3 end-to-end tests in `pkg/cmd/exitcode_cli_test.go` + 3 e2e tests for `--strict`.
- **`groot validate` and `groot inspect <archive>` (0.9.x #31, #83)**: `pkg/collector/preflight.go` adds `Service.Preflight(ctx) PreflightResult` (config + cluster handshake + RBAC `auth can-i` matrix + disk space, configurable via `collection.min_free_bytes` / `collection.warn_free_bytes`, defaults 256 MiB fail / 1 GiB warn). New `groot validate` subcommand prints the findings as text or JSON, exits 1 on config/disk/RBAC failure or 2 on Kubernetes-API failure. `pkg/collector/inspect.go` adds `InspectArchive(path)` (no cluster; reads `extras/manifest.json` from a `.tar.gz`). New `groot inspect <archive>` subcommand summarises an archive. 9 preflight tests + 5 inspect tests + 8 render tests + 7 e2e CLI tests.
- **`groot collect --summary` and signal-first jobs (0.9.x #42, #84)**: `--summary` flag writes a one-screen block after a successful collect (total / success / failed jobs, archive path, duration, plus counts of pods in `CrashLoopBackOff` / `ImagePullBackOff` / `OOMKilled` / `Pending` via `Service.CountUnhealthyPods`). Signal-first job ordering is **on by default** (use `SetHighSignalFirst(false)` to disable); moves `events-warning`, `events-all`, `cluster-info`, `nodes-wide`, `pods-all` to the head of the queue so operators see actionable signal in seconds. 6 summary tests + 4 reorder tests + 3 tally tests.
- **`run_id` + `archive_sha256` in manifest (0.9.x #81)**: `Service.Run` now generates a stable per-run identifier (format `YYYYMMDDTHHMMSSZ-<base32>`) and SHA-256 of the final `.tar.gz`; both surface in `extras/manifest.json` as `run_id` and `archive_sha256`. The manifest is re-emitted after the archive is hashed so the checksum always agrees with the bytes on disk. `Summary.RunID` propagated to notify `{{run_id}}` placeholder; S3/GCS upload metadata tags include `run_id`; SFTP remote filename includes `run_id` suffix. 4 unit tests in `pkg/collector/runid_test.go`.

### Changed

- **Planning triad refocus**: trimmed Band 3 to **1.0.0 contract only** (#30, #34, #35, #40, #48, #87); new **Band 0.9.x** for operator wins ([plan-0.9.0.md](docs/plan-0.9.0.md): #31, #42, #60, #64, #79–#86); deferred multi-cluster, stream, analyze, and 30+ items to Band 4 backlog. SPEC scope sections aligned. (ROADMAP planning)

## [0.8.0] - 2026-06-17

### Added

- **Workload resource RCA extras (0.8.x #39)**: **`extras/workload-resources.tsv`** — per-container CPU/memory requests and limits for every pod, plus controller owner kind/name. **`extras/all-pods-rca.tsv`** adds pod-level `cpu_request`, `cpu_limit`, `memory_request`, and `memory_limit` columns (summed across init + app containers) alongside usage metrics and log paths.

## [0.7.2] - 2026-06-17

### Changed

- **Node log capture**: primary host logs via `…/proxy/logs/messages` → `nodes/<node>.log`; optional kubelet Node Log Query → `nodes/<node>-kubelet.log`. Both job types are optional so managed clusters without log query (e.g. AKS) no longer report spurious job failures.
- **`--version` output**: prints `groot vX.Y.Z (commit=… branch=… built=…)` (aligned with [vision](https://github.com/hrodrig/vision)). New `groot version` subcommand.

## [0.7.1] - 2026-06-16

### Added

- **`cluster_name` config and in-cluster archive naming**: optional `cluster_name:` overrides the archive basename cluster segment. When empty, Groot resolves kubeconfig cluster metadata, then `kube-public/cluster-info`, then the API server host, before `unknown-cluster`. Helps in-cluster CronJobs and on-demand pods without kubeconfig context.

## [0.7.0] - 2026-06-15

### Added

- **SFTP post-collect upload (band 2 #36)**: `upload.sftp` pushes `.tar.gz` to a remote Linux host over SSH (public-key only, `known_hosts` verification, `BatchMode`). Env: `GROOT_UPLOAD_SFTP_*`. Same failure semantics as S3/GCS. Relay playbook → [groot-selfhosted](https://github.com/hrodrig/groot-selfhosted) `run/examples/airgapped-relay/`.
- **Airgapped relay playbook (band 2 #37)**: `run/examples/airgapped-relay/` in groot-selfhosted — end-to-end topology bastion → SFTP → rclone → OneDrive, systemd watcher, SSH hardening. No groot code.
- **Container image SBOM (band 2 #38)**: OCI attestation enabled on `dockers_v2` (`sbom: true`); release workflow uses buildx `docker-container` driver.

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
