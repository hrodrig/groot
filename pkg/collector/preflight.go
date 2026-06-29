package collector

import (
	"context"
	"fmt"
	"sort"
	"strings"
)

// PreflightSeverity tags a single Preflight finding.
type PreflightSeverity string

const (
	PreflightOK    PreflightSeverity = "ok"
	PreflightWarn  PreflightSeverity = "warn"
	PreflightError PreflightSeverity = "error"
)

// PreflightFinding is a single check result. Error findings abort the run;
// warn findings log but let the user proceed; ok findings tally clean checks.
type PreflightFinding struct {
	Severity PreflightSeverity `json:"severity"`
	Check    string            `json:"check"`
	Message  string            `json:"message"`
}

// PreflightResult is the aggregate returned by Preflight. OK is true when no
// finding has severity == error. Warnings do not flip OK.
type PreflightResult struct {
	OK        bool               `json:"ok"`
	Findings  []PreflightFinding `json:"findings"`
	Cluster   string             `json:"cluster,omitempty"`
	OutputDir string             `json:"output_dir"`
}

// Default thresholds for #83 — disk preflight. Hard fail when output_dir
// cannot hold more than this many bytes after the estimate; warn when it has
// less than warnBytes free. Override via Config.Collection.MinFreeBytes for
// the hard floor and Collection.WarnFreeBytes for the warning.
const (
	defaultMinFreeBytes  int64 = 256 * 1024 * 1024      // 256 MiB
	defaultWarnFreeBytes int64 = 1 * 1024 * 1024 * 1024 // 1 GiB

	// Estimated per-job collect footprint when no estimate is available yet
	// (manifest before #70 lands). Conservative — better to over-warn than
	// to fail mid-collect when /tmp fills up.
	defaultEstimatedJobBytes int64 = 8 * 1024 * 1024 // 8 MiB
)

// Preflight performs non-destructive checks on the runtime: config validity,
// connectivity to the cluster, RBAC for the verbs the collector will use, and
// free space on output_dir. It does not write to output_dir; it only stat()s.
//
// Plan reference: ROADMAP #31 + #83 (0.9.x).
func (s *Service) Preflight(ctx context.Context) PreflightResult {
	res := PreflightResult{
		OK:        true,
		OutputDir: strings.TrimSpace(s.cfg.OutputDir),
	}
	add := func(sev PreflightSeverity, check, msg string) {
		res.Findings = append(res.Findings, PreflightFinding{Severity: sev, Check: check, Message: msg})
		if sev == PreflightError {
			res.OK = false
		}
	}

	// 1. Config shape: a minimal structural sanity check beyond what Viper
	// does during Load(). Anything more elaborate lives in PlanValidate.
	if len(s.cfg.Collection.Namespaces) == 0 {
		add(PreflightWarn, "config.namespaces", "no namespaces listed — collect will run with broad listing")
	}

	// 2. Cluster connectivity + 3. metadata (cluster name).
	if err := s.initK8s(); err != nil {
		add(PreflightError, "kubernetes.client", fmt.Sprintf("kubernetes client: %v", err))
		return res // downstream RBAC checks would just repeat the failure
	}
	if meta, err := s.ReadKubeMetadata(ctx); err != nil {
		add(PreflightError, "kubernetes.metadata", err.Error())
	} else {
		res.Cluster = meta.Cluster
	}
	if _, err := s.k8sRunner.Run(ctx, []string{"cluster-info"}); err != nil {
		add(PreflightError, "kubernetes.cluster-info", err.Error())
	} else {
		add(PreflightOK, "kubernetes.cluster-info", "cluster-info reachable")
	}

	// 4. RBAC matrix — verbs and resources the collector uses. We test
	// SelfSubjectAccessReview against the first namespace the user listed
	// (or "" for cluster-scoped verbs). Failures show the user which verb
	// is missing.
	tasks := rbacChecks(s.cfg.Collection.Namespaces)
	for _, c := range tasks {
		s.preflightRBAC(ctx, &res, c)
	}

	// 5. Disk preflight (#83). Never writes; only statvfs or fallback.
	s.preflightDisk(&res)

	// Stable order for tests + logs.
	sort.SliceStable(res.Findings, func(i, j int) bool {
		return findingsLess(res.Findings[i], res.Findings[j])
	})
	return res
}

type rbacCheck struct {
	label     string
	verb      string
	resource  string
	namespace string
}

// rbacChecks returns the RBAC matrix Preflight exercises. Namespaced checks
// pin to the first user-supplied namespace to keep noise low; cluster-scoped
// checks pass an empty namespace.
func rbacChecks(namespaces []string) []rbacCheck {
	firstNS := ""
	if len(namespaces) > 0 {
		firstNS = strings.TrimSpace(namespaces[0])
	}
	return []rbacCheck{
		{label: "list.namespaces", verb: "list", resource: "namespaces"},
		{label: "list.pods", verb: "list", resource: "pods", namespace: firstNS},
		{label: "get.pods.log", verb: "get", resource: "pods/log", namespace: firstNS},
		{label: "list.events", verb: "list", resource: "events", namespace: firstNS},
		{label: "list.nodes", verb: "list", resource: "nodes"},
	}
}

func formatBytes(b int64) string {
	const (
		KiB = 1024
		MiB = 1024 * 1024
		GiB = 1024 * 1024 * 1024
	)
	switch {
	case b >= GiB:
		return fmt.Sprintf("%.1fGiB", float64(b)/float64(GiB))
	case b >= MiB:
		return fmt.Sprintf("%.1fMiB", float64(b)/float64(MiB))
	case b >= KiB:
		return fmt.Sprintf("%.1fKiB", float64(b)/float64(KiB))
	default:
		return fmt.Sprintf("%dB", b)
	}
}

// findingsLess orders findings by severity then check name. Pure helper so
// tests can reason about ordering deterministically.
func findingsLess(a, b PreflightFinding) bool {
	rank := map[PreflightSeverity]int{
		PreflightError: 0,
		PreflightWarn:  1,
		PreflightOK:    2,
	}
	ra, rb := rank[a.Severity], rank[b.Severity]
	if ra != rb {
		return ra < rb
	}
	return a.Check < b.Check
}

// max is a tiny shim because Go 1.20+ added built-in max but we keep it
// explicit so the code stays readable for readers on older toolchains.
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// preflightRBAC runs a single SelfSubjectAccessReview and records the finding.
func (s *Service) preflightRBAC(ctx context.Context, res *PreflightResult, c rbacCheck) {
	add := func(sev PreflightSeverity, check, msg string) {
		res.Findings = append(res.Findings, PreflightFinding{Severity: sev, Check: check, Message: msg})
		if sev == PreflightError {
			res.OK = false
		}
	}
	verb, resource := c.verb, c.resource
	ns := c.namespace
	argv := []string{"auth", "can-i", verb, resource}
	if ns != "" {
		argv = append(argv, "--namespace", ns)
	}
	out, err := s.k8sRunner.Run(ctx, argv)
	switch {
	case err != nil:
		add(PreflightError, "rbac."+c.label, err.Error())
	case strings.TrimSpace(string(out)) == "yes":
		add(PreflightOK, "rbac."+c.label, fmt.Sprintf("can-i %s %s (ns=%q)", verb, resource, ns))
	default:
		add(PreflightError, "rbac."+c.label, fmt.Sprintf("missing permission: %s %s (ns=%q)", verb, resource, ns))
	}
}

// preflightDisk runs a one-step statvfs check and records the finding.
func (s *Service) preflightDisk(res *PreflightResult) {
	add := func(sev PreflightSeverity, check, msg string) {
		res.Findings = append(res.Findings, PreflightFinding{Severity: sev, Check: check, Message: msg})
		if sev == PreflightError {
			res.OK = false
		}
	}
	free, total, derr := diskFree(s.cfg.OutputDir)
	if derr != nil {
		add(PreflightError, "disk.stat", derr.Error())
		return
	}
	minBytes := s.cfg.Collection.MinFreeBytes
	if minBytes <= 0 {
		minBytes = defaultMinFreeBytes
	}
	warnBytes := s.cfg.Collection.WarnFreeBytes
	if warnBytes <= 0 {
		warnBytes = defaultWarnFreeBytes
	}
	estimate := defaultEstimatedJobBytes * int64(max(len(s.cfg.Collection.Namespaces), 1))
	switch {
	case free < minBytes:
		add(PreflightError, "disk.free",
			fmt.Sprintf("output_dir %s has %s free; need at least %s (estimated collect footprint %s)",
				s.cfg.OutputDir, formatBytes(free), formatBytes(minBytes), formatBytes(estimate)))
	case free < warnBytes:
		add(PreflightWarn, "disk.free",
			fmt.Sprintf("output_dir %s has %s free; warn threshold %s — collect may fill the volume",
				s.cfg.OutputDir, formatBytes(free), formatBytes(warnBytes)))
	default:
		add(PreflightOK, "disk.free",
			fmt.Sprintf("output_dir %s free=%s of %s", s.cfg.OutputDir, formatBytes(free), formatBytes(total)))
	}
}
