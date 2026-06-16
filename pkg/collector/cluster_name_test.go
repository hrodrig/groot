package collector

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/hrodrig/groot/pkg/config"
	"github.com/hrodrig/groot/pkg/kubetest"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
)

func emptyKubeconfig(t *testing.T) string {
	t.Helper()
	kc := filepath.Join(t.TempDir(), "empty-kubeconfig")
	const content = `apiVersion: v1
kind: Config
current-context: ""
contexts: []
clusters: []
users: []
`
	if err := os.WriteFile(kc, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return kc
}

func TestResolveClusterName_configOverride(t *testing.T) {
	svc := New(config.Config{ClusterName: "prod-eks", Kubeconfig: emptyKubeconfig(t)})
	if got := svc.resolveClusterName(context.Background()); got != "prod-eks" {
		t.Fatalf("got %q", got)
	}
}

func TestResolveClusterName_hostFallback(t *testing.T) {
	svc := New(config.Config{Kubeconfig: emptyKubeconfig(t)})
	svc.restConfig = &rest.Config{Host: "https://api.my-cluster.example:6443"}
	if got := svc.resolveClusterName(context.Background()); got != "api.my-cluster.example" {
		t.Fatalf("got %q", got)
	}
}

func TestClusterNameFromHost(t *testing.T) {
	tests := []struct {
		host string
		want string
	}{
		{"https://kubernetes.default.svc", "kubernetes.default.svc"},
		{"https://127.0.0.1:6443", "127.0.0.1"},
		{"", ""},
	}
	for _, tc := range tests {
		if got := clusterNameFromHost(tc.host); got != tc.want {
			t.Fatalf("host %q: got %q want %q", tc.host, got, tc.want)
		}
	}
}

func TestClusterNameFromClusterInfo(t *testing.T) {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "cluster-info", Namespace: "kube-public"},
		Data: map[string]string{
			"kubeconfig": `apiVersion: v1
kind: Config
current-context: default
contexts:
- name: default
  context:
    cluster: my-k8s
    user: user
clusters:
- name: my-k8s
  cluster:
    server: https://kubernetes.default.svc
users:
- name: user
  user: {}
`,
		},
	}
	cs := fake.NewSimpleClientset(cm)
	got, err := clusterNameFromClusterInfo(context.Background(), cs)
	if err != nil {
		t.Fatal(err)
	}
	if got != "my-k8s" {
		t.Fatalf("got %q", got)
	}
}

func TestResolveClusterName_clusterInfoFallback(t *testing.T) {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "cluster-info", Namespace: "kube-public"},
		Data: map[string]string{
			"kubeconfig": `apiVersion: v1
kind: Config
current-context: default
contexts:
- name: default
  context:
    cluster: my-k8s
    user: user
clusters:
- name: my-k8s
  cluster:
    server: https://kubernetes.default.svc
users:
- name: user
  user: {}
`,
		},
	}
	svc := New(config.Config{Kubeconfig: emptyKubeconfig(t)})
	svc.clientset = fake.NewSimpleClientset(cm)
	if got := svc.resolveClusterName(context.Background()); got != "my-k8s" {
		t.Fatalf("got %q", got)
	}
}

func TestClusterNameFromClusterInfo_missingKey(t *testing.T) {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "cluster-info", Namespace: "kube-public"},
		Data:       map[string]string{},
	}
	cs := fake.NewSimpleClientset(cm)
	if _, err := clusterNameFromClusterInfo(context.Background(), cs); err == nil {
		t.Fatal("expected error")
	}
}

func TestRestHostFromKubeconfig(t *testing.T) {
	kc, cleanup := kubetest.StartAPIServer(t)
	defer cleanup()
	if host := restHostFromKubeconfig(kc); host == "" {
		t.Fatal("expected host from kubeconfig")
	}
}

func TestResolveClusterName_kubeconfigMetadata(t *testing.T) {
	dir := t.TempDir()
	kc := filepath.Join(dir, "kc")
	content := `apiVersion: v1
kind: Config
current-context: ctx1
contexts:
- name: ctx1
  context:
    cluster: prod-eks
    user: u1
clusters:
- name: prod-eks
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
	if got := svc.resolveClusterName(context.Background()); got != "prod-eks" {
		t.Fatalf("got %q", got)
	}
}
