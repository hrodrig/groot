package collector

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"strings"

	"github.com/hrodrig/groot/pkg/kubeloader"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

const defaultClusterName = "unknown-cluster"

// resolveClusterName picks the archive basename cluster segment.
// Order: config cluster_name → kubeconfig metadata → kube-public/cluster-info → API server host.
func (s *Service) resolveClusterName(ctx context.Context) string {
	if c := sanitize(strings.TrimSpace(s.cfg.ClusterName)); c != "" {
		return c
	}
	if meta, err := s.ReadKubeMetadata(ctx); err == nil {
		if c := sanitize(strings.TrimSpace(meta.Cluster)); c != "" {
			return c
		}
	}
	if s.clientset != nil {
		if c, err := clusterNameFromClusterInfo(ctx, s.clientset); err == nil {
			if c = sanitize(strings.TrimSpace(c)); c != "" {
				return c
			}
		}
	}
	if s.restConfig != nil {
		if c := clusterNameFromHost(s.restConfig.Host); c != "" {
			return c
		}
	}
	return defaultClusterName
}

func clusterNameFromClusterInfo(ctx context.Context, cs kubernetes.Interface) (string, error) {
	cm, err := cs.CoreV1().ConfigMaps("kube-public").Get(ctx, "cluster-info", metav1.GetOptions{})
	if err != nil {
		return "", err
	}
	raw, ok := cm.Data["kubeconfig"]
	if !ok || strings.TrimSpace(raw) == "" {
		return "", fmt.Errorf("cluster-info: missing kubeconfig key")
	}
	cfg, err := clientcmd.Load([]byte(raw))
	if err != nil {
		return "", fmt.Errorf("cluster-info kubeconfig: %w", err)
	}
	if cfg.CurrentContext != "" {
		if ctxCfg, ok := cfg.Contexts[cfg.CurrentContext]; ok {
			if n := strings.TrimSpace(ctxCfg.Cluster); n != "" {
				return n, nil
			}
		}
	}
	for name := range cfg.Clusters {
		if n := strings.TrimSpace(name); n != "" {
			return n, nil
		}
	}
	return "", fmt.Errorf("cluster-info: no cluster name")
}

func clusterNameFromHost(host string) string {
	host = strings.TrimSpace(host)
	if host == "" {
		return ""
	}
	if u, err := url.Parse(host); err == nil && strings.TrimSpace(u.Host) != "" {
		host = u.Host
	}
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	return sanitize(host)
}

// restHostFromKubeconfig returns the API server URL from kubeconfig when available.
func restHostFromKubeconfig(explicitPath string) string {
	cfg, err := kubeloader.RESTConfig(explicitPath)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(cfg.Host)
}
