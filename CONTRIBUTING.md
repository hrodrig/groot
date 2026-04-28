# Contributing to GROOT

Thanks for helping improve GROOT.

## Ground rules

- Follow the [Code of Conduct](CODE_OF_CONDUCT.md).
- For **security issues**, use [SECURITY.md](SECURITY.md) — do not file public issues for undisclosed vulnerabilities.

## How to contribute

- **Bugs and ideas:** Open an [issue](https://github.com/hrodrig/groot/issues). Describe what you expected, what happened, and how to reproduce (commands, config snippets, cluster context if relevant).
- **Code:** Open a pull request **against `develop`**. `main` is release-only; day-to-day work merges into `develop` first (see project git flow).

Use focused branches, for example `fix/short-topic` or `feat/short-topic`.

## Before you open a PR

1. **Format:** `make lint-fix` (applies `gofmt -s` across the tree).
2. **Verify:** `make ci` runs `go vet` and tests — same bundle as CI.
3. **Broader check (optional):** `make all` adds coverage and cyclomatic complexity checks; maintainers use `make release-check` before releases (includes security scans and GoReleaser config validation).

Keep commits scoped and messages understandable.

## Project language

Repository content (code, comments, docs, UI strings) should be **English**, per project conventions.

## Questions

If something is unclear, open an issue and we can narrow the design or scope there.

## Resources

New to open source? [Open Source Guide](https://opensource.guide/how-to-contribute/) has general contribution practices.
