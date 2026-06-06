package k8srunner

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

func (r *Runner) getConfigMaps(ctx context.Context, ns, name, output, sel string, noHeaders, allNS bool) ([]byte, error) {
	listNS := namespacedOrAll(ns, allNS)
	if name != "" {
		obj, err := r.Core.CoreV1().ConfigMaps(listNS).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		return formatObject(obj, output)
	}
	list, err := r.Core.CoreV1().ConfigMaps(listNS).List(ctx, metav1.ListOptions{LabelSelector: sel})
	if err != nil {
		return nil, err
	}
	return formatNamedRows(list, output, noHeaders, "configmap", len(list.Items), func(i int) (string, string) {
		return list.Items[i].Name, shortAge(list.Items[i].CreationTimestamp.Time)
	})
}

func (r *Runner) getPVCs(ctx context.Context, ns, name, output, sel string, noHeaders, allNS bool) ([]byte, error) {
	listNS := namespacedOrAll(ns, allNS)
	if name != "" {
		obj, err := r.Core.CoreV1().PersistentVolumeClaims(listNS).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		return formatObject(obj, output)
	}
	list, err := r.Core.CoreV1().PersistentVolumeClaims(listNS).List(ctx, metav1.ListOptions{LabelSelector: sel})
	if err != nil {
		return nil, err
	}
	return formatNamedRows(list, output, noHeaders, "pvc", len(list.Items), func(i int) (string, string) {
		item := list.Items[i]
		return item.Name, string(item.Status.Phase)
	})
}

func (r *Runner) getServices(ctx context.Context, ns, name, output, sel string, noHeaders, allNS bool) ([]byte, error) {
	listNS := namespacedOrAll(ns, allNS)
	if name != "" {
		obj, err := r.Core.CoreV1().Services(listNS).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		return formatObject(obj, output)
	}
	list, err := r.Core.CoreV1().Services(listNS).List(ctx, metav1.ListOptions{LabelSelector: sel})
	if err != nil {
		return nil, err
	}
	return formatNamedRows(list, output, noHeaders, "service", len(list.Items), func(i int) (string, string) {
		item := list.Items[i]
		return item.Name, string(item.Spec.Type)
	})
}

func (r *Runner) getIngresses(ctx context.Context, ns, name, output, sel string, noHeaders, allNS bool) ([]byte, error) {
	listNS := namespacedOrAll(ns, allNS)
	if name != "" {
		obj, err := r.Core.NetworkingV1().Ingresses(listNS).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		return formatObject(obj, output)
	}
	list, err := r.Core.NetworkingV1().Ingresses(listNS).List(ctx, metav1.ListOptions{LabelSelector: sel})
	if err != nil {
		return nil, err
	}
	return formatNamedRows(list, output, noHeaders, "ingress", len(list.Items), func(i int) (string, string) {
		return list.Items[i].Name, shortAge(list.Items[i].CreationTimestamp.Time)
	})
}

func (r *Runner) getDeployments(ctx context.Context, ns, name, output, sel string, noHeaders, allNS bool) ([]byte, error) {
	listNS := namespacedOrAll(ns, allNS)
	if name != "" {
		obj, err := r.Core.AppsV1().Deployments(listNS).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		return formatObject(obj, output)
	}
	list, err := r.Core.AppsV1().Deployments(listNS).List(ctx, metav1.ListOptions{LabelSelector: sel})
	if err != nil {
		return nil, err
	}
	return formatNamedRows(list, output, noHeaders, "deployment", len(list.Items), func(i int) (string, string) {
		d := list.Items[i]
		return d.Name, fmt.Sprintf("%d/%d", d.Status.ReadyReplicas, d.Status.Replicas)
	})
}

func (r *Runner) getReplicaSets(ctx context.Context, ns, name, output, sel string, noHeaders, allNS bool) ([]byte, error) {
	listNS := namespacedOrAll(ns, allNS)
	if name != "" {
		obj, err := r.Core.AppsV1().ReplicaSets(listNS).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		return formatObject(obj, output)
	}
	list, err := r.Core.AppsV1().ReplicaSets(listNS).List(ctx, metav1.ListOptions{LabelSelector: sel})
	if err != nil {
		return nil, err
	}
	return formatNamedRows(list, output, noHeaders, "replicaset", len(list.Items), func(i int) (string, string) {
		rs := list.Items[i]
		return rs.Name, fmt.Sprintf("%d/%d", rs.Status.ReadyReplicas, rs.Status.Replicas)
	})
}

func (r *Runner) getStatefulSets(ctx context.Context, ns, name, output, sel string, noHeaders, allNS bool) ([]byte, error) {
	listNS := namespacedOrAll(ns, allNS)
	if name != "" {
		obj, err := r.Core.AppsV1().StatefulSets(listNS).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		return formatObject(obj, output)
	}
	list, err := r.Core.AppsV1().StatefulSets(listNS).List(ctx, metav1.ListOptions{LabelSelector: sel})
	if err != nil {
		return nil, err
	}
	return formatNamedRows(list, output, noHeaders, "statefulset", len(list.Items), func(i int) (string, string) {
		sts := list.Items[i]
		return sts.Name, fmt.Sprintf("%d/%d", sts.Status.ReadyReplicas, sts.Status.Replicas)
	})
}

func (r *Runner) getDaemonSets(ctx context.Context, ns, name, output, sel string, noHeaders, allNS bool) ([]byte, error) {
	listNS := namespacedOrAll(ns, allNS)
	if name != "" {
		obj, err := r.Core.AppsV1().DaemonSets(listNS).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		return formatObject(obj, output)
	}
	list, err := r.Core.AppsV1().DaemonSets(listNS).List(ctx, metav1.ListOptions{LabelSelector: sel})
	if err != nil {
		return nil, err
	}
	return formatNamedRows(list, output, noHeaders, "daemonset", len(list.Items), func(i int) (string, string) {
		ds := list.Items[i]
		return ds.Name, fmt.Sprintf("%d/%d", ds.Status.NumberReady, ds.Status.DesiredNumberScheduled)
	})
}

func (r *Runner) describeConfigMap(ctx context.Context, ns, name string) ([]byte, error) {
	if ns == "" {
		ns = metav1.NamespaceDefault
	}
	obj, err := r.Core.CoreV1().ConfigMaps(ns).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return []byte(fmt.Sprintf("Name: %s\nNamespace: %s\nData keys: %d\n", obj.Name, obj.Namespace, len(obj.Data))), nil
}

func (r *Runner) describePVC(ctx context.Context, ns, name string) ([]byte, error) {
	if ns == "" {
		ns = metav1.NamespaceDefault
	}
	obj, err := r.Core.CoreV1().PersistentVolumeClaims(ns).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return []byte(fmt.Sprintf("Name: %s\nNamespace: %s\nPhase: %s\nStorageClass: %s\n",
		obj.Name, obj.Namespace, obj.Status.Phase, ptrStr(obj.Spec.StorageClassName))), nil
}

func (r *Runner) describeService(ctx context.Context, ns, name string) ([]byte, error) {
	if ns == "" {
		ns = metav1.NamespaceDefault
	}
	obj, err := r.Core.CoreV1().Services(ns).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return []byte(fmt.Sprintf("Name: %s\nNamespace: %s\nType: %s\nClusterIP: %s\n",
		obj.Name, obj.Namespace, obj.Spec.Type, obj.Spec.ClusterIP)), nil
}

func (r *Runner) describeIngress(ctx context.Context, ns, name string) ([]byte, error) {
	if ns == "" {
		ns = metav1.NamespaceDefault
	}
	obj, err := r.Core.NetworkingV1().Ingresses(ns).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	hosts := ingressHosts(obj)
	return []byte(fmt.Sprintf("Name: %s\nNamespace: %s\nHosts: %s\n",
		obj.Name, obj.Namespace, strings.Join(hosts, ","))), nil
}

func ingressHosts(in *networkingv1.Ingress) []string {
	var hosts []string
	for _, rule := range in.Spec.Rules {
		if rule.Host != "" {
			hosts = append(hosts, rule.Host)
		}
	}
	return hosts
}

func ptrStr(p *string) string {
	if p == nil {
		return "<none>"
	}
	return *p
}

func namespacedOrAll(ns string, allNS bool) string {
	if allNS {
		return metav1.NamespaceAll
	}
	if ns == "" {
		return metav1.NamespaceDefault
	}
	return ns
}

func formatNamedRows(list any, output string, noHeaders bool, kind string, n int, row func(i int) (name, extra string)) ([]byte, error) {
	switch output {
	case "yaml":
		return yaml.Marshal(list)
	case "json":
		return json.MarshalIndent(list, "", "  ")
	case "name":
		var b strings.Builder
		for i := 0; i < n; i++ {
			name, _ := row(i)
			fmt.Fprintf(&b, "%s/%s\n", kind, name)
		}
		return []byte(b.String()), nil
	default:
		var b strings.Builder
		if !noHeaders {
			fmt.Fprintf(&b, "NAME\tSTATUS\n")
		}
		for i := 0; i < n; i++ {
			name, extra := row(i)
			fmt.Fprintf(&b, "%s\t%s\n", name, extra)
		}
		return []byte(b.String()), nil
	}
}
