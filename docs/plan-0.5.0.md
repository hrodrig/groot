# Plan 0.5.0 — notifications and Kubernetes-native operations

**Status:** **shipped** — **`v0.5.0`** tagged on `main` (2026-06-06)  
**Target release:** `v0.5.0` ✓  
**Roadmap band:** [ROADMAP.md](ROADMAP.md) **0.5.x** items **#19–#24**  
**Agreed implementation order:** **#24 → #20 → #21 → #19 → #23 → #22**

## Why 0.5.0

After **0.4.x** (manifest, naming, `--list-jobs`, broader collector, kind E2E in CI), operators need:

1. **Honest failure alerts** — notify when collect aborts or partial failures exceed a threshold  
2. **Richer webhooks** — JSON templates, extra fields, HMAC signing  
3. **Email channel** — SMTP for teams without chat webhooks  
4. **HTTP resilience** — retry/backoff on transient notify errors  
5. **Optional secret scrub** — redact likely secrets in collected logs  
6. **In-cluster scheduling** — Helm chart + flat CronJob manifests

Homebrew, SBOM, and object-store upload stay in **0.6.x** (see [ROADMAP.md](ROADMAP.md)).

## Success criteria (0.5.0)

| # | Criterion | Roadmap |
|---|-----------|---------|
| 1 | `notify.on_failure` alerts on abort (`on_abort`) and when `failed >= min_failed_jobs` on success path; respects `--no-notify` | #19 |
| 2 | Generic webhook supports `extra_fields`, `body_template`, `hmac_secret` / `hmac_header` | #20 |
| 3 | `notify.email` SMTP channel with env overrides | #21 |
| 4 | Helm chart + `deploy/k8s/` CronJob, RBAC, ConfigMap, PVC for `/out` | #22 |
| 5 | `collection.redact_secrets` scans `*.log` with built-in + custom regex patterns | #23 |
| 6 | `notify.retry` retries transient 5xx / network errors for HTTP notify clients | #24 |
| 7 | `make ci` and `make release-check` green; merged coverage ≥ 80% | — |

---

## Implementation order

| Step | Item | Primary files |
|------|------|---------------|
| 1 | HTTP retry | `pkg/notifier/httpclient.go`, `config.go` |
| 2 | Rich generic webhooks | `pkg/notifier/notifier.go`, `template.go` |
| 3 | Email SMTP | `pkg/notifier/email.go` |
| 4 | Notify on failure | `pkg/cmd/root.go`, `config.go` |
| 5 | Secret redaction | `pkg/collector/redact.go`, `collector.go` |
| 6 | In-cluster deploy | `deploy/helm/groot/`, `deploy/k8s/cronjob.yaml` |

---

## Configuration examples (operator)

See also [README → Notifications](../../README.md#notifications), [SPEC §9](SPECIFICATIONS.md#9-configuration-examples), and [deploy/README.md](../../deploy/README.md).

### Failure alert when any optional job fails

```yaml
notify:
  on_failure:
    enabled: true
    min_failed_jobs: 1
  slack:
    enabled: true
    webhook_url: "https://hooks.slack.com/services/…"
```

### Signed generic webhook

```yaml
notify:
  generic:
    enabled: true
    webhook_url: "https://api.example/hooks/groot"
    body_template: '{"message":"{{summary}}","failed":{{failed}},"event":"{{event}}"}'
    hmac_secret: "${GROOT_WEBHOOK_SECRET}"
```

### Redact secrets before archive

```yaml
collection:
  redact_secrets: true
  redact_patterns:
    - '(?i)corp-token\s*=\s*\S+'
```

### Helm install with custom config

```bash
helm upgrade --install groot ./deploy/helm/groot \
  -n groot --create-namespace \
  --set-file config.grootYml=./groot.yml \
  --set image.tag=0.5.0
```

---

## Release checklist

1. **Code complete** on `develop`; `make ci` and `make release-check` green.  
2. **ROADMAP** — mark **0.5.x #19–#24** **Done (v0.5.0)**; add **Shipped** row; set **Current focus** to **0.6.x**.  
3. **SPECIFICATIONS** — §4 (notify, redact), §7 (notifications).  
4. **CHANGELOG** — `[0.5.0]` with `(0.5.x #N)` references.  
5. **VERSION** — `0.5.0`; README version badge if needed.  
6. **Sample** — `configs/groot.yml.sample` and `SampleYAML()` in sync.  
7. Merge **`develop` → `main`**.  
8. **Tag** — `git tag -a v0.5.0 -m "Release 0.5.0"` on `main`; push tag.
