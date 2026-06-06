#!/usr/bin/env bash
# scripts/update-homebrew-cask.sh
# ===============================
# Render `contrib/homebrew/Casks/groot.rb.template` with VERSION + SHA256
# placeholders and push the result to the hrodrig/homebrew-groot tap repo.
#
# Required environment:
#   VERSION   - bare semver, e.g. "0.6.0"
#   SHA256    - SHA256 digest(s) for the rendered cask. Either a single
#               string (used for all arches) or a comma-separated list
#               matching the arch order GoReleaser writes:
#                 arm64_darwin,intel,arm64_linux,x86_64_linux
#
#   One of:
#     TAP_REPO  - absolute path to a local clone of hrodrig/homebrew-groot
#     TAP_GIT   - git URL to clone (defaults to git@github.com:hrodrig/homebrew-groot.git
#                 when TAP_REPO is not set)
#
# Optional:
#   TAP_BRANCH - branch to commit on (default: main)
#   DRY_RUN    - "1" to render to stdout and skip git operations

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TEMPLATE="${REPO_ROOT}/contrib/homebrew/Casks/groot.rb.template"

if [[ -z "${VERSION:-}" ]]; then
  echo "VERSION env var is required (e.g. 0.6.0)" >&2
  exit 2
fi

if [[ -z "${SHA256:-}" ]]; then
  echo "SHA256 env var is required (single value or arch-keyed list)" >&2
  exit 2
fi

if [[ ! -f "${TEMPLATE}" ]]; then
  echo "Template not found: ${TEMPLATE}" >&2
  exit 2
fi

TAP_BRANCH="${TAP_BRANCH:-main}"
DRY_RUN="${DRY_RUN:-0}"

# Render via Python (more readable than sed/awk for multi-line substitutions).
export _RENDER_TEMPLATE="${TEMPLATE}"
RENDERED="$(_RENDER_TEMPLATE="${TEMPLATE}" VERSION="${VERSION}" SHA256="${SHA256}" python3 - <<'PY'
import os, sys, pathlib

template = pathlib.Path(os.environ["_RENDER_TEMPLATE"]).read_text()
version = os.environ["VERSION"]
sha256 = os.environ["SHA256"]

out = template.replace("{{VERSION}}", version).replace("{{arch}}", "x86_64")

if "," in sha256:
    parts = [p.strip() for p in sha256.split(",")]
    if len(parts) != 4:
        sys.exit("comma-separated SHA256 must have exactly 4 entries")
    arm64_darwin, intel, arm64_linux, x86_64_linux = parts
    block = (
        f'  sha256 arm64_darwin: "{arm64_darwin}",\n'
        f'         intel:        "{intel}",\n'
        f'         arm64_linux:   "{arm64_linux}",\n'
        f'         x86_64_linux:  "{x86_64_linux}"'
    )
    needle = '  sha256 "{{SHA256}}"'
    if needle not in out:
        sys.exit(f"placeholder line not found: {needle!r}")
    out = out.replace(needle, block, 1)
else:
    out = out.replace('sha256 "{{SHA256}}"', f'sha256 "{sha256}"', 1)

sys.stdout.write(out)
PY
)"

if [[ "${DRY_RUN}" == "1" ]]; then
  echo "${RENDERED}"
  exit 0
fi

if [[ -z "${TAP_REPO:-}" ]]; then
  TAP_GIT="${TAP_GIT:-git@github.com:hrodrig/homebrew-groot.git}"
  WORK="$(mktemp -d -t homebrew-groot.XXXXXX)"
  trap 'rm -rf "${WORK}"' EXIT
  git clone --depth=1 --branch="${TAP_BRANCH}" "${TAP_GIT}" "${WORK}/tap"
  TAP_REPO="${WORK}/tap"
else
  if [[ ! -d "${TAP_REPO}/.git" ]]; then
    echo "TAP_REPO=${TAP_REPO} is not a git working tree" >&2
    exit 2
  fi
  (cd "${TAP_REPO}" && git pull --ff-only origin "${TAP_BRANCH}")
fi

CASK_PATH="${TAP_REPO}/Casks/groot.rb"
mkdir -p "$(dirname "${CASK_PATH}")"
echo "${RENDERED}" > "${CASK_PATH}"

(
  cd "${TAP_REPO}"
  if git diff --quiet -- Casks/groot.rb; then
    echo "No changes in ${CASK_PATH} - nothing to commit."
    exit 0
  fi
  git add Casks/groot.rb
  git -c user.name="groot-release-bot" \
      -c user.email="groot-release-bot@users.noreply.github.com" \
      commit -m "groot: bump cask to v${VERSION}"
  git push origin "${TAP_BRANCH}"
)

echo "Updated ${CASK_PATH} for v${VERSION}."
