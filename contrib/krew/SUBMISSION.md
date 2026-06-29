# Submitting the Groot plugin to the Krew index

Groot ships a kubectl plugin (`kubectl-groot`) starting with **v0.9.x** (ROADMAP #64, #85). Krew is the most common installer for kubectl plugins and gets us discoverable install one-liners (`kubectl krew install groot`) without users having to `make install-kubectl-plugin` by hand.

This document captures the **manual** steps to submit `contrib/krew/groot.yaml` to [`krew-index`](https://github.com/kubernetes-sigs/krew-index). Krew requires a real GitHub account with sign-off on the CLA and a CLA-bot-approved PR; this cannot be fully automated from CI without granting the release workflow commit rights to a fork.

## Why this is not automated today

- Krew's PR workflow requires human review and a CLA-sign-off comment. Bots can open the PR but cannot sign the CLA on the user's behalf.
- Submitting the plugin pins a per-tag SHA-256. Generating those hashes requires running `goreleaser release --clean` for the exact tag, which is what the existing tag-release workflow already does.
- We track the **template** in this repo (placeholders `{{ .Tag }}`, `{{ .Sha256* }}`) and let the release maintainer fill them at submission time. A future enhancement could template-substitute with the same GoReleaser metadata as the Homebrew cask, but Krew's separate index repo is the hard part.

## Per-release submission steps

1. Cut a release tag as usual (`make release-check`, push `v0.9.x`).
2. After the GitHub release publishes the archives and GoReleaser pushes SHA-256 sums, fetch them:
   ```bash
   TAG="v0.9.0"
   curl -fsSL "https://github.com/hrodrig/groot/releases/download/${TAG}/checksums.txt"
   ```
3. Match each `groot_${TAG}_<os>_<arch>.tar.gz` / `.zip` line to its SHA-256.
4. Render the template at `contrib/krew/groot.yaml` by replacing:
   - `{{ .Tag }}` → the literal tag (`v0.9.0`, **with** the `v` prefix).
   - `{{ .Sha256LinuxAmd64 }}`, `{{ .Sha256LinuxArm64 }}`, `{{ .Sha256DarwinAmd64 }}`, `{{ .Sha256DarwinArm64 }}`, `{{ .Sha256WindowsAmd64 }}` → the corresponding SHA-256.
5. Fork `kubernetes-sigs/krew-index`, drop the rendered file at `plugins/groot.yaml`, and open a PR. Reference the upstream release tag in the PR body. The krew-bot will:
   - Run `krew lint plugins/groot.yaml`
   - Open a CLA-bot check (sign once per GitHub account)
   - Request review from a `@krew-index-maintainers` member
6. Once merged, users can install with:
   ```bash
   kubectl krew install groot
   kubectl groot --version   # → groot vX.Y.Z (commit=… branch=… built=…) as expected
   ```

## Validation before submission

Before opening the PR, locally validate the rendered manifest:

```bash
# Install krew locally (one-time)
kubectl krew install krew

# Lint the rendered manifest
krew lint plugins/groot.yaml

# Smoke: install from a local file and verify the plugin runs
kubectl krew install --manifest=plugins/groot.yaml --archive-grep='tar.gz|zip'
kubectl groot --version
kubectl groot collect --help
kubectl plugin list | grep groot
```

If `krew lint` complains, the most common cause is a SHA-256 that doesn't match the binary in the archive or a `bin:` path that doesn't exist as a file inside the archive.

## When to submit

| Release event | Submit to krew-index? |
|---------------|-----------------------|
| `v0.9.x` plugin launch (this PR) | **Yes** — initial publication. |
| Subsequent `v0.9.y` patch | **Yes** — submit a new SHA-256 / tag. Krew enforces "latest tag wins"; skipping a version is fine. |
| `v1.0.0` | **Yes** — same flow; mention config-freeze in PR body so reviewers know the API is stable. |
| `v0.9.x → v1.0.x` with breaking CLI changes | **Yes**, and call out the migration in the PR. |

## Caveats to users (already in the manifest)

- The Krew manifest installs **only** the `kubectl-groot` binary. The standalone `groot` binary lives in the same GitHub Release tarball; users who want both should download and unpack the tarball manually.
- The plugin reuses the standalone CLI binary verbatim. All flags behave identically.
- A sample config `groot.yml.sample` is installed alongside the binary for users who want a starting point (`cp $(kubectl groot collect --print-sample-config) ~/.config/groot/groot.yml` does **not** exist — the sample is in the plugin directory or one shell call away: `kubectl groot --print-sample-config > ~/.config/groot/groot.yml`).
