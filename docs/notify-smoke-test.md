# Notify smoke test (SMTP / Mailgun)

Validate **`notify.*`** wiring **without** running `groot collect` or contacting the Kubernetes API.

Contract: [SPECIFICATIONS.md §15](../SPECIFICATIONS.md) (`groot notify test`, 1.0.3 #95).

Sample config (no secrets): [examples/notify/mailgun-smoke.yml](../examples/notify/mailgun-smoke.yml).

---

## Quick start (Mailgun)

**1. Mailgun SMTP credentials** (Sending → Domain → SMTP credentials):

| Field | Typical value |
|-------|----------------|
| Host | `smtp.mailgun.org` (US) or `smtp.eu.mailgun.org` (EU) |
| Port | `587` (STARTTLS) or `465` (implicit TLS) |
| Username | e.g. `postmaster@mg.yourdomain.com` |
| Password | SMTP password from the panel |
| From | Authorized sender on that domain (e.g. `groot@mg.yourdomain.com`) |
| To | Your inbox; **sandbox** domains require **Authorized Recipients** |

**2. Export secrets in your shell** (do not put passwords in YAML or chat logs):

```bash
export GROOT_NOTIFY_EMAIL_HOST=smtp.mailgun.org
export GROOT_NOTIFY_EMAIL_USERNAME='postmaster@mg.yourdomain.com'
export GROOT_NOTIFY_EMAIL_PASSWORD='your-smtp-password'
export GROOT_NOTIFY_EMAIL_FROM='groot@mg.yourdomain.com'
export GROOT_NOTIFY_EMAIL_TO='you@example.com'
```

**3. Run smoke test** (from repo root, after `make build` or `go build -o ./bin/groot ./cmd/groot`):

```bash
./bin/groot notify test --config examples/notify/mailgun-smoke.yml
./bin/groot notify test --config examples/notify/mailgun-smoke.yml --event failure
```

Success: stdout `notify test: sent event "notify.test" to enabled channel(s)` and email in inbox (or Mailgun logs).

Delivery failure: exit **4** (`ExitNotifyFailed`). Config / no channels / bad `--event`: exit **1**.

**Note:** `groot notify test` **ignores** `--no-notify` and `GROOT_NO_NOTIFY` — it exists only to exercise notify.

---

## After smoke test (optional)

End-to-end with a real cluster:

```bash
./bin/groot validate --config examples/notify/mailgun-smoke.yml
./bin/groot collect --config examples/notify/mailgun-smoke.yml --quiet
```

Notify behavior on collect:

| Outcome | Email (and other enabled channels) |
|---------|-------------------------------------|
| Collect OK | Success summary |
| Collect abort | Failure alert if `notify.on_failure.enabled: true` and `on_abort: true` |
| Collect OK but `failed >= min_failed_jobs` | Success notify + failure alert (when `on_failure.enabled`) |
| Any run with `--no-notify` / `GROOT_NO_NOTIFY=1` | Skipped |

---

## Port 465 (implicit TLS)

In YAML:

```yaml
notify:
  email:
    enabled: true
    port: 465
    use_tls: true
```

Same `GROOT_NOTIFY_EMAIL_*` env vars.

---

## Troubleshooting

| Symptom | Likely cause |
|---------|----------------|
| `535 authentication failed` | Wrong Mailgun SMTP username or password |
| `email: host and from are required` | Missing `GROOT_NOTIFY_EMAIL_HOST` or `GROOT_NOTIFY_EMAIL_FROM` |
| Message rejected (sandbox) | Recipient not in Mailgun **Authorized Recipients** |
| Exit **4**, `HTTP status 500` | Wrong channel type — that message is for webhooks; for email check SMTP errors in stderr |
| Exit **1**, `no notify channel enabled` | `notify.email.enabled: false` or validation failed |

---

## Other providers

Same flow works for any SMTP relay: set `GROOT_NOTIFY_EMAIL_*`, enable `notify.email` in config, run `groot notify test`. For Slack/Teams/webhooks, enable the channel in YAML (or env) and use the same command — all enabled destinations receive the synthetic summary.

CI uses a **fake SMTP server** in unit tests (`internal/notifier/email_test.go`); no Mailgun credentials in GitHub Actions.
