// Package k8srunner executes allowlisted read-only kubectl-style argv slices using client-go.
package k8srunner

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"time"

	authorizationv1 "k8s.io/api/authorization/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/kubernetes"
	metricsv1beta1 "k8s.io/metrics/pkg/apis/metrics/v1beta1"
	metricsversioned "k8s.io/metrics/pkg/client/clientset/versioned"
	"sigs.k8s.io/yaml"

	"github.com/hrodrig/groot/internal/kubeloader"
)

// Runner holds Kubernetes API clients for extra_kubectl execution.
type Runner struct {
	Core           kubernetes.Interface
	Metrics        metricsversioned.Interface
	Discovery      discovery.DiscoveryInterface
	Host           string
	KubeconfigPath string
}

// New builds a runner. Metrics may be nil. Host is rest.Config.Host.
func New(core kubernetes.Interface, metrics metricsversioned.Interface, disc discovery.DiscoveryInterface, host, kubeconfigPath string) *Runner {
	return &Runner{
		Core:           core,
		Metrics:        metrics,
		Discovery:      disc,
		Host:           host,
		KubeconfigPath: kubeconfigPath,
	}
}

// Run executes argv like kubectl (without the binary name).
func (r *Runner) Run(ctx context.Context, argv []string) ([]byte, error) {
	if len(argv) == 0 {
		return nil, fmt.Errorf("empty command")
	}
	switch strings.ToLower(argv[0]) {
	case "get":
		return r.runGet(ctx, argv[1:])
	case "describe":
		return r.runDescribe(ctx, argv[1:])
	case "explain", "wait":
		return nil, fmt.Errorf("%q is not supported for extra_kubectl in this build", argv[0])
	case "top":
		return r.runTop(ctx, argv[1:])
	case "logs":
		return r.runLogs(ctx, argv[1:])
	case "api-resources":
		return r.runAPIResources(ctx)
	case "api-versions":
		return r.runAPIVersions(ctx)
	case "version":
		return r.runVersion(ctx)
	case "cluster-info":
		return r.runClusterInfo(ctx)
	case "config":
		return r.runConfig(argv[1:])
	case "auth":
		return r.runAuth(ctx, argv[1:])
	default:
		return nil, fmt.Errorf("unsupported command %q", argv[0])
	}
}

func (r *Runner) runVersion(ctx context.Context) ([]byte, error) {
	v, err := r.Discovery.ServerVersion()
	if err != nil {
		return nil, err
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Client Version: client-go\n")
	fmt.Fprintf(&b, "Server Version: %s\n", v.GitVersion)
	return []byte(b.String()), nil
}

func (r *Runner) runClusterInfo(ctx context.Context) ([]byte, error) {
	v, err := r.Discovery.ServerVersion()
	if err != nil {
		return nil, err
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Kubernetes control plane is running at %s\n", r.Host)
	fmt.Fprintf(&b, "Core control plane version %s\n", v.GitVersion)
	return []byte(b.String()), nil
}

func (r *Runner) runAPIVersions(ctx context.Context) ([]byte, error) {
	groups, err := r.Discovery.ServerGroups()
	if err != nil {
		return nil, err
	}
	var b strings.Builder
	for _, g := range groups.Groups {
		for _, ver := range g.Versions {
			fmt.Fprintf(&b, "%s/%s\n", g.Name, ver.Version)
		}
	}
	return []byte(b.String()), nil
}

func (r *Runner) runAPIResources(ctx context.Context) ([]byte, error) {
	lists, err := r.Discovery.ServerPreferredResources()
	if err != nil {
		return nil, err
	}
	var b strings.Builder
	fmt.Fprintf(&b, "NAME\tSHORTNAMES\tAPIVERSION\tNAMESPACED\tKIND\n")
	for _, list := range lists {
		gv, err := schema.ParseGroupVersion(list.GroupVersion)
		if err != nil {
			continue
		}
		for _, rs := range list.APIResources {
			if len(rs.Verbs) == 0 || !containsVerb(rs.Verbs, "list") {
				continue
			}
			ns := "false"
			if rs.Namespaced {
				ns = "true"
			}
			short := strings.Join(rs.ShortNames, ",")
			fmt.Fprintf(&b, "%s\t%s\t%s\t%s\t%s\n", rs.Name, short, gv.String(), ns, rs.Kind)
		}
	}
	return []byte(b.String()), nil
}

func containsVerb(vs []string, want string) bool {
	for _, v := range vs {
		if v == want {
			return true
		}
	}
	return false
}

func (r *Runner) runConfig(argv []string) ([]byte, error) {
	if len(argv) < 1 || !strings.EqualFold(argv[0], "view") {
		return nil, fmt.Errorf("config: only \"config view\" is supported")
	}
	raw, err := kubeloader.APIConfig(r.KubeconfigPath)
	if err != nil {
		return nil, err
	}
	if wantsYAML(argv[1:]) {
		b, err := yaml.Marshal(raw)
		return b, err
	}
	return json.MarshalIndent(raw, "", "  ")
}

func wantsYAML(flags []string) bool {
	for i := 0; i < len(flags); i++ {
		a := flags[i]
		if a == "-o" || a == "--output" {
			if i+1 < len(flags) && flags[i+1] == "yaml" {
				return true
			}
		}
		if strings.HasPrefix(a, "-o=yaml") || strings.HasPrefix(a, "--output=yaml") {
			return true
		}
	}
	return false
}

func (r *Runner) runAuth(ctx context.Context, argv []string) ([]byte, error) {
	if len(argv) < 2 || !strings.EqualFold(argv[0], "can-i") {
		return nil, fmt.Errorf("auth: only \"auth can-i\" is supported")
	}
	f := parseFlags(argv[1:])
	if len(f.args) < 2 {
		return nil, fmt.Errorf("auth can-i: need verb and resource")
	}
	verb, resource := f.args[0], f.args[1]
	sar := &authorizationv1.SelfSubjectAccessReview{
		Spec: authorizationv1.SelfSubjectAccessReviewSpec{
			ResourceAttributes: &authorizationv1.ResourceAttributes{
				Namespace: f.namespace,
				Verb:      verb,
				Resource:  resource,
			},
		},
	}
	resp, err := r.Core.AuthorizationV1().SelfSubjectAccessReviews().Create(ctx, sar, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}
	if resp.Status.Allowed {
		return []byte("yes\n"), nil
	}
	return []byte("no\n"), nil
}

type flagSet struct {
	namespace     string
	allNamespaces bool
	output        string
	labelSelector string
	noHeaders     bool
	args          []string
}

func parseFlags(argv []string) flagSet {
	var f flagSet
	out := make([]string, 0, len(argv))
	for i := 0; i < len(argv); i++ {
		if ni, ok := f.tryParseOneFlag(argv, i); ok {
			i = ni
			continue
		}
		out = append(out, argv[i])
	}
	f.args = out
	return f
}

func (f *flagSet) tryParseOneFlag(argv []string, i int) (int, bool) {
	a := argv[i]
	if a == "-A" || a == "--all-namespaces" {
		f.allNamespaces = true
		return i, true
	}
	if ni, ok := f.parseNamespaceFlag(argv, i, a); ok {
		return ni, true
	}
	if ni, ok := f.parseOutputFlag(argv, i, a); ok {
		return ni, true
	}
	if ni, ok := f.parseSelectorFlag(argv, i, a); ok {
		return ni, true
	}
	if a == "--no-headers" {
		f.noHeaders = true
		return i, true
	}
	if ni, ok := skipOptionalNextArg(argv, i, a, "--sort-by", "--sort-by="); ok {
		return ni, true
	}
	if ni, ok := skipOptionalNextArg(argv, i, a, "--request-timeout", "--request-timeout="); ok {
		return ni, true
	}
	return i, false
}

func (f *flagSet) parseNamespaceFlag(argv []string, i int, a string) (int, bool) {
	if v, ok := strings.CutPrefix(a, "-n="); ok {
		f.namespace = v
		return i, true
	}
	if v, ok := strings.CutPrefix(a, "--namespace="); ok {
		f.namespace = v
		return i, true
	}
	if a != "-n" && a != "--namespace" {
		return i, false
	}
	if i+1 < len(argv) {
		f.namespace = argv[i+1]
		return i + 1, true
	}
	return i, true
}

func (f *flagSet) parseOutputFlag(argv []string, i int, a string) (int, bool) {
	if v, ok := strings.CutPrefix(a, "-o="); ok {
		f.output = v
		return i, true
	}
	if v, ok := strings.CutPrefix(a, "--output="); ok {
		f.output = v
		return i, true
	}
	if a != "-o" && a != "--output" {
		return i, false
	}
	if i+1 < len(argv) {
		f.output = argv[i+1]
		return i + 1, true
	}
	return i, true
}

func (f *flagSet) parseSelectorFlag(argv []string, i int, a string) (int, bool) {
	if v, ok := strings.CutPrefix(a, "-l="); ok {
		f.labelSelector = v
		return i, true
	}
	if v, ok := strings.CutPrefix(a, "--selector="); ok {
		f.labelSelector = v
		return i, true
	}
	if a != "-l" && a != "--selector" {
		return i, false
	}
	if i+1 < len(argv) {
		f.labelSelector = argv[i+1]
		return i + 1, true
	}
	return i, true
}

func skipOptionalNextArg(argv []string, i int, a, exact, withEqPrefix string) (int, bool) {
	if a == exact {
		if i+1 < len(argv) {
			return i + 1, true
		}
		return i, true
	}
	if strings.HasPrefix(a, withEqPrefix) {
		return i, true
	}
	return i, false
}

func (r *Runner) runGet(ctx context.Context, argv []string) ([]byte, error) {
	f := parseFlags(argv)
	if len(f.args) == 0 {
		return nil, fmt.Errorf("get: missing resource")
	}
	if f.args[0] == "--raw" && len(f.args) > 1 {
		rawPath := f.args[1]
		return r.Core.CoreV1().RESTClient().Get().AbsPath(rawPath).DoRaw(ctx)
	}
	res := f.args[0]
	name := ""
	if len(f.args) > 1 && !strings.HasPrefix(f.args[1], "-") {
		name = f.args[1]
	}
	ns := f.namespace
	if f.allNamespaces {
		ns = metav1.NamespaceAll
	}
	res = strings.ToLower(res)
	if handler, ok := getDispatch()[res]; ok {
		return handler(r, ctx, getArgs{ns: ns, name: name, flags: f})
	}
	return nil, fmt.Errorf("get %s: resource not supported in extra_kubectl (supported: pods, nodes, namespaces, events, configmap, pvc, service, ingress, apps workloads, --raw)", res)
}

type getArgs struct {
	ns    string
	name  string
	flags flagSet
}

type getHandler func(*Runner, context.Context, getArgs) ([]byte, error)

func getDispatch() map[string]getHandler {
	return map[string]getHandler{
		"ns": namespaceHandler, "namespace": namespaceHandler, "namespaces": namespaceHandler,
		"nodes": nodesHandler, "node": nodesHandler,
		"pods": podsHandler, "pod": podsHandler,
		"events": eventsHandler, "event": eventsHandler,
		"configmap": configMapHandler, "configmaps": configMapHandler, "cm": configMapHandler,
		"pvc": pvcHandler, "persistentvolumeclaim": pvcHandler, "persistentvolumeclaims": pvcHandler,
		"svc": serviceHandler, "service": serviceHandler, "services": serviceHandler,
		"ingress": ingressHandler, "ingresses": ingressHandler, "ing": ingressHandler,
		"deployment": appsDeploymentHandler, "deployments": appsDeploymentHandler, "deploy": appsDeploymentHandler,
		"replicaset": appsReplicaSetHandler, "replicasets": appsReplicaSetHandler, "rs": appsReplicaSetHandler,
		"statefulset": appsStatefulSetHandler, "statefulsets": appsStatefulSetHandler, "sts": appsStatefulSetHandler,
		"daemonset": appsDaemonSetHandler, "daemonsets": appsDaemonSetHandler, "ds": appsDaemonSetHandler,
	}
}

func namespaceHandler(r *Runner, ctx context.Context, a getArgs) ([]byte, error) {
	return r.getNamespaces(ctx, a.name, a.flags.output, a.flags.noHeaders)
}
func nodesHandler(r *Runner, ctx context.Context, a getArgs) ([]byte, error) {
	return r.getNodes(ctx, a.name, a.flags.output, a.flags.labelSelector, a.flags.noHeaders)
}
func podsHandler(r *Runner, ctx context.Context, a getArgs) ([]byte, error) {
	return r.getPods(ctx, a.ns, a.name, a.flags.output, a.flags.labelSelector, a.flags.noHeaders, a.flags.allNamespaces)
}
func eventsHandler(r *Runner, ctx context.Context, a getArgs) ([]byte, error) {
	return r.getEvents(ctx, a.ns, a.flags.allNamespaces, a.flags.output, a.flags.labelSelector, a.flags.noHeaders)
}
func configMapHandler(r *Runner, ctx context.Context, a getArgs) ([]byte, error) {
	return r.getConfigMaps(ctx, a.ns, a.name, a.flags.output, a.flags.labelSelector, a.flags.noHeaders, a.flags.allNamespaces)
}
func pvcHandler(r *Runner, ctx context.Context, a getArgs) ([]byte, error) {
	return r.getPVCs(ctx, a.ns, a.name, a.flags.output, a.flags.labelSelector, a.flags.noHeaders, a.flags.allNamespaces)
}
func serviceHandler(r *Runner, ctx context.Context, a getArgs) ([]byte, error) {
	return r.getServices(ctx, a.ns, a.name, a.flags.output, a.flags.labelSelector, a.flags.noHeaders, a.flags.allNamespaces)
}
func ingressHandler(r *Runner, ctx context.Context, a getArgs) ([]byte, error) {
	return r.getIngresses(ctx, a.ns, a.name, a.flags.output, a.flags.labelSelector, a.flags.noHeaders, a.flags.allNamespaces)
}
func appsDeploymentHandler(r *Runner, ctx context.Context, a getArgs) ([]byte, error) {
	return r.getDeployments(ctx, a.ns, a.name, a.flags.output, a.flags.labelSelector, a.flags.noHeaders, a.flags.allNamespaces)
}
func appsReplicaSetHandler(r *Runner, ctx context.Context, a getArgs) ([]byte, error) {
	return r.getReplicaSets(ctx, a.ns, a.name, a.flags.output, a.flags.labelSelector, a.flags.noHeaders, a.flags.allNamespaces)
}
func appsStatefulSetHandler(r *Runner, ctx context.Context, a getArgs) ([]byte, error) {
	return r.getStatefulSets(ctx, a.ns, a.name, a.flags.output, a.flags.labelSelector, a.flags.noHeaders, a.flags.allNamespaces)
}
func appsDaemonSetHandler(r *Runner, ctx context.Context, a getArgs) ([]byte, error) {
	return r.getDaemonSets(ctx, a.ns, a.name, a.flags.output, a.flags.labelSelector, a.flags.noHeaders, a.flags.allNamespaces)
}

func (r *Runner) getNamespaces(ctx context.Context, name, output string, noHeaders bool) ([]byte, error) {
	if name != "" {
		obj, err := r.Core.CoreV1().Namespaces().Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		return formatObject(obj, output)
	}
	list, err := r.Core.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	var b strings.Builder
	switch output {
	case "name":
		for _, n := range list.Items {
			fmt.Fprintf(&b, "namespace/%s\n", n.Name)
		}
		return []byte(b.String()), nil
	case "yaml":
		return yaml.Marshal(list)
	case "json":
		return json.MarshalIndent(list, "", "  ")
	default:
		if !noHeaders {
			fmt.Fprintf(&b, "NAME\tSTATUS\tAGE\n")
		}
		for _, n := range list.Items {
			age := shortAge(n.CreationTimestamp.Time)
			fmt.Fprintf(&b, "%s\t%s\t%s\n", n.Name, n.Status.Phase, age)
		}
		return []byte(b.String()), nil
	}
}

func (r *Runner) getNodes(ctx context.Context, name, output, sel string, noHeaders bool) ([]byte, error) {
	if name != "" {
		obj, err := r.Core.CoreV1().Nodes().Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		return formatObject(obj, output)
	}
	list, err := r.Core.CoreV1().Nodes().List(ctx, metav1.ListOptions{LabelSelector: sel})
	if err != nil {
		return nil, err
	}
	var b strings.Builder
	switch output {
	case "name":
		for _, n := range list.Items {
			fmt.Fprintf(&b, "node/%s\n", n.Name)
		}
		return []byte(b.String()), nil
	case "yaml":
		return yaml.Marshal(list)
	case "json":
		return json.MarshalIndent(list, "", "  ")
	default:
		if !noHeaders {
			fmt.Fprintf(&b, "NAME\tSTATUS\tROLES\tAGE\tVERSION\n")
		}
		for _, n := range list.Items {
			ready := "NotReady"
			for _, c := range n.Status.Conditions {
				if c.Type == corev1.NodeReady && c.Status == corev1.ConditionTrue {
					ready = "Ready"
					break
				}
			}
			ver := n.Status.NodeInfo.KubeletVersion
			age := shortAge(n.CreationTimestamp.Time)
			fmt.Fprintf(&b, "%s\t%s\t\t%s\t%s\n", n.Name, ready, age, ver)
		}
		return []byte(b.String()), nil
	}
}

func (r *Runner) getPods(ctx context.Context, ns, name, output, sel string, noHeaders, allNS bool) ([]byte, error) {
	listNS := podListNamespace(ns, allNS)
	if name != "" {
		obj, err := r.Core.CoreV1().Pods(listNS).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		return formatObject(obj, output)
	}
	list, err := r.Core.CoreV1().Pods(listNS).List(ctx, metav1.ListOptions{LabelSelector: sel})
	if err != nil {
		return nil, err
	}
	return formatPodListText(list, output, noHeaders)
}

func podListNamespace(ns string, allNS bool) string {
	if allNS {
		return metav1.NamespaceAll
	}
	if ns == "" {
		return metav1.NamespaceDefault
	}
	return ns
}

func formatPodListText(list *corev1.PodList, output string, noHeaders bool) ([]byte, error) {
	switch output {
	case "wide":
		return []byte(podListWideText(list, noHeaders)), nil
	case "name":
		return []byte(podListNameText(list)), nil
	case "yaml":
		return yaml.Marshal(list)
	case "json":
		return json.MarshalIndent(list, "", "  ")
	default:
		return []byte(podListDefaultText(list, noHeaders)), nil
	}
}

func podRestartCount(p *corev1.Pod) int64 {
	var n int64
	for _, cs := range p.Status.ContainerStatuses {
		n += int64(cs.RestartCount)
	}
	return n
}

func podListWideText(list *corev1.PodList, noHeaders bool) string {
	var b strings.Builder
	if !noHeaders {
		fmt.Fprintf(&b, "NAMESPACE\tNAME\tREADY\tSTATUS\tRESTARTS\tAGE\tIP\tNODE\n")
	}
	for _, p := range list.Items {
		node := p.Spec.NodeName
		if node == "" {
			node = "<none>"
		}
		ip := p.Status.PodIP
		if ip == "" {
			ip = "<none>"
		}
		age := shortAge(p.CreationTimestamp.Time)
		phase := string(p.Status.Phase)
		fmt.Fprintf(&b, "%s\t%s\t%s\t%s\t%d\t%s\t%s\t%s\n",
			p.Namespace, p.Name, podReadyString(&p), phase, podRestartCount(&p), age, ip, node)
	}
	return b.String()
}

func podListDefaultText(list *corev1.PodList, noHeaders bool) string {
	var b strings.Builder
	if !noHeaders {
		fmt.Fprintf(&b, "NAME\tREADY\tSTATUS\tRESTARTS\tAGE\n")
	}
	for _, p := range list.Items {
		age := shortAge(p.CreationTimestamp.Time)
		phase := string(p.Status.Phase)
		fmt.Fprintf(&b, "%s\t%s\t%s\t%d\t%s\n", p.Name, podReadyString(&p), phase, podRestartCount(&p), age)
	}
	return b.String()
}

func podListNameText(list *corev1.PodList) string {
	var b strings.Builder
	for _, p := range list.Items {
		fmt.Fprintf(&b, "pod/%s\n", p.Name)
	}
	return b.String()
}

func podReadyString(p *corev1.Pod) string {
	total := len(p.Spec.Containers)
	ready := 0
	for _, c := range p.Spec.Containers {
		for _, st := range p.Status.ContainerStatuses {
			if st.Name == c.Name && st.Ready {
				ready++
				break
			}
		}
	}
	if total == 0 {
		return "0/0"
	}
	return fmt.Sprintf("%d/%d", ready, total)
}

func (r *Runner) getEvents(ctx context.Context, ns string, allNS bool, output, sel string, noHeaders bool) ([]byte, error) {
	var events []corev1.Event
	if allNS {
		nss, err := r.Core.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
		if err != nil {
			return nil, err
		}
		for _, n := range nss.Items {
			el, err := r.Core.CoreV1().Events(n.Name).List(ctx, metav1.ListOptions{LabelSelector: sel})
			if err != nil {
				continue
			}
			events = append(events, el.Items...)
		}
	} else {
		if ns == "" {
			ns = metav1.NamespaceDefault
		}
		el, err := r.Core.CoreV1().Events(ns).List(ctx, metav1.ListOptions{LabelSelector: sel})
		if err != nil {
			return nil, err
		}
		events = el.Items
	}
	sort.Slice(events, func(i, j int) bool {
		return eventTime(events[i]).Before(eventTime(events[j]))
	})
	var b strings.Builder
	if !noHeaders {
		fmt.Fprintf(&b, "LAST SEEN\tTYPE\tREASON\tOBJECT\tMESSAGE\n")
	}
	for _, e := range events {
		ts := eventTime(e).Format(time.RFC3339)
		obj := fmt.Sprintf("%s/%s", strings.ToLower(e.InvolvedObject.Kind), e.InvolvedObject.Name)
		msg := strings.ReplaceAll(e.Message, "\n", " ")
		fmt.Fprintf(&b, "%s\t%s\t%s\t%s\t%s\n", ts, e.Type, e.Reason, obj, msg)
	}
	return []byte(b.String()), nil
}

func eventTime(e corev1.Event) time.Time {
	if e.Series != nil && !e.Series.LastObservedTime.IsZero() {
		return e.Series.LastObservedTime.Time
	}
	if !e.LastTimestamp.IsZero() {
		return e.LastTimestamp.Time
	}
	return e.EventTime.Time
}

func formatObject(obj any, output string) ([]byte, error) {
	switch output {
	case "yaml":
		return yaml.Marshal(obj)
	case "json", "":
		return json.MarshalIndent(obj, "", "  ")
	default:
		return json.MarshalIndent(obj, "", "  ")
	}
}

func shortAge(t time.Time) string {
	if t.IsZero() {
		return "unknown"
	}
	d := time.Since(t)
	if d < time.Minute {
		return "<1m"
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh", int(d.Hours()))
	}
	return fmt.Sprintf("%dd", int(d.Hours()/24))
}

func splitKindName(s string) (kind, name string) {
	parts := strings.SplitN(s, "/", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return "", ""
}

type describeArgs struct {
	ns   string
	name string
}

type describeHandler func(*Runner, context.Context, describeArgs) ([]byte, error)

func describeDispatch() map[string]describeHandler {
	return map[string]describeHandler{
		"pod": describePodHandler, "pods": describePodHandler,
		"node": describeNodeHandler, "nodes": describeNodeHandler,
		"configmap": describeConfigMapHandler, "configmaps": describeConfigMapHandler, "cm": describeConfigMapHandler,
		"pvc": describePVCHandler, "persistentvolumeclaim": describePVCHandler, "persistentvolumeclaims": describePVCHandler,
		"svc": describeServiceHandler, "service": describeServiceHandler, "services": describeServiceHandler,
		"ingress": describeIngressHandler, "ingresses": describeIngressHandler, "ing": describeIngressHandler,
	}
}

func (r *Runner) runDescribe(ctx context.Context, argv []string) ([]byte, error) {
	f := parseFlags(argv)
	if len(f.args) < 1 {
		return nil, fmt.Errorf("describe: missing resource")
	}
	kind, name := splitKindName(f.args[0])
	if name == "" && len(f.args) >= 2 {
		kind, name = f.args[0], f.args[1]
	}
	if kind == "" || name == "" {
		return nil, fmt.Errorf("describe: could not parse resource/name from %v", f.args)
	}
	if handler, ok := describeDispatch()[strings.ToLower(kind)]; ok {
		return handler(r, ctx, describeArgs{ns: f.namespace, name: name})
	}
	return nil, fmt.Errorf("describe %s: not supported in extra_kubectl (supported: pod, node, configmap, pvc, service, ingress)", kind)
}

func describePodHandler(r *Runner, ctx context.Context, a describeArgs) ([]byte, error) {
	ns := a.ns
	if ns == "" {
		ns = metav1.NamespaceDefault
	}
	obj, err := r.Core.CoreV1().Pods(ns).Get(ctx, a.name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return []byte(fmt.Sprintf("Name: %s\nNamespace: %s\nNode: %s\nPhase: %s\n", obj.Name, obj.Namespace, obj.Spec.NodeName, obj.Status.Phase)), nil
}

func describeNodeHandler(r *Runner, ctx context.Context, a describeArgs) ([]byte, error) {
	obj, err := r.Core.CoreV1().Nodes().Get(ctx, a.name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Name: %s\n", obj.Name)
	fmt.Fprintf(&b, "Labels: %v\n", obj.Labels)
	fmt.Fprintf(&b, "Addresses: %v\n", obj.Status.Addresses)
	return []byte(b.String()), nil
}

func describeConfigMapHandler(r *Runner, ctx context.Context, a describeArgs) ([]byte, error) {
	return r.describeConfigMap(ctx, a.ns, a.name)
}
func describePVCHandler(r *Runner, ctx context.Context, a describeArgs) ([]byte, error) {
	return r.describePVC(ctx, a.ns, a.name)
}
func describeServiceHandler(r *Runner, ctx context.Context, a describeArgs) ([]byte, error) {
	return r.describeService(ctx, a.ns, a.name)
}
func describeIngressHandler(r *Runner, ctx context.Context, a describeArgs) ([]byte, error) {
	return r.describeIngress(ctx, a.ns, a.name)
}

func (r *Runner) runTop(ctx context.Context, argv []string) ([]byte, error) {
	if r.Metrics == nil {
		return nil, fmt.Errorf("metrics API client unavailable (install metrics-server)")
	}
	f := parseFlags(argv)
	if len(f.args) < 1 {
		return nil, fmt.Errorf("top: missing resource")
	}
	switch strings.ToLower(f.args[0]) {
	case "pods", "pod":
		list, err := r.Metrics.MetricsV1beta1().PodMetricses(metav1.NamespaceAll).List(ctx, metav1.ListOptions{})
		if err != nil {
			return nil, err
		}
		var b strings.Builder
		fmt.Fprintf(&b, "NAMESPACE\tNAME\tCPU(cores)\tMEMORY(bytes)\n")
		for _, m := range list.Items {
			cpu, mem := sumPodMetrics(&m)
			fmt.Fprintf(&b, "%s\t%s\t%s\t%s\n", m.Namespace, m.Name, cpu, mem)
		}
		return []byte(b.String()), nil
	case "nodes", "node":
		if len(f.args) < 2 {
			return nil, fmt.Errorf("top node: need node name")
		}
		nm, err := r.Metrics.MetricsV1beta1().NodeMetricses().Get(ctx, f.args[1], metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		cpu, mem := sumNodeMetrics(nm)
		var b strings.Builder
		fmt.Fprintf(&b, "NAME\tCPU(cores)\tCPU%%\tMEMORY(bytes)\tMEMORY%%\n")
		fmt.Fprintf(&b, "%s\t%s\t\t%s\t\n", nm.Name, cpu, mem)
		return []byte(b.String()), nil
	default:
		return nil, fmt.Errorf("top %s: not supported", f.args[0])
	}
}

func sumPodMetrics(m *metricsv1beta1.PodMetrics) (cpu, mem string) {
	var cpuMilli int64
	var memBytes int64
	for _, c := range m.Containers {
		if q, ok := c.Usage[corev1.ResourceCPU]; ok {
			cpuMilli += q.MilliValue()
		}
		if q, ok := c.Usage[corev1.ResourceMemory]; ok {
			memBytes += q.Value()
		}
	}
	return fmt.Sprintf("%dm", cpuMilli), formatMem(memBytes)
}

func formatMem(b int64) string {
	const mi = 1024 * 1024
	if b >= mi {
		return fmt.Sprintf("%dMi", b/mi)
	}
	return fmt.Sprintf("%d", b)
}

func sumNodeMetrics(nm *metricsv1beta1.NodeMetrics) (cpu, mem string) {
	var cpuMilli int64
	var memBytes int64
	if q, ok := nm.Usage[corev1.ResourceCPU]; ok {
		cpuMilli = q.MilliValue()
	}
	if q, ok := nm.Usage[corev1.ResourceMemory]; ok {
		memBytes = q.Value()
	}
	return fmt.Sprintf("%dm", cpuMilli), formatMem(memBytes)
}

type logArgvState struct {
	ns            string
	pod           string
	allContainers bool
	opts          *corev1.PodLogOptions
}

func parseLogArgv(argv []string) logArgvState {
	st := logArgvState{opts: &corev1.PodLogOptions{Timestamps: true}}
	for i := 0; i < len(argv); i++ {
		if ni, ok := applyLogFlag(argv, i, argv[i], &st); ok {
			i = ni
		}
	}
	return st
}

func applyLogFlag(argv []string, i int, a string, st *logArgvState) (int, bool) {
	if ni, ok := applyLogNamespace(argv, i, a, st); ok {
		return ni, true
	}
	if applyLogSwitches(a, st) {
		return i, true
	}
	if ni, ok := applyLogTail(argv, i, a, st.opts); ok {
		return ni, true
	}
	if strings.HasPrefix(a, "--since=") {
		applySinceSeconds(strings.TrimPrefix(a, "--since="), st.opts)
		return i, true
	}
	if !strings.HasPrefix(a, "-") && st.pod == "" {
		st.pod = a
		return i, true
	}
	return i, false
}

func applyLogNamespace(argv []string, i int, a string, st *logArgvState) (int, bool) {
	if a == "-n" || a == "--namespace" {
		if i+1 < len(argv) {
			st.ns = argv[i+1]
			return i + 1, true
		}
		return i, true
	}
	if v, ok := strings.CutPrefix(a, "-n="); ok {
		st.ns = v
		return i, true
	}
	if v, ok := strings.CutPrefix(a, "--namespace="); ok {
		st.ns = v
		return i, true
	}
	return i, false
}

func applyLogSwitches(a string, st *logArgvState) bool {
	switch a {
	case "-p", "--previous":
		st.opts.Previous = true
		return true
	case "--all-containers=true", "--all-containers":
		st.allContainers = true
		return true
	case "--timestamps=true", "--timestamps":
		st.opts.Timestamps = true
		return true
	default:
		return false
	}
}

func applyLogTail(argv []string, i int, a string, opts *corev1.PodLogOptions) (int, bool) {
	if a == "--tail" {
		if i+1 < len(argv) {
			setTailLines(argv[i+1], opts)
			return i + 1, true
		}
		return i, true
	}
	if v, ok := strings.CutPrefix(a, "--tail="); ok {
		setTailLines(v, opts)
		return i, true
	}
	return i, false
}

func setTailLines(s string, opts *corev1.PodLogOptions) {
	n, _ := strconv.Atoi(s)
	if n <= 0 {
		return
	}
	t := int64(n)
	opts.TailLines = &t
}

func applySinceSeconds(since string, opts *corev1.PodLogOptions) {
	d, err := time.ParseDuration(since)
	if err != nil {
		return
	}
	sec := int64(d.Round(time.Second) / time.Second)
	if sec > 0 {
		opts.SinceSeconds = &sec
	}
}

func (r *Runner) runLogs(ctx context.Context, argv []string) ([]byte, error) {
	st := parseLogArgv(argv)
	if st.pod == "" {
		return nil, fmt.Errorf("logs: need pod name")
	}
	ns := st.ns
	if ns == "" {
		ns = metav1.NamespaceDefault
	}
	if st.allContainers {
		return r.podLogsAllContainers(ctx, ns, st.pod, st.opts)
	}
	return r.podLogsStream(ctx, ns, st.pod, st.opts)
}

func (r *Runner) podLogsAllContainers(ctx context.Context, ns, pod string, opts *corev1.PodLogOptions) ([]byte, error) {
	po, err := r.Core.CoreV1().Pods(ns).Get(ctx, pod, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	for _, c := range po.Spec.Containers {
		o := *opts
		o.Container = c.Name
		stream, err := r.Core.CoreV1().Pods(ns).GetLogs(pod, &o).Stream(ctx)
		if err != nil {
			return nil, err
		}
		fmt.Fprintf(&buf, "==== %s ====\n", c.Name)
		_, err = io.Copy(&buf, stream)
		_ = stream.Close()
		if err != nil {
			return nil, err
		}
	}
	return buf.Bytes(), nil
}

func (r *Runner) podLogsStream(ctx context.Context, ns, pod string, opts *corev1.PodLogOptions) ([]byte, error) {
	req := r.Core.CoreV1().Pods(ns).GetLogs(pod, opts)
	stream, err := req.Stream(ctx)
	if err != nil {
		return nil, err
	}
	defer stream.Close()
	var buf bytes.Buffer
	_, err = io.Copy(&buf, stream)
	return buf.Bytes(), err
}
