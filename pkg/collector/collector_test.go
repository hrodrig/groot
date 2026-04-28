package collector

import (
	"os"
	"path/filepath"
	"testing"

	"groot/pkg/config"
)

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
	args := workloadPodLogArgs("ns", "pod", false, 0)
	if len(args) < 4 {
		t.Fatalf("%#v", args)
	}
	argsPrev := workloadPodLogArgs("ns", "pod", true, 10)
	found := false
	for i, a := range argsPrev {
		if a == "--previous" {
			found = true
		}
		if a == "--tail" && i+1 < len(argsPrev) && argsPrev[i+1] == "10" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected tail/previous: %#v", argsPrev)
	}
}

func TestControlPlanePodLogArgs(t *testing.T) {
	args := controlPlanePodLogArgs("kube-apiserver", true, 0)
	if len(args) < 4 {
		t.Fatalf("%#v", args)
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
