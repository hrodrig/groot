package kubetest

import (
	"net/http"
	"strings"
)

func fakeKubeHandler(w http.ResponseWriter, r *http.Request) {
	p := r.URL.Path
	if handleExactCore(w, p) {
		return
	}
	if handleNodeDetail(w, r, p) {
		return
	}
	if handlePodLog(w, p) {
		return
	}
	if handleEvents(w, p) {
		return
	}
	if handleKubeSystemPods(w, p) {
		return
	}
	if handleProxyLogs(w, p) {
		return
	}
	if handleMetrics(w, p) {
		return
	}
	if handleAuth(w, r, p) {
		return
	}
	if handleOpenAPI(w, p) {
		return
	}
	if handleAppsV1(w, r, p) {
		return
	}
	if handleNamespacedCore(w, r, p) {
		return
	}
	w.Header().Set("Content-Type", "application/json")
	http.NotFound(w, r)
}

func writeJSON(w http.ResponseWriter, body string) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(body))
}

func handleExactCore(w http.ResponseWriter, p string) bool {
	switch p {
	case "/version":
		writeJSON(w, `{"major":"1","minor":"30","gitVersion":"v1.30.0"}`)
		return true
	case "/api/v1/namespaces":
		writeJSON(w, `{"kind":"NamespaceList","apiVersion":"v1","metadata":{},"items":[{"metadata":{"name":"default"}},{"metadata":{"name":"kube-system"}}]}`)
		return true
	case "/api/v1/nodes":
		writeJSON(w, `{"kind":"NodeList","apiVersion":"v1","metadata":{},"items":[{"metadata":{"name":"node1"},"status":{"conditions":[{"type":"Ready","status":"True"}],"nodeInfo":{"kubeletVersion":"v1.30.0"}}}]}`)
		return true
	case "/api/v1/pods":
		writeJSON(w, `{"kind":"PodList","apiVersion":"v1","metadata":{},"items":[{"metadata":{"namespace":"default","name":"pod-a","labels":{"app.kubernetes.io/name":"api"}},"spec":{"nodeName":"node1"},"status":{"phase":"Running","containerStatuses":[{"name":"c","ready":true,"restartCount":0}]}}]}`)
		return true
	default:
		return false
	}
}

func handleNodeDetail(w http.ResponseWriter, r *http.Request, p string) bool {
	if !strings.HasPrefix(p, "/api/v1/nodes/") || r.Method != http.MethodGet || strings.Contains(p, "/proxy/") {
		return false
	}
	writeJSON(w, `{"kind":"Node","apiVersion":"v1","metadata":{"name":"node1"},"status":{"addresses":[],"labels":{}}}`)
	return true
}

func handlePodLog(w http.ResponseWriter, p string) bool {
	if !strings.Contains(p, "/pods/") || !strings.Contains(p, "/log") {
		return false
	}
	w.Header().Set("Content-Type", "text/plain")
	_, _ = w.Write([]byte("log line\n"))
	return true
}

func handleEvents(w http.ResponseWriter, p string) bool {
	if !strings.HasPrefix(p, "/api/v1/namespaces/") || !strings.HasSuffix(p, "/events") {
		return false
	}
	writeJSON(w, `{"kind":"EventList","apiVersion":"v1","metadata":{},"items":[]}`)
	return true
}

func handleKubeSystemPods(w http.ResponseWriter, p string) bool {
	if p != "/api/v1/namespaces/kube-system/pods" {
		return false
	}
	writeJSON(w, `{"kind":"PodList","apiVersion":"v1","metadata":{},"items":[{"metadata":{"namespace":"kube-system","name":"kube-apiserver-node1","labels":{"tier":"control-plane"}},"spec":{"nodeName":"node1"},"status":{"phase":"Running","containerStatuses":[{"name":"c","ready":true,"restartCount":0}]}}]}`)
	return true
}

func handleProxyLogs(w http.ResponseWriter, p string) bool {
	if !strings.Contains(p, "/proxy/logs/") {
		return false
	}
	w.Header().Set("Content-Type", "text/plain")
	_, _ = w.Write([]byte("mock kubelet log line\n"))
	return true
}

func handleMetrics(w http.ResponseWriter, p string) bool {
	switch {
	case strings.HasPrefix(p, "/apis/metrics.k8s.io/v1beta1/nodes/"):
		writeJSON(w, `{"kind":"NodeMetrics","apiVersion":"metrics.k8s.io/v1beta1","metadata":{"name":"node1"},"usage":{"cpu":"100m","memory":"1000Mi"}}`)
		return true
	case p == "/apis/metrics.k8s.io/v1beta1/pods":
		writeJSON(w, `{"kind":"PodMetricsList","apiVersion":"metrics.k8s.io/v1beta1","metadata":{},"items":[{"metadata":{"name":"pod-a","namespace":"default"},"containers":[{"name":"c","usage":{"cpu":"5m","memory":"10Mi"}}]}]}`)
		return true
	default:
		return false
	}
}

func handleAuth(w http.ResponseWriter, r *http.Request, p string) bool {
	if r.Method != http.MethodPost || !strings.HasPrefix(p, "/apis/authorization.k8s.io/v1/selfsubjectaccessreviews") {
		return false
	}
	writeJSON(w, `{"kind":"SelfSubjectAccessReview","apiVersion":"authorization.k8s.io/v1","status":{"allowed":true}}`)
	return true
}

func handleOpenAPI(w http.ResponseWriter, p string) bool {
	if !strings.HasPrefix(p, "/openapi/v2") {
		return false
	}
	writeJSON(w, `{"swagger":"2.0"}`)
	return true
}

func handleAppsV1(w http.ResponseWriter, r *http.Request, p string) bool {
	if !strings.HasPrefix(p, "/apis/apps/v1/namespaces/") {
		return false
	}
	if !appsV1ListPath(p) {
		http.NotFound(w, r)
		return true
	}
	kind := appsListKind(p)
	writeJSON(w, `{"kind":"`+kind+`List","apiVersion":"apps/v1","metadata":{},"items":[]}`)
	return true
}

func appsV1ListPath(p string) bool {
	return strings.Contains(p, "/deployments") ||
		strings.Contains(p, "/replicasets") ||
		strings.Contains(p, "/statefulsets") ||
		strings.Contains(p, "/daemonsets")
}

func appsListKind(p string) string {
	switch {
	case strings.Contains(p, "replicasets"):
		return "ReplicaSet"
	case strings.Contains(p, "statefulsets"):
		return "StatefulSet"
	case strings.Contains(p, "daemonsets"):
		return "DaemonSet"
	default:
		return "Deployment"
	}
}

func handleNamespacedCore(w http.ResponseWriter, r *http.Request, p string) bool {
	if !strings.HasPrefix(p, "/api/v1/namespaces/") {
		return false
	}
	switch {
	case strings.Contains(p, "/pods"):
		writeJSON(w, `{"kind":"PodList","apiVersion":"v1","metadata":{},"items":[]}`)
	case strings.Contains(p, "/services"):
		writeJSON(w, `{"kind":"ServiceList","apiVersion":"v1","metadata":{},"items":[]}`)
	default:
		http.NotFound(w, r)
	}
	return true
}
