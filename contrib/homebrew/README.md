# Homebrew cask for `groot`

This directory holds the **source-of-truth** for the `groot` Homebrew cask.
The actual cask file lives in the **[`hrodrig/homebrew-groot`](https://github.com/hrodrig/homebrew-groot)**
tap repository; this directory is the seed + the manual fallback path.

## File layout

| File | Purpose |
|------|---------|
| `Casks/groot.rb.template` | Cask source-of-truth. Placeholders (`{{VERSION}}`, `{{SHA256}}`, `{{arch}}`) are substituted either by GoReleaser on release or by `scripts/update-homebrew-cask.sh` for manual updates. |
| `../../scripts/update-homebrew-cask.sh` | Idempotent shell script that clones/pulls the tap repo, substitutes placeholders against the latest release artifacts, commits, and pushes. Used when GoReleaser automation is disabled. |

## How the tap stays in sync

On every `vX.Y.Z` tag push, the **`release`** job in `.github/workflows/release.yml`
runs `goreleaser release --clean`. The `brews:` stanza in `.goreleaser.yaml`
opens (or updates) a PR against the tap repo with the rendered cask.

Secrets required (configure once in repo settings → **Secrets and variables → Actions**):

| Secret | Scope on `hrodrig/homebrew-groot` |
|--------|-----------------------------------|
| `HOMEBREW_TAP_TOKEN` | `repo` (read+write contents, no admin) |

## Manual update (no CI)

If `HOMEBREW_TAP_TOKEN` is missing or the `brews:` stanza was disabled for a
release, render and push the cask by hand:

```bash
# Set TAP_REPO to your local clone of hrodrig/homebrew-groot
export TAP_REPO="$HOME/src/homebrew-groot"
# Set VERSION and SHA256 from the published release
export VERSION=0.6.0
# SHA256 from the rendered cask: paste from the GitHub release checksums
# (use the per-arch SHA256 arm64/amd64 + darwin/linux for the rendered file)
./scripts/update-homebrew-cask.sh
```

`update-homebrew-cask.sh` will:

1. Read `contrib/homebrew/Casks/groot.rb.template`
2. Substitute `{{VERSION}}` and `{{SHA256}}` from env
3. Write to `${TAP_REPO}/Casks/groot.rb`
4. `git commit` and `git push` from inside `${TAP_REPO}`

## Verifying the cask locally

After the tap repo has the rendered cask:

```bash
brew tap hrodrig/groot
brew install --cask hrodrig/groot/groot
groot --version
brew audit --strict --online hrodrig/groot/groot
```

The audit command runs against the live tap, so this is the only way to
validate per-arch `sha256` blocks end-to-end.

## Why a template, not a plain cask file

Homebrew **requires** a real `version` + `sha256` in every cask. The template
keeps the placeholders visible to anyone reviewing the upstream repo and lets
GoReleaser (or `update-homebrew-cask.sh`) do the substitution against the
**real** release artifacts at tag time. The tap repo never sees the
template — only the rendered file.
