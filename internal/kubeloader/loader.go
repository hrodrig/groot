// Package kubeloader loads kubeconfig and builds REST clients without invoking kubectl.
package kubeloader

import (
	"fmt"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

// RESTConfig returns a client-go rest.Config from kubeconfig loading rules.
// If explicitPath is non-empty, it is used as the kubeconfig file path.
func RESTConfig(explicitPath string) (*rest.Config, error) {
	rules := clientcmd.NewDefaultClientConfigLoadingRules()
	if explicitPath != "" {
		rules.ExplicitPath = explicitPath
	}
	cc := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		rules,
		&clientcmd.ConfigOverrides{},
	)
	cfg, err := cc.ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("kubeconfig: %w", err)
	}
	cfg.QPS = 50
	cfg.Burst = 100
	return cfg, nil
}

// APIConfig returns the merged api.Config (current context, clusters, users).
func APIConfig(explicitPath string) (clientcmdapi.Config, error) {
	rules := clientcmd.NewDefaultClientConfigLoadingRules()
	if explicitPath != "" {
		rules.ExplicitPath = explicitPath
	}
	cc := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		rules,
		&clientcmd.ConfigOverrides{},
	)
	raw, err := cc.RawConfig()
	if err != nil {
		return clientcmdapi.Config{}, fmt.Errorf("kubeconfig: %w", err)
	}
	return raw, nil
}
