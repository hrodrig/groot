# Examples

Copy a YAML, set secrets via **env** (never commit tokens), then:

```bash
groot validate --config examples/.../file.yml
# Optional channel smoke (no collect):
groot notify test --config examples/notify/slack-smoke.yml
```

Full schema: [`configs/groot.yml.sample`](../configs/groot.yml.sample) and [SPECIFICATIONS.md](../SPECIFICATIONS.md).  
Bastion cron / Helm / rclone relays: **[groot-selfhosted](https://github.com/hrodrig/groot-selfhosted)** (not this tree).

## Profiles (`profiles/`)

Scenario starting points (ROADMAP **#86** + **#46** cloud siblings):

| File | Use case |
|------|----------|
| [`incident-quick.yml`](profiles/incident-quick.yml) | Narrow NS, short `--since`, fast incident capture |
| [`compliance-full.yml`](profiles/compliance-full.yml) | Broad capture + redaction |
| [`bastion-airgap.yml`](profiles/bastion-airgap.yml) | SFTP upload, no external webhooks |
| [`eks-managed.yml`](profiles/eks-managed.yml) | EKS: skip node host logs, metrics on |
| [`gke-managed.yml`](profiles/gke-managed.yml) | GKE: skip node host logs, GCS-oriented comments |
| [`aks-managed.yml`](profiles/aks-managed.yml) | AKS: skip node host logs, metrics on |

```bash
cp examples/profiles/incident-quick.yml groot.yml
```

## Notify (`notify/`)

Channel smokes for **`groot notify test`** (no cluster required for the notify path):

| File | Channel | Secrets (env) |
|------|---------|---------------|
| [`mailgun-smoke.yml`](notify/mailgun-smoke.yml) | SMTP / Mailgun | `GROOT_NOTIFY_EMAIL_*` — [runbook](../docs/notify-smoke-test.md) |
| [`slack-smoke.yml`](notify/slack-smoke.yml) | Slack | `GROOT_NOTIFY_SLACK_WEBHOOK_URL` |
| [`teams-smoke.yml`](notify/teams-smoke.yml) | Teams | `GROOT_NOTIFY_TEAMS_WEBHOOK_URL` |
| [`webhook-generic.yml`](notify/webhook-generic.yml) | Generic JSON + optional HMAC | `GROOT_NOTIFY_GENERIC_WEBHOOK_URL`, optional `GROOT_NOTIFY_GENERIC_HMAC_SECRET` |
| [`pagerduty-smoke.yml`](notify/pagerduty-smoke.yml) | PagerDuty Events v2 | `GROOT_NOTIFY_PAGERDUTY_ROUTING_KEY` |

## Upload (`upload/`)

Post-collect archive push (credentials via env / cloud SDK):

| File | Backend | Notes |
|------|---------|-------|
| [`s3.yml`](upload/s3.yml) | S3 / S3-compatible | `GROOT_UPLOAD_S3_*` or AWS SDK default chain |
| [`gcs.yml`](upload/gcs.yml) | GCS | ADC / `GOOGLE_APPLICATION_CREDENTIALS` |
| [`sftp.yml`](upload/sftp.yml) | SFTP | `GROOT_UPLOAD_SFTP_*` + `known_hosts` |

## Collection snippets (`collection/`)

| File | Focus |
|------|-------|
| [`targets-and-extra.yml`](collection/targets-and-extra.yml) | Per-NS `targets` + allowlisted `extra_kubectl` |
| [`redact.yml`](collection/redact.yml) | `redact_secrets` + custom `redact_patterns` |
