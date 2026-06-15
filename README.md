# GROOT — Kubernetes log collector CLI

<a id="readme-top"></a>

**☸** _Collect Kubernetes logs and cluster context into one archive_

[![Release](https://img.shields.io/github/v/release/hrodrig/groot?display_name=tag&label=release&logo=github)](https://github.com/hrodrig/groot/releases)
[![Version](https://img.shields.io/badge/version-0.6.1-blue)](https://github.com/hrodrig/groot/releases)
[![Go](https://img.shields.io/badge/Go-1.26.4-00ADD8?logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/license-MIT-green)](./LICENSE)
[![pkg.go.dev](https://pkg.go.dev/badge/github.com/hrodrig/groot)](https://pkg.go.dev/github.com/hrodrig/groot)
[![CI](https://github.com/hrodrig/groot/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/hrodrig/groot/actions/workflows/ci.yml)
[![Codecov](https://codecov.io/gh/hrodrig/groot/branch/main/graph/badge.svg)](https://codecov.io/gh/hrodrig/groot)
[![gghstats clones](https://gghstats.hermesrodriguez.com/api/v1/badge/hrodrig/groot?metric=clones)](https://gghstats.hermesrodriguez.com/hrodrig/groot)
[![Go Report Card](https://goreportcard.com/badge/github.com/hrodrig/groot)](https://goreportcard.com/report/github.com/hrodrig/groot)
[![Article on DEV](https://img.shields.io/badge/dev.to-article-0A0A0A?logo=devdotto&logoColor=white)](https://dev.to/hrodrig/groot-one-archive-for-cluster-diagnostics-2d76)
[![Ask DeepWiki](https://deepwiki.com/badge.svg)](https://deepwiki.com/hrodrig/groot/)

**Repo:** [github.com/hrodrig/groot](https://github.com/hrodrig/groot) · **Releases:** [GitHub Releases](https://github.com/hrodrig/groot/releases) · **Spec:** [docs/SPECIFICATIONS.md](docs/SPECIFICATIONS.md) · **Operator:** [groot-selfhosted](https://github.com/hrodrig/groot-selfhosted) · **Changelog:** [CHANGELOG.md](CHANGELOG.md) · **Roadmap:** [docs/ROADMAP.md](docs/ROADMAP.md) · **Article:** [GROOT on DEV — one archive for cluster diagnostics](https://dev.to/hrodrig/groot-one-archive-for-cluster-diagnostics-2d76)

<p align="center">
  <img src="docs/assets/groot-readme-hero.png" alt="GROOT — Kubernetes log collector CLI" width="100%" />
</p>

<p align="center">
  <img src="docs/demo.gif" alt="groot terminal demo: help, version, sample config" width="92%" />
</p>

<p align="center"><sub>Terminal demo recorded with <a href="https://github.com/charmbracelet/vhs">VHS</a>. Regenerate: <code>make install && bash -c "vhs docs/demo.tape"</code> · <a href="docs/demo.tape"><code>docs/demo.tape</code></a></sub></p>

GROOT is a **read-only Kubernetes log and context collector**: a single **`groot collect`** pulls **pod logs**, control-plane logs, **events**, and selected API snapshots—in parallel, from YAML—and packs them into one **`.tar.gz`**. It does **not** analyze the cluster or render a diagnosis; it **archives evidence** you can attach to tickets, hand to teammates, or retain for compliance.

That workflow supports **incident response**, **troubleshooting**, and **root cause analysis (RCA)**: one reproducible bundle replaces scattered `kubectl` copy-paste, so you can reconstruct *what the cluster looked like* when you ran collect and shorten postmortems.

> **Operator deployment** (bastion, cron, `docker run`, Helm CronJob, flat manifests): **[groot-selfhosted](https://github.com/hrodrig/groot-selfhosted)** — this repo ships the CLI binary, packages, and container image only.

## Table of contents

- [README badge reference](docs/badges.md)
- [Specifications (behavior contract)](docs/SPECIFICATIONS.md)
- [Roadmap (planned work)](docs/ROADMAP.md)
- [Features](#features)
- [Requirements](#requirements)
- [Install or update](#install-or-update)
  - [Install with Homebrew (macOS / Linux, recommended for desktop)](#install-with-homebrew-macos--linux-recommended-for-desktop)
  - [Install with Go](#install-with-go)
- [Quick start](#quick-start)
- [First run](#first-run)
- [Usage examples](#usage-examples)
- [Config](#config)
- [Configuration reference (all keys)](#configuration-reference-all-keys)
- [Resolution and precedence](#resolution-and-precedence)
- [Output naming](#output-naming)
- [Console output modes](#console-output-modes)
- [Typical collected data](#typical-collected-data)
- [Notifications](#notifications)
- [Upload (S3 / GCS / SFTP)](#upload-s3--gcs--sftp)
- [Operator deployment (groot-selfhosted)](#operator-deployment-groot-selfhosted)
- [Secret redaction](#secret-redaction)
- [Container image](#container-image)
- [Security note](#security-note)
- [Get involved](#get-involved)
- [License](#license)

[↑ Back to top](#readme-top)

## Features

- Cobra CLI with `collect` command
- Viper YAML config + environment variable override
- Concurrent Kubernetes API calls for faster collection
- Worker/node and control plane oriented log gathering
- Output folder + `.tar.gz` archive generation
- Optional notifications (Slack, Discord, Teams, PagerDuty, Telegram, **email/SMTP**, generic webhooks with **templates and HMAC**)
- **Notify on failure** (collect abort or partial job failures above a threshold)
- **HTTP retry/backoff** for transient webhook errors
- Optional **secret redaction** in collected log files
- **In-cluster scheduling** supported via published image; Helm chart and manifests live in **[groot-selfhosted](https://github.com/hrodrig/groot-selfhosted)** (`run/deploy/`)
- Rootless container image (`ghcr.io/hrodrig/groot`; distroless nonroot)

**Libraries** (see [`go.mod`](go.mod)): [Cobra](https://github.com/spf13/cobra) **v1.10.2**, [Viper](https://github.com/spf13/viper) **v1.21.0**, [client-go](https://github.com/kubernetes/client-go) for cluster access (no `kubectl` binary required).

[↑ Back to top](#readme-top)

## Requirements

- A valid **kubeconfig** (or in-cluster config) and network reachability to the Kubernetes API
- RBAC permissions to read logs/resources
- **Go 1.26+** if you [build from source](#quick-start) (`make build`) or use [`go install`](#install-with-go) for the CLI

[↑ Back to top](#readme-top)

## Install or update

Pre-built **`.deb`**, **`.rpm`**, **`.tar.gz`** (and **`.zip`** on Windows) are on **[GitHub Releases](https://github.com/hrodrig/groot/releases)** and **[latest release](https://github.com/hrodrig/groot/releases/latest)**. The **release** badge at the top of this README shows the current tag at a glance.

**Why not a single `latest` URL for every file?** GitHub’s `…/releases/latest/download/<file>` only works if the **asset filename is identical** on every release. From **v0.6.0** onward, basenames embed the **git tag** (for example **`groot_v0.6.0_amd64.deb`**), matching the URL path (`…/download/v0.6.0/groot_v0.6.0_amd64.deb`). Older releases used **`groot_0.5.0_*`** (no `v` in the basename). Options: **pick names from the release page**, use the **snippet below**, or use the **badge**.

### Install latest `.deb` (Debian / Ubuntu, `amd64`)

```bash
# Latest published release tag (python3 or jq). Basename uses the tag (v0.6.0+).
TAG="$(curl -fsSL https://api.github.com/repos/hrodrig/groot/releases/latest | python3 -c 'import json,sys; print(json.load(sys.stdin)["tag_name"])')"
# Alternative: TAG="$(curl -fsSL https://api.github.com/repos/hrodrig/groot/releases/latest | jq -r .tag_name)"

[ -n "$TAG" ] || { echo "Could not resolve tag (empty). Install python3 or jq, or set TAG manually from the Releases page." >&2; exit 1; }

DEB="groot_${TAG}_amd64.deb"
URL="https://github.com/hrodrig/groot/releases/download/${TAG}/${DEB}"
TMP="/tmp/${DEB}"

# Download to /tmp so user _apt can read the file (apt often cannot read ~/.deb when $HOME is mode 700).
if ! curl -fsSL "$URL" -o "$TMP"; then
  echo "Download failed (curl exit $?). Check URL: $URL" >&2
  exit 1
fi
if [ ! -f "$TMP" ]; then
  echo "Expected $TMP after download — not found." >&2
  exit 1
fi
sudo apt install "$TMP"
```

Paste the block **as a whole**, or chain with `&&`, so **`apt` does not run** after a failed **`curl`**. **`curl -f`** exits non‑zero on HTTP errors (404, etc.).

**`apt` + `_apt` / “Permission denied” under `$HOME`:** if you `curl` the `.deb` into **`~`** and run `sudo apt install ./groot_….deb`, Debian/Ubuntu may warn that **`_apt` cannot read the file** (home directory not world-executable). Use **`/tmp`** as above, or `sudo cp "$DEB" /tmp/` then `sudo apt install "/tmp/$DEB"`.

**404 on `groot_v0.1.6_amd64.deb`:** the file on GitHub is **`groot_0.1.6_amd64.deb`** (no `v` in the basename). **Empty `TAG`:** if `jq`/`python3` failed, you get `.../download//groot__amd64.deb` and **`./groot__amd64.deb`** from `apt`.

`groot` is installed to `/usr/bin`. The package drops a **sample** at **`/etc/groot/groot.yml.sample`** (from `configs/groot.yml.sample` in the repo) as a **template**; it is **not** read unless you pass **`--config`**. With no **`--config`**, discovery is **`./groot.yml`**, then **`~/.groot/groot.yml`**, then **`/etc/groot/groot.yml`**, then built-in defaults. Use a per-user file under **`~/.groot/`**, **`sudo cp /etc/groot/groot.yml.sample /etc/groot/groot.yml`** for a machine-wide config, or **`--config /path/to/file.yaml`**. Use `arm64` in the download filename on ARM64.

### Fixed-tag examples (copy from the release page if you prefer)

| Format | Example (tag **`v0.6.0`** in URL path and basename since **v0.6.0**) |
|--------|------------------------------------------------------------------|
| **`.deb`** | `curl -fsSL -o /tmp/groot_v0.6.0_amd64.deb https://github.com/hrodrig/groot/releases/download/v0.6.0/groot_v0.6.0_amd64.deb` then `sudo apt install /tmp/groot_v0.6.0_amd64.deb` (use `/tmp` so `_apt` can read the file if `$HOME` is `700`) |
| **`.rpm`** | `curl -fsSLO https://github.com/hrodrig/groot/releases/download/v0.6.0/groot_v0.6.0_amd64.rpm` then `sudo rpm -Uvh groot_v0.6.0_amd64.rpm` or `sudo dnf install ./groot_v0.6.0_amd64.rpm` |
| **`.tar.gz`** | `curl -fsSLO https://github.com/hrodrig/groot/releases/download/v0.6.0/groot_v0.6.0_linux_amd64.tar.gz` then `tar xzf groot_v0.6.0_linux_amd64.tar.gz` and run `./groot` inside the extracted directory |

**Update:** download a newer release and run the same install command again (`rpm -Uvh`, `apt install` over the `.deb`, or replace the tarball tree).

**Basename change in 0.6.x:** release artifacts switch from `groot_0.5.0_*` to `groot_v0.6.0_*` (v-prefixed tag in the basename, matching pgwd/kzero). Scripts that used `VER="${TAG#v}"` and `groot_${VER}_*` must use `groot_${TAG}_*` instead.

**Supply chain (v0.6.0+):** each release attaches **SPDX** and **CycloneDX** SBOMs, **Cosign** signatures for `checksums.txt` and GHCR images. Verify checksums with `cosign verify-blob` (see release assets); verify images with `cosign verify --certificate-identity-regexp=… ghcr.io/hrodrig/groot:v0.6.0`.

**Windows:** use the **`.zip`** asset for your arch, unpack, and run `groot.exe` on a host that can reach the Kubernetes API with a valid **kubeconfig** (or in-cluster credentials).

Then [configure](#first-run) and run `groot collect` (or `groot --print-sample-config > groot.yml` first).

[↑ Back to top](#readme-top)

## Quick start

Build from a clone of this repository:

```bash
make build
./bin/groot --print-sample-config > groot.yml
# Edit groot.yml: replace sample values with your cluster settings (namespaces, targets,
# kubeconfig, output paths, optional notify webhooks/tokens) before collecting.
./bin/groot collect
```

If you installed from a **release package**, use `groot` on your `PATH` instead of `./bin/groot`.

### Install with Homebrew (macOS / Linux, recommended for desktop)

```bash
brew tap hrodrig/groot
brew install --cask hrodrig/groot/groot
```

**Upgrading** keeps the same tap; new releases are picked up automatically:

```bash
brew upgrade --cask hrodrig/groot/groot
```

The cask installs the **`groot`** binary to `$(brew --prefix)/bin/groot` and adds it to your `PATH` (already on it in default Homebrew setups). A **sample config** is not bundled with the cask; generate it with `groot --print-sample-config > ~/.config/groot/groot.yml` and edit.

**macOS first run:** unsigned CLI binaries may trigger Gatekeeper (“Apple could not verify…”). The cask clears the download quarantine on install. If you still see the dialog, run once:

```bash
xattr -dr com.apple.quarantine "$(command -v groot)"
```

Or use **System Settings → Privacy & Security → Open Anyway**. Same applies to binaries installed from GitHub `.tar.gz` (see [Install or update](#install-or-update)).

> The tap repo lives at **[github.com/hrodrig/homebrew-groot](https://github.com/hrodrig/homebrew-groot)** (`Casks/groot.rb`). GoReleaser updates it on every tag via the `homebrew_casks:` stanza in `.goreleaser.yaml`. CI needs secret **`HOMEBREW_TAP_TOKEN`** (PAT with `repo` scope on the tap).

### Install with Go

From any machine with Go **1.26+** (installs to `$(go env GOPATH)/bin`; ensure that directory is on your `PATH`):

```bash
go install github.com/hrodrig/groot/cmd/groot@latest
```

Use a **release tag** instead of `@latest` if you want a pinned version (for example `@v0.6.0`). Documentation for the module: [pkg.go.dev/github.com/hrodrig/groot](https://pkg.go.dev/github.com/hrodrig/groot).

Useful runtime flags (global or with `collect`):

- `--version` prints version, commit, branch, and build date
- `--test-connection` validates Kubernetes connectivity and exits
- `--verbose` shows each executed command as `CMD`, plus `OK`/`ERR` results
- `--quiet` suppresses normal **console** output (INFO/WARN/CMD/OK) and only prints errors; **notify integrations still run** (Slack, Discord, Teams, PagerDuty, Telegram, generic) unless you disable them in config or use `--no-notify`
- `--no-notify` skips all notifications after a successful collect (useful for cron when you only want the archive). Same effect as env `GROOT_NO_NOTIFY=1` (or `true` / `yes`, case-insensitive)
- `--no-upload` skips post-collect S3/GCS/SFTP upload when `upload.enabled` is true. Same effect as env `GROOT_NO_UPLOAD=1`
- `--no-color` disables ANSI colors
- `--message "label text"` appends a sanitized suffix to archive and capture-related output names
- `--kubeconfig /path/to/config` overrides kubeconfig from file/env
- **`collect` only:** `--since` limits **pod** log collection to lines newer than a duration (same semantics as the Kubernetes **`--since`** filter on pod logs). A **bare number** is treated as **hours** (for example `--since=24` → `24h`). Other forms follow Go durations (`24h`, `45m`, `90s`). Overrides `collection.pod_logs_since` from config when passed.

[↑ Back to top](#readme-top)

## First run

If you do not have a config file yet, print a sample and save it:

```bash
./bin/groot --print-sample-config > groot.yml
```

The sample YAML is written to **standard output**, so shell redirection (`>`) works as shown. If you use an **older** `groot` binary where `>` produced an empty file, redirect **stderr** instead: `groot --print-sample-config 2> groot.yml`.

The generated file is a **template only**. Open `groot.yml` and set **your own** values for your environment—for example `kubeconfig` (if not using the default), `collection.namespaces`, workloads under `collection.targets` (deployments, StatefulSets, DaemonSets, Jobs, CronJobs, Helm releases), `output_dir` / `file_prefix`, `collection.redact_secrets`, and any `notify.*` URLs or secrets. Until you do, the sample names and disabled notification blocks will not match a real cluster.

Then run:

```bash
./bin/groot collect
```

Default config discovery order (when `--config` is not provided). The **first existing file** wins; if none exist, built-in defaults apply, then `GROOT_*` environment variables override where applicable:

1. `./groot.yml`
2. `~/.groot/groot.yml`
3. `/etc/groot/groot.yml`
4. built-in defaults (then `GROOT_*` env overrides where applicable)

The **`.deb` / `.rpm`** sample at **`/etc/groot/groot.yml.sample`** is not part of this chain; copy it to **`groot.yml`** or pass **`--config /etc/groot/groot.yml.sample`** explicitly.

You can always override file discovery with `--config` (see [Usage examples](#usage-examples)).

[↑ Back to top](#readme-top)

## Usage examples

Paths below use `./bin/groot` after `make build`; if you installed from [Releases](#install-or-update) or `make install`, use `groot` on your `PATH` the same way (for example `groot collect ...`).

### Use a specific config file

Paths **`./groot.yml`**, **`~/.groot/groot.yml`**, and **`/etc/groot/groot.yml`** are discovered automatically (see [First run](#first-run)). Any other path—including **`/etc/groot/groot.yml.sample`**—**must** be passed explicitly:

```bash
./bin/groot collect --config /path/to/my-groot.yml
./bin/groot collect --config ./groot-mi-test.yml
```

From the repository root, after editing your copy:

```bash
./bin/groot collect --config groot-mi-test.yml
```

### Check Kubernetes access and config (no collection)

```bash
./bin/groot --config ./groot-mi-test.yml --test-connection
./bin/groot collect --config ./groot-mi-test.yml --test-connection
```

### Cron: quiet console, no outbound notifications

Console only (Slack/Discord/etc. still run if enabled in YAML):

```bash
./bin/groot collect --config /path/to/groot.yml --quiet
```

Skip **all** notify channels for this run (archive still created); same as env `GROOT_NO_NOTIFY=1` / `true` / `yes`:

```bash
./bin/groot collect --config /path/to/groot.yml --quiet --no-notify
0 * * * * GROOT_NO_NOTIFY=1 /usr/local/bin/groot collect --config /home/you/.groot/prod.yml --quiet
```

### Preview planned jobs (`--list-jobs`)

Print API jobs and output paths without writing disk or sending notify:

```bash
./bin/groot collect --config groot.yml --list-jobs
# pod-logs-default-api -> default/my-pod__node-1.log args=[logs -n default my-pod --all-containers]
```

### Notify on failure

When `notify.on_failure.enabled: true`, Groot can alert on **abort** (archive error, timeout, …) or when **`failed >= min_failed_jobs`** on a completed run (in addition to the normal success notify). Respects `--no-notify`.

```yaml
notify:
  slack:
    enabled: true
    webhook_url: "https://hooks.slack.com/services/..."
  on_failure:
    enabled: true
    on_abort: true
    min_failed_jobs: 2
```

### Secret redaction (optional)

Scan collected `*.log` files and replace likely secrets before archiving:

```yaml
collection:
  redact_secrets: true
  redact_patterns:
    - '(?i)my-internal-token\s*=\s*\S+'
```

### Custom capture label (`--message`)

```bash
./bin/groot collect --config groot.yml --message "staging-network-audit-2026-04-28"
```

### Pod logs: last N hours (`--since`)

Same as `collection.pod_logs_since` in YAML; **bare number = hours** (here, last 24 hours of pod logs):

```bash
./bin/groot collect --config groot.yml --since=24
```

**Empty `*.log` files are normal** when you narrow the window: with **`--since`**, the API only returns lines **newer than that duration**. If a pod was quiet during the window, Groot still writes the file (often **0 bytes**) — that is **not** a Groot bug, it means **no stdout/stderr in that interval**. Widen the window, drop **`--since`** for a test run, or raise **`pod_log_tail_lines`** to confirm the workload emitted output during the capture.

### Override kubeconfig for one run

```bash
./bin/groot collect --config groot.yml --kubeconfig /path/to/other-kubeconfig
```

[↑ Back to top](#readme-top)

## Config

Edit `groot.yml` (or any file passed with `--config`) and align every section with your cluster and operational needs. Do not rely on the shipped sample as a drop-in configuration.

The **annotated** template (every key explained in comments) is **[`configs/groot.yml.sample`](configs/groot.yml.sample)** — identical to `groot --print-sample-config` output. Use that file when you want line-by-line guidance next to the YAML.

Sample config (abbreviated; full annotated template in **[`configs/groot.yml.sample`](configs/groot.yml.sample)** — same as `groot --print-sample-config`):

```yaml
kubeconfig: ""
output_dir: "./out"
file_prefix: "groot-capture"

collection:
  timeout: 20m
  worker_concurrency: 6
  namespaces:
    - kube-system
    - default
  targets:
    default:
      deployments:
        - api
      jobs:
        - batch-import
      cronjobs:
        - nightly-sync
  include_pod_logs: true
  pod_log_tail_lines: 1500
  # pod_logs_since: "24"
  include_node_details: true
  include_node_logs: true
  include_pod_metrics: true
  redact_secrets: false
  extra_kubectl:
    - "get ingress -A"
    - "get pvc -A"

notify:
  on_failure:
    enabled: false
    on_abort: true
    min_failed_jobs: 1
  retry:
    max_attempts: 3
    initial_backoff: 1s
    max_backoff: 10s
  slack:
    enabled: false
    webhook_url: ""
  generic:
    enabled: false
    webhook_url: ""
    json_key: "text"
    # body_template: '{"text":"{{summary}}","failed":{{failed}},"event":"{{event}}"}'
    # hmac_secret: ""   # or env GROOT_NOTIFY_GENERIC_HMAC_SECRET
  email:
    enabled: false
    host: ""
    port: 587
    from: ""
    to: ""
```

### Configuration reference (all keys)

#### Top-level

| Key | What it does |
|-----|----------------|
| **`kubeconfig`** | Path to the kubeconfig file used to build the **client-go** REST config (same discovery rules as **client-go** / **`clientcmd`**). Empty: use **`KUBECONFIG`** if set, then the default kubeconfig locations (for example **`~/.kube/config`**), or in-cluster credentials when Groot runs as a pod. **`groot --kubeconfig`** overrides this for a single run (see [Resolution and precedence](#resolution-and-precedence)). |
| **`output_dir`** | Base directory: each run creates **`<file_prefix>-<timestamp>[-since-<slug>]/`**, then **`<sessionBase>-<cluster>[-<message>].tar.gz`** beside it. Supports **`~`** and **`${VAR}`** expansion. |
| **`file_prefix`** | Prefix for capture directory and archive basename (default **`groot-capture`**). Example session: **`groot-capture-20260606-120000-my-cluster.tar.gz`**. |
| **`collection`** | Tuning for timeouts, parallelism, namespaces, pod logs, optional **`extra_kubectl`** argv lines, redaction, etc. (see below). |
| **`notify`** | Optional webhooks, email, and failure alerts after collect (see [Notifications](#notifications)). |

#### `collection`

Pod ↔ node placement at capture start is in **`extras/all-pod-node-placement.tsv`** (fourth column **`pod_log_file`** when Groot collects that pod’s log). After all jobs finish, **`extras/all-pods-rca.tsv`** merges that placement with **cluster-wide pod metrics** from **metrics.k8s.io** (when **`include_pod_metrics`** is on — the same snapshot **`top pods -A`** would show) so you get **namespace, pod, node, cpu_cores, memory_bytes, pod_log_file** in one table — **cluster-wide** and aligned with Groot’s log paths for RCA handoff.

| Key | What it does |
|-----|----------------|
| **`timeout`** | Maximum wall time for the whole **`groot collect`** run (Go `context` deadline). |
| **`worker_concurrency`** | Number of **parallel** collection workers (concurrent API jobs). |
| **`namespaces`** | For each entry, Groot lists namespace-scoped resources through the API (pods, services, Deployments, ReplicaSets, StatefulSets, DaemonSets), writes **JSON sections** to **`<ns>/resources.txt`**, and ensures **`<ns>/`** exists under the capture tree. |
| **`targets`** | Per-namespace **pod log** filters only. Keys are **namespace names**. Under each: **`deployments`**, **`statefulsets`**, **`daemonsets`**, **`jobs`**, **`cronjobs`**, **`helm_releases`** (string lists). If a namespace has **at least one** non-empty list, only pods whose labels match those workloads get **log** jobs. Empty/missing entry → broad pod logs for that NS. |
| **`include_pod_logs`** | When **`true`**, collects **pod logs** for workload and control-plane pods via the API (subject to **`targets`**, **`pod_log_tail_lines`**, **`pod_logs_since`**). When **`false`**, skips all pod log jobs. |
| **`include_previous_logs`** | When **`true`**, also collects **previous-container** logs into **`*.previous.log`** (same semantics as **`--previous`** on pod logs; marked optional so a missing previous container does not fail the run). |
| **`pod_log_tail_lines`** | When **`>0`**, passes **`--tail N`** to pod log commands. **`0`** means **no `--tail`** (full log stream — can be very large). |
| **`pod_logs_since`** | When set, passes **`--since=…`** to **pod log** commands only (digits-only = **hours**, e.g. **`24`** → **`24h`**; otherwise a Go duration like **`24h`**, **`45m`**). **`groot collect --since`** overrides this when the flag is set. The capture directory and **`.tar.gz`** basename include **`since-<slug>`** after the timestamp so runs with a log window are identifiable on disk (see [Output naming](#output-naming)). |
| **`include_node_details`** | When **`true`**, for each node writes **describe**-style summaries and **node metrics** (when the metrics API is available) under **`nodes/`**. |
| **`include_node_logs`** | When **`true`**, for each node: (1) **GET** **`/api/v1/nodes/<node>/proxy/logs/?query=kubelet`** (optional **`&tailLines=N`**) → **`nodes/<node>-kubelet.log`** (kubelet via **node log query**; Kubernetes **1.27+**, RBAC **`nodes/proxy`**, kubelet log-query settings — see [Node log query](https://kubernetes.io/blog/2023/04/21/node-log-query-alpha/)); (2) **GET** **`/api/v1/nodes/<node>/proxy/logs/messages`** → **`nodes/<node>-messages.log`** (host **`/var/log/messages`** when the kubelet serves it). The **messages** job is **optional** (failure does not fail the run) because many nodes use journald only or do not expose that path. |
| **`node_log_tail_lines`** | When **`>0`**, appends **`tailLines`** to the kubelet log query (**default `5000`**). **`0`** omits **`tailLines`** (server default limit). |
| **`include_pod_metrics`** | When **`true`**, writes cluster-wide pod CPU/memory to **`extras/all-pods-top.txt`** (via **metrics.k8s.io**; requires **metrics-server** or an equivalent metrics provider). |
| **`redact_secrets`** | When **`true`**, scans collected **`*.log`** files and replaces likely secret values with **`[REDACTED]`** before the manifest and archive (see [Secret redaction](#secret-redaction)). Default **`false`**. |
| **`redact_patterns`** | Optional list of extra regex patterns (RE2 syntax). Invalid patterns fail at collect time. |
| **`extra_kubectl`** | List of extra **read-only** argv lines (allowlisted verbs; split on whitespace, **no shell**). Groot executes them in-process with **client-go**. See the note below on allowed verbs. |

#### `notify` (each channel)

| Block / field | What it does |
|----------------|----------------|
| **`on_failure`**: **`enabled`**, **`on_abort`**, **`min_failed_jobs`** | Optional alerts when collect **aborts** or when **`failed >= min_failed_jobs`** on a completed run. Respects **`--no-notify`**. |
| **`retry`**: **`max_attempts`**, **`initial_backoff`**, **`max_backoff`** | Retries transient **5xx** and network errors for HTTP notify clients (webhooks, PagerDuty). |
| **`slack`**, **`discord`**, **`teams`**: **`enabled`**, **`webhook_url`** | POST a one-line summary to incoming webhook URL(s). Multiple URLs: **`;`**. Env: **`GROOT_NOTIFY_*_WEBHOOK_URL`**. |
| **`pagerduty`**: **`enabled`**, **`routing_key`**, **`severity`**, **`source`** | Events API v2 trigger. Env: **`GROOT_NOTIFY_PAGERDUTY_ROUTING_KEY`**. |
| **`telegram`**: **`enabled`**, **`token`**, **`chat_id`** | Bot API. Env: **`GROOT_NOTIFY_TELEGRAM_TOKEN`**, **`GROOT_NOTIFY_TELEGRAM_CHAT_ID`**. |
| **`generic`**: **`webhook_url`**, **`json_key`**, **`headers`**, **`extra_fields`**, **`body_template`**, **`hmac_secret`**, **`hmac_header`** | Custom JSON POST; see [Notifications](#notifications). Env: **`GROOT_NOTIFY_GENERIC_WEBHOOK_URL`**, **`GROOT_NOTIFY_GENERIC_HMAC_SECRET`**. |
| **`email`**: **`host`**, **`port`**, **`username`**, **`password`**, **`from`**, **`to`**, **`use_tls`** | Plain-text summary via SMTP (STARTTLS on 587 by default). Env: **`GROOT_NOTIFY_EMAIL_*`**. |

Environment variables use the `GROOT_` prefix (Viper). Nested YAML keys map to env names by replacing `.` with `_` (for example `collection.timeout` → `GROOT_COLLECTION_TIMEOUT`). `kubeconfig` in YAML still loses to the process `KUBECONFIG` env when that is set (see [Resolution and precedence](#resolution-and-precedence)).

Common examples:

- `GROOT_OUTPUT_DIR`, `GROOT_FILE_PREFIX`
- `GROOT_COLLECTION_TIMEOUT`, `GROOT_COLLECTION_WORKER_CONCURRENCY`, `GROOT_COLLECTION_INCLUDE_POD_LOGS` (boolean), `GROOT_COLLECTION_POD_LOG_TAIL_LINES`, `GROOT_COLLECTION_POD_LOGS_SINCE`, …
- Notify secrets (also read when `enabled: true` and the YAML field is empty): `GROOT_NOTIFY_SLACK_WEBHOOK_URL`, `GROOT_NOTIFY_DISCORD_WEBHOOK_URL`, `GROOT_NOTIFY_TEAMS_WEBHOOK_URL`, `GROOT_NOTIFY_TELEGRAM_TOKEN`, `GROOT_NOTIFY_TELEGRAM_CHAT_ID`, `GROOT_NOTIFY_GENERIC_WEBHOOK_URL`, `GROOT_NOTIFY_GENERIC_HMAC_SECRET`, `GROOT_NOTIFY_PAGERDUTY_ROUTING_KEY`, `GROOT_NOTIFY_EMAIL_HOST`, `GROOT_NOTIFY_EMAIL_USERNAME`, `GROOT_NOTIFY_EMAIL_PASSWORD`, `GROOT_NOTIFY_EMAIL_FROM`, `GROOT_NOTIFY_EMAIL_TO`
- `GROOT_NO_NOTIFY=1` (or `true` / `yes`): same as `--no-notify` for a run

**`collection.extra_kubectl`:** Each string is split on whitespace into argv tokens (no shell). At load time, Groot only accepts **read-oriented** leading verbs: `get`, `describe`, `top`, `logs`, `api-resources`, `api-versions`, `version`, `cluster-info`, plus `config view …` and `auth can-i …`. Anything else fails `collect` immediately with a configuration error so a typo or copy-paste cannot turn extras into destructive verbs (`delete`, `exec`, `apply`, etc.).

When a notification channel is enabled and required credentials are missing, `groot` fails fast with a clear configuration error.

[↑ Back to top](#readme-top)

## Resolution and precedence

Configuration file precedence:

1. `--config` explicit path
2. `./groot.yml`
3. `~/.groot/groot.yml`
4. `/etc/groot/groot.yml`
5. defaults

`kubeconfig` precedence:

1. `--kubeconfig /path/to/config`
2. `KUBECONFIG`
3. `kubeconfig` value in YAML
4. if all empty, **client-go** / **`clientcmd`** default kubeconfig discovery (including in-cluster when applicable)

Workload filter behavior (`collection.targets`):

- per namespace, you can define `deployments`, `statefulsets`, `daemonsets`, `jobs`, `cronjobs`, and `helm_releases`
- if a namespace has targets with at least one non-empty list, pod logs for that namespace are limited to matching pods
- if a namespace has no targets entry, or all lists are empty, pod logs stay broad for that namespace
- `jobs` / `cronjobs` match label keys plus **`job-name`** on Job pods
- `helm_releases` matches `app.kubernetes.io/instance`

`pod_log_tail_lines` behavior:

- `0`: collect full logs (no `--tail`; use when you need the entire log stream)
- `>0`: collect only the last N lines per pod
- applies to both current and `--previous` pod logs

`pod_logs_since` and **`collect --since`** (pod logs only):

- applies **`--since`** / time-window filtering to workload and control-plane **pod log** jobs; other capture jobs are unchanged
- in YAML or env, a **string of digits only** is interpreted as **whole hours** (`"24"` → `24h`); otherwise the value must parse as a Go duration (`24h`, `45m`, …)
- **`groot collect --since=…`** overrides `collection.pod_logs_since` for that run when the flag is set

`include_previous_logs` behavior:

- `true`: also collects **previous-container** pod logs into `*.previous.log` (optional jobs; same idea as **`--previous`** on pod logs)
- `false`: collects only current pod logs

`output_dir` path expansion:

- supports `~` (home directory), for example `~/tmp/groot-out`
- supports environment variables, for example `${HOME}/tmp/groot-out`

[↑ Back to top](#readme-top)

## Output naming

Capture output names use **`file_prefix`** (default **`groot-capture`**):

- **directory:** `<file_prefix>-<timestamp>` or `<file_prefix>-<timestamp>-since-<slug>` when **`pod_logs_since`** / **`--since`** is set
- **archive:** `<sessionBase>-<cluster>[-<message>].tar.gz` (for example **`groot-capture-20260606-120000-my-cluster.tar.gz`**)

When **`pod_logs_since`** is set, **`<slug>`** is a filesystem-safe form of the duration (for example **`12h`**, **`45m`**).

`--message` is sanitized before use:

- lowercase
- trims leading/trailing spaces
- removes accents/diacritics
- converts spaces and `_` to `-`
- removes unsupported filesystem characters
- collapses repeated dashes

Example:

- input: `--message "network routing issue"`
- suffix: `network-routing-issue`
- output: `groot-capture-20260428-123200-my-cluster-network-routing-issue.tar.gz`
- with **`pod_logs_since`** (or **`--since`**) set to **`12h`** and no message: `groot-capture-20260428-123200-since-12h-my-cluster.tar.gz`

Directory layout:

- `nodes/`
- `extras/`
- one directory per configured namespace (for example `kube-system/`, `default/`)
- pod log files: `<pod>__<node>.log` (and `.previous.log` when enabled), same pattern for control-plane pods under `kube-system/`
- after archive creation, the timestamp directory is automatically removed

Inside the `.tar.gz`, every path is prefixed with the capture folder name (`<session>/…`, for example `20260502-174207/kube-system/…` or `20260503-081049-since-12h/kube-system/…`). Extracting into a shared directory (for example `~/tmp/groot-out`) keeps each run under its own subdirectory instead of mixing `kube-system/`, `production/`, etc. at the extraction root. Archives produced by older Groot versions may still have a flat layout at the tar root.

[↑ Back to top](#readme-top)

## Console output modes

- default: summary `INFO` lines
- `--verbose`: adds per-command `CMD` / `OK` / `ERR`
- `--quiet`: suppresses normal **console** output, prints only errors; does **not** disable webhooks/API notifications
- `--no-notify`: skips every notify channel for this run (config can still have `enabled: true`; use from cron when you want silence to external systems). Env equivalent: `GROOT_NO_NOTIFY=1`
- `--no-color`: disables ANSI colors

[↑ Back to top](#readme-top)

## Typical collected data

These artifacts mirror common read-only inspection commands (all via **client-go**):

- **`extras/cluster-info.txt`** — discovery / server summary (`cluster-info`)
- **`extras/nodes-wide.txt`** — all nodes, wide columns (`get nodes -o wide`)
- **`extras/all-pods-wide.txt`** — all pods cluster-wide, wide columns (`get pods -A -o wide`)
- **`extras/all-cluster-events.log`** — all events, sorted by last timestamp (`get events -A`)
- Under **`nodes/`** — per-node **describe**-style output and **node metrics** when enabled (`describe node`, `top node`)
- Pod logs — streams all containers like **`logs -n <ns> <pod> --all-containers`** → files named `<pod>__<node>.log` under each namespace directory (pending/unscheduled pods use `unknown-node`)
- Control plane pod logs in `kube-system` (`tier=control-plane`, when available) use the same `<pod>__<node>.log` pattern
- **`extras/kubeconfig.txt`** derived from kubeconfig (`context`, `cluster`, `user`, `server`)
- **`extras/manifest.json`** — archive manifest (version, cluster, job counts, file paths)

[↑ Back to top](#readme-top)

## Notifications

Groot sends a **one-line summary** after collect. Channels are independent; enable only what you need.

**Success message (all channels):**

```text
GROOT finished. total=42 success=40 failed=2 duration=3m12s output=/out/groot-capture-… archive=/out/groot-capture-….tar.gz
```

**Failure messages** (when `notify.on_failure.enabled: true`):

```text
GROOT FAILED. reason=archive logs: … total=42 success=40 failed=2 …
GROOT finished with failures. total=42 success=40 failed=2 …
```

### Slack / Discord / Teams

```yaml
notify:
  slack:
    enabled: true
    webhook_url: "https://hooks.slack.com/services/T…/B…/…;https://hooks.slack.com/services/…"
  discord:
    enabled: true
    webhook_url: "https://discord.com/api/webhooks/…"
```

Env fallbacks: `GROOT_NOTIFY_SLACK_WEBHOOK_URL`, `GROOT_NOTIFY_DISCORD_WEBHOOK_URL`, `GROOT_NOTIFY_TEAMS_WEBHOOK_URL`. Discord truncates content to 2000 runes.

### PagerDuty Events v2

```yaml
notify:
  pagerduty:
    enabled: true
    routing_key: "your-integration-key"
    severity: warning   # critical | error | warning | info
    source: groot
```

Env: `GROOT_NOTIFY_PAGERDUTY_ROUTING_KEY`. Expects HTTP **202**. `custom_details` includes job counts, duration, paths.

### Telegram

```yaml
notify:
  telegram:
    enabled: true
    token: "123456:ABC…"
    chat_id: "-1001234567890;123456789"
```

Env: `GROOT_NOTIFY_TELEGRAM_TOKEN`, `GROOT_NOTIFY_TELEGRAM_CHAT_ID`.

### Generic webhook (JSON template, extra fields, HMAC)

**Simple (default shape):**

```yaml
notify:
  generic:
    enabled: true
    webhook_url: "https://internal.example/hooks/groot"
    json_key: "text"
    extra_fields:
      source: "groot"
      environment: "production"
```

POST body: `{"text":"<summary>","source":"groot","environment":"production"}`.

**Custom JSON template** (placeholders: `{{summary}}`, `{{text}}`, `{{event}}`, `{{total}}`, `{{success}}`, `{{failed}}`, `{{duration}}`, `{{output_dir}}`, `{{archive_path}}`, `{{reason}}`):

```yaml
notify:
  generic:
    enabled: true
    webhook_url: "https://internal.example/hooks/groot"
    body_template: '{"event":"{{event}}","message":"{{summary}}","stats":{"total":{{total}},"failed":{{failed}}}}'
    headers:
      Authorization: "Bearer ${INTERNAL_TOKEN}"
    hmac_secret: "shared-signing-key"
    hmac_header: "X-Groot-Signature"
```

When `hmac_secret` is set, Groot sends **`X-Groot-Signature: sha256=<hex>`** (HMAC-SHA256 over the raw POST body). Env: `GROOT_NOTIFY_GENERIC_HMAC_SECRET`.

### Email (SMTP)

```yaml
notify:
  email:
    enabled: true
    host: smtp.example.com
    port: 587
    username: groot-bot
    password: "${SMTP_PASSWORD}"
    from: groot@example.com
    to: "ops@example.com;oncall@example.com"
    use_tls: false   # true for implicit TLS (e.g. port 465)
```

Env: `GROOT_NOTIFY_EMAIL_HOST`, `GROOT_NOTIFY_EMAIL_USERNAME`, `GROOT_NOTIFY_EMAIL_PASSWORD`, `GROOT_NOTIFY_EMAIL_FROM`, `GROOT_NOTIFY_EMAIL_TO`.

### Notify on failure and retry

```yaml
notify:
  on_failure:
    enabled: true
    on_abort: true          # alert when collect aborts (timeout, archive error, …)
    min_failed_jobs: 1      # also alert when failed jobs >= this (success path)
  retry:
    max_attempts: 3
    initial_backoff: 1s
    max_backoff: 10s
  slack:
    enabled: true
    webhook_url: "https://hooks.slack.com/services/…"
```

Partial-failure alerts are **in addition to** the normal success notify. **`--no-notify`** / **`GROOT_NO_NOTIFY=1`** skips **all** channels including failure alerts.

HTTP clients retry transient **5xx** and network errors only; **4xx** fails immediately.

[↑ Back to top](#readme-top)

## Upload (S3 / GCS / SFTP)

After a successful collect, optionally upload the **`.tar.gz`** to object storage. Credentials come from the standard **AWS** / **Google** SDK env vars (not from long-lived keys in YAML).

```yaml
upload:
  enabled: true
  continue_on_error: true
  s3:
    enabled: true
    bucket: my-archives
    region: us-east-1
    key_prefix: groot/prod
```

Env overrides: `GROOT_UPLOAD_S3_BUCKET`, `GROOT_UPLOAD_S3_REGION`, `GROOT_UPLOAD_S3_KEY_PREFIX`, `GROOT_UPLOAD_S3_ENDPOINT` (S3-compatible), `GROOT_UPLOAD_GCS_BUCKET`, `GROOT_UPLOAD_GCS_KEY_PREFIX`. Upload runs **after** notify; failures are logged but **do not fail** the collect. Skip with **`--no-upload`** or **`GROOT_NO_UPLOAD=1`**.

Minimum IAM: S3 `s3:PutObject` on the bucket/prefix; GCS `roles/storage.objectCreator` (or tighter custom role).

### SFTP (airgapped clusters)

For clusters **without outbound internet**, GROOT runs on a **bastion** (has kubeconfig) and pushes `.tar.gz` via SFTP to a **relay host** that syncs to cloud storage.

```yaml
upload:
  enabled: true
  sftp:
    enabled: true
    host: ipA.example.com
    port: 22
    user: groot-inbox
    remote_dir: inbox
    known_hosts_file: /etc/groot/known_hosts
```

Identity from env (never in YAML): `GROOT_UPLOAD_SFTP_IDENTITY_FILE=/home/groot/.ssh/id_ed25519`

Auth: **public-key only** (BatchMode — password rejected). Host key verified against `known_hosts_file`; fails closed on mismatch. Same failure semantics as S3/GCS.

Full relay playbook (systemd watcher → rclone → OneDrive): **[groot-selfhosted → airgapped-relay](https://github.com/hrodrig/groot-selfhosted/blob/main/run/examples/airgapped-relay/README.md)**.

[↑ Back to top](#readme-top)

## Operator deployment (groot-selfhosted)

This repository ships the **CLI**, **packages**, and **`ghcr.io/hrodrig/groot`** image. How you **run** GROOT in production — bastion Docker, cron, Helm CronJob, flat Kubernetes manifests — lives in the operator repo:

**[github.com/hrodrig/groot-selfhosted](https://github.com/hrodrig/groot-selfhosted)** → start at [`run/README.md`](https://github.com/hrodrig/groot-selfhosted/blob/main/run/README.md)

| Mode | Where |
|------|--------|
| Docker / Podman on a bastion | [run/docker/](https://github.com/hrodrig/groot-selfhosted/tree/main/run/docker) |
| Helm CronJob (in-cluster) | [run/deploy/](https://github.com/hrodrig/groot-selfhosted/tree/main/run/deploy) · **`helm repo add groot https://hrodrig.github.io/groot-selfhosted`** |
| Flat CronJob YAML | [run/deploy/k8s/cronjob.yaml](https://github.com/hrodrig/groot-selfhosted/blob/main/run/deploy/k8s/cronjob.yaml) |
| cron / systemd (Releases binary) | [run/standalone/](https://github.com/hrodrig/groot-selfhosted/tree/main/run/standalone) |

Pull the image from here:

```bash
docker pull ghcr.io/hrodrig/groot:0.6.1
```

In-cluster behavior (CronJob, RBAC, `/out` volume) is documented in [SPEC §8](docs/SPECIFICATIONS.md#8-runtime-and-kubernetes-access).

[↑ Back to top](#readme-top)

## Secret redaction

Collected logs may contain passwords, tokens, or API keys. Enable an optional scrub pass **before** the archive is written:

```yaml
collection:
  redact_secrets: true
  redact_patterns:
    - '(?i)corp-internal-key\s*=\s*\S+'
```

**Behavior:**

- Scans only **`*.log`** files under the capture tree
- Built-in patterns match common key names (`password`, `token`, `Bearer …`, `api_key`, …)
- Matches are replaced with **`[REDACTED]`**
- Off by default; does not guarantee all secrets are removed — treat archives as sensitive

Example line before/after:

```text
authorization: Bearer eyJhbGciOiJIUzI1NiIs…
authorization: [REDACTED]
```

[↑ Back to top](#readme-top)

## Container image

Published image: **`ghcr.io/hrodrig/groot`** (distroless **nonroot**). Build locally with `make docker-build` when hacking on this repo.

For **`docker run`** / Podman with kubeconfig mounts, cron, Helm, and in-cluster manifests, see **[groot-selfhosted](https://github.com/hrodrig/groot-selfhosted)**.

[↑ Back to top](#readme-top)

## Security note

Collected logs may contain sensitive data (secrets, PII, credentials). Handle archives according to your security policy.

Optional **`collection.redact_secrets`** reduces accidental exposure in **`*.log`** files but is **not** a guarantee — review before sharing archives externally. Restrict access to **`output_dir`** and in-cluster PVCs; keep notify credentials in env/Secrets, not committed ConfigMaps.

[↑ Back to top](#readme-top)

## Get involved

Found Groot useful? We'd love your help to make it better. You can:

- **Report bugs** or **suggest features** — [open an issue](https://github.com/hrodrig/groot/issues)
- **Contribute code** — see [CONTRIBUTING.md](./CONTRIBUTING.md) for how to submit a pull request
- **Star the repo** — it helps others discover Groot

Thanks for using Groot.

[↑ Back to top](#readme-top)

## License

Groot is distributed under the **MIT License**. The full text is in **[`LICENSE`](LICENSE)** in this repository.
