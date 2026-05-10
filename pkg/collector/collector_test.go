package collector

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hrodrig/groot/pkg/config"
)

func TestKubeletLogQueryRawPath(t *testing.T) {
	if got := kubeletLogQueryRawPath("node-1", 100); got != "/api/v1/nodes/node-1/proxy/logs/?query=kubelet&tailLines=100" {
		t.Fatalf("got %q", got)
	}
	if got := kubeletLogQueryRawPath("my.node.local", 0); got != "/api/v1/nodes/my.node.local/proxy/logs/?query=kubelet" {
		t.Fatalf("got %q", got)
	}
}

func TestNodeVarLogMessagesRawPath(t *testing.T) {
	if got := nodeVarLogMessagesRawPath("node-1"); got != "/api/v1/nodes/node-1/proxy/logs/messages" {
		t.Fatalf("got %q", got)
	}
}

func TestPodLogArtifactRelPath(t *testing.T) {
	got := podLogArtifactRelPath("default", "my-pod", "node-1")
	if got != "default/my-pod__node-1.log" {
		t.Fatalf("got %q", got)
	}
	if podLogArtifactRelPath("ns", "p", "") != "ns/p__unknown-node.log" {
		t.Fatal("empty node")
	}
}

func TestSanitize(t *testing.T) {
	if got := sanitize("a/b:c d"); got != "a_b_c_d" {
		t.Fatalf("got %q", got)
	}
}

func TestSanitizeMessage(t *testing.T) {
	if sanitizeMessage("  ") != "" {
		t.Fatal("blank should be empty")
	}
	got := sanitizeMessage("Hello  World!!")
	if got == "" || got[0] == '-' {
		t.Fatalf("unexpected %q", got)
	}
}

func TestEmptyAsUnknown(t *testing.T) {
	if emptyAsUnknown("") != "unknown" {
		t.Fatal()
	}
	if emptyAsUnknown("x") != "x" {
		t.Fatal()
	}
}

func TestParsePodLines(t *testing.T) {
	refs := parsePodLines([]string{
		"ns1 pod1 node1",
		"bad",
		"ns2 pod2",
	})
	if len(refs) != 2 {
		t.Fatalf("len=%d %#v", len(refs), refs)
	}
	if refs[0].Namespace != "ns1" || refs[0].Name != "pod1" || refs[0].Node != "node1" {
		t.Fatalf("first ref %#v", refs[0])
	}
	if refs[1].Node != "unknown-node" {
		t.Fatalf("node %#v", refs[1])
	}
}

func TestParseNameNodeLines(t *testing.T) {
	refs := parseNameNodeLines([]string{
		"kube-apiserver-node1	node1",
		"etcd-masternode",
		"",
		"   ",
	})
	if len(refs) != 2 {
		t.Fatalf("len=%d %#v", len(refs), refs)
	}
	if refs[0].Name != "kube-apiserver-node1" || refs[0].Node != "node1" {
		t.Fatalf("first %#v", refs[0])
	}
	if refs[1].Name != "etcd-masternode" || refs[1].Node != "unknown-node" {
		t.Fatalf("second %#v", refs[1])
	}
}

func TestHasTargets(t *testing.T) {
	if hasTargets(config.NamespaceTargets{}) {
		t.Fatal("empty should be false")
	}
	if !hasTargets(config.NamespaceTargets{Deployments: []string{"a"}}) {
		t.Fatal("deployments should count")
	}
	if !hasTargets(config.NamespaceTargets{HelmReleases: []string{"r"}}) {
		t.Fatal("helm should count")
	}
}

func TestContains(t *testing.T) {
	if !contains([]string{" a ", "b"}, "a") {
		t.Fatal("trim match")
	}
	if contains([]string{"x"}, "y") {
		t.Fatal()
	}
}

func TestMatchesTargetsByLabels(t *testing.T) {
	targets := config.NamespaceTargets{Deployments: []string{"api"}}
	if matchesTargetsByLabels(nil, targets) {
		t.Fatal("nil labels")
	}
	if !matchesTargetsByLabels(map[string]string{
		"app.kubernetes.io/name": "api",
	}, targets) {
		t.Fatal("name label")
	}
	if !matchesTargetsByLabels(map[string]string{
		"app.kubernetes.io/instance": "myrel",
	}, config.NamespaceTargets{HelmReleases: []string{"myrel"}}) {
		t.Fatal("helm instance")
	}
	if !matchesTargetsByLabels(map[string]string{
		"app": "worker",
	}, config.NamespaceTargets{DaemonSets: []string{"worker"}}) {
		t.Fatal("legacy app label")
	}
}

func TestWorkloadPodLogArgs(t *testing.T) {
	args := workloadPodLogArgs("ns", "pod", false, 0, "")
	if len(args) < 4 {
		t.Fatalf("%#v", args)
	}
	argsPrev := workloadPodLogArgs("ns", "pod", true, 10, "")
	foundPrev := false
	foundTail := false
	for i, a := range argsPrev {
		if a == "--previous" {
			foundPrev = true
		}
		if a == "--tail" && i+1 < len(argsPrev) && argsPrev[i+1] == "10" {
			foundTail = true
		}
	}
	if !foundPrev || !foundTail {
		t.Fatalf("expected tail and previous: %#v", argsPrev)
	}
}

func TestWorkloadPodLogArgs_since(t *testing.T) {
	args := workloadPodLogArgs("ns", "pod", false, 100, "24h")
	foundSince := false
	foundTail := false
	for i, a := range args {
		if a == "--since=24h" {
			foundSince = true
		}
		if a == "--tail" && i+1 < len(args) && args[i+1] == "100" {
			foundTail = true
		}
	}
	if !foundSince || !foundTail {
		t.Fatalf("expected --since and --tail: %#v", args)
	}
}

func TestControlPlanePodLogArgs(t *testing.T) {
	args := controlPlanePodLogArgs("kube-apiserver", true, 0, "")
	if len(args) < 4 {
		t.Fatalf("%#v", args)
	}
}

func TestCaptureSessionBase(t *testing.T) {
	if got := captureSessionBase("20260102-150405", ""); got != "20260102-150405" {
		t.Fatalf("empty since: %q", got)
	}
	if got := captureSessionBase("20260102-150405", "12h"); got != "20260102-150405-since-12h" {
		t.Fatalf("12h: %q", got)
	}
	if got := captureSessionBase("20260102-150405", "45m"); got != "20260102-150405-since-45m" {
		t.Fatalf("45m: %q", got)
	}
}

func TestKubectlArgs(t *testing.T) {
	s := New(config.Config{Kubeconfig: ""})
	if got := s.kubectlArgs([]string{"get", "pods"}); len(got) != 2 || got[0] != "get" {
		t.Fatalf("%#v", got)
	}
	s2 := New(config.Config{Kubeconfig: "/tmp/k"})
	got2 := s2.kubectlArgs([]string{"get", "pods"})
	if len(got2) != 4 || got2[0] != "--kubeconfig" || got2[1] != "/tmp/k" {
		t.Fatalf("%#v", got2)
	}
}

func TestEnsureGroupDirs(t *testing.T) {
	root := t.TempDir()
	s := New(config.Config{
		Collection: config.CollectionCfg{
			Namespaces: []string{"  ", "app-ns"},
		},
	})
	if err := s.ensureGroupDirs(root); err != nil {
		t.Fatal(err)
	}
	for _, rel := range []string{"nodes", "extras", "app-ns"} {
		st, err := os.Stat(filepath.Join(root, rel))
		if err != nil || !st.IsDir() {
			t.Fatalf("missing dir %s: %v", rel, err)
		}
	}
}
