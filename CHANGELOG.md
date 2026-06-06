# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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
