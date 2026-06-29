package collector

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/hrodrig/groot/pkg/archive"
	"github.com/hrodrig/groot/pkg/config"
	"github.com/hrodrig/groot/pkg/k8srunner"
	"golang.org/x/text/unicode/norm"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	metricsversioned "k8s.io/metrics/pkg/client/clientset/versioned"
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

// Service executes Kubernetes API collection (client-go; no kubectl binary).
type Service struct {
	cfg             config.Config
	message         string
	buildInfo       BuildInfo
	highSignalFirst bool
	RunID           string // ROADMAP #81 — stable per-run id, set on Run
	archiveSHA256   string // ROADMAP #81 — populated after archive writer returns
	clientset       kubernetes.Interface
	metricsCS       *metricsversioned.Clientset
	restConfig      *rest.Config
	k8sRunner       *k8srunner.Runner
	onStart         func(name string, args []string)
	onDone          func(name string)
	onFailed        func(name string, err error)
	hooksMu         sync.Mutex // serializes hook callbacks; workers invoke hooks concurrently otherwise
}

// New returns a collector service. Signal-first ordering (ROADMAP #84) is
// enabled by default — pass SetHighSignalFirst(false) to disable.
func New(cfg config.Config) *Service {
	return &Service{cfg: cfg, highSignalFirst: true}
}

// SetMessage adds a custom suffix for output names.
func (s *Service) SetMessage(message string) {
	s.message = message
}

// SetHighSignalFirst (ROADMAP #84 — 0.9.x) reorders the buildJobs output so
// that "signal" jobs (Warning+ events, etc.) run before bulk namespace logs.
// Default is on; pass false to restore the historical order for repro tests.
func (s *Service) SetHighSignalFirst(enabled bool) {
	s.highSignalFirst = enabled
}

// SetHooks attaches command execution hooks. Callbacks run under an internal
// mutex so they are not invoked concurrently from collection workers.
func (s *Service) SetHooks(
	onStart func(name string, args []string),
	onDone func(name string),
	onFailed func(name string, err error),
) {
	s.hooksMu.Lock()
	defer s.hooksMu.Unlock()
	s.onStart = onStart
	s.onDone = onDone
	s.onFailed = onFailed
}

func (s *Service) invokeOnStart(name string, args []string) {
	s.hooksMu.Lock()
	defer s.hooksMu.Unlock()
	if s.onStart != nil {
		s.onStart(name, args)
	}
}

func (s *Service) invokeOnDone(name string) {
	s.hooksMu.Lock()
	defer s.hooksMu.Unlock()
	if s.onDone != nil {
		s.onDone(name)
	}
}

func (s *Service) invokeOnFailed(name string, err error) {
	s.hooksMu.Lock()
	defer s.hooksMu.Unlock()
	if s.onFailed != nil {
		s.onFailed(name, err)
	}
}

// Run executes the collection workflow.
func (s *Service) Run(ctx context.Context) (Summary, error) {
	start := time.Now()

	// ROADMAP #81: a stable per-run id is generated once at the start of Run
	// and threaded through every downstream artifact (manifest, notify
	// metadata, upload metadata). Format is YYYYMMDDTHHMMSSZ-<short> where
	// <short> is base32(crypto/rand first 4 bytes).
	s.RunID = newRunID()

	timestamp := time.Now().Format("20060102-150405")
	sessionBase := captureSessionBase(s.cfg.FilePrefix, timestamp, s.cfg.Collection.PodLogsSince)
	captureDir := filepath.Join(s.cfg.OutputDir, sessionBase)
	if err := os.MkdirAll(captureDir, 0o755); err != nil {
		return Summary{}, fmt.Errorf("create output dir: %w", err)
	}
	if err := s.ensureGroupDirs(captureDir); err != nil {
		return Summary{}, fmt.Errorf("prepare output groups: %w", err)
	}
	if err := s.initK8s(); err != nil {
		return Summary{}, fmt.Errorf("kubernetes client: %w", err)
	}
	if err := s.writeMetadata(ctx, captureDir); err != nil {
		s.invokeOnFailed("metadata", err)
	}
	if err := s.writePodNodePlacement(ctx, captureDir); err != nil {
		s.invokeOnFailed("pod-node-placement", err)
	}

	jobs, err := s.buildJobs(ctx)
	if err != nil {
		return Summary{}, fmt.Errorf("build jobs: %w", err)
	}

	summary := s.runJobs(ctx, captureDir, jobs)
	summary.OutputDir = captureDir
	summary.Duration = time.Since(start)

	if s.cfg.Collection.RedactSecrets {
		s.runRedactPass(captureDir)
	}

	podResources, err := s.writeWorkloadResourcesTable(ctx, captureDir)
	if err != nil {
		s.invokeOnFailed("workload-resources", err)
	}
	if err := s.writePodRCATable(captureDir, podResources); err != nil {
		s.invokeOnFailed("pod-rca-table", err)
	}

	clusterName := s.resolveClusterName(ctx)
	archiveName := archiveBasename(sessionBase, clusterName, s.message)

	if err := s.writeManifest(ctx, captureDir, sessionBase, archiveName, summary); err != nil {
		s.invokeOnFailed("manifest", err)
	}

	archivePath := filepath.Join(s.cfg.OutputDir, archiveName+".tar.gz")
	if err := archive.DirToTarGz(captureDir, archivePath); err != nil {
		summary.ArchivePath = archivePath
		return summary, fmt.Errorf("archive logs: %w", err)
	}
	// Compute the SHA-256 over the freshly built archive so operators can
	// pin the exact bytes if they keep the archive around (ROADMAP #81).
	if sha, shaErr := fileSHA256(archivePath); shaErr == nil {
		s.archiveSHA256 = sha
		// Re-emit the manifest so it carries the checksum. Best-effort;
		// the archive itself is the source of truth.
		if mErr := s.writeManifest(ctx, captureDir, sessionBase, archiveName, summary); mErr != nil {
			s.invokeOnFailed("manifest-with-sha", mErr)
		}
	} else {
		s.invokeOnFailed("archive-sha256", shaErr)
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

	jobs, err = s.appendNodeLogJobs(ctx, jobs)
	if err != nil {
		return nil, err
	}

	jobs = s.appendNamespaceResourceJobs(jobs)

	jobs, err = s.appendWorkloadPodLogJobs(ctx, jobs)
	if err != nil {
		return nil, err
	}

	jobs = s.appendControlPlanePodLogJobs(ctx, jobs)
	if s.highSignalFirst {
		jobs = prioritizeSignalJobs(jobs)
	}
	return jobs, nil
}

// prioritizeSignalJobs reorders jobs so that "signal" jobs (Warning+ events,
// unhealthy pod listings, cluster metadata) run before bulk namespace and pod
// log jobs. The relative order among signal jobs and among bulk jobs is
// preserved.
//
// ROADMAP #84 (0.9.x). The intent is operator-experience: surface actionable
// signal in seconds even when pod log collection takes minutes.
func prioritizeSignalJobs(in []job) []job {
	signalNames := map[string]struct{}{
		"events-warning": {},
		"events-all":     {}, // all-cluster events are high-signal enough to lead
		"cluster-info":   {},
		"nodes-wide":     {},
		"pods-all":       {},
	}
	signal := make([]job, 0, len(signalNames))
	bulk := make([]job, 0, len(in))
	for _, j := range in {
		if _, ok := signalNames[j.Name]; ok {
			signal = append(signal, j)
			continue
		}
		bulk = append(bulk, j)
	}
	out := make([]job, 0, len(in))
	out = append(out, signal...)
	out = append(out, bulk...)
	return out
}

func (s *Service) baseKubectlJobs() []job {
	jobs := []job{
		{Name: "cluster-info", Args: []string{"cluster-info"}, FileName: filepath.Join("extras", "cluster-info.txt")},
		{Name: "nodes-wide", Args: []string{"get", "nodes", "-o", "wide"}, FileName: filepath.Join("extras", "nodes-wide.txt")},
		{Name: "events-all", Args: []string{"get", "events", "-A", "--sort-by=.lastTimestamp"}, FileName: filepath.Join("extras", "all-cluster-events.log")},
		// Signal-first slice (ROADMAP #84). Cheap to run, surfaces operator
		// signal in seconds even on large clusters.
		{Name: "events-warning", Optional: true, Args: []string{"get", "events", "-A", "--field-selector", "type=Warning", "--sort-by=.lastTimestamp"}, FileName: filepath.Join("extras", "warning-events.log")},
		{Name: "pods-all", Args: []string{"get", "pods", "-A", "-o", "wide"}, FileName: filepath.Join("extras", "all-pods-wide.txt")},
	}
	if s.cfg.Collection.IncludePodMetrics {
		jobs = append(jobs, job{
			Name:     "pods-top-all",
			Args:     []string{"top", "pods", "-A"},
			FileName: filepath.Join("extras", "all-pods-top.txt"),
		})
	}
	return jobs
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
	nodes, err := s.listNodesAsResources(ctx)
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

// kubeletLogQueryRawPath is the apiserver raw URL for kubelet logs via the node log query API
// (Kubernetes 1.27+; kubelet must expose log query — see README).
func kubeletLogQueryRawPath(nodeName string, tailLines int) string {
	q := url.Values{}
	q.Set("query", "kubelet")
	if tailLines > 0 {
		q.Set("tailLines", strconv.Itoa(tailLines))
	}
	return fmt.Sprintf("/api/v1/nodes/%s/proxy/logs/?%s", url.PathEscape(nodeName), q.Encode())
}

// nodeVarLogMessagesRawPath matches kubectl get --raw …/proxy/logs/messages (host /var/log/messages
// when the kubelet exposes it on the node).
func nodeVarLogMessagesRawPath(nodeName string) string {
	return fmt.Sprintf("/api/v1/nodes/%s/proxy/logs/messages", url.PathEscape(nodeName))
}

// appendNodeLogJobs captures per-node host logs. Primary path: GET …/proxy/logs/messages
// → nodes/<node>.log (common on managed clouds such as AKS). Kubelet log query (Node Log
// Query API, 1.27+) is best-effort when the cluster exposes it.
func (s *Service) appendNodeLogJobs(ctx context.Context, jobs []job) ([]job, error) {
	if !s.cfg.Collection.IncludeNodeLogs {
		return jobs, nil
	}
	nodes, err := s.listNodesAsResources(ctx)
	if err != nil {
		return nil, fmt.Errorf("list nodes for node logs: %w", err)
	}
	tail := s.cfg.Collection.NodeLogTailLines
	for _, node := range nodes {
		nodeSafe := sanitize(node)
		nodeName := strings.TrimPrefix(node, "node/")
		jobs = append(jobs, job{
			Name:     "node-log-" + nodeSafe,
			Args:     []string{"get", "--raw", nodeVarLogMessagesRawPath(nodeName)},
			FileName: filepath.Join("nodes", nodeSafe+".log"),
			Optional: true,
		})
		raw := kubeletLogQueryRawPath(nodeName, tail)
		jobs = append(jobs, job{
			Name:     "node-log-kubelet-" + nodeSafe,
			Args:     []string{"get", "--raw", raw},
			FileName: filepath.Join("nodes", nodeSafe+"-kubelet.log"),
			Optional: true,
		})
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

func workloadPodLogArgs(ns, name string, previous bool, tail int, since string) []string {
	args := []string{"logs", "-n", ns, name, "--all-containers=true", "--timestamps=true"}
	if previous {
		args = append(args, "--previous")
	}
	if since != "" {
		args = append(args, "--since="+since)
	}
	if tail > 0 {
		args = append(args, "--tail", fmt.Sprintf("%d", tail))
	}
	return args
}

func controlPlanePodLogArgs(pod string, previous bool, tail int, since string) []string {
	args := []string{"logs", "-n", "kube-system", pod, "--timestamps=true"}
	if previous {
		args = append(args, "--previous")
	}
	if since != "" {
		args = append(args, "--since="+since)
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
	since := s.cfg.Collection.PodLogsSince
	for _, item := range pods {
		baseName := sanitize(item.Name) + "__" + sanitize(item.Node)
		jobs = append(jobs, job{
			Name:     fmt.Sprintf("pod-log-%s-%s", item.Namespace, item.Name),
			Args:     workloadPodLogArgs(item.Namespace, item.Name, false, tail, since),
			FileName: filepath.Join(item.Namespace, baseName+".log"),
		})
		if s.cfg.Collection.IncludePreviousLogs {
			jobs = append(jobs, job{
				Name:     fmt.Sprintf("pod-log-previous-%s-%s", item.Namespace, item.Name),
				Args:     workloadPodLogArgs(item.Namespace, item.Name, true, tail, since),
				FileName: filepath.Join(item.Namespace, baseName+".previous.log"),
				Optional: true,
			})
		}
	}
	return jobs, nil
}

func (s *Service) appendControlPlanePodLogJobs(ctx context.Context, jobs []job) []job {
	refs, _ := s.listControlPlanePods(ctx)
	tail := s.cfg.Collection.PodLogTailLines
	since := s.cfg.Collection.PodLogsSince
	for _, item := range refs {
		if item.Name == "" {
			continue
		}
		node := item.Node
		baseName := sanitize(item.Name) + "__" + sanitize(node)
		jobs = append(jobs, job{
			Name:     "control-plane-" + item.Name,
			Args:     controlPlanePodLogArgs(item.Name, false, tail, since),
			FileName: filepath.Join("kube-system", baseName+".log"),
		})
		if s.cfg.Collection.IncludePreviousLogs {
			jobs = append(jobs, job{
				Name:     "control-plane-previous-" + item.Name,
				Args:     controlPlanePodLogArgs(item.Name, true, tail, since),
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

// captureSessionBase is the capture folder name and the leading part of the archive basename.
// Format: "<file_prefix>-<timestamp>" or "<file_prefix>-<timestamp>-since-<slug>" when pod_logs_since is set.
func captureSessionBase(filePrefix, timestamp, podLogsSince string) string {
	base := sanitizeFilePrefix(filePrefix) + "-" + timestamp
	s := strings.TrimSpace(podLogsSince)
	if s == "" {
		return base
	}
	slug := sanitizeMessage(s)
	if slug == "" {
		return base
	}
	return base + "-since-" + slug
}

func sanitizeFilePrefix(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		p = "groot-capture"
	}
	return sanitize(p)
}

func archiveBasename(sessionBase, clusterName, message string) string {
	name := fmt.Sprintf("%s-%s", sessionBase, clusterName)
	if suffix := sanitizeMessage(message); suffix != "" {
		name += "-" + suffix
	}
	return name
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

// podLogArtifactRelPath is the capture-relative path for the main pod log file
// (<namespace>/<pod>__<node>.log with sanitized segments).
func podLogArtifactRelPath(namespace, podName, node string) string {
	n := strings.TrimSpace(node)
	if n == "" {
		n = "unknown-node"
	}
	return filepath.ToSlash(filepath.Join(namespace, sanitize(podName)+"__"+sanitize(n)+".log"))
}

// buildPodLogPathIndex maps "namespace/podName" to the relative log path groot writes when it
// collects logs (workloads when include_pod_logs; kube-system control-plane pods always).
func (s *Service) buildPodLogPathIndex(ctx context.Context) map[string]string {
	out := make(map[string]string)
	if s.cfg.Collection.IncludePodLogs {
		if refs, err := s.resolvePodsForLogs(ctx); err == nil {
			for _, r := range refs {
				name := strings.TrimSpace(r.Name)
				if name == "" {
					continue
				}
				ns := strings.TrimSpace(r.Namespace)
				node := strings.TrimSpace(r.Node)
				if node == "" {
					node = "unknown-node"
				}
				out[ns+"/"+name] = podLogArtifactRelPath(ns, name, node)
			}
		}
	}
	refs, _ := s.listControlPlanePods(ctx)
	for _, item := range refs {
		if item.Name == "" {
			continue
		}
		node := item.Node
		if node == "" {
			node = "unknown-node"
		}
		out["kube-system/"+item.Name] = podLogArtifactRelPath("kube-system", item.Name, node)
	}
	return out
}

type topPodMetrics struct {
	CPU string
	Mem string
}

// parseKubectlTopPodsAll parses kubectl top pods -A output (tab- or space-separated rows).
func parseKubectlTopPodsAll(content string) map[string]topPodMetrics {
	out := make(map[string]topPodMetrics)
	for _, line := range strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n") {
		line = strings.TrimSpace(strings.TrimSuffix(line, "\r"))
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}
		if fields[0] == "NAMESPACE" {
			continue
		}
		ns, name, cpu, mem := fields[0], fields[1], fields[2], fields[3]
		out[ns+"/"+name] = topPodMetrics{CPU: cpu, Mem: mem}
	}
	return out
}

// writePodRCATable merges all-pod-node-placement.tsv with all-pods-top.txt (when present)
// and pod resource totals into extras/all-pods-rca.tsv — one table for RCA handoff.
func (s *Service) writePodRCATable(captureDir string, podResources map[string]podResourceTotals) error {
	placementPath := filepath.Join(captureDir, "extras", "all-pod-node-placement.tsv")
	raw, err := os.ReadFile(placementPath)
	if err != nil {
		return fmt.Errorf("read all-pod-node-placement.tsv: %w", err)
	}
	topPath := filepath.Join(captureDir, "extras", "all-pods-top.txt")
	var metrics map[string]topPodMetrics
	if b, readErr := os.ReadFile(topPath); readErr == nil {
		metrics = parseKubectlTopPodsAll(string(b))
	}

	var b strings.Builder
	b.WriteString("namespace\tpod\tnode\tcpu_cores\tmemory_bytes\tcpu_request\tcpu_limit\tmemory_request\tmemory_limit\tpod_log_file\n")
	for _, line := range strings.Split(strings.ReplaceAll(string(raw), "\r\n", "\n"), "\n") {
		line = strings.TrimSpace(strings.TrimSuffix(line, "\r"))
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "namespace\tpod\tnode") {
			continue
		}
		parts := strings.Split(line, "\t")
		if len(parts) < 4 {
			continue
		}
		ns, pod, node, logFile := parts[0], parts[1], parts[2], parts[3]
		cpu, mem := "", ""
		if metrics != nil {
			if m, ok := metrics[ns+"/"+pod]; ok {
				cpu, mem = m.CPU, m.Mem
			}
		}
		cpuReq, cpuLim, memReq, memLim := "", "", "", ""
		if podResources != nil {
			if totals, ok := podResources[ns+"/"+pod]; ok {
				cpuReq, cpuLim = totals.CPURequest, totals.CPULimit
				memReq, memLim = totals.MemoryRequest, totals.MemoryLimit
			}
		}
		b.WriteString(fmt.Sprintf("%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			ns, pod, node, cpu, mem, cpuReq, cpuLim, memReq, memLim, logFile))
	}

	target := filepath.Join(captureDir, "extras", "all-pods-rca.tsv")
	if err := os.WriteFile(target, []byte(b.String()), 0o644); err != nil {
		return fmt.Errorf("write all-pods-rca.tsv: %w", err)
	}
	return nil
}

func (s *Service) writePodNodePlacement(ctx context.Context, captureDir string) error {
	list, err := s.listAllPods(ctx)
	if err != nil {
		return fmt.Errorf("list pods for placement: %w", err)
	}
	refs := podRefsFromList(list)
	logPaths := s.buildPodLogPathIndex(ctx)

	var b strings.Builder
	b.WriteString("namespace\tpod\tnode\tpod_log_file\n")
	for _, ref := range refs {
		logRel := ""
		if p, ok := logPaths[ref.Namespace+"/"+ref.Name]; ok {
			logRel = p
		}
		b.WriteString(fmt.Sprintf("%s\t%s\t%s\t%s\n", ref.Namespace, ref.Name, ref.Node, logRel))
	}

	target := filepath.Join(captureDir, "extras", "all-pod-node-placement.tsv")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return fmt.Errorf("create placement dir: %w", err)
	}
	if err := os.WriteFile(target, []byte(b.String()), 0o644); err != nil {
		return fmt.Errorf("write placement: %w", err)
	}
	return nil
}

func (s *Service) writeMetadata(ctx context.Context, captureDir string) error {
	meta, err := s.ReadKubeMetadata(ctx)
	if err != nil {
		return err
	}
	cluster := s.resolveClusterName(ctx)
	if s.restConfig == nil && strings.TrimSpace(meta.Server) == "" {
		meta.Server = restHostFromKubeconfig(s.cfg.Kubeconfig)
	} else if strings.TrimSpace(meta.Server) == "" && s.restConfig != nil {
		meta.Server = strings.TrimSpace(s.restConfig.Host)
	}

	content := fmt.Sprintf(
		"context: %s\ncluster: %s\nuser: %s\nserver: %s\n",
		emptyAsUnknown(meta.Context),
		emptyAsUnknown(cluster),
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

func (s *Service) resolvePodsForLogs(ctx context.Context) ([]podRef, error) {
	list, err := s.listAllPods(ctx)
	if err != nil {
		return nil, err
	}
	allRefs := podRefsFromList(list)

	if len(s.cfg.Collection.Targets) == 0 {
		return allRefs, nil
	}

	filtered := make([]podRef, 0)
	seen := map[string]struct{}{}

	for _, pod := range list.Items {
		ns := strings.TrimSpace(pod.Namespace)
		targets, ok := s.cfg.Collection.Targets[ns]
		if !ok || !hasTargets(targets) {
			continue
		}
		if !matchesTargetsByLabels(pod.Labels, targets) {
			continue
		}

		node := strings.TrimSpace(pod.Spec.NodeName)
		if node == "" {
			node = "unknown-node"
		}
		ref := podRef{
			Namespace: ns,
			Name:      strings.TrimSpace(pod.Name),
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
	for _, ref := range allRefs {
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

// parseNameNodeLines parses NAME,NODE rows (first field pod name, rest node).
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

func hasTargets(t config.NamespaceTargets) bool {
	return len(t.Deployments) > 0 || len(t.StatefulSets) > 0 || len(t.DaemonSets) > 0 ||
		len(t.Jobs) > 0 || len(t.CronJobs) > 0 || len(t.HelmReleases) > 0
}

func matchesTargetsByLabels(labels map[string]string, targets config.NamespaceTargets) bool {
	if len(labels) == 0 {
		return false
	}
	nameLabel := strings.TrimSpace(labels["app.kubernetes.io/name"])
	instanceLabel := strings.TrimSpace(labels["app.kubernetes.io/instance"])
	jobName := strings.TrimSpace(labels["job-name"])
	appLabel := strings.TrimSpace(labels["app"])

	// Helm releases: match by Helm-canonical labels, and reject non-Helm owners
	// that copy app.kubernetes.io/* without being a Helm-managed workload.
	if len(targets.HelmReleases) > 0 {
		if helmMatches(labels, targets.HelmReleases) {
			return true
		}
	}

	// Standard and Jobs/CronJob label sets.
	for _, lists := range [][]string{
		targets.Deployments, targets.StatefulSets, targets.DaemonSets,
		targets.Jobs, targets.CronJobs,
	} {
		if labelMatches(lists, nameLabel, instanceLabel, jobName, appLabel) {
			return true
		}
	}
	return false
}

// helmMatches reports whether labels belong to any of the listed Helm releases.
//
// Helm-canonical labels are app.kubernetes.io/instance + app.kubernetes.io/name,
// usually paired with app.kubernetes.io/managed-by=Helm. Legacy Helm 2 used
// heritage=Tiller + chart=NAME. We honour the modern labels first and fall back
// to the legacy pair, while refusing to match non-Helm workloads that happen to
// carry app.kubernetes.io/instance (e.g. Kustomize, Operators): if a managed-by
// label is present and identifies a non-Helm owner, the pod is excluded even
// when instance or name match the target list.
func helmMatches(labels map[string]string, releases []string) bool {
	instance := strings.TrimSpace(labels["app.kubernetes.io/instance"])
	name := strings.TrimSpace(labels["app.kubernetes.io/name"])
	managedBy := strings.TrimSpace(labels["app.kubernetes.io/managed-by"])
	heritage := strings.TrimSpace(labels["heritage"])
	legacyRelease := strings.TrimSpace(labels["release"])

	// Explicit non-Helm owner rejects the match — even if other labels coincide.
	if managedBy != "" && !strings.EqualFold(managedBy, "Helm") {
		return false
	}

	// Modern labels: instance and name must hit the target list (either suffices).
	if contains(releases, instance) || contains(releases, name) {
		return true
	}

	// Legacy Helm 2 (Tiller-managed): heritage=Tiller + release=<target>.
	if strings.EqualFold(heritage, "Tiller") && contains(releases, legacyRelease) {
		return true
	}

	return false
}

// labelMatches reports whether any label among the given keys hits one of the
// candidate target names. Passing the same slice of candidates through every
// label key is intentional: it keeps the matching policy identical for the
// standard workload families and Job/CronJob.
func labelMatches(candidates []string, keys ...string) bool {
	for _, k := range keys {
		if k == "" {
			continue
		}
		if contains(candidates, k) {
			return true
		}
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

func (s *Service) runRedactPass(captureDir string) {
	if err := RedactCaptureLogs(captureDir, s.cfg.Collection.RedactPatterns); err != nil {
		s.invokeOnFailed("redact-secrets", err)
	}
}
