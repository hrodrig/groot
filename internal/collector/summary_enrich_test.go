package collector

import (
	"context"
	"testing"
	"time"

	"github.com/hrodrig/groot/internal/config"
)

func TestEnrichSummary_nilSafe(t *testing.T) {
	s := New(config.Config{})
	s.enrichSummary(context.Background(), nil) // must not panic
}

func TestEnrichSummary_defaults(t *testing.T) {
	s := New(config.Config{ClientID: "bastion-01", ClusterName: "prod-cluster"})
	s.SetBuildInfo("", "", "", "")
	var sum Summary
	s.enrichSummary(context.Background(), &sum)
	if sum.ClientID != "bastion-01" {
		t.Fatalf("ClientID=%q", sum.ClientID)
	}
	if sum.Cluster != "prod-cluster" {
		t.Fatalf("Cluster=%q", sum.Cluster)
	}
	if sum.Version != "dev" {
		t.Fatalf("Version=%q", sum.Version)
	}
	if sum.Mode != "collect" {
		t.Fatalf("Mode=%q", sum.Mode)
	}
	if sum.FinishedAt.IsZero() {
		t.Fatal("FinishedAt zero")
	}
	// Preserves Mode / FinishedAt when already set.
	fixed := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	sum2 := Summary{Mode: "notify.test", FinishedAt: fixed, Version: "1.0.0"}
	s.SetBuildInfo("1.2.2", "", "", "")
	s.enrichSummary(context.Background(), &sum2)
	if sum2.Mode != "notify.test" || !sum2.FinishedAt.Equal(fixed) {
		t.Fatalf("%+v", sum2)
	}
	if sum2.Version != "1.2.2" {
		t.Fatalf("Version=%q", sum2.Version)
	}
}
