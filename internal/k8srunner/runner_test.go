package k8srunner

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
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
	mc.PrependReactor("list", "pods", func(action ktesting.Action) (bool, runtime.Object, error) {
		_ = action
		return true, &metricsv1beta1api.PodMetricsList{Items: []metricsv1beta1api.PodMetrics{*pm}}, nil
	})
	mc.PrependReactor("get", "nodes", func(action ktesting.Action) (bool, runtime.Object, error) {
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

// ---- Helpers for extended-resource tests ----

func seedConfigMap() runtime.Object {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "cm1", Namespace: "default"},
		Data:       map[string]string{"k": "v"},
	}
}

func seedPVC() runtime.Object {
	sc := "standard"
	return &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{Name: "pvc1", Namespace: "default"},
		Spec:       corev1.PersistentVolumeClaimSpec{StorageClassName: &sc},
		Status:     corev1.PersistentVolumeClaimStatus{Phase: corev1.ClaimBound, AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce}},
	}
}

func seedService() runtime.Object {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: "svc1", Namespace: "default"},
		Spec: corev1.ServiceSpec{
			Type:      corev1.ServiceTypeClusterIP,
			ClusterIP: "10.0.0.1",
			Ports:     []corev1.ServicePort{{Port: 80, Protocol: corev1.ProtocolTCP}},
		},
	}
}

func seedIngress() runtime.Object {
	pathType := networkingv1.PathTypePrefix
	return &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{Name: "ing1", Namespace: "default"},
		Spec: networkingv1.IngressSpec{
			Rules: []networkingv1.IngressRule{
				{Host: "a.example.com", IngressRuleValue: networkingv1.IngressRuleValue{
					HTTP: &networkingv1.HTTPIngressRuleValue{
						Paths: []networkingv1.HTTPIngressPath{{Path: "/", PathType: &pathType}},
					},
				}},
				{Host: "b.example.com", IngressRuleValue: networkingv1.IngressRuleValue{
					HTTP: &networkingv1.HTTPIngressRuleValue{
						Paths: []networkingv1.HTTPIngressPath{{Path: "/", PathType: &pathType}},
					},
				}},
			},
		},
	}
}

func seedDeployment() runtime.Object {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "dep1", Namespace: "default"},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "dep1"}},
			Replicas: int32Ptr(3),
		},
		Status: appsv1.DeploymentStatus{Replicas: 3, ReadyReplicas: 2},
	}
}

func seedReplicaSet() runtime.Object {
	return &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{Name: "rs1", Namespace: "default"},
		Status:     appsv1.ReplicaSetStatus{Replicas: 3, ReadyReplicas: 3},
	}
}

func seedStatefulSet() runtime.Object {
	return &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{Name: "sts1", Namespace: "default"},
		Status:     appsv1.StatefulSetStatus{Replicas: 2, ReadyReplicas: 1},
	}
}

func seedDaemonSet() runtime.Object {
	return &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{Name: "ds1", Namespace: "default"},
		Status:     appsv1.DaemonSetStatus{NumberReady: 4, DesiredNumberScheduled: 5},
	}
}

func int32Ptr(i int32) *int32 { return &i }

// TestRunner_Run_get_extended covers the "get" dispatch and get-handlers for
// the new resource families: configmap, pvc, service, ingress, deployment,
// replicaset, statefulset, daemonset. For each, it exercises both the list
// path and the get-by-name path, plus at least one alias.
func TestRunner_Run_get_extended(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	type seedFn func() runtime.Object

	cases := []struct {
		name    string
		argv    []string
		seed    seedFn
		wantSub []string
	}{
		// configmap
		{"configmap_list", []string{"get", "configmap", "-n", "default"}, seedConfigMap, []string{"cm1"}},
		{"configmap_name", []string{"get", "configmap", "cm1", "-n", "default", "-o", "yaml"}, seedConfigMap, []string{"cm1"}},
		{"configmap_alias_cm", []string{"get", "cm", "-n", "default"}, seedConfigMap, []string{"cm1"}},
		// pvc
		{"pvc_list", []string{"get", "pvc", "-n", "default"}, seedPVC, []string{"pvc1"}},
		{"pvc_name", []string{"get", "pvc", "pvc1", "-n", "default", "-o", "json"}, seedPVC, []string{"pvc1"}},
		{"pvc_alias_persistentvolumeclaim", []string{"get", "persistentvolumeclaim", "-n", "default"}, seedPVC, []string{"pvc1"}},
		// service
		{"service_list", []string{"get", "service", "-n", "default"}, seedService, []string{"svc1", "ClusterIP"}},
		{"service_name", []string{"get", "service", "svc1", "-n", "default", "-o", "yaml"}, seedService, []string{"svc1"}},
		{"service_alias_svc", []string{"get", "svc", "-n", "default"}, seedService, []string{"svc1"}},
		// ingress
		{"ingress_list", []string{"get", "ingress", "-n", "default"}, seedIngress, []string{"ing1"}},
		{"ingress_name", []string{"get", "ingress", "ing1", "-n", "default", "-o", "yaml"}, seedIngress, []string{"ing1", "a.example.com"}},
		{"ingress_alias_ing", []string{"get", "ing", "-n", "default"}, seedIngress, []string{"ing1"}},
		// deployment
		{"deployment_list", []string{"get", "deployment", "-n", "default"}, seedDeployment, []string{"dep1", "2/3"}},
		{"deployment_name", []string{"get", "deployment", "dep1", "-n", "default", "-o", "yaml"}, seedDeployment, []string{"dep1"}},
		{"deployment_alias_deploy", []string{"get", "deploy", "-n", "default"}, seedDeployment, []string{"dep1"}},
		// replicaset
		{"replicaset_list", []string{"get", "replicaset", "-n", "default"}, seedReplicaSet, []string{"rs1", "3/3"}},
		{"replicaset_name", []string{"get", "replicaset", "rs1", "-n", "default", "-o", "yaml"}, seedReplicaSet, []string{"rs1"}},
		{"replicaset_alias_rs", []string{"get", "rs", "-n", "default"}, seedReplicaSet, []string{"rs1"}},
		// statefulset
		{"statefulset_list", []string{"get", "statefulset", "-n", "default"}, seedStatefulSet, []string{"sts1", "1/2"}},
		{"statefulset_name", []string{"get", "statefulset", "sts1", "-n", "default", "-o", "yaml"}, seedStatefulSet, []string{"sts1"}},
		{"statefulset_alias_sts", []string{"get", "sts", "-n", "default"}, seedStatefulSet, []string{"sts1"}},
		// daemonset
		{"daemonset_list", []string{"get", "daemonset", "-n", "default"}, seedDaemonSet, []string{"ds1", "4/5"}},
		{"daemonset_name", []string{"get", "daemonset", "ds1", "-n", "default", "-o", "yaml"}, seedDaemonSet, []string{"ds1"}},
		{"daemonset_alias_ds", []string{"get", "ds", "-n", "default"}, seedDaemonSet, []string{"ds1"}},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			cs := fake.NewSimpleClientset(tc.seed())
			r := New(cs, nil, cs.Discovery(), "https://api.test", "")
			out, err := r.Run(ctx, tc.argv)
			if err != nil {
				t.Fatalf("err=%v out=%s", err, out)
			}
			s := string(out)
			for _, w := range tc.wantSub {
				if !strings.Contains(s, w) {
					t.Fatalf("argv=%v: missing %q in output:\n%s", tc.argv, w, s)
				}
			}
		})
	}
}

// TestRunner_Run_describe_extended covers describe dispatch + describe-handlers
// for the new resource families.
func TestRunner_Run_describe_extended(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	type seedFn func() runtime.Object
	cases := []struct {
		name    string
		argv    []string
		seed    seedFn
		wantSub []string
	}{
		{
			"configmap",
			[]string{"describe", "configmap", "cm1", "-n", "default"},
			seedConfigMap,
			[]string{"Name: cm1", "Namespace: default", "Data keys: 1"},
		},
		{
			"configmap_alias_cm",
			[]string{"describe", "cm", "cm1", "-n", "default"},
			seedConfigMap,
			[]string{"Name: cm1"},
		},
		{
			"pvc",
			[]string{"describe", "pvc", "pvc1", "-n", "default"},
			seedPVC,
			[]string{"Name: pvc1", "Namespace: default", "Phase: Bound", "StorageClass: standard"},
		},
		{
			"pvc_alias",
			[]string{"describe", "persistentvolumeclaim", "pvc1", "-n", "default"},
			seedPVC,
			[]string{"Name: pvc1"},
		},
		{
			"service",
			[]string{"describe", "svc", "svc1", "-n", "default"},
			seedService,
			[]string{"Name: svc1", "Namespace: default", "Type: ClusterIP", "ClusterIP: 10.0.0.1"},
		},
		{
			"service_alias",
			[]string{"describe", "service", "svc1", "-n", "default"},
			seedService,
			[]string{"Name: svc1"},
		},
		{
			"ingress",
			[]string{"describe", "ingress", "ing1", "-n", "default"},
			seedIngress,
			[]string{"Name: ing1", "Namespace: default", "Hosts: a.example.com,b.example.com"},
		},
		{
			"ingress_alias",
			[]string{"describe", "ing", "ing1", "-n", "default"},
			seedIngress,
			[]string{"Name: ing1"},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			cs := fake.NewSimpleClientset(tc.seed())
			r := New(cs, nil, cs.Discovery(), "https://api.test", "")
			out, err := r.Run(ctx, tc.argv)
			if err != nil {
				t.Fatalf("err=%v out=%s", err, out)
			}
			s := string(out)
			for _, w := range tc.wantSub {
				if !strings.Contains(s, w) {
					t.Fatalf("argv=%v: missing %q in output:\n%s", tc.argv, w, s)
				}
			}
		})
	}
}

// TestFormatNamedRows covers the output modes of formatNamedRows.
func TestFormatNamedRows(t *testing.T) {
	t.Parallel()
	row := func(i int) (string, string) {
		names := []string{"a", "b"}
		status := []string{"Ready", "Pending"}
		return names[i], status[i]
	}
	list := &corev1.PodList{Items: nil} // unused, but needed for yaml/json path
	_ = list

	cases := []struct {
		mode    string
		assert  func(*testing.T, string)
		wantErr bool
	}{
		{"name", func(t *testing.T, s string) {
			if !strings.Contains(s, "thing/a") || !strings.Contains(s, "thing/b") {
				t.Fatalf("name output: %q", s)
			}
		}, false},
		{"yaml", func(t *testing.T, s string) {
			if !strings.Contains(s, "metadata:") {
				t.Fatalf("yaml output missing metadata: %q", s)
			}
		}, false},
		{"json", func(t *testing.T, s string) {
			if !strings.Contains(s, "{") || !strings.Contains(s, "}") {
				t.Fatalf("json output: %q", s)
			}
		}, false},
		{"", func(t *testing.T, s string) {
			if !strings.Contains(s, "NAME	STATUS") {
				t.Fatalf("default text missing header: %q", s)
			}
			if !strings.Contains(s, "a	Ready") {
				t.Fatalf("default text missing row: %q", s)
			}
		}, false},
		{"wide", func(t *testing.T, s string) {
			// unknown output mode falls through to default text path.
			if !strings.Contains(s, "NAME	STATUS") {
				t.Fatalf("wide fallback: %q", s)
			}
		}, false},
	}

	for _, tc := range cases {
		tc := tc
		t.Run("mode="+tc.mode, func(t *testing.T) {
			t.Parallel()
			out, err := formatNamedRows(list, tc.mode, false, "thing", 2, row)
			if (err != nil) != tc.wantErr {
				t.Fatalf("err=%v", err)
			}
			tc.assert(t, string(out))
		})
	}

	// Also exercise the no-headers branch on default text.
	out, err := formatNamedRows(list, "", true, "thing", 2, row)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(out), "NAME	STATUS") {
		t.Fatalf("noHeaders=true should suppress header; got %q", out)
	}
}

// TestNamespacedOrAll covers the namespace-resolution helper.
func TestNamespacedOrAll(t *testing.T) {
	t.Parallel()
	if got := namespacedOrAll("", true); got != "" /* NamespaceAll */ {
		if got == "default" {
			t.Fatalf("allNS=true must not return default: %q", got)
		}
	}
	if got := namespacedOrAll("", false); got != "default" {
		t.Fatalf("ns='' allNS=false: %q", got)
	}
	if got := namespacedOrAll("foo", false); got != "foo" {
		t.Fatalf("ns='foo' allNS=false: %q", got)
	}
}

// TestPtrStr covers the *string->string helper.
func TestPtrStr(t *testing.T) {
	t.Parallel()
	if got := ptrStr(nil); got != "<none>" {
		t.Fatalf("nil: %q", got)
	}
	s := "hello"
	if got := ptrStr(&s); got != "hello" {
		t.Fatalf("&s: %q", got)
	}
}

// TestIngressHosts covers ingressHosts across empty, missing, and populated cases.
func TestIngressHosts(t *testing.T) {
	t.Parallel()
	if got := ingressHosts(&networkingv1.Ingress{}); len(got) != 0 {
		t.Fatalf("empty ingress: %v", got)
	}
	in := &networkingv1.Ingress{Spec: networkingv1.IngressSpec{Rules: []networkingv1.IngressRule{
		{Host: ""},
		{Host: "x.example.com"},
		{Host: "y.example.com"},
	}}}
	got := ingressHosts(in)
	if diff := cmp.Diff([]string{"x.example.com", "y.example.com"}, got); diff != "" {
		t.Fatalf("hosts diff (-want +got):\n%s", diff)
	}
}

// TestRunner_Run_get_nodes covers nodesHandler and the underlying getNodes for
// both the list and get-by-name paths.
func TestRunner_Run_get_nodes(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	node := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{Name: "node1"},
		Status: corev1.NodeStatus{
			Conditions: []corev1.NodeCondition{{Type: corev1.NodeReady, Status: corev1.ConditionTrue}},
			NodeInfo:   corev1.NodeSystemInfo{KubeletVersion: "v1.30.0"},
		},
	}
	cs := fake.NewSimpleClientset(node)
	r := New(cs, nil, cs.Discovery(), "https://api.test", "")

	out, err := r.Run(ctx, []string{"get", "nodes"})
	if err != nil || !strings.Contains(string(out), "node1") {
		t.Fatalf("list err=%v out=%s", err, out)
	}
	out, err = r.Run(ctx, []string{"get", "node", "node1", "-o", "yaml"})
	if err != nil || !strings.Contains(string(out), "node1") {
		t.Fatalf("name err=%v", err)
	}
	out, err = r.Run(ctx, []string{"get", "nodes", "-o", "json"})
	if err != nil || !strings.Contains(string(out), "node1") {
		t.Fatalf("json err=%v out=%s", err, out)
	}
}
