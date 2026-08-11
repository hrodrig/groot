---
last_mapped_commit: 805eba0
analysis_date: 2026-08-10
---

# Testing Patterns

**Analysis Date:** 2026-08-10

## Test Framework

**Runner:**
- Go `testing` package (stdlib)
- Config: Makefile targets (`test`, `cover`, `ci`, `release-check`); CI in `.github/workflows/ci.yml`
- Go version from `go.mod` (`go 1.26.5`) via `actions/setup-go` `go-version-file`

**Assertion Library:**
- Stdlib only (`t.Fatal`, `t.Fatalf`, `t.Errorf`, `t.Error`)
- `github.com/google/go-cmp/cmp` for structured diffs (notably `internal/k8srunner/runner_test.go`)
- No testify / `require` / `assert` packages

**Run Commands:**
```bash
make test                 # go test -count=1 ./... -race (CI parity)
make cover                # race + atomic coverprofile=coverage.out; gate COVER_MIN (default 80; needs bc)
make ci                   # fmt-check + lint (vet) + gocyclo + test
make release-check        # VERSION semver + goreleaser check + fmt-check + lint + cover + security
make release-check STRICT_RELEASE=1   # also docker-scan
make test-e2e-kind        # build + kind cluster E2E (alias: e2e-kind)
make lint-fix && make ci  # recommended local PR loop (CONTRIBUTING.md)
```

## Test File Organization

**Location:**
- Co-located with source: `internal/<pkg>/*_test.go`, `cmd/groot/main_test.go`
- ~40 `*_test.go` files across `cmd/groot`, `internal/archive`, `cmd`, `collector`, `config`, `k8srunner`, `kubeloader`, `kubetest`, `logx`, `notifier`, `uploader`
- Product E2E (kind): `testing/scripts/test-e2e-kind.sh` + `testing/k8s/e2e-workload.yaml` (see `testing/README.md`, `docs/e2e-kind.md`)
- No committed `testdata/` tree for golden archives — golden fixture is built in-test

**Naming:**
- `*_test.go` same package as production code (white-box)
- Integration: `*_integration_test.go` or descriptive CLI e2e names (`validate_inspect_e2e_test.go`)
- Golden: `inspect_golden_test.go` (`TestInspectArchive_goldenFixture`, ROADMAP **#87**)

**Structure:**
```
cmd/groot/main_test.go
internal/<package>/*_test.go          # unit + package integration
internal/kubetest/                    # shared fake K8s API HTTP server
testing/
  k8s/e2e-workload.yaml
  scripts/test-e2e-kind.sh
  README.md
```

## Test Structure

**Suite Organization:**
```go
func TestExitCodeOf_exitError(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want int
	}{
		{"config", NewExitError(ExitConfigError, errors.New("bad yaml")), ExitConfigError},
		{"kubernetes", NewExitError(ExitKubernetesError, errors.New("client")), ExitKubernetesError},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ExitCodeOf(tc.err); got != tc.want {
				t.Fatalf("got %d want %d", got, tc.want)
			}
		})
	}
}
```

**Patterns:**
- Prefer focused `TestXxx_yyy` functions; table-driven where many inputs share logic (`exitcode_test.go`, `k8srunner` `cases := []struct{...}`)
- Helpers call `t.Helper()` (`resetPersistentFlags`, `kubetest.StartAPIServer`)
- Temp dirs: `t.TempDir()`; env: `t.Setenv`; cleanup: `t.Cleanup` / `defer cleanup()`
- CLI tests: `resetPersistentFlags(t)` then `rootCmd.SetArgs` / `SetOut` / `SetErr` / `Execute()` (`internal/cmd/root_test.go`)
- Parallel: used heavily in `internal/k8srunner/runner_test.go` via `t.Parallel()` at test start; most other packages stay sequential (CLI/global flag state)

## Mocking

**Framework:** Interfaces + fakes — no mock codegen

**Patterns:**
```go
// Fake Kubernetes API (httptest) + kubeconfig on disk
kc, cleanup := kubetest.StartAPIServer(t)
defer cleanup()

// client-go fake clientset for in-memory objects
cs := fake.NewSimpleClientset(/* objects... */)

// Custom httptest / ServeMux for RBAC / edge cases (validate_inspect_e2e_test.go)
```

**What to Mock:**
- Cluster API: `internal/kubetest` for collect/validate/inspect CLI paths
- Object-level APIs: `k8s.io/client-go/kubernetes/fake` (+ reactors) in collector/k8srunner tests
- Cloud/upload edges: package-local fakes / `fake-gcs-server` where used in uploader tests

**What NOT to Mock:**
- Archive/tar.gz and inspect path logic — exercise real `archive.DirToTarGz` + `InspectArchive`
- Config load/validate — real Viper/YAML through `config.Load`
- Prefer lightest layer that proves the change (`CONTRIBUTING.md` collector guide)

## Fixtures and Factories

**Test Data:**
```go
// Golden inspect fixture (#87) — build archive in t.TempDir, no checked-in blob
func TestInspectArchive_goldenFixture(t *testing.T) {
	src := t.TempDir()
	manifest := filepath.Join(src, "extras", "manifest.json")
	// write minimal extras/manifest.json with archive_layout_version
	archivePath := filepath.Join(t.TempDir(), "inspect-golden.tar.gz")
	_ = archive.DirToTarGz(src, archivePath)
	info, err := InspectArchive(archivePath)
	// assert FileCount, ManifestJSON, ArchiveLayoutVersion
}
```

**Location:**
- Inline YAML strings written to temp config files (CLI / config tests)
- Kind workload: `testing/k8s/e2e-workload.yaml` (namespace `e2e-groot`, Deployment `log-generator`)
- Sample config source of truth for humans: `configs/groot.yml.sample` / `config.SampleYAML()` (also asserted in tests)

**Shared helpers:**
- `internal/kubetest.StartAPIServer` — primary cluster substitute
- CLI flag reset: `ResetPersistentCLI` / `resetPersistentFlags` in `internal/cmd`

## Coverage

**Requirements:**
- Local/release gate: `COVER_MIN` default **80** (merged statement coverage via `go tool cover -func=coverage.out`)
- `make cover` fails when total `% < COVER_MIN` (requires `bc` when `COVER_MIN > 0`)
- Codecov project target **80%** (`codecov.yml`, threshold 1%); patch status informational
- CI test job uploads `coverage.out` with `codecov-action`; `fail_ci_if_error: false`
- `make release-check` includes `cover` (hence `COVER_MIN`) plus security tooling

**View Coverage:**
```bash
make cover
go tool cover -func=coverage.out | tail -1
go tool cover -html=coverage.out
```

## Test Types

**Unit Tests:**
- Pure helpers, exit codes, redaction, config defaults/rejects, job plan, flag parsing
- Packages: `internal/config`, `internal/collector` (many files), `internal/cmd/exitcode_test.go`, `internal/logx`, `internal/archive`

**Integration Tests:**
- `internal/collector/run_integration_test.go` — `Service.Run` against kubetest / fake clientset; asserts archive on disk
- `internal/collector/manifest_integration_test.go` — manifest layout
- `internal/cmd/validate_inspect_e2e_test.go` — validate/inspect CLI against fake API; assert JSON + exit codes
- Uploader/notifier package tests with HTTP fakes / local SMTP-related helpers as present

**E2E Tests:**
- Kind harness: `make test-e2e-kind` → `testing/scripts/test-e2e-kind.sh`
- Requires Docker (bounded `docker info`), kind, kubectl (script setup only; groot uses client-go)
- Flow: create cluster → apply workload → sleep for logs → `groot collect` → assert `.tar.gz` → delete cluster (`trap`)
- Overrides: `GROOT_E2E_CLUSTER`, `GROOT_BIN`, `GROOT_E2E_ARCHIVE`, `GROOT_DOCKER_WAIT_SECS`
- E2E config disables node details/logs for speed; empty `nodes/` in archive is expected
- CI job `test-e2e-kind` in `.github/workflows/ci.yml` with `continue-on-error: true` (not a hard gate)
- Not part of `make ci` / default unit path

**Golden fixtures (#87):**
- Contract: inspect/archive regression without a live cluster
- Implementation: `internal/collector/inspect_golden_test.go`
- Cited in CHANGELOG / ROADMAP 1.0.0 band and `CONTRIBUTING.md` test matrix

## Common Patterns

**Async Testing:**
```go
sum, err := svc.Run(context.Background())
if err != nil {
	t.Fatal(err)
}
```
- Prefer explicit contexts; use timeouts in runner tests when exercising cancellation

**Error Testing:**
```go
err := rootCmd.Execute()
if err == nil {
	t.Fatal("expected error")
}
if got := ExitCodeOf(err); got != ExitConfigError {
	t.Fatalf("exit code = %d, want %d", got, ExitConfigError)
}
```

**Collector PR checklist (from CONTRIBUTING):**
- Unit or kubetest coverage for new paths
- Optional kind smoke after deep collect changes: `make test-e2e-kind`
- `make ci` green before PR; maintainers use `make release-check` before tags

**Notify smoke (manual, not unit suite):**
- `groot notify test` with examples under `examples/notify/` — see `docs/notify-smoke-test.md`

---

*Testing analysis: 2026-08-10*
