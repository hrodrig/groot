package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNotifyTest_sendsToSlack(t *testing.T) {
	resetPersistentFlags(t)
	resetPersistentFlags(t)
	var got struct {
		Text        string `json:"text"`
		Attachments []struct {
			Fallback string `json:"fallback"`
			Text     string `json:"text"`
		} `json:"attachments"`
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&got)
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "c.yaml")
	yaml := "output_dir: " + filepath.ToSlash(dir) + "\nnotify:\n  slack:\n    enabled: true\n    webhook_url: " + srv.URL + "\n  retry:\n    max_attempts: 1\n"
	if err := os.WriteFile(cfgPath, []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"notify", "test", "--config", cfgPath})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("notify test: %v", err)
	}
	if !strings.Contains(buf.String(), `event "notify.test"`) {
		t.Fatalf("stdout=%q", buf.String())
	}
	if got.Text != "" {
		t.Fatalf("top-level text must be empty, got %q", got.Text)
	}
	if len(got.Attachments) != 1 {
		t.Fatalf("attachments=%d", len(got.Attachments))
	}
	att := got.Attachments[0]
	if !strings.Contains(att.Fallback, "GROOT notify test") {
		t.Fatalf("fallback=%q", att.Fallback)
	}
	if !strings.Contains(att.Text, "GROOT notify test") {
		t.Fatalf("attachment text=%q", att.Text)
	}
}

func TestNotifyTest_noChannelsExitConfig(t *testing.T) {
	resetPersistentFlags(t)
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "c.yaml")
	yaml := "output_dir: " + filepath.ToSlash(dir) + "\nnotify:\n  slack:\n    enabled: false\n"
	if err := os.WriteFile(cfgPath, []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}

	rootCmd.SetOut(bytes.NewBuffer(nil))
	rootCmd.SetErr(bytes.NewBuffer(nil))
	rootCmd.SetArgs([]string{"notify", "test", "--config", cfgPath})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error")
	}
	if got := ExitCodeOf(err); got != ExitConfigError {
		t.Fatalf("exit=%d want %d", got, ExitConfigError)
	}
}

func TestNotifyTest_badEventExitConfig(t *testing.T) {
	resetPersistentFlags(t)
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "c.yaml")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	yaml := "output_dir: " + filepath.ToSlash(dir) + "\nnotify:\n  slack:\n    enabled: true\n    webhook_url: " + srv.URL + "\n"
	if err := os.WriteFile(cfgPath, []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}

	rootCmd.SetOut(bytes.NewBuffer(nil))
	rootCmd.SetErr(bytes.NewBuffer(nil))
	rootCmd.SetArgs([]string{"notify", "test", "--config", cfgPath, "--event", "nope"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error")
	}
	if got := ExitCodeOf(err); got != ExitConfigError {
		t.Fatalf("exit=%d want %d", got, ExitConfigError)
	}
}

func TestNotifyTest_deliveryFailureExit4(t *testing.T) {
	resetPersistentFlags(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "c.yaml")
	yaml := "output_dir: " + filepath.ToSlash(dir) + "\nnotify:\n  slack:\n    enabled: true\n    webhook_url: " + srv.URL + "\n  retry:\n    max_attempts: 1\n"
	if err := os.WriteFile(cfgPath, []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}

	rootCmd.SetOut(bytes.NewBuffer(nil))
	rootCmd.SetErr(bytes.NewBuffer(nil))
	rootCmd.SetArgs([]string{"notify", "test", "--config", cfgPath})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error")
	}
	if got := ExitCodeOf(err); got != ExitNotifyFailed {
		t.Fatalf("exit=%d want %d", got, ExitNotifyFailed)
	}
}
