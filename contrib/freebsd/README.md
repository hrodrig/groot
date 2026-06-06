# FreeBSD port for `groot`

This directory holds the **FreeBSD ports collection** makefile + metadata
for `groot`. It is **not** in the upstream FreeBSD ports tree yet (would
need a maintainer review on `freebsd-ports@`); the file layout is the
same as if it were, so the port can be dropped into `/usr/ports/sysutils/`
verbatim and built today.

## File layout

| File | Purpose |
|------|---------|
| `Makefile` | Go-module port (FreeBSD 14.x with `lang/go` 1.26.4). Mirrors nfpms layout: binary to `/usr/local/bin/groot`, sample to `/usr/local/etc/groot/groot.yml.sample`, docs to `/usr/local/share/doc/groot/`. |
| `distinfo` | Placeholder SHA256 hashes; CI's `release-ports-freebsd` job regenerates with `makesum` against the actual source tarball. |
| `pkg-descr` | Short description for `pkg-descr(5)`. |
| `pkg-plist` | File list for `pkg-plist(5)`; uses `@sample` for `groot.yml.sample` so installs don't clobber operator edits. |
| `files/groot.in` | `rc.d` script (`service groot onestart`). No-op until 0.7.x adds watch mode; documented as such. |
| `README.md` | This file. |

## Building locally on FreeBSD 14

```sh
# As root, with the FreeBSD ports tree installed
git clone https://github.com/hrodrig/groot
mkdir -p /usr/ports/sysutils/groot
cp -r groot/contrib/freebsd/* /usr/ports/sysutils/groot/
cd /usr/ports/sysutils/groot

# Regenerate distinfo against the real source tarball
make distinfo

# Build + install
make package
pkg add ./Work/pkg/groot-0.6.0.pkg
groot --version
```

## Using the rc.d script

```sh
sysrc groot_enable=YES
sysrc groot_config=/usr/local/etc/groot/groot.yml
cp /usr/local/etc/groot/groot.yml.sample /usr/local/etc/groot/groot.yml
# Edit the file with your kubeconfig and notify settings

# Today: returns exit 0 with a no-op notice. Use cron(8) to actually
# run scheduled collects until 0.7.x adds the long-running watch mode.
service groot onestart
```

## CI integration

The `release-ports-freebsd` job in `.github/workflows/release.yml` runs
on `tags: ['v*']` and uses `vmactions/freebsd-vm@v1` to:

1. Spin up a FreeBSD 14 VM (QEMU-backed)
2. Install `lang/go` 1.26.4 and the ports tree skeleton
3. Drop `contrib/freebsd/` into `/usr/ports/sysutils/groot`
4. `make distinfo` to refresh SHA256 hashes
5. `make package` to produce `Work/pkg/groot-0.6.0.pkg`
6. If `distinfo` changed, push a follow-up commit
7. Upload the `.pkg` to the GitHub Release assets

The maintainer-side override for when this lives in the official FreeBSD
ports tree: drop the `DISTVERSIONPREFIX` / `USE_GITHUB` block and rely on
the tree's `ports/`, `distfiles/`, and `INDEX`.
