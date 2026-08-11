---
last_mapped_commit: 805eba0
analysis_date: 2026-08-10
---

# External Integrations

**Analysis Date:** 2026-08-10

## APIs & External Services

**Kubernetes API:**
- Target cluster Kubernetes API server - Read-only diagnostics collection (resources, events, logs, metrics, node proxy logs)
  - SDK/Client: `k8s.io/client-go` v0.32.5 (`internal/kubeloader/loader.go`, `internal/k8srunner/`, `internal/collector/`)
  - Auth: kubeconfig (`--kubeconfig` / `kubeconfig:` / `KUBECONFIG`) or in-cluster service account; client-go default loading rules
  - Optional: metrics-server for pod/node top (`k8s.io/metrics`); kubelet log query / node proxy when enabled in config

**Chat / Incident webhooks (notify):**
- Slack Incoming Webhooks - Collect success/failure summaries
  - Integration method: HTTPS POST JSON (`internal/notifier/notifier.go`)
  - Auth: webhook URL in `notify.slack.webhook_url` or `GROOT_NOTIFY_SLACK_WEBHOOK_URL` (`;`-separated multi-URL)
- Discord Incoming Webhooks - Same notify fan-out
  - Auth: `notify.discord.webhook_url` or `GROOT_NOTIFY_DISCORD_WEBHOOK_URL`
- Microsoft Teams Incoming Webhooks - Same notify fan-out
  - Auth: `notify.teams.webhook_url` or `GROOT_NOTIFY_TEAMS_WEBHOOK_URL`
- Generic JSON webhook - Custom POST body / headers / optional HMAC-SHA256
  - Auth: `notify.generic.webhook_url` or `GROOT_NOTIFY_GENERIC_WEBHOOK_URL`; HMAC via `hmac_secret` / `GROOT_NOTIFY_GENERIC_HMAC_SECRET`
  - Implementation: `internal/notifier/` (`appendGenericSenders`, HMAC helpers)

**PagerDuty:**
- PagerDuty Events API v2 - Trigger incidents on notify
  - Endpoint: `https://events.pagerduty.com/v2/enqueue` (`internal/notifier/pagerduty.go`)
  - Auth: routing/integration key in `notify.pagerduty.routing_key` or `GROOT_NOTIFY_PAGERDUTY_ROUTING_KEY`

**Telegram:**
- Telegram Bot API - `sendMessage` via `https://api.telegram.org` (`internal/notifier/notifier.go`)
  - Auth: bot `token` + `chat_id` (`GROOT_NOTIFY_TELEGRAM_TOKEN`, `GROOT_NOTIFY_TELEGRAM_CHAT_ID`)

**Email/SMS:**
- SMTP email - Collect summaries via STARTTLS (587) or implicit TLS (465)
  - Implementation: `internal/notifier/email.go` (stdlib `net/smtp` + TLS)
  - Auth: `GROOT_NOTIFY_EMAIL_HOST`, `_USERNAME`, `_PASSWORD`, `_FROM`, `_TO` (or YAML `notify.email.*`)
  - SMS: Not detected

**Payment Processing:**
- Not applicable

## Data Storage

**Databases:**
- None - No application database; ephemeral local capture tree + `.tar.gz` archive under `output_dir`

**File Storage:**
- Local filesystem - Capture directories and archives (`output_dir` in config; collector archive path)
- Amazon S3 / S3-compatible - Optional post-collect upload (`internal/uploader/s3.go`)
  - SDK/Client: AWS SDK Go v2 S3 + upload manager
  - Auth: `AWS_ACCESS_KEY_ID` / `AWS_SECRET_ACCESS_KEY` / optional `AWS_SESSION_TOKEN`, or default credential chain; region via `GROOT_UPLOAD_S3_REGION` / `AWS_REGION` / `AWS_DEFAULT_REGION`
  - Config: `upload.s3.bucket` (`GROOT_UPLOAD_S3_BUCKET`), `key_prefix`, optional `endpoint` (`GROOT_UPLOAD_S3_ENDPOINT`) for MinIO/etc.
- Google Cloud Storage - Optional upload (`internal/uploader/gcs.go`)
  - SDK/Client: `cloud.google.com/go/storage`
  - Auth: Application Default Credentials / `GOOGLE_APPLICATION_CREDENTIALS` / workload identity; emulator via `STORAGE_EMULATOR_HOST`
  - Config: `upload.gcs.bucket` (`GROOT_UPLOAD_GCS_BUCKET`), `key_prefix`
- SFTP over SSH - Optional air-gapped relay upload (`internal/uploader/sftp.go`)
  - SDK/Client: `github.com/pkg/sftp` + `golang.org/x/crypto/ssh` (public-key only, known_hosts)
  - Auth: `GROOT_UPLOAD_SFTP_IDENTITY_FILE`; host verification `known_hosts_file` / `GROOT_UPLOAD_SFTP_KNOWN_HOSTS`
  - Config: `GROOT_UPLOAD_SFTP_HOST`, `_USER`, `_PORT`, `_REMOTE_DIR`

**Caching:**
- None

## Authentication & Identity

**Auth Provider:**
- Kubernetes user/cluster auth via client-go (certificates, tokens, exec plugins as defined in kubeconfig) — no end-user OAuth login in the product
- Cloud object storage: AWS default credential chain; GCP ADC; SFTP SSH private key
- Notify channels: shared secrets (webhook URLs, PagerDuty routing key, Telegram bot token, SMTP password) — prefer env over YAML for secrets

**OAuth Integrations:**
- Not applicable for end-user sign-in (GCP OAuth libraries may appear transitively for ADC only)

## Monitoring & Observability

**Error Tracking:**
- None in-process (no Sentry/Datadog SDK)
- GitHub CodeQL workflow (`.github/workflows/codeql.yml`) for static analysis alerts

**Analytics:**
- Codecov - Coverage upload from CI (`codecov/codecov-action@v5` in `.github/workflows/ci.yml`; secret `CODECOV_TOKEN`)

**Logs:**
- Stderr/stdout structured logging via `internal/logx/` — operator captures process logs externally
- Optional outbound notify on collect failure/abort (`notify.on_failure`)

## CI/CD & Deployment

**Hosting:**
- GitHub Releases - Binary archives + checksums + SBOMs
- GitHub Container Registry - `ghcr.io/hrodrig/groot` (GoReleaser `dockers_v2` + cosign keyless signing)
- Homebrew tap - `hrodrig/homebrew-groot` (requires `HOMEBREW_TAP_TOKEN`)
- Operator runbooks/charts - External repo [groot-selfhosted](https://github.com/hrodrig/groot-selfhosted) (not in this tree)

**CI Pipeline:**
- GitHub Actions
  - Workflows: `.github/workflows/ci.yml` (gofmt, vet, gocyclo, tests, kind E2E), `security.yml` (govulncheck, Grype image), `release.yml` (tag `v*` → GoReleaser), `codeql.yml`
  - Secrets (names only): `CODECOV_TOKEN`, `HOMEBREW_TAP_TOKEN`, `GITHUB_TOKEN` (GHCR/OIDC for cosign)
  - Branches: `main` / `develop` for CI; releases only from annotated tags on release flow

## Environment Configuration

**Required env vars:**
- None strictly required for a local collect with a readable kubeconfig and YAML file
- Cluster: `KUBECONFIG` when not using default path / in-cluster / `--kubeconfig`
- When notify enabled: corresponding `GROOT_NOTIFY_*` secrets (Slack/Discord/Teams/Telegram/PagerDuty/Email/Generic)
- When upload enabled: `GROOT_UPLOAD_S3_*` or `AWS_*`; `GROOT_UPLOAD_GCS_*` + ADC; or `GROOT_UPLOAD_SFTP_*` + identity file + known_hosts
- Test-only: `STORAGE_EMULATOR_HOST` for fake GCS

**Secrets location:**
- Operator environment / Kubernetes Secrets (typical for CronJob) — documented for deployment in groot-selfhosted
- CI: GitHub Actions repository secrets
- Do not commit credentials; sample config leaves webhook/password fields empty (`configs/groot.yml.sample`)

**Mock/stub services:**
- `internal/kubetest/` - Fake K8s API for unit tests
- `fake-gcs-server` + `STORAGE_EMULATOR_HOST` - GCS upload tests
- Example smoke configs under `examples/notify/` and `examples/upload/`

## Webhooks & Callbacks

**Incoming:**
- None - groot is a client CLI; it does not expose HTTP listen endpoints for third-party callbacks

**Outgoing:**
- Slack / Discord / Teams / generic HTTPS webhooks - After collect (or `groot notify test`); retry on 5xx/network via `notify.retry` (`internal/notifier/httpclient.go`)
- PagerDuty Events v2 enqueue - Same notify path
- Telegram Bot API - Same notify path
- SMTP submit - Email channel
- S3 PutObject / GCS object write / SFTP create - Post-collect archive upload when `upload.enabled`

---

*Integration audit: 2026-08-10*
*Update when adding/removing external services*
