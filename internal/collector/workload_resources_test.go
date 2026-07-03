package collector

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

func TestPodResourceTotalsFromPod_sumsContainers(t *testing.T) {
	cpu500m := resource.MustParse("500m")
	cpu1 := resource.MustParse("1")
	mem512Mi := resource.MustParse("512Mi")
	mem4Gi := resource.MustParse("4Gi")

	pod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "apps",
			Name:      "metrics-db-0",
			OwnerReferences: []metav1.OwnerReference{
				{Kind: "StatefulSet", Name: "metrics-db", Controller: ptr.To(true)},
			},
		},
		Spec: corev1.PodSpec{
			NodeName: "node-a",
			Containers: []corev1.Container{{
				Name: "postgres",
				Resources: corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:    cpu500m,
						corev1.ResourceMemory: mem512Mi,
					},
					Limits: corev1.ResourceList{
						corev1.ResourceCPU:    cpu1,
						corev1.ResourceMemory: mem4Gi,
					},
				},
			}},
		},
	}

	totals := podResourceTotalsFromPod(pod)
	if totals.CPURequest != "500m" || totals.CPULimit != "1" {
		t.Fatalf("cpu totals: req=%q lim=%q", totals.CPURequest, totals.CPULimit)
	}
	if totals.MemoryRequest != "512Mi" || totals.MemoryLimit != "4Gi" {
		t.Fatalf("memory totals: req=%q lim=%q", totals.MemoryRequest, totals.MemoryLimit)
	}

	rows := containerResourceRowsFromPod(pod)
	if len(rows) != 1 {
		t.Fatalf("rows = %d, want 1", len(rows))
	}
	if rows[0].OwnerKind != "StatefulSet" || rows[0].OwnerName != "metrics-db" {
		t.Fatalf("owner = %s/%s", rows[0].OwnerKind, rows[0].OwnerName)
	}
	if rows[0].MemoryLimit != "4Gi" {
		t.Fatalf("memory limit = %q", rows[0].MemoryLimit)
	}
}

func TestPodResourceTotalsFromPod_initAndAppContainers(t *testing.T) {
	pod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Namespace: "ns", Name: "pod"},
		Spec: corev1.PodSpec{
			InitContainers: []corev1.Container{{
				Name: "init",
				Resources: corev1.ResourceRequirements{
					Limits: corev1.ResourceList{corev1.ResourceMemory: resource.MustParse("128Mi")},
				},
			}},
			Containers: []corev1.Container{{
				Name: "app",
				Resources: corev1.ResourceRequirements{
					Limits: corev1.ResourceList{corev1.ResourceMemory: resource.MustParse("256Mi")},
				},
			}},
		},
	}

	totals := podResourceTotalsFromPod(pod)
	if totals.MemoryLimit != "384Mi" {
		t.Fatalf("aggregated memory limit = %q, want 384Mi", totals.MemoryLimit)
	}

	rows := containerResourceRowsFromPod(pod)
	if len(rows) != 2 {
		t.Fatalf("rows = %d, want 2", len(rows))
	}
	if !rows[0].InitContainer || rows[1].InitContainer {
		t.Fatalf("init flags: %v, %v", rows[0].InitContainer, rows[1].InitContainer)
	}
}

func TestWriteWorkloadResourcesTSV_headerAndRow(t *testing.T) {
	dir := t.TempDir()
	rows := []containerResourceRow{{
		Namespace: "default", Pod: "p", Node: "n1", Container: "c",
		CPURequest: "100m", MemoryLimit: "1Gi", OwnerKind: "Deployment", OwnerName: "api",
	}}
	if err := writeWorkloadResourcesTSV(dir, rows); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(dir, "extras", "workload-resources.tsv"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "namespace\tpod\tnode\tcontainer") {
		t.Fatalf("header missing: %s", raw)
	}
	if !strings.Contains(string(raw), "default\tp\tn1\tc\tfalse\t100m") {
		t.Fatalf("row missing: %s", raw)
	}
}
