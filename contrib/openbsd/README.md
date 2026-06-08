# OpenBSD — groot

**groot** is a **CLI** only: there is no bundled **rc.d** script in this repository. Operators run **`groot collect`** from cron, Helm CronJob, CI, or their own wrappers.

Official port skeleton: **`contrib/openbsd/port/`** (submit to **ports@openbsd.org**).

Release tarballs match **`DISTFILES`** in that port (binary **`groot`** plus **`share/doc/groot/LICENSE`** and **`share/examples/groot/groot.yml.sample`**). Build a matching tarball locally with **`make dist-openbsd`** from the repo root.
