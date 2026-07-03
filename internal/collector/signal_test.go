package collector

import (
	"testing"

	"github.com/hrodrig/groot/internal/config"
)

func TestPrioritizeSignalJobs_signalLeadsBulk(t *testing.T) {
	in := []job{
		{Name: "namespace-resources-default"},
		{Name: "events-warning"},
		{Name: "pod-logs-default"},
		{Name: "events-all"},
		{Name: "cluster-info"},
		{Name: "nodes-wide"},
		{Name: "pod-logs-kube-system"},
		{Name: "pods-all"},
	}
	got := prioritizeSignalJobs(in)

	// First 5 are signal-class names in the order found in input.
	wantFirst := []string{"events-warning", "events-all", "cluster-info", "nodes-wide", "pods-all"}
	for i, w := range wantFirst {
		if got[i].Name != w {
			t.Fatalf("pos %d: got %q want %q (got=%+v)", i, got[i].Name, w, jobNames(got))
		}
	}
	// The bulk slice must preserve original order.
	wantBulk := []string{"namespace-resources-default", "pod-logs-default", "pod-logs-kube-system"}
	if len(got) != len(wantFirst)+len(wantBulk) {
		t.Fatalf("len mismatch: got=%d want=%d (%+v)", len(got), len(wantFirst)+len(wantBulk), jobNames(got))
	}
	for i, w := range wantBulk {
		pos := len(wantFirst) + i
		if got[pos].Name != w {
			t.Fatalf("bulk pos %d: got %q want %q", pos, got[pos].Name, w)
		}
	}
}

func TestPrioritizeSignalJobs_empty(t *testing.T) {
	if got := prioritizeSignalJobs(nil); len(got) != 0 {
		t.Fatalf("expected empty, got %+v", got)
	}
}

func TestPrioritizeSignalJobs_noSignalClass(t *testing.T) {
	in := []job{
		{Name: "namespace-resources-default"},
		{Name: "pod-logs-default"},
	}
	got := prioritizeSignalJobs(in)
	for i, j := range in {
		if got[i].Name != j.Name {
			t.Fatalf("pos %d: order should be preserved, got %q want %q", i, got[i].Name, j.Name)
		}
	}
}

func TestSetHighSignalFirst_defaultsToTrue(t *testing.T) {
	s := New(config.Config{})
	if !s.highSignalFirst {
		t.Fatal("default should be true")
	}
	s.SetHighSignalFirst(false)
	if s.highSignalFirst {
		t.Fatal("expected false after disable")
	}
}

func jobNames(in []job) []string {
	out := make([]string, len(in))
	for i, j := range in {
		out[i] = j.Name
	}
	return out
}
