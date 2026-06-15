# contrib — packaging helpers for groot

**groot is a CLI**, not a daemon or a shipped systemd service. Packages install the binary and an example config path.

This tree holds files referenced by **GoReleaser** (`.goreleaser.yaml`): maintainer scripts for Linux **`.deb`** / **`.rpm`**, plus **FreeBSD** and **OpenBSD** port skeletons under **`contrib/freebsd/`** and **`contrib/openbsd/`**.

## Linux packages (.deb / .rpm)

GoReleaser **nfpm** integration produces, per Linux architecture:

- **`*.deb`** and **`*.rpm`** — install **`/usr/bin/groot`** and **`/etc/groot/groot.yml.sample`** (`config|noreplace` on upgrades)
- **`contrib/deb/prerm.sh`** — no-op (CLI-only package)
- **`contrib/deb/postrm.sh`** — on **`apt purge`**, removes **`/etc/groot`** so no stale config remains

**Tarballs / zip:** the **`archives`** section in `.goreleaser.yaml` emits **`*.tar.gz`** (and **`*.zip`** on Windows), including **FreeBSD** and **OpenBSD** builds. Each archive includes **`share/doc/groot/LICENSE`** and **`share/examples/groot/groot.yml.sample`** (same layout as the BSD port distfiles).

## FreeBSD and OpenBSD ports

Maintainer-facing skeletons live under **`contrib/freebsd/`** and **`contrib/openbsd/port/`** (submit to the official trees when ready).

- **`make port-freebsd-sync`** / **`make port-openbsd-sync`** — refresh **`PORTVERSION`** / **`DISTNAME`** / **`PKGNAME`** / **`MASTER_SITES`** / **`DISTFILES`** from the repo **`VERSION`** file.
- **`make dist-freebsd`** — cross-build **`dist/groot_v<semver>_freebsd_<arch>.tar.gz`** (default **`FREEBSD_ARCH=amd64`**).
- **`make dist-openbsd`** — cross-build **`dist/groot_v<semver>_openbsd_<arch>.tar.gz`** (default **`OPENBSD_ARCH=amd64`**).

groot is a **CLI** only: these ports do not install **rc.d** or systemd units. See **`contrib/freebsd/README.md`**, **`contrib/openbsd/README.md`**, and **`contrib/openbsd/port/README.md`**.

## Homebrew cask

The cask lives in the separate tap repo **[hrodrig/homebrew-groot](https://github.com/hrodrig/homebrew-groot)** (`Casks/groot.rb`). GoReleaser updates it on every tag via the **`homebrew_casks:`** stanza in `.goreleaser.yaml`.

## Scheduling (optional, operator-owned)

If you want periodic runs, invoke **`groot collect`** from **cron**, **Helm CronJob**, **CI**, or your own wrappers. Examples and manifests: **[groot-selfhosted](https://github.com/hrodrig/groot-selfhosted)** (`run/standalone/`, `run/deploy/`). The packages do not enable or ship any long-running service.
