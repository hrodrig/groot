---
last_mapped_commit: 805eba0
analysis_date: 2026-08-10
---

# Technology Stack

**Analysis Date:** 2026-08-10

## Languages

**Primary:**
- Go 1.26.6 - All application code (`cmd/`, `internal/`); module `github.com/hrodrig/groot` in `go.mod`

**Secondary:**
- YAML - Operator config (`configs/groot.yml.sample`, `examples/`)
- Shell - Packaging hooks and CI helpers (`contrib/deb/*.sh`, Makefile targets)
- Dockerfile - Container builds (`Dockerfile`, `Dockerfile.release`)
- man(7) / Makefile - FreeBSD/OpenBSD ports and man pages under `contrib/`

## Runtime

**Environment:**
- Go toolchain 1.26.6 (pinned in `go.mod`; Docker builder `golang:1.26.6-alpine`)
- CGO disabled for release binaries (`CGO_ENABLED=0` in `.goreleaser.yaml` and `Dockerfile`)
- No Node/Python/browser runtime — pure CLI process

**Package Manager:**
- Go modules (`go mod` / `go.sum`)
- Lockfile: `go.sum` present

## Frameworks

**Core:**
- `github.com/spf13/cobra` v1.10.2 - CLI commands (`groot collect`, `notify`, etc.) in `internal/cmd/`
- `github.com/spf13/viper` v1.21.0 - YAML config load + `GROOT_*` env overrides in `internal/config/config.go`
- Kubernetes client-go stack (`k8s.io/client-go` v0.32.5, `k8s.io/api`, `k8s.io/apimachinery`, `k8s.io/metrics`) - API collection without shelling out to `kubectl` (`internal/kubeloader/`, `internal/k8srunner/`, `internal/collector/`)

**Testing:**
- Go standard `testing` package with `-race` (`make test`)
- `github.com/google/go-cmp` v0.7.0 - Deep comparisons in unit tests
- `github.com/fsouza/fake-gcs-server` v1.52.3 - GCS upload tests (`STORAGE_EMULATOR_HOST`)
- `internal/kubetest/` - Fake Kubernetes HTTP handlers for collector/runner tests
- kind-based E2E under `testing/` (`make test-e2e-kind`; CI job `test-e2e-kind`)

**Build/Dev:**
- GNU Make (`Makefile`) - build, test, cover gate (`COVER_MIN` default 80%), lint, security, docker, release-check
- GoReleaser v2 (`.goreleaser.yaml`) - multi-OS binaries, nFPM deb/rpm, GHCR images, SBOM (syft), cosign, Homebrew cask
- `gofmt -s`, `go vet`, `gocyclo` (complexity ≥ 15 fails)
- `govulncheck`, Anchore Grype - vulnerability / image scans (`make security`, `.github/workflows/security.yml`)

## Key Dependencies

**Critical:**
- `k8s.io/client-go` v0.32.5 - Cluster REST client, kubeconfig / in-cluster config (`internal/kubeloader/loader.go`)
- `github.com/spf13/cobra` v1.10.2 - Command tree and flags (`internal/cmd/`)
- `github.com/spf13/viper` v1.21.0 - Config file + env binding (`internal/config/config.go`)
- `github.com/aws/aws-sdk-go-v2` (+ `service/s3`, `feature/s3/manager`) - S3 / S3-compatible archive upload (`internal/uploader/s3.go`)
- `cloud.google.com/go/storage` v1.62.3 - GCS archive upload (`internal/uploader/gcs.go`)
- `github.com/pkg/sftp` + `golang.org/x/crypto/ssh` - SFTP archive relay (`internal/uploader/sftp.go`)
- `sigs.k8s.io/yaml` v1.4.0 - YAML helpers used with K8s object encoding

**Infrastructure:**
- `golang.org/x/sys`, `golang.org/x/text` - OS/text utilities
- `google.golang.org/api` - Google API support for GCS client stack
- Distroless runtime image `gcr.io/distroless/static-debian13:nonroot` (`Dockerfile`)

## Configuration

**Environment:**
- Primary: YAML config file (sample `configs/groot.yml.sample`; `--config` / `--print-sample-config`)
- Viper prefix `GROOT` + `AutomaticEnv` in `internal/config/config.go` (e.g. `GROOT_COLLECTION_INCLUDE_POD_LOGS`)
- Explicit secrets/overrides for notify/upload (webhook URLs, SMTP, PagerDuty key, S3/GCS/SFTP) — see `INTEGRATIONS.md`
- Cluster access: `kubeconfig` field, `--kubeconfig`, or process `KUBECONFIG` / client-go default rules / in-cluster

**Build:**
- `go.mod` / `go.sum` - module and versions
- `VERSION` - product version (currently `1.0.6`); injected via `-ldflags` into `main.version` etc. (`cmd/groot/main.go`)
- `Makefile` - local and CI targets
- `.goreleaser.yaml` - release artifacts
- `Dockerfile` / `Dockerfile.release` - local vs GoReleaser image packaging
- `.github/workflows/{ci,release,security,codeql}.yml` - CI/CD

## Platform Requirements

**Development:**
- Go 1.26.6+ matching `go.mod`
- Optional: Docker/buildx, kind + kubectl for E2E (`make test-e2e-kind`)
- Optional: `bc` when enforcing `COVER_MIN` > 0; goreleaser for `make snapshot` / `release-check`
- Any OS with a Go toolchain; releases also target FreeBSD/OpenBSD via contrib ports

**Production:**
- Static binary (`groot` and `kubectl-groot` plugin basename) for linux/darwin/windows/freebsd/openbsd (amd64/arm64; Windows arm64 ignored)
- Container: `ghcr.io/hrodrig/groot` (tags `vX.Y.Z` and `latest`)
- Packages: deb/rpm via nFPM; Homebrew cask tap `hrodrig/homebrew-groot`
- Operator deployment (Helm/cron) lives in sibling repo **groot-selfhosted**, not this product tree
- Runs against a reachable Kubernetes API; optional outbound HTTPS for notify/upload

---

*Stack analysis: 2026-08-10*
*Update after major dependency changes*
