<!-- last_mapped_commit: 805eba0 -->
# Codebase Structure

**Analysis Date:** 2026-08-10

## Directory Layout

```
groot/                          # Product CLI (module github.com/hrodrig/groot)
├── cmd/groot/                  # Binary entry only (main + thin tests)
├── internal/                   # All application packages (private)
│   ├── cmd/                    # Cobra commands, exit codes, plugin helpers
│   ├── collector/              # Collect / preflight / inspect domain
│   ├── config/                 # YAML/env config schema + sample
│   ├── k8srunner/              # Allowlisted client-go “kubectl argv”
│   ├── kubeloader/             # Kubeconfig → rest.Config
│   ├── archive/                # Dir → .tar.gz
│   ├── notifier/               # Post-collect notification fan-out
│   ├── uploader/               # Post-collect S3/GCS/SFTP fan-out
│   ├── logx/                   # Console logger
│   └── kubetest/               # Fake K8s API for tests
├── configs/                    # Sample config shipped with product
├── contrib/                    # Packaging (deb/freebsd/openbsd/krew/man)
├── docs/                       # Plans, demos, e2e notes (not SPEC triad)
├── examples/                   # Example YAML profiles (notify/upload/…)
├── testing/                    # Product E2E (kind scripts + manifests)
├── scripts/                    # Maintainer helper scripts
├── .github/                    # CI, CodeQL, release, security workflows
├── .agents/skills/             # Project agent skills (golang-*)
├── .cursor/rules/              # Cursor/project guardrails
├── SPECIFICATIONS.md           # Behavior contract (root)
├── ROADMAP.md                  # Planned work / global IDs (root)
├── CHANGELOG.md                # Shipped releases (root)
├── VERSION                     # Semver string for releases
├── Makefile                    # build/test/release-check targets
├── Dockerfile*                 # Image build for ghcr.io/hrodrig/groot
├── .goreleaser.yaml            # Multi-OS release + kubectl-groot
├── go.mod / go.sum
└── AGENTS.md                   # Product vs groot-selfhosted scope
```

**Not product source (local/scratch — do not treat as layout contract):** `.gocache/`, `.gotmp/`, `.tmp/`, `dist/`, `bin/`, `.no-va-al-repo/`, `coverage.out`.

**Companion repo (out of tree):** operator deployment → **groot-selfhosted**.

## Directory Purposes

**`cmd/groot/`:**
- Purpose: Process entrypoint only.
- Contains: `main.go`, `main_test.go`.
- Key files: `cmd/groot/main.go` — signals + `cmd.ExecuteContext` + `ExitCodeOf`.

**`internal/cmd/`:**
- Purpose: Full Cobra CLI surface.
- Contains: `root.go` (collect + root flags), `validate.go` (validate + inspect), `notify.go`, `version.go`, `completion.go`, `exitcode.go`, `plugin.go`, co-located `*_test.go`.
- Key files: `internal/cmd/root.go`, `internal/cmd/validate.go`, `internal/cmd/exitcode.go`.

**`internal/collector/`:**
- Purpose: Core product engine — jobs, archive artifacts, preflight, inspect.
- Contains: `collector.go` (largest), `preflight.go`, `inspect.go`, `manifest.go`, `redact.go`, `summary.go`, `jobs_plan.go`, platform `diskfree_*.go`, tests + golden inspect.
- Key files: `internal/collector/collector.go`, `preflight.go`, `inspect.go`.

**`internal/config/`:**
- Purpose: Configuration types and load path.
- Contains: `config.go`, `sample.go`, pod_logs_since / extra_kubectl helpers, tests.
- Key files: `internal/config/config.go`, `configs/groot.yml.sample` (user-facing sample mirror).

**`internal/k8srunner/`:**
- Purpose: Execute allowlisted read-only API operations.
- Contains: `runner.go`, `get_extended.go`, tests.
- Key files: `internal/k8srunner/runner.go`.

**`internal/kubeloader/`:**
- Purpose: Kubeconfig loading without kubectl.
- Key files: `internal/kubeloader/loader.go`.

**`internal/archive/`:**
- Purpose: Pack capture directory to `.tar.gz`.
- Key files: `internal/archive/targz.go`.

**`internal/notifier/` / `internal/uploader/`:**
- Purpose: Optional post-collect integrations.
- Contains: Channel/provider files (`email.go`, `pagerduty.go`, `s3.go`, `gcs.go`, `sftp.go`, …).

**`internal/logx/`:**
- Purpose: Human-readable console logging for collect UX.

**`internal/kubetest/`:**
- Purpose: Shared fake API server for Go tests.

**`configs/`:**
- Purpose: Canonical sample YAML for docs and `--print-sample-config` alignment.
- Key files: `configs/groot.yml.sample`.

**`contrib/`:**
- Purpose: Distro packaging and kubectl plugin metadata — not runtime Go code.
- Contains: `contrib/deb/`, `contrib/freebsd/`, `contrib/openbsd/`, `contrib/krew/`, `contrib/man/`.

**`docs/`:**
- Purpose: Band plans (`plan-*.md`), demo assets, e2e/notify smoke notes.
- Note: Behavior contract is **`SPECIFICATIONS.md` at repo root**, not under `docs/`.

**`examples/`:**
- Purpose: Copy-paste YAML for profiles, notify, upload, collection options.
- Key files: `examples/README.md`, `examples/profiles/*.yml`.

**`testing/`:**
- Purpose: Kind-based product E2E scripts and workload YAML.
- Key files: `testing/scripts/test-e2e-kind.sh`, `testing/k8s/e2e-workload.yaml`.

**`.planning/codebase/`:**
- Purpose: GSD codebase maps (this file and siblings).
- Generated: By `/gsd-map-codebase` agents.
- Committed: Yes when planning workflow includes them.

## Key File Locations

**Entry Points:**
- `cmd/groot/main.go`: Binary main.
- `internal/cmd/root.go`: Cobra root + `collect` + command registration.
- `internal/cmd/validate.go`: `validate` and `inspect` command factories.

**Configuration:**
- `internal/config/config.go`: Schema + `Load`.
- `configs/groot.yml.sample`: Sample for operators.
- `internal/config/sample.go`: Embedded sample for `--print-sample-config`.

**Core Logic:**
- `internal/collector/collector.go`: `Service.Run`, job plan, workers.
- `internal/collector/preflight.go`: `Service.Preflight`.
- `internal/collector/inspect.go`: `InspectArchive` (analyze #69 absent).
- `internal/k8srunner/runner.go`: API execution allowlist.
- `internal/notifier/notifier.go` / `internal/uploader/uploader.go`: Fan-outs.

**Contract / planning triad:**
- `SPECIFICATIONS.md`: Testable behavior today.
- `ROADMAP.md`: Bands and item IDs (e.g. #69 analyze pending).
- `CHANGELOG.md`: Released notes.

**Testing:**
- Co-located `*_test.go` under each `internal/*` package.
- `internal/kubetest/`: Fake cluster for collect/validate tests.
- `testing/`: Kind E2E.

**Build / release:**
- `Makefile`, `.goreleaser.yaml`, `Dockerfile`, `Dockerfile.release`, `VERSION`.

## Naming Conventions

**Files:**
- Package name matches directory (`collector`, `notifier`).
- Platform files: `diskfree_unix.go`, `diskfree_windows.go`, `diskfree_openbsd.go`.
- Tests: `foo_test.go` next to `foo.go`; integration/golden suffixes when specialized (`*_integration_test.go`, `inspect_golden_test.go`).

**Directories:**
- Lowercase, single purpose under `internal/`.
- No public `pkg/` — everything product-private under `internal/`.

**Types / functions:**
- Exported services: `Service`, `FanOut`, `Runner`, `New` / `NewFanOut`.
- Cobra factories: `newValidateCmd`, `newInspectCmd`, `newNotifyCmd` (unexported constructors).
- CLI packages named `cmd` at both `cmd/groot` (main) and `internal/cmd` (library).

**Commands (user-facing):**
- Implemented: `collect`, `validate`, `inspect`, `notify test`, `version`, `completion`.
- Not present: `analyze` (ROADMAP #69).

## Where to Add New Code

**New CLI subcommand (e.g. future `analyze` #69):**
- Factory + flags: `internal/cmd/<name>.go` (register in `root.go` `init` `AddCommand`).
- Domain logic: prefer `internal/collector/` if archive-centric, or new `internal/<domain>/` if substantial.
- Tests: co-located `*_test.go`; reuse golden archive patterns from `internal/collector/inspect_golden_test.go`.
- Do **not** add until ROADMAP #69 is in an active plan — inspect stays inventory-only.

**New collect job / API snapshot:**
- Plan in `buildJobs` / append helpers in `internal/collector/collector.go` (or split file if large).
- If new argv shape: extend allowlist in `internal/k8srunner/runner.go` (+ tests).
- Update `SPECIFICATIONS.md` and sample config when user-visible.

**New notify or upload provider:**
- Implement behind existing interfaces in `internal/notifier/` or `internal/uploader/`.
- Wire in `NewFanOut`; add config fields in `internal/config/config.go` + sample YAML.
- Example YAML under `examples/notify/` or `examples/upload/`.

**New config field:**
- Struct + mapstructure tags in `internal/config/config.go`.
- Defaults/validation in `Load`; document in `SPECIFICATIONS.md` and `configs/groot.yml.sample`.

**Packaging / man / krew only:**
- `contrib/` — no Go package changes required unless binary name/flags change.

**Operator deploy (Helm, CronJob, systemd):**
- **Wrong place:** this repo.
- **Right place:** groot-selfhosted.

**Utilities:**
- Shared non-domain helpers: small packages under `internal/` (follow `logx`, `archive`).
- Avoid new top-level packages outside `internal/` / `cmd/`.

## Special Directories

**`dist/`, `bin/`:**
- Purpose: Local build outputs.
- Generated: Yes.
- Committed: No (treat as artifacts).

**`.planning/`:**
- Purpose: GSD planning + codebase maps.
- Generated: Partially (maps refreshed by agents).
- Committed: When included in planning workflow.

**`.agents/skills/`:**
- Purpose: Project skills (`golang-pro`, `golang-cli-cobra-viper`).
- Generated: No.
- Committed: Yes.

**`.no-va-al-repo/`:**
- Purpose: Local scratch / research dumps.
- Generated: Manual.
- Committed: No (gitignore whitelist).

**`coverage.out`:**
- Purpose: Coverage from `make cover`.
- Generated: Yes.
- Committed: No.

---

*Structure analysis: 2026-08-10*
