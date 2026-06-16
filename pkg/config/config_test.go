package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSampleYAML(t *testing.T) {
	s := SampleYAML()
	if !strings.Contains(s, "groot-capture") {
		t.Fatalf("sample should mention file_prefix default: %q", s)
	}
	if !strings.Contains(s, "cluster_name:") {
		t.Fatalf("sample should document cluster_name: %q", s)
	}
	if !strings.Contains(s, "upload:") {
		t.Fatalf("sample should document upload block: %q", s)
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

	oldEtc := defaultEtcConfigPaths
	defaultEtcConfigPaths = nil
	t.Cleanup(func() { defaultEtcConfigPaths = oldEtc })

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

func TestLoad_systemEtcWhenNoLocalOrHome(t *testing.T) {
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

	sysPath := filepath.Join(root, "site.yaml")
	wantOut := filepath.Join(root, "from-etc")
	content := `
kubeconfig: ""
output_dir: "` + wantOut + `"
file_prefix: "etc-site"
collection:
  timeout: 1m
  worker_concurrency: 2
  namespaces: ["ns-etc"]
notify:
  slack:
    enabled: false
`
	if err := os.WriteFile(sysPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	oldEtc := defaultEtcConfigPaths
	defaultEtcConfigPaths = []string{sysPath}
	t.Cleanup(func() { defaultEtcConfigPaths = oldEtc })

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.OutputDir != wantOut {
		t.Fatalf("output_dir: got %q want %q", cfg.OutputDir, wantOut)
	}
	if cfg.FilePrefix != "etc-site" {
		t.Fatalf("file_prefix: %q", cfg.FilePrefix)
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

func TestLoad_extraKubectlAllowed(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "extra.yaml")
	content := `
collection:
  extra_kubectl:
    - "get ns -o name"
    - "config view --minify"
    - "auth can-i list pods"
notify:
  slack: { enabled: false }
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("KUBECONFIG", "")
	if _, err := Load(path); err != nil {
		t.Fatalf("Load: %v", err)
	}
}

func TestLoad_extraKubectlRejected(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad-extra.yaml")
	content := `
collection:
  extra_kubectl:
    - "delete pods --all"
notify:
  slack: { enabled: false }
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("KUBECONFIG", "")
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for disallowed extra_kubectl")
	}
	if !strings.Contains(err.Error(), "extra_kubectl") {
		t.Fatalf("error should mention extra_kubectl: %v", err)
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

func TestLoad_podLogsSinceInvalid(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "since.yaml")
	if err := os.WriteFile(path, []byte(`
collection:
  pod_logs_since: "not-a-duration"
notify:
  slack: { enabled: false }
`), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("KUBECONFIG", "")
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for invalid pod_logs_since")
	}
	if !strings.Contains(err.Error(), "pod_logs_since") {
		t.Fatalf("error should mention pod_logs_since: %v", err)
	}
}

func TestLoad_podLogsSinceNormalized(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "since-ok.yaml")
	if err := os.WriteFile(path, []byte(`
collection:
  pod_logs_since: "48"
notify:
  slack: { enabled: false }
`), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("KUBECONFIG", "")
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Collection.PodLogsSince != "48h" {
		t.Fatalf("pod_logs_since: got %q want 48h", cfg.Collection.PodLogsSince)
	}
}

func TestLoad_uploadEnabledRequiresProvider(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "upload.yaml")
	if err := os.WriteFile(path, []byte(`
upload:
  enabled: true
notify:
  slack: { enabled: false }
`), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := Load(path)
	if err == nil || !strings.Contains(err.Error(), "upload.s3.enabled") {
		t.Fatalf("err=%v", err)
	}
}

func TestLoad_uploadS3EnvBucket(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "upload-s3.yaml")
	if err := os.WriteFile(path, []byte(`
upload:
  enabled: true
  s3:
    enabled: true
notify:
  slack: { enabled: false }
`), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GROOT_UPLOAD_S3_BUCKET", "my-bucket")
	t.Setenv("GROOT_UPLOAD_S3_REGION", "eu-west-1")
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Upload.S3.Bucket != "my-bucket" || cfg.Upload.S3.Region != "eu-west-1" {
		t.Fatalf("%+v", cfg.Upload.S3)
	}
}

func TestValidateUploadConfig_s3BucketRequired(t *testing.T) {
	err := validateUploadConfig(Config{Upload: UploadCfg{Enabled: true, S3: S3UploadCfg{Enabled: true}}})
	if err == nil || !strings.Contains(err.Error(), "bucket") {
		t.Fatalf("err=%v", err)
	}
}

func TestValidateUploadConfig_gcsBucketRequired(t *testing.T) {
	err := validateUploadConfig(Config{Upload: UploadCfg{Enabled: true, GCS: GCSUploadCfg{Enabled: true}}})
	if err == nil || !strings.Contains(err.Error(), "bucket") {
		t.Fatalf("err=%v", err)
	}
}

func TestNormalizeUpload_defaultTimeout(t *testing.T) {
	cfg := Config{}
	normalizeUpload(&cfg)
	if cfg.Upload.Timeout != 5*time.Minute {
		t.Fatalf("timeout=%s", cfg.Upload.Timeout)
	}
}

func TestValidateUploadConfig_disabledOK(t *testing.T) {
	if err := validateUploadConfig(Config{}); err != nil {
		t.Fatal(err)
	}
}

func TestResolveUploadSecrets_env(t *testing.T) {
	cfg := Config{Upload: UploadCfg{GCS: GCSUploadCfg{Enabled: true}}}
	t.Setenv("GROOT_UPLOAD_GCS_BUCKET", "gcs-bucket")
	t.Setenv("GROOT_UPLOAD_GCS_KEY_PREFIX", "pfx")
	resolveUploadSecrets(&cfg)
	if cfg.Upload.GCS.Bucket != "gcs-bucket" || cfg.Upload.GCS.KeyPrefix != "pfx" {
		t.Fatalf("%+v", cfg.Upload.GCS)
	}
}

func TestValidateUploadConfig_sftpHostRequired(t *testing.T) {
	err := validateUploadConfig(Config{Upload: UploadCfg{Enabled: true, SFTP: SFTPUploadCfg{Enabled: true}}})
	if err == nil || !strings.Contains(err.Error(), "host") {
		t.Fatalf("err=%v", err)
	}
}

func TestResolveUploadSecrets_sftpEnv(t *testing.T) {
	cfg := Config{Upload: UploadCfg{SFTP: SFTPUploadCfg{Enabled: true, Port: 22}}}
	t.Setenv("GROOT_UPLOAD_SFTP_HOST", "relay.example.com")
	t.Setenv("GROOT_UPLOAD_SFTP_USER", "groot-inbox")
	t.Setenv("GROOT_UPLOAD_SFTP_REMOTE_DIR", "inbox")
	t.Setenv("GROOT_UPLOAD_SFTP_IDENTITY_FILE", "/home/groot/.ssh/id_ed25519")
	t.Setenv("GROOT_UPLOAD_SFTP_KNOWN_HOSTS", "/etc/groot/known_hosts")
	t.Setenv("GROOT_UPLOAD_SFTP_PORT", "2222")
	resolveUploadSecrets(&cfg)
	s := cfg.Upload.SFTP
	if s.Host != "relay.example.com" || s.User != "groot-inbox" || s.RemoteDir != "inbox" {
		t.Fatalf("%+v", s)
	}
	if s.IdentityFile != "/home/groot/.ssh/id_ed25519" || s.KnownHostsFile != "/etc/groot/known_hosts" {
		t.Fatalf("identity=%s known_hosts=%s", s.IdentityFile, s.KnownHostsFile)
	}
	if s.Port != 2222 {
		t.Fatalf("port=%d", s.Port)
	}
}

func TestValidateUploadConfig_sftpWithAllThreeProviders(t *testing.T) {
	cfg := Config{Upload: UploadCfg{
		Enabled: true,
		S3:      S3UploadCfg{Enabled: true, Bucket: "b"},
		GCS:     GCSUploadCfg{Enabled: true, Bucket: "g"},
		SFTP:    SFTPUploadCfg{Enabled: true, Host: "h", Port: 22},
	}}
	if err := validateUploadConfig(cfg); err != nil {
		t.Fatal(err)
	}
}

func TestValidateUploadConfig_sftpPortRange(t *testing.T) {
	tests := []struct {
		port int
		ok   bool
	}{
		{0, false},
		{1, true},
		{22, true},
		{65535, true},
		{65536, false},
	}
	for _, tc := range tests {
		cfg := Config{Upload: UploadCfg{
			Enabled: true,
			SFTP:    SFTPUploadCfg{Enabled: true, Host: "h", Port: tc.port},
		}}
		err := validateUploadConfig(cfg)
		if tc.ok && err != nil {
			t.Errorf("port=%d expected ok, got %v", tc.port, err)
		}
		if !tc.ok && err == nil {
			t.Errorf("port=%d expected error", tc.port)
		}
	}
}
