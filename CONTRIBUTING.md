# Contributing to GROOT

Thanks for helping improve GROOT.

## Ground rules

- Follow the [Code of Conduct](CODE_OF_CONDUCT.md).
- For **security issues**, use [SECURITY.md](SECURITY.md) — do not file public issues for undisclosed vulnerabilities.

## How to contribute

- **Bugs and ideas:** Open an [issue](https://github.com/hrodrig/groot/issues). Describe what you expected, what happened, and how to reproduce (commands, config snippets, cluster context if relevant).
- **Code:** Open a pull request **against `develop`**. `main` is release-only; day-to-day work merges into `develop` first (see project git flow).

Use focused branches, for example `fix/short-topic` or `feat/short-topic`.

## Planning docs (SPEC, ROADMAP, CHANGELOG)

Behavior contract: **[SPECIFICATIONS.md](SPECIFICATIONS.md)**. Planned work: **[ROADMAP.md](ROADMAP.md)**.

**Triad sync (kzero-style):** when you ship user-facing behavior—

1. Update **SPEC** if the observable contract changed.
2. Mark the roadmap item **Done** in **ROADMAP** (reference global `#` IDs).
3. Add a **CHANGELOG** bullet under `[Unreleased]` with **`(band #N)`** when applicable (e.g. `(0.4.x #12)`).

On release: move `[Unreleased]` to a version section, add a **Shipped** row in ROADMAP, sync **VERSION** and README badges, run **`make man-sync`** (man `.TH`), **`make port-freebsd-sync`** / **`make port-openbsd-sync`**, refresh VHS if UX changed, then **`make release-check`** before tag.

## Before you open a PR

1. **Format:** `make lint-fix` (applies `gofmt -s` across the tree).
2. **Verify:** `make ci` runs **`fmt-check`** (same `gofmt -s` gate as CI), **`golangci-lint`** (`.golangci.yml`), **`gocyclo`**, and **`go test -race`** — it matches the GitHub Actions **lint** and **test** jobs.
3. **Broader check (optional):** `make all` adds merged coverage and a build; maintainers run **`make release-check`** before tagging a release (GoReleaser config, **`fmt-check`**, **`lint`**, **`cover`** with `COVER_MIN`, **`security`**, and optionally **`STRICT_RELEASE=1`** for **`docker-scan`**).

Keep commits scoped and messages understandable.

## Collector guide (ROADMAP #47)

GROOT is a **read-only** collector: one **`groot collect`** builds a timestamped **`.tar.gz`**. Runtime path is **client-go** via **`internal/k8srunner`** (argv slices, **no shell**). There is **no plugin registry** — jobs are wired in Go.

### Where things live

| Concern | Path |
|---------|------|
| Job plan / `buildJobs` | `internal/collector/collector.go` |
| Builtin / pattern dispatch | `internal/collector/k8s_exec.go` |
| Argv → API (`Runner.Run`) | `internal/k8srunner/` |
| YAML `collection:` schema | `internal/config/config.go` (`CollectionCfg`) |
| Sample config | `configs/groot.yml.sample`, `internal/config/sample.go` |
| `extra_kubectl` allowlist | `internal/config/extra_kubectl.go` + SPEC §6 |
| CLI entry (`collect`, `--list-jobs`) | `internal/cmd/root.go` |
| Behavior contract | [SPECIFICATIONS.md](SPECIFICATIONS.md) |

### Adding a built-in collection job

1. **Shape** — jobs are `Name` / `Args` / `FileName` / `Optional` (`job` in `collector.go`). Prefer a clear name that `--list-jobs` can show.
2. **Plan** — append from `buildJobs` (or a new `append*Jobs` helper called from there). Keep signal-first behavior in mind (`prioritizeSignalJobs`, ROADMAP #84) if the job is high-signal.
3. **Execute** — teach `runNamedBuiltinJob` and/or `runJobByNamePattern` in `k8s_exec.go` to produce bytes (usually via `k8srunner` / existing helpers). Mark jobs **`Optional`** when failure should not abort the whole collect.
4. **Config** — if operators need a toggle or knobs, add fields on `CollectionCfg`, defaults in `setDefaults` / Viper, sample YAML, and a SPEC table row. Do **not** document behavior only in README.
5. **Archive path** — pick a stable relative path under the archive; update SPEC “collected layout” if the tree changes.
6. **Triad** — SPEC + ROADMAP `#` + CHANGELOG `(band #N)`.

### Prefer `extra_kubectl` when possible

Operators can add **read-only** argv lists under `collection.extra_kubectl` without a code change. Validation runs at **config load** (`ValidateExtraKubectl`); runtime rejects unknown verbs/resources again in **`k8srunner`**.

Allowed verbs (summary — full table in SPEC §6): `get`, `describe`, `top`, `logs`, discovery (`api-resources`, `api-versions`, `version`, `cluster-info`), `config view`, `auth can-i`. **Rejected:** `explain`, `wait`, and anything mutating. Args are whitespace-split — **no** shell pipelines or redirects.

Use a new built-in job only when the work needs multi-step client-go logic, structured tables (e.g. RCA TSV), or behavior that cannot be expressed as a safe argv list.

### Tests (pick the lightest that proves the change)

| Layer | When | Where / how |
|-------|------|-------------|
| Unit | helpers, job plan order, allowlist, config rejects | `internal/collector/*_test.go`, `internal/config/*_test.go`, `internal/k8srunner/*_test.go` |
| Fake API | `Run` / archive contents without a real cluster | `internal/collector/run_integration_test.go` + `internal/kubetest/` |
| Golden | inspect / fixture layout (#87) | `internal/collector/inspect_golden_test.go` |
| Kind E2E | optional smoke after deeper collect changes | `make test-e2e-kind` — [docs/e2e-kind.md](docs/e2e-kind.md), [testing/README.md](testing/README.md) |

Local loop: `make lint-fix && make ci`. Coverage gate: `make cover` / `COVER_MIN` (default 80%).

### Design constraints (do not violate)

- **Read-only** — no create/update/delete/exec/attach/port-forward as product features.
- **No kubectl binary** on the runtime path for built-ins — client-go + `k8srunner` only.
- **Partial job failures** are counted in `Summary` and must not fail collect unless **`--strict`** (see SPEC exit codes).
- **Redaction** — if you write new text that may contain secrets, respect `redact_secrets` / `redact_patterns`.
- Operator deployment (Helm CronJob, bastion wrappers) belongs in **[groot-selfhosted](https://github.com/hrodrig/groot-selfhosted)**, not this repo ([AGENTS.md](AGENTS.md)).

### Quick checklist for a collector PR

- [ ] Job appears in `groot collect --list-jobs` when enabled
- [ ] SPEC updated for config keys and/or archive paths
- [ ] Sample YAML / `--print-sample-config` updated if config changed
- [ ] Unit or kubetest coverage for the new path
- [ ] `extra_kubectl` still reject-tested if you touched the allowlist
- [ ] ROADMAP `#` + CHANGELOG under `[Unreleased]`
- [ ] `make ci` green

## Codecov (repository maintainers)

CI uploads **`coverage.out`** from `go test … -coverprofile=coverage.out` via [codecov-action](https://github.com/codecov/codecov-action) (see `.github/workflows/ci.yml`). Configuration lives in **`codecov.yml`** (project target aligns with `make cover` / `COVER_MIN` in the Makefile).

1. In [Codecov](https://app.codecov.io/gh/hrodrig/groot), open the repo **Settings** and copy the **repository upload token** (or finish the GitHub App flow Codecov shows for new repos).
2. In GitHub: **Settings → Secrets and variables → Actions → New repository secret** named **`CODECOV_TOKEN`** with that value.
3. Push to **`main`** or **`develop`** (or open a PR) so the **test** workflow runs once; the dashboard should populate after the first successful upload.

Forks and contributors without the secret: the upload step is a no-op token-wise; **`fail_ci_if_error`** is **`false`** so CI still passes.

## Project language

Repository content (code, comments, docs, UI strings) should be **English**, per project conventions.

## Questions

If something is unclear, open an issue and we can narrow the design or scope there.

## Resources

New to open source? [Open Source Guide](https://opensource.guide/how-to-contribute/) has general contribution practices.
