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

Behavior contract: **[docs/SPECIFICATIONS.md](docs/SPECIFICATIONS.md)**. Planned work: **[docs/ROADMAP.md](docs/ROADMAP.md)**.

**Triad sync (kzero-style):** when you ship user-facing behavior—

1. Update **SPEC** if the observable contract changed.
2. Mark the roadmap item **Done** in **ROADMAP** (reference global `#` IDs).
3. Add a **CHANGELOG** bullet under `[Unreleased]` with **`(band #N)`** when applicable (e.g. `(0.4.x #12)`).

On release: move `[Unreleased]` to a version section, add a **Shipped** row in ROADMAP, sync **VERSION** and README badges, run **`make man-sync`** (man `.TH`), **`make port-freebsd-sync`** / **`make port-openbsd-sync`**, refresh VHS if UX changed, then **`make release-check`** before tag.

## Before you open a PR

1. **Format:** `make lint-fix` (applies `gofmt -s` across the tree).
2. **Verify:** `make ci` runs **`fmt-check`** (same `gofmt -s` gate as CI), **`go vet`**, **`gocyclo`**, and **`go test -race`** — it matches the GitHub Actions **lint** and **test** jobs.
3. **Broader check (optional):** `make all` adds merged coverage and a build; maintainers run **`make release-check`** before tagging a release (GoReleaser config, **`fmt-check`**, **`lint`**, **`cover`** with `COVER_MIN`, **`security`**, and optionally **`STRICT_RELEASE=1`** for **`docker-scan`**).

Keep commits scoped and messages understandable.

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
