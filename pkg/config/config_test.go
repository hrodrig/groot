package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSampleYAML(t *testing.T) {
	s := SampleYAML()
	if !strings.Contains(s, "groot-capture") {
		t.Fatalf("sample should mention file_prefix default: %q", s)
	}
	if !strings.Contains(s, "collection:") {
		t.Fatal("sample should include collection block")
	}
}

func TestLoad_defaultsWhenNoFiles(t *testing.T) {
	root := t.TempDir()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", root)
	t.Setenv("KUBECONFIG", "")
	t.Setenv("GROOT_NOTIFY_SLACK_WEBHOOK_URL", "")
	t.Setenv("GROOT_NOTIFY_TEAMS_WEBHOOK_URL", "")
	t.Setenv("GROOT_NOTIFY_TELEGRAM_TOKEN", "")
	t.Setenv("GROOT_NOTIFY_TELEGRAM_CHAT_ID", "")

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Collection.WorkerConcurrency < 1 {
		t.Fatalf("worker concurrency: %d", cfg.Collection.WorkerConcurrency)
	}
	if !strings.HasSuffix(cfg.OutputDir, "out") && cfg.OutputDir != "" {
		t.Fatalf("unexpected output_dir: %q", cfg.OutputDir)
	}
}

func TestLoad_validFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cfg.yaml")
	content := `
kubeconfig: ""
output_dir: "` + filepath.Join(dir, "captures") + `"
file_prefix: "pfx"
collection:
  timeout: 1m
  worker_concurrency: 2
  namespaces: ["ns-a"]
  include_pod_logs: false
notify:
  slack:
    enabled: false
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("KUBECONFIG", "")
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.FilePrefix != "pfx" {
		t.Fatalf("file_prefix: %q", cfg.FilePrefix)
	}
	if cfg.Collection.WorkerConcurrency != 2 {
		t.Fatalf("worker_concurrency: %d", cfg.Collection.WorkerConcurrency)
	}
	if len(cfg.Collection.Namespaces) != 1 || cfg.Collection.Namespaces[0] != "ns-a" {
		t.Fatalf("namespaces: %#v", cfg.Collection.Namespaces)
	}
	if cfg.Collection.IncludePodLogs {
		t.Fatal("expected include_pod_logs false")
	}
}

func TestSplitSemicolonList(t *testing.T) {
	if got := SplitSemicolonList(""); len(got) != 0 {
		t.Fatalf("empty: %#v", got)
	}
	got := SplitSemicolonList(" https://a/x ; ;https://b/y ")
	if len(got) != 2 || got[0] != "https://a/x" || got[1] != "https://b/y" {
		t.Fatalf("got %#v", got)
	}
	chats := SplitSemicolonList("-1001; -1002 ")
	if len(chats) != 2 || chats[0] != "-1001" || chats[1] != "-1002" {
		t.Fatalf("chats %#v", chats)
	}
	if got := SplitWebhookURLs("a;b"); len(got) != 2 {
		t.Fatalf("SplitWebhookURLs alias: %#v", got)
	}
}

func TestLoad_slackEnabledMultipleWebhooks(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ok.yaml")
	if err := os.WriteFile(path, []byte(`
notify:
  slack:
    enabled: true
    webhook_url: "https://hooks.slack.com/services/A;https://hooks.slack.com/services/B"
`), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GROOT_NOTIFY_SLACK_WEBHOOK_URL", "")
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(SplitSemicolonList(cfg.Notify.Slack.WebhookURL)) != 2 {
		t.Fatalf("urls: %#v", SplitSemicolonList(cfg.Notify.Slack.WebhookURL))
	}
}

func TestLoad_telegramEnabledMultipleChatIDs(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tg.yaml")
	if err := os.WriteFile(path, []byte(`
notify:
  telegram:
    enabled: true
    token: "123:abc"
    chat_id: "-1001; -1002"
`), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GROOT_NOTIFY_TELEGRAM_TOKEN", "")
	t.Setenv("GROOT_NOTIFY_TELEGRAM_CHAT_ID", "")
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(SplitSemicolonList(cfg.Notify.Telegram.ChatID)) != 2 {
		t.Fatalf("chat ids: %#v", SplitSemicolonList(cfg.Notify.Telegram.ChatID))
	}
}

func TestLoad_genericEnabledMissingURL(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "gen.yaml")
	if err := os.WriteFile(path, []byte(`
notify:
  generic:
    enabled: true
    webhook_url: ""
`), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GROOT_NOTIFY_GENERIC_WEBHOOK_URL", "")
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error when generic enabled without webhook")
	}
	if !strings.Contains(err.Error(), "generic") {
		t.Fatalf("error: %v", err)
	}
}

func TestLoad_genericJSONKeyDefault(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "gen2.yaml")
	if err := os.WriteFile(path, []byte(`
notify:
  generic:
    enabled: false
    webhook_url: ""
    json_key: ""
`), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Notify.Generic.JSONKey != "text" {
		t.Fatalf("json_key: %q", cfg.Notify.Generic.JSONKey)
	}
}

func TestLoad_genericContentKey(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "gen3.yaml")
	if err := os.WriteFile(path, []byte(`
notify:
  generic:
    enabled: false
    webhook_url: ""
    json_key: "content"
`), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Notify.Generic.JSONKey != "content" {
		t.Fatalf("json_key: %q", cfg.Notify.Generic.JSONKey)
	}
}

func TestLoad_pagerdutyEnabledMissingRoutingKey(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "pd.yaml")
	if err := os.WriteFile(path, []byte(`
notify:
  pagerduty:
    enabled: true
    routing_key: ""
`), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GROOT_NOTIFY_PAGERDUTY_ROUTING_KEY", "")
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error when pagerduty enabled without routing_key")
	}
	if !strings.Contains(err.Error(), "pagerduty") {
		t.Fatalf("error: %v", err)
	}
}

func TestLoad_pagerdutyRoutingKeyFromEnv(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "pd3.yaml")
	if err := os.WriteFile(path, []byte(`
notify:
  pagerduty:
    enabled: true
    routing_key: ""
`), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GROOT_NOTIFY_PAGERDUTY_ROUTING_KEY", "env-rk-123")
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	l := SplitSemicolonList(cfg.Notify.PagerDuty.RoutingKey)
	if len(l) != 1 || l[0] != "env-rk-123" {
		t.Fatalf("%#v", l)
	}
}

func TestLoad_pagerdutyInvalidSeverity(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "pd2.yaml")
	if err := os.WriteFile(path, []byte(`
notify:
  pagerduty:
    enabled: true
    routing_key: "abc"
    severity: "banana"
`), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for invalid severity")
	}
}

func TestLoad_discordEnabledMissingURL(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "disc.yaml")
	if err := os.WriteFile(path, []byte(`
notify:
  discord:
    enabled: true
    webhook_url: ""
`), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GROOT_NOTIFY_DISCORD_WEBHOOK_URL", "")
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error when discord enabled without webhook")
	}
	if !strings.Contains(err.Error(), "discord") {
		t.Fatalf("error: %v", err)
	}
}

func TestLoad_slackEnabledOnlySeparatorsFails(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad2.yaml")
	if err := os.WriteFile(path, []byte(`
notify:
  slack:
    enabled: true
    webhook_url: " ; ; "
`), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GROOT_NOTIFY_SLACK_WEBHOOK_URL", "")
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error when slack enabled with no real webhook URLs")
	}
}

func TestLoad_slackEnabledMissingURL(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.yaml")
	if err := os.WriteFile(path, []byte(`
notify:
  slack:
    enabled: true
    webhook_url: ""
`), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GROOT_NOTIFY_SLACK_WEBHOOK_URL", "")
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error when slack enabled without webhook")
	}
	if !strings.Contains(err.Error(), "slack") {
		t.Fatalf("error: %v", err)
	}
}

func TestLoad_podLogTailLinesNegativeCoerced(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tail.yaml")
	if err := os.WriteFile(path, []byte(`
collection:
  pod_log_tail_lines: -5
notify:
  slack: { enabled: false }
`), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Collection.PodLogTailLines != 1000 {
		t.Fatalf("expected negative coerced to 1000, got %d", cfg.Collection.PodLogTailLines)
	}
}
