# AGENTS.md — groot (product)

This repository is the **GROOT product**: CLI, collector engine, behavior contract, tests, and release artifacts (binaries, packages, container image).

| Repo | Role |
|------|------|
| **groot** (this repo) | `groot collect`, SPEC, ROADMAP, CHANGELOG, GoReleaser, `ghcr.io/hrodrig/groot` |
| **[groot-selfhosted](https://github.com/hrodrig/groot-selfhosted)** | Operator deployment: Docker/Podman, Helm CronJob, flat manifests, cron/systemd |
| **[groot-trigger](https://github.com/hrodrig/groot-trigger)** | In-cluster HTTP → Job `groot collect` (on-demand; not a long-lived HTTP server in this CLI) |
| **[groot-share](https://github.com/hrodrig/groot-share)** (**gfs**) | VPS web/API door for `.tar.gz` archives: auth, ingest, list, download, audit, retention |
| **[groot-share-selfhosted](https://github.com/hrodrig/groot-share-selfhosted)** | Operator deploy for **gfs**: Compose, systemd, Helm (`vps` / `vps-s3`) |

## Scope

- **`cmd/`**, **`internal/`**, **`docs/SPECIFICATIONS.md`**, **`configs/groot.yml.sample`**, **`contrib/`** (packaging), **`testing/`** (product E2E).
- Do **not** add Helm charts, bastion runbooks, or cron wrappers here — those belong in **groot-selfhosted**.
- Do **not** add on-demand HTTP or gfs (archive catalog / VPS door) here — those belong in **groot-trigger**, **groot-share**, and **groot-share-selfhosted**.

## Operator deployment

For Helm, in-cluster CronJob, `docker run` with kubeconfig, and standalone scheduling, link to **[groot-selfhosted](https://github.com/hrodrig/groot-selfhosted)** (`run/README.md`). On-demand collect: **[groot-trigger](https://github.com/hrodrig/groot-trigger)**. VPS archive door: **[groot-share](https://github.com/hrodrig/groot-share)**; gfs deploy: **[groot-share-selfhosted](https://github.com/hrodrig/groot-share-selfhosted)**.

## Language

English only for all project artifacts.
