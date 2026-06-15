# AGENTS.md — groot (product)

This repository is the **GROOT product**: CLI, collector engine, behavior contract, tests, and release artifacts (binaries, packages, container image).

| Repo | Role |
|------|------|
| **groot** (this repo) | `groot collect`, SPEC, ROADMAP, CHANGELOG, GoReleaser, `ghcr.io/hrodrig/groot` |
| **[groot-selfhosted](https://github.com/hrodrig/groot-selfhosted)** | Operator deployment: Docker/Podman, Helm CronJob, flat manifests, cron/systemd |

## Scope

- **`cmd/`**, **`pkg/`**, **`docs/SPECIFICATIONS.md`**, **`configs/groot.yml.sample`**, **`contrib/`** (packaging), **`testing/`** (product E2E).
- Do **not** add Helm charts, bastion runbooks, or cron wrappers here — those belong in **groot-selfhosted**.

## Operator deployment

For Helm, in-cluster CronJob, `docker run` with kubeconfig, and standalone scheduling, link to **[groot-selfhosted](https://github.com/hrodrig/groot-selfhosted)** (`run/README.md`).

## Language

English only for all project artifacts.
