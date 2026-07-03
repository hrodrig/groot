// Package collector: human-readable summary footer for `groot collect --summary`
// (ROADMAP #42, 0.9.x).
//
// The footer is meant for operators running collect interactively or from a
// bastion — short enough to fit in one screen, with the actionable numbers
// (total / success / failed jobs, archive path, duration, and counts of pods
// in the most common unhealthy states).
//
// This file is pure formatting; it does not touch Summary or any persisted
// archive artifact.
package collector

import (
	"fmt"
	"io"
	"sort"
	"strings"
)

// SummaryLine is one row of the human-readable collect footer. We keep the
// type minimal so it composes cleanly with both text and JSON output (#40
// will arrive with v1.0.0; until then this struct is the canonical shape).
type SummaryLine struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

// FormatSummary renders the collect footer as a single block of text. It takes
// the existing Summary plus the optional extras needed to count unhealthy
// pods. The extras are computed on demand by the CLI; here we just format.
func FormatSummary(s Summary, archive string, unhealthy UnhealthyPodCounts) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Summary\n")
	fmt.Fprintf(&b, "  total jobs : %d\n", s.Total)
	fmt.Fprintf(&b, "  succeeded  : %d\n", s.Success)
	fmt.Fprintf(&b, "  failed     : %d\n", s.Failed)
	fmt.Fprintf(&b, "  duration   : %s\n", s.Duration.Round(1e9))
	if s.OutputDir != "" {
		fmt.Fprintf(&b, "  output dir : %s\n", s.OutputDir)
	}
	arc := archive
	if arc == "" {
		arc = s.ArchivePath
	}
	if arc != "" {
		fmt.Fprintf(&b, "  archive    : %s\n", arc)
	}
	if len(s.Failures) > 0 {
		fmt.Fprintf(&b, "\nFailures (%d):\n", len(s.Failures))
		for _, f := range s.Failures {
			fmt.Fprintf(&b, "  - %s\n", f)
		}
	}
	// Unhealthy pods, only if a counting pass ran. We don't compute them
	// here — the CLI performs a list-pods walk and passes the result in.
	if unhealthy.HasAny() {
		fmt.Fprintf(&b, "\nUnhealthy pods:\n")
		fmt.Fprintf(&b, "  CrashLoopBackOff : %d\n", unhealthy.CrashLoopBackOff)
		fmt.Fprintf(&b, "  ImagePullBackOff : %d\n", unhealthy.ImagePullBackOff)
		fmt.Fprintf(&b, "  OOMKilled        : %d\n", unhealthy.OOMKilled)
		fmt.Fprintf(&b, "  Pending          : %d\n", unhealthy.Pending)
	}
	return b.String()
}

// WriteSummary prints the footer to w. Wraps FormatSummary for symmetry with
// other helpers that take an io.Writer in this package.
func WriteSummary(w io.Writer, s Summary, archive string, unhealthy UnhealthyPodCounts) {
	_, _ = io.WriteString(w, FormatSummary(s, archive, unhealthy))
}

// UnhealthyPodCounts groups the four common operator-relevant pod states.
// Zero on every field means "did not compute" — callers should leave the
// counters alone rather than guessing.
type UnhealthyPodCounts struct {
	CrashLoopBackOff int `json:"crash_loop_back_off"`
	ImagePullBackOff int `json:"image_pull_back_off"`
	OOMKilled        int `json:"oom_killed"`
	Pending          int `json:"pending"`
}

// HasAny reports whether any counter has a non-zero value (the CLI uses this
// to decide whether to render the unhealthy section at all).
func (u UnhealthyPodCounts) HasAny() bool {
	return u.CrashLoopBackOff != 0 || u.ImagePullBackOff != 0 ||
		u.OOMKilled != 0 || u.Pending != 0
}

// SortedKeys is exposed here so other call sites can sort the summary map
// when extended later. Currently unused but kept stable for tests.
func (u UnhealthyPodCounts) SortedKeys() []string {
	keys := []string{
		"CrashLoopBackOff", "ImagePullBackOff", "OOMKilled", "Pending",
	}
	sort.Strings(keys)
	return keys
}
