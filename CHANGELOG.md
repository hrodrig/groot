# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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
