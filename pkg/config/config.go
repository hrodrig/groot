package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config represents the YAML and env configuration.
type Config struct {
	Kubeconfig  string        `mapstructure:"kubeconfig"`
	ClusterName string        `mapstructure:"cluster_name"`
	OutputDir   string        `mapstructure:"output_dir"`
	FilePrefix  string        `mapstructure:"file_prefix"`
	Collection  CollectionCfg `mapstructure:"collection"`
	Notify      NotifyCfg     `mapstructure:"notify"`
	Upload      UploadCfg     `mapstructure:"upload"`
}

type CollectionCfg struct {
	Timeout             time.Duration               `mapstructure:"timeout"`
	WorkerConcurrency   int                         `mapstructure:"worker_concurrency"`
	Namespaces          []string                    `mapstructure:"namespaces"`
	Targets             map[string]NamespaceTargets `mapstructure:"targets"`
	ExtraKubectl        []string                    `mapstructure:"extra_kubectl"`
	IncludePodLogs      bool                        `mapstructure:"include_pod_logs"`
	IncludePreviousLogs bool                        `mapstructure:"include_previous_logs"`
	PodLogTailLines     int                         `mapstructure:"pod_log_tail_lines"`
	PodLogsSince        string                      `mapstructure:"pod_logs_since"`
	IncludeNodeDetails  bool                        `mapstructure:"include_node_details"`
	IncludeNodeLogs     bool                        `mapstructure:"include_node_logs"`
	NodeLogTailLines    int                         `mapstructure:"node_log_tail_lines"`
	IncludePodMetrics   bool                        `mapstructure:"include_pod_metrics"`
	RedactSecrets       bool                        `mapstructure:"redact_secrets"`
	RedactPatterns      []string                    `mapstructure:"redact_patterns"`
	MinFreeBytes        int64                       `mapstructure:"min_free_bytes"`
	WarnFreeBytes       int64                       `mapstructure:"warn_free_bytes"`
}

type NamespaceTargets struct {
	Deployments  []string `mapstructure:"deployments"`
	StatefulSets []string `mapstructure:"statefulsets"`
	DaemonSets   []string `mapstructure:"daemonsets"`
	Jobs         []string `mapstructure:"jobs"`
	CronJobs     []string `mapstructure:"cronjobs"`
	HelmReleases []string `mapstructure:"helm_releases"`
}

type NotifyCfg struct {
	Slack     WebhookCfg        `mapstructure:"slack"`
	Discord   WebhookCfg        `mapstructure:"discord"`
	Telegram  TelegramCfg       `mapstructure:"telegram"`
	Teams     WebhookCfg        `mapstructure:"teams"`
	Generic   GenericWebhookCfg `mapstructure:"generic"`
	PagerDuty PagerDutyCfg      `mapstructure:"pagerduty"`
	Email     EmailCfg          `mapstructure:"email"`
	OnFailure OnFailureCfg      `mapstructure:"on_failure"`
	Retry     NotifyRetryCfg    `mapstructure:"retry"`
}

type WebhookCfg struct {
	Enabled    bool   `mapstructure:"enabled"`
	WebhookURL string `mapstructure:"webhook_url"`
}

type TelegramCfg struct {
	Enabled bool   `mapstructure:"enabled"`
	Token   string `mapstructure:"token"`
	ChatID  string `mapstructure:"chat_id"`
}

// GenericWebhookCfg is an optional HTTP POST JSON notifier (custom services, Discord-style webhooks, etc.).
type GenericWebhookCfg struct {
	Enabled      bool              `mapstructure:"enabled"`
	WebhookURL   string            `mapstructure:"webhook_url"`
	JSONKey      string            `mapstructure:"json_key"`
	Headers      map[string]string `mapstructure:"headers"`
	ExtraFields  map[string]string `mapstructure:"extra_fields"`
	BodyTemplate string            `mapstructure:"body_template"`
	HMACSecret   string            `mapstructure:"hmac_secret"`
	HMACHeader   string            `mapstructure:"hmac_header"`
}

// EmailCfg sends a plain-text summary via SMTP (STARTTLS on 587 by default).
type EmailCfg struct {
	Enabled    bool   `mapstructure:"enabled"`
	Host       string `mapstructure:"host"`
	Port       int    `mapstructure:"port"`
	Username   string `mapstructure:"username"`
	Password   string `mapstructure:"password"`
	From       string `mapstructure:"from"`
	To         string `mapstructure:"to"`
	UseTLS     bool   `mapstructure:"use_tls"`
	SkipVerify bool   `mapstructure:"skip_verify"`
}

// OnFailureCfg controls optional alerts when collect aborts or job failures exceed a threshold.
type OnFailureCfg struct {
	Enabled       bool `mapstructure:"enabled"`
	OnAbort       bool `mapstructure:"on_abort"`
	MinFailedJobs int  `mapstructure:"min_failed_jobs"`
}

// NotifyRetryCfg configures transient HTTP notify retries (5xx and network errors).
type NotifyRetryCfg struct {
	MaxAttempts    int           `mapstructure:"max_attempts"`
	InitialBackoff time.Duration `mapstructure:"initial_backoff"`
	MaxBackoff     time.Duration `mapstructure:"max_backoff"`
}

// UploadCfg controls optional post-collect archive upload to object storage.
type UploadCfg struct {
	Enabled         bool          `mapstructure:"enabled"`
	ContinueOnError bool          `mapstructure:"continue_on_error"`
	Timeout         time.Duration `mapstructure:"timeout"`
	S3              S3UploadCfg   `mapstructure:"s3"`
	GCS             GCSUploadCfg  `mapstructure:"gcs"`
	SFTP            SFTPUploadCfg `mapstructure:"sftp"`
}

// S3UploadCfg uploads the archive to S3 or an S3-compatible endpoint.
type S3UploadCfg struct {
	Enabled      bool              `mapstructure:"enabled"`
	Bucket       string            `mapstructure:"bucket"`
	Region       string            `mapstructure:"region"`
	KeyPrefix    string            `mapstructure:"key_prefix"`
	Endpoint     string            `mapstructure:"endpoint"`
	StorageClass string            `mapstructure:"storage_class"`
	ContentType  string            `mapstructure:"content_type"`
	SSE          string            `mapstructure:"sse"`
	SSEKMSKeyID  string            `mapstructure:"sse_kms_key_id"`
	ACL          string            `mapstructure:"acl"`
	Metadata     map[string]string `mapstructure:"metadata"`
}

// GCSUploadCfg uploads the archive to Google Cloud Storage.
type GCSUploadCfg struct {
	Enabled       bool              `mapstructure:"enabled"`
	Bucket        string            `mapstructure:"bucket"`
	KeyPrefix     string            `mapstructure:"key_prefix"`
	ContentType   string            `mapstructure:"content_type"`
	CacheControl  string            `mapstructure:"cache_control"`
	KMSKey        string            `mapstructure:"kms_key"`
	PredefinedACL string            `mapstructure:"predefined_acl"`
	Metadata      map[string]string `mapstructure:"metadata"`
}

// SFTPUploadCfg uploads the archive to a remote Linux host via SFTP over SSH.
// Credentials (identity file) come from env only — never stored in YAML.
type SFTPUploadCfg struct {
	Enabled        bool   `mapstructure:"enabled"`
	Host           string `mapstructure:"host"`
	Port           int    `mapstructure:"port"`
	User           string `mapstructure:"user"`
	RemoteDir      string `mapstructure:"remote_dir"`
	KnownHostsFile string `mapstructure:"known_hosts_file"`
	IdentityFile   string `mapstructure:"-"` // env only
}

// PagerDutyCfg sends Events API v2 triggers (https://developer.pagerduty.com/docs/events-api-v2-overview).
type PagerDutyCfg struct {
	Enabled    bool   `mapstructure:"enabled"`
	RoutingKey string `mapstructure:"routing_key"`
	Severity   string `mapstructure:"severity"`
	Source     string `mapstructure:"source"`
}

// Load reads configuration from YAML and environment.
func Load(configFile string) (Config, error) {
	v := viper.New()
	v.SetConfigType("yaml")
	v.SetEnvPrefix("GROOT")
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	setDefaults(v)

	if configFile != "" {
		v.SetConfigFile(configFile)
		if err := v.ReadInConfig(); err != nil {
			if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
				return Config{}, fmt.Errorf("read config: %w", err)
			}
		}
	} else {
		if err := readDefaultConfig(v); err != nil {
			return Config{}, fmt.Errorf("read config: %w", err)
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return Config{}, fmt.Errorf("unmarshal config: %w", err)
	}

	if err := applyLoadedConfig(&cfg); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func applyLoadedConfig(cfg *Config) error {
	if envKubeconfig := os.Getenv("KUBECONFIG"); envKubeconfig != "" {
		cfg.Kubeconfig = envKubeconfig
	}

	if cfg.Collection.WorkerConcurrency < 1 {
		cfg.Collection.WorkerConcurrency = 4
	}
	if cfg.Collection.PodLogTailLines < 0 {
		cfg.Collection.PodLogTailLines = 1000
	}
	if strings.TrimSpace(cfg.Notify.Generic.JSONKey) == "" {
		cfg.Notify.Generic.JSONKey = "text"
	} else {
		cfg.Notify.Generic.JSONKey = strings.TrimSpace(cfg.Notify.Generic.JSONKey)
	}
	if err := normalizePagerDuty(cfg); err != nil {
		return err
	}
	if err := ValidateExtraKubectl(cfg.Collection.ExtraKubectl); err != nil {
		return err
	}
	since, err := NormalizePodLogsSince(cfg.Collection.PodLogsSince)
	if err != nil {
		return err
	}
	cfg.Collection.PodLogsSince = since
	cfg.OutputDir = expandPath(cfg.OutputDir)
	resolveNotificationSecrets(cfg)
	resolveUploadSecrets(cfg)
	if err := validateNotificationConfig(*cfg); err != nil {
		return err
	}
	normalizeNotifyRetry(cfg)
	normalizeOnFailure(cfg)
	normalizeUpload(cfg)
	return validateUploadConfig(*cfg)
}

func normalizeNotifyRetry(cfg *Config) {
	r := &cfg.Notify.Retry
	if r.MaxAttempts < 1 {
		r.MaxAttempts = 3
	}
	if r.InitialBackoff <= 0 {
		r.InitialBackoff = time.Second
	}
	if r.MaxBackoff <= 0 {
		r.MaxBackoff = 10 * time.Second
	}
}

func normalizeOnFailure(cfg *Config) {
	of := &cfg.Notify.OnFailure
	if !of.Enabled {
		return
	}
	if of.MinFailedJobs < 1 {
		of.MinFailedJobs = 1
	}
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("output_dir", "./out")
	v.SetDefault("file_prefix", "groot-capture")
	v.SetDefault("collection.timeout", "20m")
	v.SetDefault("collection.worker_concurrency", 6)
	v.SetDefault("collection.namespaces", []string{"kube-system"})
	v.SetDefault("collection.targets", map[string]NamespaceTargets{})
	v.SetDefault("collection.extra_kubectl", []string{})
	v.SetDefault("collection.include_pod_logs", true)
	v.SetDefault("collection.include_previous_logs", true)
	v.SetDefault("collection.include_node_details", true)
	v.SetDefault("collection.include_node_logs", true)
	v.SetDefault("collection.node_log_tail_lines", 5000)
	v.SetDefault("collection.include_pod_metrics", true)
	v.SetDefault("collection.pod_log_tail_lines", 1500)
	v.SetDefault("notify.slack.enabled", false)
	v.SetDefault("notify.discord.enabled", false)
	v.SetDefault("notify.teams.enabled", false)
	v.SetDefault("notify.telegram.enabled", false)
	v.SetDefault("notify.generic.enabled", false)
	v.SetDefault("notify.generic.json_key", "text")
	v.SetDefault("notify.pagerduty.enabled", false)
	v.SetDefault("notify.pagerduty.severity", "warning")
	v.SetDefault("notify.pagerduty.source", "groot")
	v.SetDefault("notify.email.enabled", false)
	v.SetDefault("notify.email.port", 587)
	v.SetDefault("notify.on_failure.enabled", false)
	v.SetDefault("notify.on_failure.on_abort", true)
	v.SetDefault("notify.on_failure.min_failed_jobs", 1)
	v.SetDefault("notify.retry.max_attempts", 3)
	v.SetDefault("notify.retry.initial_backoff", "1s")
	v.SetDefault("notify.retry.max_backoff", "10s")
	v.SetDefault("collection.redact_secrets", false)
	v.SetDefault("upload.enabled", false)
	v.SetDefault("upload.continue_on_error", true)
	v.SetDefault("upload.timeout", "5m")
	v.SetDefault("upload.s3.enabled", false)
	v.SetDefault("upload.gcs.enabled", false)
	v.SetDefault("upload.sftp.enabled", false)
	v.SetDefault("upload.sftp.port", 22)
}

// defaultEtcConfigPaths are tried after ./groot.yml and ~/.groot/groot.yml (first existing file wins).
// Site-wide config is /etc/groot/groot.yml. A packaged groot.yml.sample is documentation only;
// it is not auto-loaded (use --config or copy to groot.yml).
var defaultEtcConfigPaths = []string{
	"/etc/groot/groot.yml",
}

func readDefaultConfig(v *viper.Viper) error {
	local := viper.New()
	local.SetConfigType("yaml")
	local.SetConfigFile("groot.yml")
	if err := local.ReadInConfig(); err == nil {
		return mergeConfig(v, local)
	} else if !isConfigNotFound(err) {
		return err
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("resolve home directory: %w", err)
	}

	home := viper.New()
	home.SetConfigType("yaml")
	home.SetConfigFile(filepath.Join(homeDir, ".groot", "groot.yml"))
	if err := home.ReadInConfig(); err == nil {
		return mergeConfig(v, home)
	} else if !isConfigNotFound(err) {
		return err
	}

	for _, p := range defaultEtcConfigPaths {
		etc := viper.New()
		etc.SetConfigType("yaml")
		etc.SetConfigFile(p)
		if err := etc.ReadInConfig(); err == nil {
			return mergeConfig(v, etc)
		} else if !isConfigNotFound(err) {
			return err
		}
	}

	return nil
}

func mergeConfig(dst, src *viper.Viper) error {
	for key, value := range src.AllSettings() {
		dst.Set(key, value)
	}
	return nil
}

func isConfigNotFound(err error) bool {
	if err == nil {
		return false
	}
	var notFound viper.ConfigFileNotFoundError
	if errors.As(err, &notFound) {
		return true
	}
	return errors.Is(err, os.ErrNotExist)
}

func resolveNotificationSecrets(cfg *Config) {
	cfg.Notify.Slack.WebhookURL = firstNonEmpty(
		cfg.Notify.Slack.WebhookURL,
		os.Getenv("GROOT_NOTIFY_SLACK_WEBHOOK_URL"),
	)
	cfg.Notify.Discord.WebhookURL = firstNonEmpty(
		cfg.Notify.Discord.WebhookURL,
		os.Getenv("GROOT_NOTIFY_DISCORD_WEBHOOK_URL"),
	)
	cfg.Notify.Teams.WebhookURL = firstNonEmpty(
		cfg.Notify.Teams.WebhookURL,
		os.Getenv("GROOT_NOTIFY_TEAMS_WEBHOOK_URL"),
	)
	cfg.Notify.Telegram.Token = firstNonEmpty(
		cfg.Notify.Telegram.Token,
		os.Getenv("GROOT_NOTIFY_TELEGRAM_TOKEN"),
	)
	cfg.Notify.Telegram.ChatID = firstNonEmpty(
		cfg.Notify.Telegram.ChatID,
		os.Getenv("GROOT_NOTIFY_TELEGRAM_CHAT_ID"),
	)
	cfg.Notify.Generic.WebhookURL = firstNonEmpty(
		cfg.Notify.Generic.WebhookURL,
		os.Getenv("GROOT_NOTIFY_GENERIC_WEBHOOK_URL"),
	)
	cfg.Notify.PagerDuty.RoutingKey = firstNonEmpty(
		cfg.Notify.PagerDuty.RoutingKey,
		os.Getenv("GROOT_NOTIFY_PAGERDUTY_ROUTING_KEY"),
	)
	cfg.Notify.Email.Host = firstNonEmpty(cfg.Notify.Email.Host, os.Getenv("GROOT_NOTIFY_EMAIL_HOST"))
	cfg.Notify.Email.Username = firstNonEmpty(cfg.Notify.Email.Username, os.Getenv("GROOT_NOTIFY_EMAIL_USERNAME"))
	cfg.Notify.Email.Password = firstNonEmpty(cfg.Notify.Email.Password, os.Getenv("GROOT_NOTIFY_EMAIL_PASSWORD"))
	cfg.Notify.Email.From = firstNonEmpty(cfg.Notify.Email.From, os.Getenv("GROOT_NOTIFY_EMAIL_FROM"))
	cfg.Notify.Email.To = firstNonEmpty(cfg.Notify.Email.To, os.Getenv("GROOT_NOTIFY_EMAIL_TO"))
	cfg.Notify.Generic.HMACSecret = firstNonEmpty(
		cfg.Notify.Generic.HMACSecret,
		os.Getenv("GROOT_NOTIFY_GENERIC_HMAC_SECRET"),
	)
}

func validateNotificationConfig(cfg Config) error {
	if err := validateWebhookChannels(cfg); err != nil {
		return err
	}
	if err := validateTelegram(cfg); err != nil {
		return err
	}
	if err := validateGeneric(cfg); err != nil {
		return err
	}
	if err := validatePagerDuty(cfg); err != nil {
		return err
	}
	return validateEmail(cfg)
}

func validateWebhookChannels(cfg Config) error {
	if cfg.Notify.Slack.Enabled && len(SplitSemicolonList(cfg.Notify.Slack.WebhookURL)) == 0 {
		return fmt.Errorf("notify.slack.enabled=true requires webhook_url or env GROOT_NOTIFY_SLACK_WEBHOOK_URL (semicolon-separated for multiple webhooks)")
	}
	if cfg.Notify.Discord.Enabled && len(SplitSemicolonList(cfg.Notify.Discord.WebhookURL)) == 0 {
		return fmt.Errorf("notify.discord.enabled=true requires webhook_url or env GROOT_NOTIFY_DISCORD_WEBHOOK_URL (semicolon-separated for multiple webhooks)")
	}
	if cfg.Notify.Teams.Enabled && len(SplitSemicolonList(cfg.Notify.Teams.WebhookURL)) == 0 {
		return fmt.Errorf("notify.teams.enabled=true requires webhook_url or env GROOT_NOTIFY_TEAMS_WEBHOOK_URL (semicolon-separated for multiple webhooks)")
	}
	return nil
}

func validateTelegram(cfg Config) error {
	if !cfg.Notify.Telegram.Enabled {
		return nil
	}
	if strings.TrimSpace(cfg.Notify.Telegram.Token) == "" {
		return fmt.Errorf("notify.telegram.enabled=true requires token or env GROOT_NOTIFY_TELEGRAM_TOKEN")
	}
	if len(SplitSemicolonList(cfg.Notify.Telegram.ChatID)) == 0 {
		return fmt.Errorf("notify.telegram.enabled=true requires chat_id or env GROOT_NOTIFY_TELEGRAM_CHAT_ID (semicolon-separated for multiple chat ids)")
	}
	return nil
}

func validateGeneric(cfg Config) error {
	if cfg.Notify.Generic.Enabled && len(SplitSemicolonList(cfg.Notify.Generic.WebhookURL)) == 0 {
		return fmt.Errorf("notify.generic.enabled=true requires webhook_url or env GROOT_NOTIFY_GENERIC_WEBHOOK_URL (semicolon-separated for multiple URLs)")
	}
	if cfg.Notify.Generic.Enabled && strings.TrimSpace(cfg.Notify.Generic.BodyTemplate) != "" {
		if !strings.Contains(cfg.Notify.Generic.BodyTemplate, "{") {
			return fmt.Errorf("notify.generic.body_template must be JSON with placeholders (see SPEC)")
		}
	}
	return nil
}

func validatePagerDuty(cfg Config) error {
	if cfg.Notify.PagerDuty.Enabled && len(SplitSemicolonList(cfg.Notify.PagerDuty.RoutingKey)) == 0 {
		return fmt.Errorf("notify.pagerduty.enabled=true requires routing_key or env GROOT_NOTIFY_PAGERDUTY_ROUTING_KEY (semicolon-separated for multiple integration keys)")
	}
	return nil
}

func validateEmail(cfg Config) error {
	if !cfg.Notify.Email.Enabled {
		return nil
	}
	if strings.TrimSpace(cfg.Notify.Email.Host) == "" {
		return fmt.Errorf("notify.email.enabled=true requires host or env GROOT_NOTIFY_EMAIL_HOST")
	}
	if strings.TrimSpace(cfg.Notify.Email.From) == "" {
		return fmt.Errorf("notify.email.enabled=true requires from or env GROOT_NOTIFY_EMAIL_FROM")
	}
	if len(SplitSemicolonList(cfg.Notify.Email.To)) == 0 {
		return fmt.Errorf("notify.email.enabled=true requires to or env GROOT_NOTIFY_EMAIL_TO (semicolon-separated for multiple recipients)")
	}
	return nil
}

func normalizePagerDuty(cfg *Config) error {
	p := &cfg.Notify.PagerDuty
	if strings.TrimSpace(p.Severity) == "" {
		p.Severity = "warning"
	} else {
		p.Severity = strings.ToLower(strings.TrimSpace(p.Severity))
	}
	if strings.TrimSpace(p.Source) == "" {
		p.Source = "groot"
	} else {
		p.Source = strings.TrimSpace(p.Source)
	}
	if p.Enabled && !isPagerDutySeverity(p.Severity) {
		return fmt.Errorf("notify.pagerduty.severity must be one of: critical, error, warning, info (got %q)", p.Severity)
	}
	return nil
}

func isPagerDutySeverity(s string) bool {
	switch s {
	case "critical", "error", "warning", "info":
		return true
	default:
		return false
	}
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

// SplitSemicolonList splits a string into non-empty trimmed segments separated by ';'.
// Used for Slack/Discord/Teams/generic webhook URLs, Telegram chat_id lists, and PagerDuty routing keys.
func SplitSemicolonList(raw string) []string {
	parts := strings.Split(raw, ";")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		u := strings.TrimSpace(p)
		if u != "" {
			out = append(out, u)
		}
	}
	return out
}

// SplitWebhookURLs splits Slack or Teams webhook_url values (alias of SplitSemicolonList).
func SplitWebhookURLs(raw string) []string {
	return SplitSemicolonList(raw)
}

func expandPath(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return value
	}

	expanded := os.ExpandEnv(trimmed)
	if strings.HasPrefix(expanded, "~") {
		home, err := os.UserHomeDir()
		if err == nil && home != "" {
			switch expanded {
			case "~":
				expanded = home
			default:
				if strings.HasPrefix(expanded, "~/") {
					expanded = filepath.Join(home, strings.TrimPrefix(expanded, "~/"))
				}
			}
		}
	}
	return expanded
}

func normalizeUpload(cfg *Config) {
	u := &cfg.Upload
	if u.Timeout <= 0 {
		u.Timeout = 5 * time.Minute
	}
}

func resolveUploadSecrets(cfg *Config) {
	u := &cfg.Upload
	u.S3.Bucket = firstNonEmpty(u.S3.Bucket, os.Getenv("GROOT_UPLOAD_S3_BUCKET"))
	u.S3.Region = firstNonEmpty(u.S3.Region, os.Getenv("GROOT_UPLOAD_S3_REGION"), os.Getenv("AWS_REGION"), os.Getenv("AWS_DEFAULT_REGION"))
	u.S3.KeyPrefix = firstNonEmpty(u.S3.KeyPrefix, os.Getenv("GROOT_UPLOAD_S3_KEY_PREFIX"))
	u.S3.Endpoint = firstNonEmpty(u.S3.Endpoint, os.Getenv("GROOT_UPLOAD_S3_ENDPOINT"))
	u.GCS.Bucket = firstNonEmpty(u.GCS.Bucket, os.Getenv("GROOT_UPLOAD_GCS_BUCKET"))
	u.GCS.KeyPrefix = firstNonEmpty(u.GCS.KeyPrefix, os.Getenv("GROOT_UPLOAD_GCS_KEY_PREFIX"))

	u.SFTP.Host = firstNonEmpty(u.SFTP.Host, os.Getenv("GROOT_UPLOAD_SFTP_HOST"))
	u.SFTP.User = firstNonEmpty(u.SFTP.User, os.Getenv("GROOT_UPLOAD_SFTP_USER"))
	u.SFTP.RemoteDir = firstNonEmpty(u.SFTP.RemoteDir, os.Getenv("GROOT_UPLOAD_SFTP_REMOTE_DIR"))
	u.SFTP.IdentityFile = firstNonEmpty(u.SFTP.IdentityFile, os.Getenv("GROOT_UPLOAD_SFTP_IDENTITY_FILE"))
	u.SFTP.KnownHostsFile = firstNonEmpty(u.SFTP.KnownHostsFile, os.Getenv("GROOT_UPLOAD_SFTP_KNOWN_HOSTS"))
	if port := os.Getenv("GROOT_UPLOAD_SFTP_PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil && p > 0 {
			u.SFTP.Port = p
		}
	}
}

func validateUploadConfig(cfg Config) error {
	u := cfg.Upload
	if !u.Enabled {
		return nil
	}
	anyEnabled := u.S3.Enabled || u.GCS.Enabled || u.SFTP.Enabled
	if !anyEnabled {
		return fmt.Errorf("upload.enabled=true requires at least one provider (upload.s3.enabled, upload.gcs.enabled, or upload.sftp.enabled)")
	}
	if u.S3.Enabled && strings.TrimSpace(u.S3.Bucket) == "" {
		return fmt.Errorf("upload.s3.enabled=true requires bucket or env GROOT_UPLOAD_S3_BUCKET")
	}
	if u.GCS.Enabled && strings.TrimSpace(u.GCS.Bucket) == "" {
		return fmt.Errorf("upload.gcs.enabled=true requires bucket or env GROOT_UPLOAD_GCS_BUCKET")
	}
	if u.SFTP.Enabled {
		if strings.TrimSpace(u.SFTP.Host) == "" {
			return fmt.Errorf("upload.sftp.enabled=true requires host or env GROOT_UPLOAD_SFTP_HOST")
		}
		if u.SFTP.Port < 1 || u.SFTP.Port > 65535 {
			return fmt.Errorf("upload.sftp.port must be in range 1-65535 (got %d)", u.SFTP.Port)
		}
	}
	return nil
}
