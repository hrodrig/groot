// Package collector: helpers for the --summary footer (ROADMAP #42).
//
// CountUnhealthyPods performs a single list-pods walk across the configured
// namespaces and tallies how many pods are in each of the high-signal
// unhealthy states (CrashLoopBackOff, ImagePullBackOff, OOMKilled, Pending).
//
// The walk is bounded: we request every namespace and use a single List() per
// namespace with no field-selector (the API server handles the rest). It is
// deliberately independent of the rest of the collect pipeline so the CLI can
// call it AFTER archive-write without disturbing the archived list.
package collector

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CountUnhealthyPods tallies pods across the configured namespaces. Empty
// namespaces list means "all namespaces"; otherwise only listed namespaces
// are walked.
func (s *Service) CountUnhealthyPods(ctx context.Context) (UnhealthyPodCounts, error) {
	out := UnhealthyPodCounts{}
	if err := s.initK8s(); err != nil {
		return out, fmt.Errorf("k8s init: %w", err)
	}
	if s.clientset == nil {
		return out, fmt.Errorf("clientset not initialized")
	}

	ns := s.cfg.Collection.Namespaces
	if len(ns) == 0 {
		list, err := s.clientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
		if err != nil {
			return out, fmt.Errorf("list pods (all ns): %w", err)
		}
		tally(list.Items, &out)
		return out, nil
	}

	for _, n := range ns {
		if n == "" {
			continue
		}
		list, err := s.clientset.CoreV1().Pods(n).List(ctx, metav1.ListOptions{})
		if err != nil {
			return out, fmt.Errorf("list pods (ns=%s): %w", n, err)
		}
		tally(list.Items, &out)
	}
	return out, nil
}

// tally inspects each pod's status and increments the appropriate counter.
// The function is split out so the all-ns and per-ns branches share the
// exact same logic.
func tally(pods []corev1.Pod, out *UnhealthyPodCounts) {
	for i := range pods {
		p := &pods[i]
		// Pending is a pod-level state, not a container state. We only
		// flag Pending when no containers have ever reached Running, so a
		// transient initContainer doesn't read as healthy-but-pending.
		if p.Status.Phase == corev1.PodPending {
			out.Pending++
			// Don't short-circuit — ImagePullBackOff can present as
			// Pending too and we want both tallies.
		}
		for _, cs := range p.Status.ContainerStatuses {
			if cs.State.Waiting != nil {
				switch cs.State.Waiting.Reason {
				case "CrashLoopBackOff":
					out.CrashLoopBackOff++
				case "ImagePullBackOff":
					out.ImagePullBackOff++
				}
			}
			// OOMKilled surfaces as a Terminated reason on LastTerminationState
			// when the kubelet is about to restart the container. Count it
			// regardless of the current State so the operator sees the
			// chronic-OOM signal.
			if cs.LastTerminationState.Terminated != nil &&
				cs.LastTerminationState.Terminated.Reason == "OOMKilled" {
				out.OOMKilled++
			}
		}
	}
}
