package k8srunner

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	ktesting "k8s.io/client-go/testing"
	metricsv1beta1api "k8s.io/metrics/pkg/apis/metrics/v1beta1"
	metricsfake "k8s.io/metrics/pkg/client/clientset/versioned/fake"
)

func TestParseFlags(t *testing.T) {
	t.Parallel()
	cases := []struct {
		argv             []string
		ns, out, sel     string
		allNS, noHeaders bool
		args             []string
	}{
		{[]string{"-A", "pods"}, "", "", "", true, false, []string{"pods"}},
		{[]string{"-n", "kube", "pods"}, "kube", "", "", false, false, []string{"pods"}},
		{[]string{"-n=kube", "pods"}, "kube", "", "", false, false, []string{"pods"}},
		{[]string{"--namespace=kube", "x"}, "kube", "", "", false, false, []string{"x"}},
		{[]string{"-o", "wide", "pods"}, "", "wide", "", false, false, []string{"pods"}},
		{[]string{"-o=yaml", "pods"}, "", "yaml", "", false, false, []string{"pods"}},
		{[]string{"--output=json", "pods"}, "", "json", "", false, false, []string{"pods"}},
		{[]string{"-l", "app=x", "pods"}, "", "", "app=x", false, false, []string{"pods"}},
		{[]string{"-l=app=x", "pods"}, "", "", "app=x", false, false, []string{"pods"}},
		{[]string{"--selector", "app=x", "pods"}, "", "", "app=x", false, false, []string{"pods"}},
		{[]string{"--selector=app=x", "pods"}, "", "", "app=x", false, false, []string{"pods"}},
		{[]string{"--no-headers", "pods"}, "", "", "", false, true, []string{"pods"}},
		{[]string{"--sort-by", ".metadata.name", "pods"}, "", "", "", false, false, []string{"pods"}},
		{[]string{"--sort-by=.metadata.name", "pods"}, "", "", "", false, false, []string{"pods"}},
		{[]string{"--request-timeout", "5s", "pods"}, "", "", "", false, false, []string{"pods"}},
		{[]string{"--request-timeout=5s", "pods"}, "", "", "", false, false, []string{"pods"}},
		{[]string{"get", "pods"}, "", "", "", false, false, []string{"get", "pods"}},
	}
	for _, tc := range cases {
		f := parseFlags(tc.argv)
		if f.namespace != tc.ns || f.output != tc.out || f.labelSelector != tc.sel ||
			f.allNamespaces != tc.allNS || f.noHeaders != tc.noHeaders {
			t.Errorf("parseFlags(%q): got %#v", tc.argv, f)
		}
		if diff := cmp.Diff(tc.args, f.args); diff != "" {
			t.Errorf("parseFlags(%q) args diff (-want +got):\n%s", tc.argv, diff)
		}
	}
}

func TestParseLogArgv(t *testing.T) {
	t.Parallel()
	st := parseLogArgv([]string{
		"-n", "ns1", "-p", "--all-containers", "--timestamps", "--tail", "100",
		"--since=1h", "my-pod",
	})
	if st.ns != "ns1" || st.pod != "my-pod" || !st.allContainers || !st.opts.Previous {
		t.Fatalf("%#v", st)
	}
	if st.opts.Timestamps == false || st.opts.TailLines == nil || *st.opts.TailLines != 100 {
		t.Fatalf("opts %#v", st.opts)
	}
	if st.opts.SinceSeconds == nil || *st.opts.SinceSeconds <= 0 {
		t.Fatalf("since %#v", st.opts)
	}
	st2 := parseLogArgv([]string{"--tail=50", "p2"})
	if st2.pod != "p2" || st2.opts.TailLines == nil || *st2.opts.TailLines != 50 {
		t.Fatalf("%#v", st2)
	}
}

func TestShortAge_and_podReady(t *testing.T) {
	t.Parallel()
	if g := shortAge(time.Time{}); g != "unknown" {
		t.Fatalf("zero: %q", g)
	}
	ts := time.Now().Add(-30 * time.Second)
	if !strings.Contains(shortAge(ts), "m") {
		t.Fatalf("recent: %q", shortAge(ts))
	}
	p := &corev1.Pod{
		Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "a"}, {Name: "b"}}},
		Status: corev1.PodStatus{ContainerStatuses: []corev1.ContainerStatus{
			{Name: "a", Ready: true},
			{Name: "b", Ready: false},
		}},
	}
	if podReadyString(p) != "1/2" {
		t.Fatalf("%s", podReadyString(p))
	}
	if podReadyString(&corev1.Pod{}) != "0/0" {
		t.Fatalf("%s", podReadyString(&corev1.Pod{}))
	}
}

func TestFormatPodListText(t *testing.T) {
	t.Parallel()
	list := &corev1.PodList{
		Items: []corev1.Pod{{
			ObjectMeta: metav1.ObjectMeta{Name: "p1", Namespace: "d"},
			Spec:       corev1.PodSpec{NodeName: "n1", Containers: []corev1.Container{{Name: "c"}}},
			Status: corev1.PodStatus{
				Phase:             corev1.PodRunning,
				PodIP:             "10.0.0.1",
				ContainerStatuses: []corev1.ContainerStatus{{Name: "c", Ready: true, RestartCount: 2}},
			},
		}},
	}
	list.Items[0].CreationTimestamp = metav1.NewTime(time.Now().Add(-2 * time.Hour))

	for _, mode := range []string{"wide", "default", "name", "yaml", "json"} {
		out := mode
		if mode == "default" {
			out = ""
		}
		b, err := formatPodListText(list, out, true)
		if err != nil {
			t.Fatalf("%s: %v", mode, err)
		}
		if len(b) == 0 {
			t.Fatalf("%s: empty", mode)
		}
	}
}

func TestSplitKindName(t *testing.T) {
	t.Parallel()
	k, n := splitKindName("pod/foo")
	if k != "pod" || n != "foo" {
		t.Fatalf("%q %q", k, n)
	}
	k, n = splitKindName("pod")
	if k != "" || n != "" {
		t.Fatalf("%q %q", k, n)
	}
}

func TestEventTime(t *testing.T) {
	t.Parallel()
	now := metav1.Now()
	e := corev1.Event{LastTimestamp: now}
	if eventTime(e).IsZero() {
		t.Fatal()
	}
}

func testPod() *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "pod1"},
		Spec: corev1.PodSpec{
			NodeName:   "node1",
			Containers: []corev1.Container{{Name: "c1"}},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			ContainerStatuses: []corev1.ContainerStatus{
				{Name: "c1", Ready: true},
			},
		},
	}
}

func TestRunner_Run_get_pods_and_namespaces(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	pod := testPod()
	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "default"}}
	cs := fake.NewSimpleClientset(ns, pod)
	r := New(cs, nil, cs.Discovery(), "https://api.test", "")

	out, err := r.Run(ctx, []string{"get", "pods", "-n", "default"})
	if err != nil || !strings.Contains(string(out), "pod1") {
		t.Fatalf("err=%v out=%s", err, out)
	}
	out, err = r.Run(ctx, []string{"get", "pods", "pod1", "-n", "default", "-o", "yaml"})
	if err != nil || !strings.Contains(string(out), "pod1") {
		t.Fatalf("err=%v", err)
	}
	out, err = r.Run(ctx, []string{"get", "ns"})
	if err != nil || !strings.Contains(string(out), "default") {
		t.Fatalf("err=%v out=%s", err, out)
	}
}

func TestRunner_Run_version_clusterInfo_apiVersions(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	cs := fake.NewSimpleClientset()
	r := New(cs, nil, cs.Discovery(), "https://h.example", "")

	out, err := r.Run(ctx, []string{"version"})
	if err != nil || !strings.Contains(string(out), "Server Version") {
		t.Fatalf("err=%v out=%s", err, out)
	}
	out, err = r.Run(ctx, []string{"cluster-info"})
	if err != nil || !strings.Contains(string(out), "https://h.example") {
		t.Fatalf("err=%v out=%s", err, out)
	}
	_, err = r.Run(ctx, []string{"api-versions"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = r.Run(ctx, []string{"api-resources"})
	if err != nil {
		t.Fatal(err)
	}
}

func TestRunner_Run_errors_and_unsupported(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	cs := fake.NewSimpleClientset()
	r := New(cs, nil, cs.Discovery(), "https://x", "")

	if _, err := r.Run(ctx, []string{}); err == nil {
		t.Fatal("expected empty argv error")
	}
	if _, err := r.Run(ctx, []string{"explain", "pods"}); err == nil {
		t.Fatal("expected explain unsupported")
	}
	if _, err := r.Run(ctx, []string{"top", "pods"}); err == nil {
		t.Fatal("expected top without metrics")
	}
	if _, err := r.Run(ctx, []string{"top", "nodes"}); err == nil {
		t.Fatal("expected top node missing name")
	}
	if _, err := r.Run(ctx, []string{"get"}); err == nil {
		t.Fatal("expected get missing resource")
	}
	if _, err := r.Run(ctx, []string{"describe"}); err == nil {
		t.Fatal("expected describe missing resource")
	}
}

func TestRunner_Run_describe_and_events(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	pod := testPod()
	node := &corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: "node1"}}
	ev := &corev1.Event{
		ObjectMeta:     metav1.ObjectMeta{Namespace: "default", Name: "e1"},
		InvolvedObject: corev1.ObjectReference{Kind: "Pod", Name: "pod1"},
		Message:        "hello",
		Type:           "Normal",
		Reason:         "Started",
		EventTime:      metav1.NewMicroTime(time.Now().Add(-time.Minute)),
		LastTimestamp:  metav1.Time{Time: time.Now().Add(-time.Minute)},
	}
	cs := fake.NewSimpleClientset(pod, node, ev)
	r := New(cs, nil, cs.Discovery(), "https://x", "")

	out, err := r.Run(ctx, []string{"describe", "pod", "pod1", "-n", "default"})
	if err != nil || !strings.Contains(string(out), "pod1") {
		t.Fatalf("err=%v out=%s", err, out)
	}
	out, err = r.Run(ctx, []string{"describe", "nodes/node1"})
	if err != nil || !strings.Contains(string(out), "node1") {
		t.Fatalf("err=%v out=%s", err, out)
	}
	out, err = r.Run(ctx, []string{"get", "events", "-n", "default"})
	if err != nil || !strings.Contains(string(out), "Started") {
		t.Fatalf("err=%v out=%s", err, out)
	}
}

func TestRunner_Run_logs(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	pod := testPod()
	cs := fake.NewSimpleClientset(pod)
	r := New(cs, nil, cs.Discovery(), "https://x", "")

	out, err := r.Run(ctx, []string{"logs", "-n", "default", "pod1"})
	if err != nil {
		t.Fatal(err)
	}
	_ = out
}

func TestRunner_Run_top_with_metrics_fake(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	now := metav1.Now()
	win := metav1.Duration{Duration: time.Minute}
	pm := &metricsv1beta1api.PodMetrics{
		TypeMeta:   metav1.TypeMeta{APIVersion: "metrics.k8s.io/v1beta1", Kind: "PodMetrics"},
		ObjectMeta: metav1.ObjectMeta{Name: "p1", Namespace: "default"},
		Timestamp:  now,
		Window:     win,
		Containers: []metricsv1beta1api.ContainerMetrics{
			{
				Name: "c1",
				Usage: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("5m"),
					corev1.ResourceMemory: resource.MustParse("2Mi"),
				},
			},
		},
	}
	nm := &metricsv1beta1api.NodeMetrics{
		TypeMeta:   metav1.TypeMeta{APIVersion: "metrics.k8s.io/v1beta1", Kind: "NodeMetrics"},
		ObjectMeta: metav1.ObjectMeta{Name: "node1"},
		Timestamp:  now,
		Window:     win,
		Usage: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("100m"),
			corev1.ResourceMemory: resource.MustParse("64Mi"),
		},
	}
	mc := metricsfake.NewSimpleClientset()
	mc.Fake.PrependReactor("list", "pods", func(action ktesting.Action) (bool, runtime.Object, error) {
		_ = action
		return true, &metricsv1beta1api.PodMetricsList{Items: []metricsv1beta1api.PodMetrics{*pm}}, nil
	})
	mc.Fake.PrependReactor("get", "nodes", func(action ktesting.Action) (bool, runtime.Object, error) {
		ga, ok := action.(ktesting.GetAction)
		if !ok || ga.GetName() != "node1" {
			return false, nil, nil
		}
		return true, nm, nil
	})
	cs := fake.NewSimpleClientset()
	r := New(cs, mc, cs.Discovery(), "https://x", "")

	out, err := r.Run(ctx, []string{"top", "pods"})
	if err != nil || !strings.Contains(string(out), "p1") {
		t.Fatalf("err=%v out=%s", err, out)
	}
	out, err = r.Run(ctx, []string{"top", "nodes", "node1"})
	if err != nil || !strings.Contains(string(out), "node1") {
		t.Fatalf("err=%v out=%s", err, out)
	}
}

func TestRunner_Run_auth_can_i(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	cs := fake.NewSimpleClientset()
	r := New(cs, nil, cs.Discovery(), "https://x", "")

	out, err := r.Run(ctx, []string{"auth", "can-i", "get", "pods", "-n", "default"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), "yes") && !strings.Contains(string(out), "no") {
		t.Fatalf("unexpected: %s", out)
	}
}

func TestRunner_runConfig_view_json(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	kc := filepath.Join(dir, "config")
	content := `apiVersion: v1
kind: Config
current-context: c1
contexts:
- name: c1
  context: {cluster: cl, user: u}
clusters:
- name: cl
  cluster: {server: https://example}
users:
- name: u
  user: {}
`
	if err := os.WriteFile(kc, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	cs := fake.NewSimpleClientset()
	r := New(cs, nil, cs.Discovery(), "https://x", kc)

	out, err := r.Run(ctx, []string{"config", "view"})
	if err != nil || !strings.Contains(string(out), "example") {
		t.Fatalf("err=%v out=%s", err, out)
	}
	out, err = r.Run(ctx, []string{"config", "view", "-o", "yaml"})
	if err != nil || !strings.Contains(string(out), "current-context") {
		t.Fatalf("err=%v", err)
	}
}

func TestContainsVerb(t *testing.T) {
	t.Parallel()
	if !containsVerb([]string{"list", "get"}, "list") {
		t.Fatal()
	}
	if containsVerb([]string{"get"}, "list") {
		t.Fatal()
	}
}

func TestWantsYAML(t *testing.T) {
	t.Parallel()
	if !wantsYAML([]string{"-o", "yaml"}) {
		t.Fatal()
	}
	if !wantsYAML([]string{"-o=yaml"}) {
		t.Fatal()
	}
	if wantsYAML([]string{"-o", "json"}) {
		t.Fatal()
	}
}
