---
last_mapped_commit: 805eba0
analysis_date: 2026-08-10
---

# Coding Conventions

**Analysis Date:** 2026-08-10

## Naming Patterns

**Files:**
- Production: short Go package filenames (`collector.go`, `runner.go`, `exitcode.go`, `targz.go`) under `internal/<package>/`
- Tests: co-located `*_test.go` in the same package (e.g. `internal/collector/redact_test.go`, `internal/cmd/exitcode_test.go`)
- Platform variants: build-tagged siblings (`diskfree_unix.go`, `diskfree_windows.go`, `diskfree_openbsd.go`)
- Integration / golden: descriptive suffixes — `run_integration_test.go`, `inspect_golden_test.go`, `validate_inspect_e2e_test.go`, `manifest_integration_test.go`

**Functions:**
- Exported: PascalCase with verb-first names (`Load`, `InspectArchive`, `RedactCaptureLogs`, `NewExitError`, `ExitCodeOf`)
- Unexported: camelCase helpers (`buildJobs`, `parseFlags`, `fakeKubeHandler`, `kubeconfigYAML`)
- Constructors: `New` / `NewXxx` (`logx.New`, `collector.New`, `cmd.NewExitError`)
- Test names: `TestType_behavior` or `TestType_scenario` (`TestInspectArchive_goldenFixture`, `TestRoot_printSampleConfig`, `TestService_Run_minimal`)

**Variables:**
- Package-level CLI flags: camelCase (`cfgFile`, `strictMode`, `collectOutputForm`) in `internal/cmd/root.go`
- Locals: short idiomatic Go (`cfg`, `err`, `svc`, `sum`, `kc`, `buf`)
- Constants: PascalCase for exported API (`ArchiveLayoutVersion`, `ExitConfigError`); mixed SCREAMING for small private tokens when needed

**Types:**
- Structs: PascalCase nouns (`Summary`, `Service`, `ExitError`, `CollectionCfg`, `Logger`)
- Interfaces: small, behavior-named when present (`exitCoder` in `internal/cmd/exitcode.go`)
- Config nested types live on `config.Config` / `CollectionCfg` in `internal/config/config.go`

## Code Style

**Formatting:**
- Tool: `gofmt` with simplify (`gofmt -s`)
- Apply: `make lint-fix` (`gofmt -s -w .`)
- Gate: `make fmt-check` — same as CI (`.github/workflows/ci.yml` lint job); non-empty `gofmt -s -l .` fails
- `make fmt` is `gofmt -w .` without `-s`; prefer `lint-fix` before PRs (`CONTRIBUTING.md`)

**Linting:**
- `make lint` / `make vet` → `go vet ./...` only (formatting is separate)
- Cyclomatic complexity: `make gocyclo` via `gocyclo -over 14` (fail if complexity ≥ 15)
- No `golangci-lint` config in-repo; do not assume extra linters beyond Makefile targets
- Language: English only for code, comments, docs, and UI strings

## Import Organization

**Order (standard gofmt groups):**
1. Standard library (`context`, `fmt`, `testing`, …)
2. External modules (`github.com/spf13/cobra`, `k8s.io/client-go/...`, `github.com/google/go-cmp/cmp`)
3. Module-internal (`github.com/hrodrig/groot/internal/...`)

**Path Aliases:**
- Not applicable (no Go import aliases required by convention)
- Occasional import aliases for clarity: `ktesting "k8s.io/client-go/testing"`, `metricsfake "k8s.io/metrics/..."`, `metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"`

**Package placement:**
- Entrypoint only in `cmd/groot/` — thin `main` that calls `internal/cmd`
- All product logic under `internal/` (not importable by external modules)

## Error Handling

**Patterns:**
- Wrap with `fmt.Errorf("…: %w", err)` so `errors.Is` / `errors.As` work
- CLI boundary: return `*cmd.ExitError` from Cobra `RunE` via `NewExitError` / `NewExitErrorf` (`internal/cmd/exitcode.go`)
- Process exit: `cmd/groot/main.go` prints error to stderr and maps via `cmd.ExitCodeOf`
- Exit taxonomy (SPEC / `exitcode.go`): `0` success, `1` config, `2` Kubernetes, `3` collect aborted, `4` notify failed, `5` partial under `--strict`
- Plain untyped errors default to config exit code (`ExitCodeOf`)
- Cobra: `SilenceUsage: true`, `SilenceErrors: true` on root — main owns printing
- Config validation: return descriptive `fmt.Errorf(...)` from validators in `internal/config/config.go` (field paths in message text)
- Prefer explicit `if err != nil { t.Fatal(err) }` in tests; no testify

**Do this:**
```go
cfg, err := config.Load(cfgFile)
if err != nil {
    return NewExitError(ExitConfigError, fmt.Errorf("load config: %w", err))
}
```

## Logging

**Framework:** Custom `internal/logx` (`Logger`), not `log/slog` or zap

**Patterns:**
- Construct with `logx.New(verbose, quiet, noColor)`
- Levels/labels: `Info`, `Warn`, `Error` (stderr), `Cmd` / `OK` (verbose-only)
- Respect `--quiet` / `--verbose` / `--no-color`
- Do not log secrets; redaction belongs in collector (`RedactCaptureLogs`)

## Comments

**When to Comment:**
- Exported package/types/funcs: short godoc sentences
- ROADMAP / SPEC cross-refs in comments when behavior is contractual (`// ROADMAP #81`, `// 1.0.0 #34`, `// Golden-style … (#87)`)
- Non-obvious CLI contracts (stdout vs stderr for `--print-sample-config`)

**JSDoc/TSDoc:**
- Not applicable (Go godoc only)

## Function Design

**Size:**
- Keep cyclomatic complexity ≤ 14 (`make gocyclo`); split helpers when approaching the gate
- Collector job plan: prefer `append*Jobs` helpers called from `buildJobs` (`CONTRIBUTING.md`)

**Parameters:**
- Pass `context.Context` as first arg on blocking / cancellable work (`Service.Run`, k8srunner `Run`)
- Prefer concrete structs (`config.Config`) over option sprawl at call sites
- CLI overrides applied in `internal/cmd` before constructing services

**Return Values:**
- `(T, error)` for fallible ops; `Summary` + error for collect
- Exit codes only via `*ExitError` at the CLI boundary — not scattered `os.Exit` in libraries

## Module Design

**Exports:**
- Export only what other packages need (`collector.Service`, `config.Load`, `kubetest.StartAPIServer`)
- Keep job structs / parse helpers unexported when package-local

**Barrel Files:**
- Not used; import concrete packages (`internal/collector`, `internal/config`, …)

**Collector constraints (must follow):**
- Read-only product surface — no mutate/exec/attach as features
- Runtime path: client-go + `internal/k8srunner` argv slices — no kubectl binary for built-ins
- Operator deploy (Helm/cron) belongs in `groot-selfhosted`, not this repo

**Docs triad when shipping behavior:**
1. Update `docs/SPECIFICATIONS.md` (or root SPEC path as linked from CONTRIBUTING)
2. Mark ROADMAP global `#` Done
3. CHANGELOG under `[Unreleased]` with `(band #N)`

---

*Convention analysis: 2026-08-10*
