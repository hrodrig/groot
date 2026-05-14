# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.3.0] - 2026-05-14

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
