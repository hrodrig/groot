package collector

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode"

	"golang.org/x/text/unicode/norm"
	"groot/pkg/archive"
	"groot/pkg/config"
)

// Summary reports collection execution details.
type Summary struct {
	OutputDir   string
	ArchivePath string
	Duration    time.Duration
	Total       int
	Success     int
	Failed      int
	Failures    []string
}

type job struct {
	Name     string
	Args     []string
	FileName string
	Optional bool
}

// Service executes kubectl log collection.
type Service struct {
	cfg      config.Config
	message  string
	onStart  func(name string, args []string)
	onDone   func(name string)
	onFailed func(name string, err error)
}

// New returns a collector service.
func New(cfg config.Config) *Service {
	return &Service{cfg: cfg}
}

// SetMessage adds a custom suffix for output names.
func (s *Service) SetMessage(message string) {
	s.message = message
}

// SetHooks attaches command execution hooks.
func (s *Service) SetHooks(
	onStart func(name string, args []string),
	onDone func(name string),
	onFailed func(name string, err error),
) {
	s.onStart = onStart
	s.onDone = onDone
	s.onFailed = onFailed
}

// Run executes the collection workflow.
func (s *Service) Run(ctx context.Context) (Summary, error) {
	start := time.Now()

	timestamp := time.Now().Format("20060102-150405")
	captureDir := filepath.Join(s.cfg.OutputDir, timestamp)
	if err := os.MkdirAll(captureDir, 0o755); err != nil {
		return Summary{}, fmt.Errorf("create output dir: %w", err)
	}
	if err := s.ensureGroupDirs(captureDir); err != nil {
		return Summary{}, fmt.Errorf("prepare output groups: %w", err)
	}
	if err := s.writeMetadata(ctx, captureDir); err != nil && s.onFailed != nil {
		s.onFailed("metadata", err)
	}

	jobs, err := s.buildJobs(ctx)
	if err != nil {
		return Summary{}, fmt.Errorf("build jobs: %w", err)
	}

	summary := s.runJobs(ctx, captureDir, jobs)
	summary.OutputDir = captureDir
	summary.Duration = time.Since(start)

	clusterName := "unknown-cluster"
	if meta, err := s.ReadKubeMetadata(ctx); err == nil {
		if strings.TrimSpace(meta.Cluster) != "" {
			clusterName = sanitize(meta.Cluster)
		}
	} else if s.onFailed != nil {
		s.onFailed("cluster-name", err)
	}

	archiveName := fmt.Sprintf("%s-%s", timestamp, clusterName)
	if suffix := sanitizeMessage(s.message); suffix != "" {
		archiveName += "-" + suffix
	}
	archivePath := filepath.Join(s.cfg.OutputDir, archiveName+".tar.gz")
	if err := archive.DirToTarGz(captureDir, archivePath); err != nil {
		return Summary{}, fmt.Errorf("archive logs: %w", err)
	}
	if err := os.RemoveAll(captureDir); err != nil {
		return Summary{}, fmt.Errorf("cleanup capture dir %s: %w", captureDir, err)
	}
	summary.ArchivePath = archivePath

	return summary, nil
}

func (s *Service) buildJobs(ctx context.Context) ([]job, error) {
	jobs := s.baseKubectlJobs()
	jobs = s.appendExtraKubectlJobs(jobs)

	var err error
	jobs, err = s.appendNodeDetailJobs(ctx, jobs)
	if err != nil {
		return nil, err
	}

	jobs = s.appendNamespaceResourceJobs(jobs)

	jobs, err = s.appendWorkloadPodLogJobs(ctx, jobs)
	if err != nil {
		return nil, err
	}

	jobs = s.appendControlPlanePodLogJobs(ctx, jobs)
	return jobs, nil
}

func (s *Service) baseKubectlJobs() []job {
	return []job{
		{Name: "cluster-info", Args: []string{"cluster-info"}, FileName: filepath.Join("extras", "cluster-info.txt")},
		{Name: "nodes-wide", Args: []string{"get", "nodes", "-o", "wide"}, FileName: filepath.Join("extras", "nodes-wide.txt")},
		{Name: "events-all", Args: []string{"get", "events", "-A", "--sort-by=.lastTimestamp"}, FileName: filepath.Join("extras", "all-cluster-events.log")},
		{Name: "pods-all", Args: []string{"get", "pods", "-A", "-o", "wide"}, FileName: filepath.Join("extras", "all-pods-wide.txt")},
	}
}

func (s *Service) appendExtraKubectlJobs(jobs []job) []job {
	for i, cmd := range s.cfg.Collection.ExtraKubectl {
		parts := strings.Fields(cmd)
		if len(parts) == 0 {
			continue
		}
		jobs = append(jobs, job{
			Name:     fmt.Sprintf("extra-%d", i+1),
			Args:     parts,
			FileName: fmt.Sprintf("extra-%d.txt", i+1),
		})
	}
	return jobs
}

func (s *Service) appendNodeDetailJobs(ctx context.Context, jobs []job) ([]job, error) {
	if !s.cfg.Collection.IncludeNodeDetails {
		return jobs, nil
	}
	nodes, err := s.list(ctx, []string{"get", "nodes", "-o", "name"})
	if err != nil {
		return nil, fmt.Errorf("list nodes: %w", err)
	}
	for _, node := range nodes {
		nodeSafe := sanitize(node)
		nodeName := strings.TrimPrefix(node, "node/")
		jobs = append(jobs,
			job{Name: "describe-" + nodeSafe, Args: []string{"describe", node}, FileName: filepath.Join("nodes", nodeSafe+"-describe.txt")},
			job{Name: "top-" + nodeSafe, Args: []string{"top", "node", nodeName}, FileName: filepath.Join("nodes", nodeSafe+"-top.txt")},
		)
	}
	return jobs, nil
}

func (s *Service) appendNamespaceResourceJobs(jobs []job) []job {
	for _, ns := range s.cfg.Collection.Namespaces {
		jobs = append(jobs, job{
			Name:     "namespace-" + ns,
			Args:     []string{"get", "all", "-n", ns, "-o", "wide"},
			FileName: filepath.Join(ns, "resources.txt"),
		})
	}
	return jobs
}

func workloadPodLogArgs(ns, name string, previous bool, tail int) []string {
	args := []string{"logs", "-n", ns, name, "--all-containers=true", "--timestamps=true"}
	if previous {
		args = append(args, "--previous")
	}
	if tail > 0 {
		args = append(args, "--tail", fmt.Sprintf("%d", tail))
	}
	return args
}

func controlPlanePodLogArgs(pod string, previous bool, tail int) []string {
	args := []string{"logs", "-n", "kube-system", pod, "--timestamps=true"}
	if previous {
		args = append(args, "--previous")
	}
	if tail > 0 {
		args = append(args, "--tail", fmt.Sprintf("%d", tail))
	}
	return args
}

func (s *Service) appendWorkloadPodLogJobs(ctx context.Context, jobs []job) ([]job, error) {
	if !s.cfg.Collection.IncludePodLogs {
		return jobs, nil
	}
	pods, err := s.resolvePodsForLogs(ctx)
	if err != nil {
		return nil, fmt.Errorf("list pods: %w", err)
	}
	tail := s.cfg.Collection.PodLogTailLines
	for _, item := range pods {
		baseName := sanitize(item.Name) + "__" + sanitize(item.Node)
		jobs = append(jobs, job{
			Name:     fmt.Sprintf("pod-log-%s-%s", item.Namespace, item.Name),
			Args:     workloadPodLogArgs(item.Namespace, item.Name, false, tail),
			FileName: filepath.Join(item.Namespace, baseName+".log"),
		})
		if s.cfg.Collection.IncludePreviousLogs {
			jobs = append(jobs, job{
				Name:     fmt.Sprintf("pod-log-previous-%s-%s", item.Namespace, item.Name),
				Args:     workloadPodLogArgs(item.Namespace, item.Name, true, tail),
				FileName: filepath.Join(item.Namespace, baseName+".previous.log"),
				Optional: true,
			})
		}
	}
	return jobs, nil
}

func (s *Service) appendControlPlanePodLogJobs(ctx context.Context, jobs []job) []job {
	lines, _ := s.list(ctx, []string{"get", "pods", "-n", "kube-system", "-l", "tier=control-plane", "--no-headers", "-o", "custom-columns=NAME:.metadata.name,NODE:.spec.nodeName"})
	tail := s.cfg.Collection.PodLogTailLines
	for _, item := range parseNameNodeLines(lines) {
		if item.Name == "" {
			continue
		}
		node := item.Node
		if node == "" {
			node = "unknown-node"
		}
		baseName := sanitize(item.Name) + "__" + sanitize(node)
		jobs = append(jobs, job{
			Name:     "control-plane-" + item.Name,
			Args:     controlPlanePodLogArgs(item.Name, false, tail),
			FileName: filepath.Join("kube-system", baseName+".log"),
		})
		if s.cfg.Collection.IncludePreviousLogs {
			jobs = append(jobs, job{
				Name:     "control-plane-previous-" + item.Name,
				Args:     controlPlanePodLogArgs(item.Name, true, tail),
				FileName: filepath.Join("kube-system", baseName+".previous.log"),
				Optional: true,
			})
		}
	}
	return jobs
}

func (s *Service) runJobs(ctx context.Context, captureDir string, jobs []job) Summary {
	var (
		wg       sync.WaitGroup
		jobCh    = make(chan job)
		resCh    = make(chan error, len(jobs))
		failMu   sync.Mutex
		failures []string
	)

	for i := 0; i < s.cfg.Collection.WorkerConcurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range jobCh {
				err := s.execToFile(ctx, captureDir, j)
				if err != nil {
					failMu.Lock()
					failures = append(failures, fmt.Sprintf("%s: %v", j.Name, err))
					failMu.Unlock()
				}
				resCh <- err
			}
		}()
	}

	for _, j := range jobs {
		jobCh <- j
	}
	close(jobCh)
	wg.Wait()
	close(resCh)

	total := len(jobs)
	failed := 0
	for err := range resCh {
		if err != nil {
			failed++
		}
	}

	return Summary{
		Total:    total,
		Success:  total - failed,
		Failed:   failed,
		Failures: failures,
	}
}

func (s *Service) execToFile(ctx context.Context, captureDir string, j job) error {
	if s.onStart != nil {
		s.onStart(j.Name, j.Args)
	}

	cmd := exec.CommandContext(ctx, "kubectl", s.kubectlArgs(j.Args)...)
	var out bytes.Buffer
	var stdErr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stdErr

	err := cmd.Run()
	content := out.String()
	if stdErr.Len() > 0 {
		content = content + "\n--- stderr ---\n" + stdErr.String()
	}

	target := filepath.Join(captureDir, j.FileName)
	if mkErr := os.MkdirAll(filepath.Dir(target), 0o755); mkErr != nil {
		return fmt.Errorf("create dir for %s: %w", target, mkErr)
	}

	if writeErr := os.WriteFile(target, []byte(content), 0o644); writeErr != nil {
		return fmt.Errorf("write %s: %w", target, writeErr)
	}

	if err != nil {
		if j.Optional {
			return nil
		}
		if s.onFailed != nil {
			s.onFailed(j.Name, err)
		}
		return fmt.Errorf("kubectl %s: %w", strings.Join(j.Args, " "), err)
	}

	if s.onDone != nil {
		s.onDone(j.Name)
	}
	return nil
}

func (s *Service) list(ctx context.Context, args []string) ([]string, error) {
	cmd := exec.CommandContext(ctx, "kubectl", s.kubectlArgs(args)...)
	b, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	rows := strings.Split(strings.TrimSpace(string(b)), "\n")
	clean := make([]string, 0, len(rows))
	for _, row := range rows {
		row = strings.TrimSpace(row)
		if row != "" {
			clean = append(clean, row)
		}
	}
	return clean, nil
}

func (s *Service) kubectlArgs(args []string) []string {
	if s.cfg.Kubeconfig == "" {
		return args
	}
	return append([]string{"--kubeconfig", s.cfg.Kubeconfig}, args...)
}

func sanitize(value string) string {
	value = strings.ReplaceAll(value, "/", "_")
	value = strings.ReplaceAll(value, ":", "_")
	value = strings.ReplaceAll(value, " ", "_")
	return value
}

func sanitizeMessage(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}

	normalized := norm.NFD.String(strings.ToLower(trimmed))
	var b strings.Builder
	for _, r := range normalized {
		if unicode.Is(unicode.Mn, r) {
			continue
		}
		b.WriteRune(r)
	}

	clean := b.String()
	clean = strings.ReplaceAll(clean, "_", "-")
	clean = strings.ReplaceAll(clean, " ", "-")

	nonSafe := regexp.MustCompile(`[^a-z0-9.-]+`)
	clean = nonSafe.ReplaceAllString(clean, "-")
	dashes := regexp.MustCompile(`-+`)
	clean = dashes.ReplaceAllString(clean, "-")
	clean = strings.Trim(clean, "-.")
	return clean
}

func (s *Service) writeMetadata(ctx context.Context, captureDir string) error {
	meta, err := s.ReadKubeMetadata(ctx)
	if err != nil {
		return err
	}

	content := fmt.Sprintf(
		"context: %s\ncluster: %s\nuser: %s\nserver: %s\n",
		emptyAsUnknown(meta.Context),
		emptyAsUnknown(meta.Cluster),
		emptyAsUnknown(meta.User),
		emptyAsUnknown(meta.Server),
	)

	target := filepath.Join(captureDir, "extras", "kubeconfig.txt")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return fmt.Errorf("create metadata dir: %w", err)
	}
	if err := os.WriteFile(target, []byte(content), 0o644); err != nil {
		return fmt.Errorf("write metadata: %w", err)
	}
	return nil
}

func emptyAsUnknown(value string) string {
	if strings.TrimSpace(value) == "" {
		return "unknown"
	}
	return value
}

func (s *Service) ensureGroupDirs(captureDir string) error {
	dirs := []string{
		filepath.Join(captureDir, "nodes"),
		filepath.Join(captureDir, "extras"),
	}

	for _, ns := range s.cfg.Collection.Namespaces {
		trimmed := strings.TrimSpace(ns)
		if trimmed == "" {
			continue
		}
		dirs = append(dirs, filepath.Join(captureDir, trimmed))
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	return nil
}

type podRef struct {
	Namespace string
	Name      string
	Node      string
}

type podsList struct {
	Items []struct {
		Metadata struct {
			Name      string            `json:"name"`
			Namespace string            `json:"namespace"`
			Labels    map[string]string `json:"labels"`
		} `json:"metadata"`
		Spec struct {
			NodeName string `json:"nodeName"`
		} `json:"spec"`
	} `json:"items"`
}

func (s *Service) resolvePodsForLogs(ctx context.Context) ([]podRef, error) {
	raw, err := s.list(ctx, []string{"get", "pods", "-A", "--no-headers", "-o", "custom-columns=NS:.metadata.namespace,NAME:.metadata.name,NODE:.spec.nodeName"})
	if err != nil {
		return nil, err
	}

	if len(s.cfg.Collection.Targets) == 0 {
		return parsePodLines(raw), nil
	}

	filtered := make([]podRef, 0)
	allPods, err := s.listPodsJSON(ctx)
	if err != nil {
		return nil, err
	}
	seen := map[string]struct{}{}

	for _, pod := range allPods.Items {
		ns := strings.TrimSpace(pod.Metadata.Namespace)
		targets, ok := s.cfg.Collection.Targets[ns]
		if !ok || !hasTargets(targets) {
			continue
		}
		if !matchesTargetsByLabels(pod.Metadata.Labels, targets) {
			continue
		}

		node := strings.TrimSpace(pod.Spec.NodeName)
		if node == "" {
			node = "unknown-node"
		}
		ref := podRef{
			Namespace: ns,
			Name:      strings.TrimSpace(pod.Metadata.Name),
			Node:      node,
		}
		key := ref.Namespace + "/" + ref.Name
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		filtered = append(filtered, ref)
	}

	// Fallback behavior: namespaces without explicit targets keep broad collection.
	for _, ref := range parsePodLines(raw) {
		targets, ok := s.cfg.Collection.Targets[ref.Namespace]
		if ok && hasTargets(targets) {
			continue
		}
		key := ref.Namespace + "/" + ref.Name
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		filtered = append(filtered, ref)
	}

	return filtered, nil
}

func parsePodLines(lines []string) []podRef {
	out := make([]podRef, 0, len(lines))
	for _, item := range lines {
		parts := strings.Fields(item)
		if len(parts) < 2 {
			continue
		}
		node := "unknown-node"
		if len(parts) >= 3 && strings.TrimSpace(parts[2]) != "" {
			node = parts[2]
		}
		out = append(out, podRef{
			Namespace: parts[0],
			Name:      parts[1],
			Node:      node,
		})
	}
	return out
}

// parseNameNodeLines parses kubectl NAME,NODE custom-columns rows (first field pod name, rest node).
func parseNameNodeLines(lines []string) []podRef {
	out := make([]podRef, 0, len(lines))
	for _, item := range lines {
		parts := strings.Fields(item)
		if len(parts) < 1 {
			continue
		}
		name := strings.TrimSpace(parts[0])
		if name == "" {
			continue
		}
		node := "unknown-node"
		if len(parts) >= 2 {
			node = strings.TrimSpace(strings.Join(parts[1:], " "))
			if node == "" {
				node = "unknown-node"
			}
		}
		out = append(out, podRef{Name: name, Node: node})
	}
	return out
}

func (s *Service) listPodsJSON(ctx context.Context) (podsList, error) {
	cmd := exec.CommandContext(ctx, "kubectl", s.kubectlArgs([]string{"get", "pods", "-A", "-o", "json"})...)
	data, err := cmd.Output()
	if err != nil {
		return podsList{}, err
	}
	var parsed podsList
	if err := json.Unmarshal(data, &parsed); err != nil {
		return podsList{}, err
	}
	return parsed, nil
}

func hasTargets(t config.NamespaceTargets) bool {
	return len(t.Deployments) > 0 || len(t.StatefulSets) > 0 || len(t.DaemonSets) > 0 || len(t.HelmReleases) > 0
}

func matchesTargetsByLabels(labels map[string]string, targets config.NamespaceTargets) bool {
	if len(labels) == 0 {
		return false
	}

	nameLabel := strings.TrimSpace(labels["app.kubernetes.io/name"])
	instanceLabel := strings.TrimSpace(labels["app.kubernetes.io/instance"])

	if contains(targets.HelmReleases, instanceLabel) {
		return true
	}

	if contains(targets.Deployments, nameLabel) || contains(targets.Deployments, instanceLabel) {
		return true
	}
	if contains(targets.StatefulSets, nameLabel) || contains(targets.StatefulSets, instanceLabel) {
		return true
	}
	if contains(targets.DaemonSets, nameLabel) || contains(targets.DaemonSets, instanceLabel) {
		return true
	}

	// Fallback for charts that still use "app" label.
	appLabel := strings.TrimSpace(labels["app"])
	if contains(targets.Deployments, appLabel) ||
		contains(targets.StatefulSets, appLabel) ||
		contains(targets.DaemonSets, appLabel) {
		return true
	}
	return false
}

func contains(items []string, target string) bool {
	for _, item := range items {
		if strings.TrimSpace(item) == target {
			return true
		}
	}
	return false
}
