package collector

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

type containerResourceRow struct {
	Namespace     string
	Pod           string
	Node          string
	Container     string
	InitContainer bool
	CPURequest    string
	CPULimit      string
	MemoryRequest string
	MemoryLimit   string
	OwnerKind     string
	OwnerName     string
}

type podResourceTotals struct {
	CPURequest    string
	CPULimit      string
	MemoryRequest string
	MemoryLimit   string
}

func quantityString(q *resource.Quantity) string {
	if q == nil || q.IsZero() {
		return ""
	}
	return q.String()
}

func podControllerOwner(pod *corev1.Pod) (kind, name string) {
	for i := range pod.OwnerReferences {
		ref := pod.OwnerReferences[i]
		if ref.Controller != nil && *ref.Controller {
			return ref.Kind, ref.Name
		}
	}
	if len(pod.OwnerReferences) > 0 {
		ref := pod.OwnerReferences[0]
		return ref.Kind, ref.Name
	}
	return "", ""
}

func containerResourceRowsFromPod(pod corev1.Pod) []containerResourceRow {
	node := strings.TrimSpace(pod.Spec.NodeName)
	if node == "" {
		node = "unknown-node"
	}
	ownerKind, ownerName := podControllerOwner(&pod)

	var rows []containerResourceRow
	appendContainers := func(containers []corev1.Container, init bool) {
		for _, c := range containers {
			reqCPU, reqMem, limCPU, limMem := containerResources(c.Resources)
			rows = append(rows, containerResourceRow{
				Namespace:     pod.Namespace,
				Pod:           pod.Name,
				Node:          node,
				Container:     c.Name,
				InitContainer: init,
				CPURequest:    reqCPU,
				CPULimit:      limCPU,
				MemoryRequest: reqMem,
				MemoryLimit:   limMem,
				OwnerKind:     ownerKind,
				OwnerName:     ownerName,
			})
		}
	}
	appendContainers(pod.Spec.InitContainers, true)
	appendContainers(pod.Spec.Containers, false)
	return rows
}

func containerResources(res corev1.ResourceRequirements) (cpuReq, memReq, cpuLim, memLim string) {
	if res.Requests != nil {
		if q, ok := res.Requests[corev1.ResourceCPU]; ok {
			cpuReq = quantityString(&q)
		}
		if q, ok := res.Requests[corev1.ResourceMemory]; ok {
			memReq = quantityString(&q)
		}
	}
	if res.Limits != nil {
		if q, ok := res.Limits[corev1.ResourceCPU]; ok {
			cpuLim = quantityString(&q)
		}
		if q, ok := res.Limits[corev1.ResourceMemory]; ok {
			memLim = quantityString(&q)
		}
	}
	return cpuReq, memReq, cpuLim, memLim
}

func addQuantity(dst *resource.Quantity, src *resource.Quantity) {
	if src == nil || src.IsZero() {
		return
	}
	if dst.IsZero() {
		*dst = src.DeepCopy()
		return
	}
	dst.Add(*src)
}

func podResourceTotalsFromPod(pod corev1.Pod) podResourceTotals {
	var cpuReq, cpuLim, memReq, memLim resource.Quantity
	addContainerTotals := func(containers []corev1.Container) {
		for _, c := range containers {
			if c.Resources.Requests != nil {
				if q, ok := c.Resources.Requests[corev1.ResourceCPU]; ok {
					addQuantity(&cpuReq, &q)
				}
				if q, ok := c.Resources.Requests[corev1.ResourceMemory]; ok {
					addQuantity(&memReq, &q)
				}
			}
			if c.Resources.Limits != nil {
				if q, ok := c.Resources.Limits[corev1.ResourceCPU]; ok {
					addQuantity(&cpuLim, &q)
				}
				if q, ok := c.Resources.Limits[corev1.ResourceMemory]; ok {
					addQuantity(&memLim, &q)
				}
			}
		}
	}
	addContainerTotals(pod.Spec.InitContainers)
	addContainerTotals(pod.Spec.Containers)
	return podResourceTotals{
		CPURequest:    quantityString(&cpuReq),
		CPULimit:      quantityString(&cpuLim),
		MemoryRequest: quantityString(&memReq),
		MemoryLimit:   quantityString(&memLim),
	}
}

func buildPodResourceTotalsMap(list *corev1.PodList) map[string]podResourceTotals {
	out := make(map[string]podResourceTotals, len(list.Items))
	for i := range list.Items {
		pod := list.Items[i]
		out[pod.Namespace+"/"+pod.Name] = podResourceTotalsFromPod(pod)
	}
	return out
}

func writeWorkloadResourcesTSV(captureDir string, rows []containerResourceRow) error {
	var b strings.Builder
	b.WriteString("namespace\tpod\tnode\tcontainer\tinit_container\tcpu_request\tcpu_limit\tmemory_request\tmemory_limit\towner_kind\towner_name\n")
	for _, row := range rows {
		initFlag := "false"
		if row.InitContainer {
			initFlag = "true"
		}
		b.WriteString(fmt.Sprintf("%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			row.Namespace, row.Pod, row.Node, row.Container, initFlag,
			row.CPURequest, row.CPULimit, row.MemoryRequest, row.MemoryLimit,
			row.OwnerKind, row.OwnerName,
		))
	}
	target := filepath.Join(captureDir, "extras", "workload-resources.tsv")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return fmt.Errorf("create workload-resources dir: %w", err)
	}
	if err := os.WriteFile(target, []byte(b.String()), 0o644); err != nil {
		return fmt.Errorf("write workload-resources.tsv: %w", err)
	}
	return nil
}

func (s *Service) writeWorkloadResourcesTable(ctx context.Context, captureDir string) (map[string]podResourceTotals, error) {
	list, err := s.listAllPods(ctx)
	if err != nil {
		return nil, fmt.Errorf("list pods for workload resources: %w", err)
	}
	var rows []containerResourceRow
	for i := range list.Items {
		rows = append(rows, containerResourceRowsFromPod(list.Items[i])...)
	}
	if err := writeWorkloadResourcesTSV(captureDir, rows); err != nil {
		return nil, err
	}
	return buildPodResourceTotalsMap(list), nil
}
