# FreeBSD port for groot

Port files for building and installing **groot** (CLI only; no rc.d or daemon).

## Install from port

When the port is in the official tree:

```bash
cd /usr/ports/sysutils/groot
make install
```

Local port (copy `Makefile`, `pkg-plist`, `pkg-descr` from this directory):

```bash
cd ~/ports/sysutils/groot
make install
```

After changing port files: `make deinstall && make clean && make install`.

## Test with a local distfile

1. From the **groot** repo root, sync **PORTVERSION** with **`VERSION`**:

   ```bash
   make port-freebsd-sync
   ```

2. Build the tarball expected by **DISTFILES** (default arch **amd64**; override with **`FREEBSD_ARCH=arm64`**):

   ```bash
   make dist-freebsd
   ```

   Output: `dist/groot_v<version>_freebsd_<arch>.tar.gz`.

3. Copy into **DISTDIR** or use **`MASTER_SITES=file:///.../`** as in the [FreeBSD Porter's Handbook](https://docs.freebsd.org/en/books/porters-handbook/).

The tarball contains: `groot`, `share/man/man1/groot.1`, `share/man/man1/kubectl-groot.1`, `share/doc/groot/LICENSE`, `share/examples/groot/groot.yml.sample`. After install, **`man groot`** works.
