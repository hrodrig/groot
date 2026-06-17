package collector

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
	ktesting "k8s.io/client-go/testing"

	"github.com/hrodrig/groot/pkg/config"
	"github.com/hrodrig/groot/pkg/kubetest"
)

func TestService_Run_minimal(t *testing.T) {
	kc, cleanup := kubetest.StartAPIServer(t)
	defer cleanup()

	out := t.TempDir()
	cfg := config.Config{
		Kubeconfig: kc,
		OutputDir:  out,
		FilePrefix: "pfx",
		Collection: config.CollectionCfg{
			Timeout:             30 * time.Second,
			WorkerConcurrency:   2,
			Namespaces:          nil,
			Targets:             nil,
			ExtraKubectl:        nil,
			IncludePodLogs:      false,
			IncludePreviousLogs: false,
			IncludeNodeDetails:  false,
			IncludeNodeLogs:     false,
			IncludePodMetrics:   false,
			PodLogTailLines:     0,
		},
	}

	svc := New(cfg)
	var started int
	svc.SetHooks(
		func(name string, args []string) { started++ },
		func(string) {},
		func(string, error) {},
	)

	sum, err := svc.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if sum.ArchivePath == "" {
		t.Fatalf("archive: %q", sum.ArchivePath)
	}
	if !filepath.IsAbs(sum.ArchivePath) {
		t.Fatalf("expected absolute archive path: %q", sum.ArchivePath)
	}
	if _, err := os.Stat(sum.ArchivePath); err != nil {
		t.Fatal(err)
	}
	if sum.Total < 1 {
		t.Fatalf("expected jobs, got %#v", sum)
	}
	if started != sum.Total {
		t.Fatalf("hooks started=%d total=%d", started, sum.Total)
	}
}

func TestService_Run_fullFeatures(t *testing.T) {
	kc, cleanup := kubetest.StartAPIServer(t)
	defer cleanup()

	out := t.TempDir()
	cfg := config.Config{
		Kubeconfig: kc,
		OutputDir:  out,
		FilePrefix: "pfx",
		Collection: config.CollectionCfg{
			Timeout:           45 * time.Second,
			WorkerConcurrency: 4,
			Namespaces:        []string{"default"},
			Targets: map[string]config.NamespaceTargets{
				"default": {Deployments: []string{"api"}},
			},
			ExtraKubectl:        []string{"get ns -o name"},
			IncludePodLogs:      true,
			IncludePreviousLogs: true,
			IncludeNodeDetails:  true,
			IncludeNodeLogs:     true,
			IncludePodMetrics:   true,
			PodLogTailLines:     50,
			NodeLogTailLines:    100,
		},
	}

	svc := New(cfg)
	svc.SetMessage("run-label-test")
	svc.SetHooks(
		func(string, []string) {},
		func(string) {},
		func(string, error) {},
	)

	sum, err := svc.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if sum.Failed > sum.Total {
		t.Fatalf("failed=%d total=%d", sum.Failed, sum.Total)
	}
	if sum.ArchivePath == "" {
		t.Fatal("no archive")
	}
}

func TestService_Run_writesPodNodePlacementInArchive(t *testing.T) {
	kc, cleanup := kubetest.StartAPIServer(t)
	defer cleanup()

	out := t.TempDir()
	cfg := config.Config{
		Kubeconfig: kc,
		OutputDir:  out,
		Collection: config.CollectionCfg{
			Timeout:            30 * time.Second,
			WorkerConcurrency:  2,
			IncludePodLogs:     true,
			IncludeNodeDetails: false,
			IncludeNodeLogs:    false,
			IncludePodMetrics:  false,
			PodLogTailLines:    10,
		},
	}

	sum, err := New(cfg).Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	f, err := os.Open(sum.ArchivePath)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	gzr, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	defer gzr.Close()

	var placement []byte
	tr := tar.NewReader(gzr)
	for {
		h, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		if !strings.HasSuffix(h.Name, "extras/all-pod-node-placement.tsv") {
			continue
		}
		placement, err = io.ReadAll(tr)
		if err != nil {
			t.Fatal(err)
		}
		break
	}
	if len(placement) == 0 {
		t.Fatal("all-pod-node-placement.tsv missing from archive")
	}
	if !bytes.Contains(placement, []byte("namespace\tpod\tnode\tpod_log_file")) {
		t.Fatalf("header missing: %s", placement)
	}
	if !bytes.Contains(placement, []byte("default\tpod-a\tnode1")) {
		t.Fatalf("row missing: %s", placement)
	}
}

func TestService_Run_writesPodRCAInArchive(t *testing.T) {
	kc, cleanup := kubetest.StartAPIServer(t)
	defer cleanup()

	out := t.TempDir()
	cfg := config.Config{
		Kubeconfig: kc,
		OutputDir:  out,
		Collection: config.CollectionCfg{
			Timeout:            30 * time.Second,
			WorkerConcurrency:  2,
			IncludePodLogs:     true,
			IncludeNodeDetails: false,
			IncludeNodeLogs:    false,
			IncludePodMetrics:  true,
			PodLogTailLines:    10,
		},
	}

	sum, err := New(cfg).Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	f, err := os.Open(sum.ArchivePath)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	gzr, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	defer gzr.Close()

	var rca []byte
	tr := tar.NewReader(gzr)
	for {
		h, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		if !strings.HasSuffix(h.Name, "extras/all-pods-rca.tsv") {
			continue
		}
		rca, err = io.ReadAll(tr)
		if err != nil {
			t.Fatal(err)
		}
		break
	}
	if len(rca) == 0 {
		t.Fatal("all-pods-rca.tsv missing from archive")
	}
	if !bytes.Contains(rca, []byte("namespace\tpod\tnode\tcpu_cores\tmemory_bytes\tcpu_request\tcpu_limit\tmemory_request\tmemory_limit\tpod_log_file")) {
		t.Fatalf("header: %s", rca)
	}
	if !bytes.Contains(rca, []byte("default\tpod-a\tnode1\t5m\t10Mi")) {
		t.Fatalf("expected merged metrics row: %s", rca)
	}
}

func TestService_Run_writesWorkloadResourcesInArchive(t *testing.T) {
	kc, cleanup := kubetest.StartAPIServer(t)
	defer cleanup()

	out := t.TempDir()
	cfg := config.Config{
		Kubeconfig: kc,
		OutputDir:  out,
		Collection: config.CollectionCfg{
			Timeout:            30 * time.Second,
			WorkerConcurrency:  2,
			IncludePodLogs:     false,
			IncludeNodeDetails: false,
			IncludeNodeLogs:    false,
		},
	}

	sum, err := New(cfg).Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	f, err := os.Open(sum.ArchivePath)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	gzr, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	defer gzr.Close()

	var wr []byte
	tr := tar.NewReader(gzr)
	for {
		h, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		if !strings.HasSuffix(h.Name, "extras/workload-resources.tsv") {
			continue
		}
		wr, err = io.ReadAll(tr)
		if err != nil {
			t.Fatal(err)
		}
		break
	}
	if len(wr) == 0 {
		t.Fatal("workload-resources.tsv missing from archive")
	}
	if !bytes.Contains(wr, []byte("namespace\tpod\tnode\tcontainer\tinit_container")) {
		t.Fatalf("header: %s", wr)
	}
	if !bytes.Contains(wr, []byte("default\tpod-a\tnode1\t")) {
		t.Fatalf("row missing: %s", wr)
	}
}

func TestReadKubeMetadata_fromKubeconfigFile(t *testing.T) {
	dir := t.TempDir()
	kc := filepath.Join(dir, "kc")
	content := `apiVersion: v1
kind: Config
current-context: ctx1
contexts:
- name: ctx1
  context:
    cluster: my-cluster
    user: u1
clusters:
- name: my-cluster
  cluster:
    server: https://127.0.0.1
users:
- name: u1
  user: {}
`
	if err := os.WriteFile(kc, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	svc := New(config.Config{Kubeconfig: kc})
	meta, err := svc.ReadKubeMetadata(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if meta.Context != "ctx1" || meta.Cluster != "my-cluster" {
		t.Fatalf("%#v", meta)
	}
}

func TestService_Run_invalidKubeconfig(t *testing.T) {
	out := t.TempDir()
	cfg := config.Config{
		Kubeconfig: filepath.Join(t.TempDir(), "nonexistent-kubeconfig"),
		OutputDir:  out,
		Collection: config.CollectionCfg{
			Timeout:            5 * time.Second,
			WorkerConcurrency:  1,
			IncludePodLogs:     false,
			IncludeNodeDetails: false,
			IncludeNodeLogs:    false,
			IncludePodMetrics:  false,
		},
	}
	_, err := New(cfg).Run(context.Background())
	if err == nil {
		t.Fatal("expected error when kubeconfig file is missing")
	}
}

func TestService_buildJobs_nodeListFails(t *testing.T) {
	cs := fake.NewSimpleClientset()
	cs.Fake.PrependReactor("list", "nodes", func(_ ktesting.Action) (bool, runtime.Object, error) {
		return true, &corev1.NodeList{}, fmt.Errorf("simulated node list failure")
	})
	svc := New(config.Config{
		Collection: config.CollectionCfg{
			IncludeNodeDetails: true,
		},
	})
	svc.clientset = cs
	svc.restConfig = &rest.Config{Host: "http://test.local"}
	if err := svc.initK8s(); err != nil {
		t.Fatal(err)
	}
	_, err := svc.buildJobs(context.Background())
	if err == nil {
		t.Fatal("expected error listing nodes")
	}
}

func TestService_resolvePodsForLogs_noTargetsUsesLines(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "pod-a", Labels: map[string]string{"app.kubernetes.io/name": "api"}},
		Spec:       corev1.PodSpec{NodeName: "node1"},
		Status:     corev1.PodStatus{Phase: corev1.PodRunning},
	}
	cs := fake.NewSimpleClientset(pod)
	svc := New(config.Config{
		Collection: config.CollectionCfg{
			IncludePodLogs: true,
			Targets:        nil,
		},
	})
	svc.clientset = cs
	svc.restConfig = &rest.Config{Host: "http://test.local"}
	if err := svc.initK8s(); err != nil {
		t.Fatal(err)
	}
	refs, err := svc.resolvePodsForLogs(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(refs) != 1 || refs[0].Name != "pod-a" {
		t.Fatalf("%#v", refs)
	}
}
