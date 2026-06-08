package config

// SampleYAML returns a ready-to-use configuration template.
func SampleYAML() string {
	return `# GROOT sample configuration (see README → Config and SPECIFICATIONS.md).
# Boolean env overrides use GROOT_* (e.g. GROOT_COLLECTION_INCLUDE_POD_LOGS).

# Path to kubeconfig. Empty = use process KUBECONFIG or client-go default rules
# (CLI flag --kubeconfig still overrides when passed).
kubeconfig: ""

# Base directory for each run: a capture folder is created here, then archived to a .tar.gz
# beside it. Supports ~ and ${VAR} expansion.
output_dir: "./out"

# Prefix for capture directory and archive basename: <file_prefix>-<timestamp>[-since-<slug>]-<cluster>
file_prefix: "groot-capture"

collection:
  # Maximum duration for the whole collect run (context deadline); entire job aborts after this.
  timeout: 20m

  # Parallel API collection jobs (worker pool size).
  worker_concurrency: 6

  # Namespaces where Groot writes per-namespace workload JSON into <ns>/resources.txt
  # and creates namespace directories under the capture tree.
  namespaces:
    - kube-system
    - default

  # Per-namespace workload filters for POD LOGS ONLY. If a namespace is listed here with at
  # least one target list, only pods matching those workloads get log jobs. Namespaces with
  # no entry, or an entry with all lists empty, keep broad pod log collection for that NS.
  # Matching uses labels: app.kubernetes.io/name, app.kubernetes.io/instance, job-name, and app.
  targets:
    default:
      deployments:
        - api
      statefulsets:
        - redis
      daemonsets:
        - node-agent
      jobs:
        - batch-job
      cronjobs:
        - nightly-sync
      helm_releases:
        - my-release

  # When true, collect pod and control-plane logs (subject to targets/tail/since).
  include_pod_logs: true

  # When true, also fetch previous container logs into *.previous.log (optional jobs).
  include_previous_logs: true

  # Max lines per log stream when >0; 0 means no --tail (full logs, can be large).
  pod_log_tail_lines: 1500

  # Optional: time window for pod logs only. Digits-only string = hours (24 -> 24h);
  # or a Go duration (24h, 45m). collect --since overrides this when set.
  # When set, capture dir and .tar.gz name include "-since-<slug>" after the timestamp.
  # pod_logs_since: "24"

  # When true, per-node describe and top under nodes/.
  include_node_details: true

  # When true, per-node kubelet query + …/proxy/logs/messages (optional; see README).
  include_node_logs: true

  # When >0, adds tailLines to kubelet log query; 0 uses API default length.
  node_log_tail_lines: 5000

  # When true, cluster-wide pod metrics into extras/all-pods-top.txt (requires metrics-server).
  include_pod_metrics: true

  # Optional extra read-only commands (split on spaces, no shell). Allowlisted verbs only; see SPEC.
  extra_kubectl:
    - "get ingress -A"
    - "get pvc -A"

  # Optional: scan collected *.log files and replace likely secret values (off by default).
  redact_secrets: false
  # redact_patterns:
  #   - '(?i)my-custom-secret\s*=\s*\S+'

notify:
  slack:
    enabled: false
    # One URL, or several separated by ';' (e.g. team A; team B webhooks)
    webhook_url: ""

  discord:
    enabled: false
    # Discord server Settings → Integrations → Webhooks (same ';' for multiple URLs)
    webhook_url: ""

  teams:
    enabled: false
    # Same ';' convention as Slack for multiple Teams incoming webhooks
    webhook_url: ""

  pagerduty:
    enabled: false
    # Events API v2 integration key(s); multiple keys separated by ';'
    routing_key: ""
    # critical | error | warning | info
    severity: "warning"
    # Shown as the event source in PagerDuty
    source: "groot"

  telegram:
    enabled: false
    token: ""
    # One chat id, or several (group/user) ids separated by ';' with the same bot
    chat_id: ""

  generic:
    enabled: false
    # POST JSON. Default: {"<json_key>":"<summary>"}. Or set body_template for richer JSON (see SPEC).
    webhook_url: ""
    json_key: "text"
    # Optional fixed fields merged into the default JSON payload (values support {{summary}} placeholders).
    # extra_fields:
    #   event: "{{event}}"
    # Optional JSON body template (placeholders: {{summary}}, {{total}}, {{failed}}, {{event}}, …).
    # body_template: '{"text":"{{summary}}","failed":{{failed}}}'
    # Optional HMAC-SHA256 signature header (sha256=<hex>) over the raw POST body.
    # hmac_secret: ""
    # hmac_header: "X-Groot-Signature"
    # Optional HTTP headers on the POST (e.g. Authorization)
    headers: {}

  email:
    enabled: false
    host: ""
    port: 587
    username: ""
    password: ""
    from: ""
    # Semicolon-separated recipients
    to: ""
    # Implicit TLS (port 465). Default path uses STARTTLS on 587 when use_tls is false.
    use_tls: false

  on_failure:
    # Alert when collect aborts or job failures exceed min_failed_jobs (respects --no-notify).
    enabled: false
    on_abort: true
    min_failed_jobs: 1

  retry:
    # Transient HTTP notify retries (5xx and network errors).
    max_attempts: 3
    initial_backoff: 1s
    max_backoff: 10s

# Optional post-collect upload of the .tar.gz archive (credentials via env; off by default).
upload:
  enabled: false
  continue_on_error: true
  timeout: 5m
  s3:
    enabled: false
    bucket: ""
    region: ""
    key_prefix: "groot-archives"
    # endpoint: "https://s3.amazonaws.com"  # optional S3-compatible endpoint
  gcs:
    enabled: false
    bucket: ""
    key_prefix: "groot-archives"
`
}
