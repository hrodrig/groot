<p align="center">
  <img src="docs/assets/groot-readme-hero.png" alt="GROOT — Kubernetes diagnostics CLI" width="100%" />
</p>

# GROOT - Go Kubernetes Emergency Logger
[![Version](https://img.shields.io/badge/version-0.1.0-blue)](#)
[![Release](https://img.shields.io/github/v/release/hrodrig/groot?label=release)](https://github.com/hrodrig/groot/releases)
[![Go](https://img.shields.io/badge/Go-1.26-00ADD8?logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/license-MIT-green)](./LICENSE)

GROOT is a Go CLI (Cobra + Viper) that collects broad Kubernetes diagnostics, including worker/node details, control plane logs, namespace resources, pod logs, and events.

## Table of contents

- [Features](#features)
- [Requirements](#requirements)
- [Quick start](#quick-start)
- [First run](#first-run)
- [Config](#config)
- [Resolution and precedence](#resolution-and-precedence)
- [Output naming](#output-naming)
- [Console output modes](#console-output-modes)
- [Typical collected data](#typical-collected-data)
- [Notifications](#notifications)
- [Rootless container](#rootless-container)
- [Security note](#security-note)

## Features

- Cobra CLI with `collect` command
- Viper YAML config + environment variable override
- Concurrent `kubectl` execution for faster collection
- Worker/node and control plane oriented log gathering
- Output folder + `.tar.gz` archive generation
- Optional notifications to Slack, Telegram, and Teams
- Rootless container image support

## Requirements

- Go 1.26+
- `kubectl` configured against target cluster
- RBAC permissions to read logs/resources

## Quick start

```bash
make build
./bin/groot --print-sample-config > groot.yml
# Edit groot.yml: replace sample values with your cluster settings (namespaces, targets,
# kubeconfig, output paths, optional notify webhooks/tokens) before collecting.
./bin/groot collect
```

Useful runtime flags (global or with `collect`):

- `--version` prints version, commit, branch, and build date
- `--test-connection` validates Kubernetes connectivity and exits
- `--verbose` shows each executed command as `CMD`, plus `OK`/`ERR` results
- `--quiet` suppresses normal console output and only prints errors
- `--no-color` disables ANSI colors
- `--message "RCA text"` appends a sanitized suffix to output names
- `--kubeconfig /path/to/config` overrides kubeconfig from file/env

## First run

If you do not have a config file yet, print a sample and save it:

```bash
./bin/groot --print-sample-config > groot.yml
```

The generated file is a **template only**. Open `groot.yml` and set **your own** values for your environment—for example `kubeconfig` (if not using the default), `collection.namespaces`, workloads under `collection.targets` (deployments, StatefulSets, DaemonSets, Helm releases), `output_dir` / `file_prefix`, and any `notify.*` URLs or secrets. Until you do, the sample names and disabled notification blocks will not match a real cluster.

Then run:

```bash
./bin/groot collect
```

Default config discovery order (when `--config` is not provided):

1. `./groot.yml`
2. `~/.groot/groot.yml`
3. built-in defaults and `GROOT_*` environment variables

You can always override file discovery with:

```bash
./bin/groot collect --config /path/to/custom.yml
```

## Config

Edit `groot.yml` (or any file passed with `--config`) and align every section with your cluster and operational needs. Do not rely on the shipped sample as a drop-in configuration.

Sample config:

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
      statefulsets:
        - redis
      daemonsets:
        - node-agent
      helm_releases:
        - my-release
  include_pod_logs: true
  include_previous_logs: true
  pod_log_tail_lines: 1500
  include_node_details: true
  extra_kubectl:
    - "get componentstatuses"
    - "get csr"

notify:
  slack:
    enabled: false
    webhook_url: ""
  teams:
    enabled: false
    webhook_url: ""
  telegram:
    enabled: false
    token: ""
    chat_id: ""
```

Environment variables are also supported via Viper prefix `GROOT_`, for example:

- `GROOT_KUBECONFIG`
- `GROOT_OUTPUT_DIR`
- `GROOT_COLLECTION_TIMEOUT`
- `GROOT_NOTIFY_SLACK_WEBHOOK_URL`
- `GROOT_NOTIFY_TEAMS_WEBHOOK_URL`
- `GROOT_NOTIFY_TELEGRAM_TOKEN`
- `GROOT_NOTIFY_TELEGRAM_CHAT_ID`

When a notification channel is enabled and required credentials are missing, `groot` fails fast with a clear configuration error.

## Resolution and precedence

Configuration file precedence:

1. `--config` explicit path
2. `./groot.yml`
3. `~/.groot/groot.yml`
4. defaults

`kubeconfig` precedence:

1. `--kubeconfig /path/to/config`
2. `KUBECONFIG`
3. `kubeconfig` value in YAML
4. if all empty, `kubectl` default behavior

Workload filter behavior (`collection.targets`):

- per namespace, you can define `deployments`, `statefulsets`, `daemonsets`, and `helm_releases`
- if a namespace has targets, pod logs for that namespace are limited to those workloads
- if a namespace has no targets, pod logs keep the default broad behavior
- `helm_releases` matches `app.kubernetes.io/instance`

`pod_log_tail_lines` behavior:

- `0`: collect full logs (no `--tail`, recommended for RCA)
- `>0`: collect only the last N lines per pod
- applies to both current and `--previous` pod logs

`include_previous_logs` behavior:

- `true`: also collects `kubectl logs --previous` per pod into `*.previous.log`
- `false`: collects only current pod logs

`output_dir` path expansion:

- supports `~` (home directory), for example `~/tmp/groot-out`
- supports environment variables, for example `${HOME}/tmp/groot-out`

## Output naming

Capture output names are:

- directory: `<timestamp>`
- archive: `<timestamp>-<cluster>[-<message>].tar.gz`

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
- output: `20260428-123200-my-cluster-network-routing-issue.tar.gz`

Directory layout:

- `nodes/`
- `extras/`
- one directory per configured namespace (for example `kube-system/`, `default/`)
- after archive creation, the timestamp directory is automatically removed

## Console output modes

- default: summary `INFO` lines
- `--verbose`: adds per-command `CMD` / `OK` / `ERR`
- `--quiet`: suppresses normal output, prints only errors
- `--no-color`: disables ANSI colors

## Typical collected data

- `kubectl cluster-info`
- `kubectl get nodes -o wide`
- `kubectl get pods -A -o wide`
- `kubectl get events -A`
- `kubectl describe node <node>`
- `kubectl top node <node>`
- `kubectl logs -n <ns> <pod> --all-containers`
- Control plane pod logs in `kube-system` (`tier=control-plane`, when available)
- `extras/kubeconfig.txt` derived from kubeconfig (`context`, `cluster`, `user`, `server`)

## Notifications

Enable each channel in config:

- Slack Incoming Webhook
- Teams Incoming Webhook
- Telegram bot token + chat id

## Rootless container

```bash
make docker-build
make docker-buildx
make scan

docker run --rm \
  -v "$HOME/.kube:/home/nonroot/.kube:ro" \
  -v "$(pwd)/out:/app/out" \
  groot:local
```

For strict rootless runtime, use Podman:

```bash
podman build -t groot:local .
podman run --rm \
  -v "$HOME/.kube:/home/nonroot/.kube:ro" \
  -v "$(pwd)/out:/app/out" \
  groot:local
```

## Security note

Collected logs may contain sensitive data. Handle archives according to your security policy.
