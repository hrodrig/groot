package collector

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/hrodrig/groot/pkg/k8srunner"
	"github.com/hrodrig/groot/pkg/kubeloader"
	metricsversioned "k8s.io/metrics/pkg/client/clientset/versioned"
)

// initK8s builds the Kubernetes clients. Tests may pre-set s.clientset to skip loading kubeconfig.
func (s *Service) initK8s() error {
	if s.clientset != nil {
		if s.k8sRunner == nil {
			host := "https://kubernetes.default.svc"
			if s.restConfig != nil && strings.TrimSpace(s.restConfig.Host) != "" {
				host = s.restConfig.Host
			}
			s.k8sRunner = k8srunner.New(s.clientset, s.metricsCS, s.clientset.Discovery(), host, s.cfg.Kubeconfig)
		}
		return nil
	}
	rc, err := kubeloader.RESTConfig(s.cfg.Kubeconfig)
	if err != nil {
		return err
	}
	cs, err := kubernetes.NewForConfig(rc)
	if err != nil {
		return err
	}
	var ms *metricsversioned.Clientset
	if m2, err := metricsversioned.NewForConfig(rc); err == nil {
		ms = m2
	}
	s.restConfig = rc
	s.clientset = cs
	s.metricsCS = ms
	s.k8sRunner = k8srunner.New(cs, ms, cs.Discovery(), rc.Host, s.cfg.Kubeconfig)
	return nil
}

func (s *Service) runJobOutput(ctx context.Context, j job) ([]byte, error) {
	if s.k8sRunner == nil {
		return nil, fmt.Errorf("kubernetes client not initialized")
	}
	if strings.HasPrefix(j.Name, "extra-") {
		return s.k8sRunner.Run(ctx, j.Args)
	}
	if b, ok, err := s.runNamedBuiltinJob(ctx, j); ok {
		return b, err
	}
	return s.runJobByNamePattern(ctx, j)
}

func (s *Service) runNamedBuiltinJob(ctx context.Context, j job) ([]byte, bool, error) {
	switch j.Name {
	case "cluster-info":
		b, err := s.k8sRunner.Run(ctx, []string{"cluster-info"})
		return b, true, err
	case "nodes-wide":
		b, err := s.k8sRunner.Run(ctx, []string{"get", "nodes", "-o", "wide"})
		return b, true, err
	case "events-all":
		b, err := s.k8sRunner.Run(ctx, []string{"get", "events", "-A", "--sort-by=.lastTimestamp"})
		return b, true, err
	case "pods-all":
		b, err := s.k8sRunner.Run(ctx, []string{"get", "pods", "-A", "-o", "wide"})
		return b, true, err
	case "pods-top-all":
		b, err := s.k8sRunner.Run(ctx, []string{"top", "pods", "-A"})
		return b, true, err
	default:
		return nil, false, nil
	}
}

func (s *Service) runJobByNamePattern(ctx context.Context, j job) ([]byte, error) {
	if strings.HasPrefix(j.Name, "namespace-") {
		ns := namespaceFromJobArgs(j.Args)
		if ns == "" {
			return nil, fmt.Errorf("namespace job missing -n")
		}
		return s.namespaceAllWide(ctx, ns)
	}
	if strings.HasPrefix(j.Name, "describe-") {
		return s.k8sRunner.Run(ctx, j.Args)
	}
	if strings.HasPrefix(j.Name, "top-") && len(j.Args) >= 2 && j.Args[0] == "top" {
		return s.k8sRunner.Run(ctx, j.Args)
	}
	if strings.HasPrefix(j.Name, "node-log-") && len(j.Args) >= 3 && j.Args[0] == "get" && j.Args[1] == "--raw" {
		return s.clientset.CoreV1().RESTClient().Get().AbsPath(j.Args[2]).DoRaw(ctx)
	}
	if len(j.Args) > 0 && j.Args[0] == "logs" {
		return s.k8sRunner.Run(ctx, j.Args)
	}
	return nil, fmt.Errorf("unhandled job %s", j.Name)
}

func namespaceFromJobArgs(args []string) string {
	for i := 0; i < len(args)-1; i++ {
		if args[i] == "-n" || args[i] == "--namespace" {
			return args[i+1]
		}
	}
	return ""
}

type nsResourceStep struct {
	title string
	list  func(context.Context) (interface{}, error)
}

func (s *Service) namespaceAllWide(ctx context.Context, ns string) ([]byte, error) {
	steps := []nsResourceStep{
		{"pods", func(c context.Context) (interface{}, error) {
			return s.clientset.CoreV1().Pods(ns).List(c, metav1.ListOptions{})
		}},
		{"services", func(c context.Context) (interface{}, error) {
			return s.clientset.CoreV1().Services(ns).List(c, metav1.ListOptions{})
		}},
		{"deployments.apps", func(c context.Context) (interface{}, error) {
			return s.clientset.AppsV1().Deployments(ns).List(c, metav1.ListOptions{})
		}},
		{"replicasets.apps", func(c context.Context) (interface{}, error) {
			return s.clientset.AppsV1().ReplicaSets(ns).List(c, metav1.ListOptions{})
		}},
		{"statefulsets.apps", func(c context.Context) (interface{}, error) {
			return s.clientset.AppsV1().StatefulSets(ns).List(c, metav1.ListOptions{})
		}},
		{"daemonsets.apps", func(c context.Context) (interface{}, error) {
			return s.clientset.AppsV1().DaemonSets(ns).List(c, metav1.ListOptions{})
		}},
	}
	var b strings.Builder
	for _, step := range steps {
		list, err := step.list(ctx)
		if err != nil {
			return nil, err
		}
		if err := appendNamespaceSection(&b, step.title, list); err != nil {
			return nil, err
		}
	}
	return []byte(b.String()), nil
}

func appendNamespaceSection(b *strings.Builder, title string, list interface{}) error {
	raw, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return err
	}
	body := strings.TrimSpace(string(raw))
	fmt.Fprintf(b, "== %s ==\n%s\n", title, body)
	if body != "" && !strings.HasSuffix(body, "\n") {
		b.WriteByte('\n')
	}
	b.WriteByte('\n')
	return nil
}

func (s *Service) execToFile(ctx context.Context, captureDir string, j job) error {
	s.invokeOnStart(j.Name, j.Args)

	body, err := s.runJobOutput(ctx, j)
	content := string(body)
	if err != nil {
		content = content + "\n--- error ---\n" + err.Error()
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
		s.invokeOnFailed(j.Name, err)
		return fmt.Errorf("api %s: %w", strings.Join(j.Args, " "), err)
	}

	s.invokeOnDone(j.Name)
	return nil
}

func (s *Service) listNodesAsResources(ctx context.Context) ([]string, error) {
	nl, err := s.clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(nl.Items))
	for _, n := range nl.Items {
		out = append(out, "node/"+n.Name)
	}
	return out, nil
}

func (s *Service) listAllPods(ctx context.Context) (*corev1.PodList, error) {
	return s.clientset.CoreV1().Pods(metav1.NamespaceAll).List(ctx, metav1.ListOptions{})
}

func (s *Service) listControlPlanePods(ctx context.Context) ([]podRef, error) {
	list, err := s.clientset.CoreV1().Pods("kube-system").List(ctx, metav1.ListOptions{
		LabelSelector: "tier=control-plane",
	})
	if err != nil {
		return nil, err
	}
	out := make([]podRef, 0, len(list.Items))
	for _, p := range list.Items {
		node := p.Spec.NodeName
		if node == "" {
			node = "unknown-node"
		}
		out = append(out, podRef{Namespace: p.Namespace, Name: p.Name, Node: node})
	}
	return out, nil
}

func podRefsFromList(list *corev1.PodList) []podRef {
	out := make([]podRef, 0, len(list.Items))
	for _, p := range list.Items {
		node := p.Spec.NodeName
		if node == "" {
			node = "unknown-node"
		}
		out = append(out, podRef{
			Namespace: p.Namespace,
			Name:      p.Name,
			Node:      node,
		})
	}
	return out
}
