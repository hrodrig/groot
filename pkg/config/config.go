package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config represents the YAML and env configuration.
type Config struct {
	Kubeconfig string        `mapstructure:"kubeconfig"`
	OutputDir  string        `mapstructure:"output_dir"`
	FilePrefix string        `mapstructure:"file_prefix"`
	Collection CollectionCfg `mapstructure:"collection"`
	Notify     NotifyCfg     `mapstructure:"notify"`
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
	IncludeNodeDetails  bool                        `mapstructure:"include_node_details"`
}

type NamespaceTargets struct {
	Deployments  []string `mapstructure:"deployments"`
	StatefulSets []string `mapstructure:"statefulsets"`
	DaemonSets   []string `mapstructure:"daemonsets"`
	HelmReleases []string `mapstructure:"helm_releases"`
}

type NotifyCfg struct {
	Slack    WebhookCfg  `mapstructure:"slack"`
	Telegram TelegramCfg `mapstructure:"telegram"`
	Teams    WebhookCfg  `mapstructure:"teams"`
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

	if envKubeconfig := os.Getenv("KUBECONFIG"); envKubeconfig != "" {
		cfg.Kubeconfig = envKubeconfig
	}

	if cfg.Collection.WorkerConcurrency < 1 {
		cfg.Collection.WorkerConcurrency = 4
	}
	if cfg.Collection.PodLogTailLines < 0 {
		cfg.Collection.PodLogTailLines = 1000
	}
	cfg.OutputDir = expandPath(cfg.OutputDir)
	resolveNotificationSecrets(&cfg)
	if err := validateNotificationConfig(cfg); err != nil {
		return Config{}, err
	}

	return cfg, nil
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
	v.SetDefault("collection.pod_log_tail_lines", 1500)
	v.SetDefault("notify.slack.enabled", false)
	v.SetDefault("notify.teams.enabled", false)
	v.SetDefault("notify.telegram.enabled", false)
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
}

func validateNotificationConfig(cfg Config) error {
	if cfg.Notify.Slack.Enabled && strings.TrimSpace(cfg.Notify.Slack.WebhookURL) == "" {
		return fmt.Errorf("notify.slack.enabled=true requires webhook_url or env GROOT_NOTIFY_SLACK_WEBHOOK_URL")
	}
	if cfg.Notify.Teams.Enabled && strings.TrimSpace(cfg.Notify.Teams.WebhookURL) == "" {
		return fmt.Errorf("notify.teams.enabled=true requires webhook_url or env GROOT_NOTIFY_TEAMS_WEBHOOK_URL")
	}
	if cfg.Notify.Telegram.Enabled {
		if strings.TrimSpace(cfg.Notify.Telegram.Token) == "" {
			return fmt.Errorf("notify.telegram.enabled=true requires token or env GROOT_NOTIFY_TELEGRAM_TOKEN")
		}
		if strings.TrimSpace(cfg.Notify.Telegram.ChatID) == "" {
			return fmt.Errorf("notify.telegram.enabled=true requires chat_id or env GROOT_NOTIFY_TELEGRAM_CHAT_ID")
		}
	}
	return nil
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
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
