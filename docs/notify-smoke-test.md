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
| `535 authentication failed` | Wrong SMTP username or password |
| `email: host and from are required` | Missing `GROOT_NOTIFY_EMAIL_HOST` or `GROOT_NOTIFY_EMAIL_FROM` |
| Message rejected (sandbox) | Recipient not in Mailgun **Authorized Recipients** |
| `SSL_connect … wrong version number` | Port/TLS mismatch — 587 needs `use_tls: false` (STARTTLS); 465 needs `use_tls: true` (see [GitLab SMTP TLS notes](https://docs.gitlab.com/omnibus/settings/smtp/#wrong-version-number-when-using-ssltls)) |
| Exit **4**, `HTTP status 500` | Wrong channel type — that message is for webhooks; for email check SMTP errors in stderr |
| Exit **1**, `no notify channel enabled` | `notify.email.enabled: false` or validation failed |

---

## Mapping from GitLab Omnibus (and similar docs)

If you already send mail from **GitLab** (or another app with SMTP examples), reuse the same host/credentials for Groot. GitLab documents dozens of providers in **[SMTP settings (GitLab Omnibus)](https://docs.gitlab.com/omnibus/settings/smtp/)** — translate `gitlab.rb` keys to Groot as follows:

| GitLab (`/etc/gitlab/gitlab.rb`) | Groot |
|----------------------------------|-------|
| `gitlab_rails['smtp_address']` | `GROOT_NOTIFY_EMAIL_HOST` |
| `gitlab_rails['smtp_port']` | `notify.email.port` in YAML |
| `gitlab_rails['smtp_user_name']` | `GROOT_NOTIFY_EMAIL_USERNAME` |
| `gitlab_rails['smtp_password']` | `GROOT_NOTIFY_EMAIL_PASSWORD` |
| `gitlab_rails['gitlab_email_from']` | `GROOT_NOTIFY_EMAIL_FROM` |
| (recipients — not in GitLab config) | `GROOT_NOTIFY_EMAIL_TO` (`;`-separated) |

**TLS mode** (same rules as GitLab troubleshooting — [wrong version number](https://docs.gitlab.com/omnibus/settings/smtp/#wrong-version-number-when-using-ssltls)):

| GitLab-style setup | Groot YAML |
|--------------------|------------|
| Port **587** + `smtp_enable_starttls_auto: true` (STARTTLS) | `port: 587`, `use_tls: false` |
| Port **465** + `smtp_tls: true` / implicit TLS | `port: 465`, `use_tls: true` |
| Port **25** plain (local relay, no TLS) | `port: 25`, `use_tls: false` |
| Self-signed / `smtp_openssl_verify_mode: 'none'` | add `skip_verify: true` under `notify.email` |

Example: copy working Mailgun block from GitLab docs into env:

```bash
# From GitLab Mailgun example (docs.gitlab.com/omnibus/settings/smtp/#mailgun)
export GROOT_NOTIFY_EMAIL_HOST=smtp.mailgun.org
export GROOT_NOTIFY_EMAIL_USERNAME='postmaster@mg.example.com'
export GROOT_NOTIFY_EMAIL_PASSWORD='…'
export GROOT_NOTIFY_EMAIL_FROM='groot@mg.example.com'
export GROOT_NOTIFY_EMAIL_TO='ops@example.com'
# YAML: port 587, use_tls: false
```

If inbox shows **mailed-by / signed-by** your GitLab host (e.g. `gitlab.example.com`) while **From** is your app address, the relay is working — Groot only needs the same SMTP endpoint GitLab uses, not GitLab itself.

---

## Provider cheat sheet

Values below match the [GitLab SMTP examples](https://docs.gitlab.com/omnibus/settings/smtp/#example-configurations). Always confirm credentials in your provider panel.

### Mailgun

| | |
|-|-|
| Host | `smtp.mailgun.org` (US), `smtp.eu.mailgun.org` (EU) |
| Port / TLS | 587 + STARTTLS → `use_tls: false`; 465 → `use_tls: true` |
| User | `postmaster@mg.yourdomain.com` |

### Amazon SES

| | |
|-|-|
| Host | `email-smtp.<region>.amazon.com` |
| Port / TLS | 587 + STARTTLS → `use_tls: false`; 465 → `use_tls: true` |
| User / pass | IAM SMTP credentials (not AWS access keys for API) |

### SendGrid

| | |
|-|-|
| Host | `smtp.sendgrid.net` |
| Port | 587, STARTTLS |
| User | API key auth: username literally `apikey`, password = API key |

### Gmail / Google relay

| | |
|-|-|
| Host | `smtp.gmail.com` or `smtp-relay.gmail.com` |
| Port | 587, STARTTLS |
| Notes | App password or relay allowlist; Gmail has low sending limits — prefer transactional provider for ops alerts |

### Microsoft 365 / Outlook

| | |
|-|-|
| Host | `smtp.office365.com` or `smtp-mail.outlook.com` |
| Port | 587, STARTTLS |

### Local / GitLab relay (port 25)

Internal MTA or GitLab’s outbound server on localhost:

```yaml
notify:
  email:
    enabled: true
    port: 25
    use_tls: false
```

```bash
export GROOT_NOTIFY_EMAIL_HOST=localhost
export GROOT_NOTIFY_EMAIL_FROM='groot@example.com'
export GROOT_NOTIFY_EMAIL_TO='ops@example.com'
# username/password only if relay requires AUTH
```

For any provider not listed, search the GitLab page above (SendGrid, Postmark, SparkPost, Brevo, etc.) and map fields with the table at the top of this section.

---

## Other channels

Same smoke flow for non-email: enable Slack/Teams/webhooks/PagerDuty in config (or env), run `groot notify test`. All enabled destinations receive the synthetic summary.

CI uses a **fake SMTP server** in unit tests (`internal/notifier/email_test.go`); no live provider credentials in GitHub Actions.
