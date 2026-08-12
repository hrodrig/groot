package collector

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hrodrig/groot/internal/config"
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

func TestParseKubectlTopPodsAll(t *testing.T) {
	in := "NAMESPACE     NAME                     CPU(cores)   MEMORY(bytes)   \ndefault       pod-a                    5m           10Mi\n"
	m := parseKubectlTopPodsAll(in)
	got, ok := m["default/pod-a"]
	if !ok || got.CPU != "5m" || got.Mem != "10Mi" {
		t.Fatalf("%#v", m)
	}
	if len(m) != 1 {
		t.Fatalf("expected 1 row, got %d", len(m))
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

func TestHelmMatches(t *testing.T) {
	releases := []string{"myrel"}

	cases := []struct {
		name   string
		labels map[string]string
		want   bool
		why    string
	}{
		{
			name:   "modern instance only",
			labels: map[string]string{"app.kubernetes.io/instance": "myrel"},
			want:   true,
			why:    "standard Helm 3 label",
		},
		{
			name: "modern instance + name + managed-by",
			labels: map[string]string{
				"app.kubernetes.io/instance":   "myrel",
				"app.kubernetes.io/name":       "myrel-chart",
				"app.kubernetes.io/managed-by": "Helm",
			},
			want: true,
			why:  "Helm 3 canonical triple",
		},
		{
			name: "modern name only (no instance)",
			labels: map[string]string{
				"app.kubernetes.io/name":       "myrel",
				"app.kubernetes.io/managed-by": "helm",
			},
			want: true,
			why:  "name match with managed-by=Helm (case-insensitive)",
		},
		{
			name: "legacy helm2 tiller + release",
			labels: map[string]string{
				"heritage": "Tiller",
				"release":  "myrel",
				"chart":    "myrel-chart",
			},
			want: true,
			why:  "Helm 2 Tiller-managed pod",
		},
		{
			name:   "non-helm owner rejected even with matching instance",
			labels: map[string]string{"app.kubernetes.io/instance": "myrel", "app.kubernetes.io/managed-by": "Kustomize"},
			want:   false,
			why:    "Kustomize sets the same labels but is not Helm",
		},
		{
			name:   "non-helm owner rejected even with matching name",
			labels: map[string]string{"app.kubernetes.io/name": "myrel", "app.kubernetes.io/managed-by": "operator-sdk"},
			want:   false,
			why:    "operator labels must not match helm_releases",
		},
		{
			name:   "instance set but not in target list",
			labels: map[string]string{"app.kubernetes.io/instance": "other-rel"},
			want:   false,
			why:    "different release name",
		},
		{
			name:   "no helm labels at all",
			labels: map[string]string{"app": "worker", "tier": "backend"},
			want:   false,
			why:    "non-Helm pod without canonical labels",
		},
		{
			name: "legacy tiller but release name differs",
			labels: map[string]string{
				"heritage": "Tiller",
				"release":  "other-rel",
			},
			want: false,
			why:  "legacy release name not in target list",
		},
		{
			name: "modern instance matches even when managed-by is absent",
			labels: map[string]string{
				"app.kubernetes.io/instance": "myrel",
				"app.kubernetes.io/name":     "myrel-chart",
			},
			want: true,
			why:  "managed-by is optional; charts may omit it",
		},
		{
			name: "managed-by=Helm but neither instance nor name match target list",
			labels: map[string]string{
				"app.kubernetes.io/managed-by": "Helm",
				"app.kubernetes.io/instance":   "other-rel",
				"app.kubernetes.io/name":       "other-chart",
			},
			want: false,
			why:  "Helm owner but different release",
		},
		{
			name: "non-helm heritage (Flux) does not match as legacy",
			labels: map[string]string{
				"heritage": "Flux",
				"release":  "myrel",
			},
			want: false,
			why:  "Flux reuses some Helm-style labels; reject",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := helmMatches(tc.labels, releases)
			if got != tc.want {
				t.Fatalf("%s: got=%v want=%v — %s", tc.name, got, tc.want, tc.why)
			}
		})
	}

	// Belt-and-braces: matchesTargetsByLabels must use helmMatches when
	// helm_releases is populated, so the rejection behavior above flows
	// through end-to-end. A Kustomize-owned pod with a matching instance must
	// NOT be considered a Helm target by the umbrella helper.
	nonHelm := map[string]string{
		"app.kubernetes.io/instance":   "myrel",
		"app.kubernetes.io/managed-by": "Kustomize",
	}
	if matchesTargetsByLabels(nonHelm, config.NamespaceTargets{HelmReleases: releases}) {
		t.Fatal("matchesTargetsByLabels must not treat non-Helm owners as Helm releases")
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
	if got := captureSessionBase("groot-capture", "7kqv2xy", "20260102-150405", ""); got != "groot-capture-7kqv2xy-20260102-150405" {
		t.Fatalf("empty since: %q", got)
	}
	if got := captureSessionBase("groot-capture", "7kqv2xy", "20260102-150405", "12h"); got != "groot-capture-7kqv2xy-20260102-150405-since-12h" {
		t.Fatalf("12h: %q", got)
	}
	if got := captureSessionBase("", "abc1234", "20260102-150405", "45m"); got != "groot-capture-abc1234-20260102-150405-since-45m" {
		t.Fatalf("45m: %q", got)
	}
	a := captureSessionBase("groot-capture", "aaaaaaa", "20260102-150405", "")
	b := captureSessionBase("groot-capture", "bbbbbbb", "20260102-150405", "")
	if a == b {
		t.Fatalf("same timestamp different shorts must not collide: %q", a)
	}
}

func TestArchiveBasename(t *testing.T) {
	got := archiveBasename("groot-capture-7kqv2xy-20260102-150405", "prod", "RCA run")
	if !strings.Contains(got, "groot-capture-7kqv2xy-20260102-150405-prod") {
		t.Fatalf("got %q", got)
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
