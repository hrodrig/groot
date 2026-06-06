# OpenBSD port for `groot`

This directory holds the **OpenBSD ports tree** makefile + metadata for
`groot`. It mirrors FreeBSD in scope (binary + sample + docs) but uses
OpenBSD's `MODGO*` conventions. The port is **not** in the upstream
OpenBSD ports tree yet; the file layout is the same as if it were, so it
can be dropped into `/usr/ports/sysutils/groot/` verbatim.

## File layout

| File | Purpose |
|------|---------|
| `Makefile` | OpenBSD Go-module port. Uses `MODGO_MOD_NAME`, `MODGO_VERSION`, `MODGO_LDFLAGS` to build with the right module path and version metadata. |
| `distinfo` | Placeholder SHA256 hashes; CI's `release-ports-openbsd` job regenerates with `make makesum`. |
| `pkg/DESCR` | Long description (OpenBSD uses `pkg/DESCR`, not `pkg-descr`). |
| `pkg/PLIST` | File list (OpenBSD uses `pkg/PLIST`, capital-P). |
| `README.md` | This file. |

## Building locally on OpenBSD 7.5

```sh
doas pkg_add go git
git clone https://github.com/hrodrig/groot

# Option A: drop into the ports tree
doas mkdir -p /usr/ports/sysutils
doas cp -r groot/contrib/openbsd /usr/ports/sysutils/groot
cd /usr/ports/sysutils/groot
make makesum
make package
doas pkg_add -D unsigned ./groot-0.6.0.tgz
groot --version

# Option B: build in place against the local tree
cd groot/contrib/openbsd
make makesum
make package
doas pkg_add -D unsigned ./groot-0.6.0.tgz
groot --version
```

The `-D unsigned` flag is required because OpenBSD will refuse to
install an unsigned package. For a production setup, sign the package
with `signify(1)` before distribution.

## Why no rc.d script?

OpenBSD uses `rc.d` for system daemons only. `groot` is a one-shot CLI;
the recommended scheduling approach is `crontab(1)`:

```cron
# /etc/crontab - run groot every 6 hours
0 */6 * * * groot /usr/local/bin/groot collect --config /etc/groot/groot.yml
```

The sample config lives in `/usr/local/share/examples/groot/groot.yml.sample`
after install. Copy it to a writable location before editing.

## CI integration

The `release-ports-openbsd` job in `.github/workflows/release.yml` runs
on `tags: ['v*']` and uses `vmactions/openbsd-vm@v1` to:

1. Spin up an OpenBSD 7.5 VM (QEMU-backed)
2. Install `go` 1.26 and a ports skeleton
3. Drop `contrib/openbsd/` into `/usr/ports/sysutils/groot`
4. `make makesum` to refresh SHA256 hashes
5. `make package` to produce `groot-0.6.0.tgz`
6. If `distinfo` changed, push a follow-up commit
7. Upload the `.tgz` to the GitHub Release assets

## Cross-port notes

Both FreeBSD and OpenBSD ports derive the binary the same way; the
differences are conventions (paths, file names, package signing) rather
than build behavior. Keeping them in lock-step means a release tag
publishes both `*.pkg` and `*.tgz` artifacts with matching versions and
identical `groot --version` output.
