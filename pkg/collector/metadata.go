package collector

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

type kubeconfigView struct {
	CurrentContext string `json:"current-context"`
	Contexts       []struct {
		Name    string `json:"name"`
		Context struct {
			Cluster string `json:"cluster"`
			User    string `json:"user"`
		} `json:"context"`
	} `json:"contexts"`
	Clusters []struct {
		Name    string `json:"name"`
		Cluster struct {
			Server string `json:"server"`
		} `json:"cluster"`
	} `json:"clusters"`
}

type kubeMeta struct {
	Context string
	Cluster string
	User    string
	Server  string
}

// ReadKubeMetadata extracts context, cluster, and server metadata from kubeconfig.
func (s *Service) ReadKubeMetadata(ctx context.Context) (kubeMeta, error) {
	cmd := exec.CommandContext(ctx, "kubectl", s.kubectlArgs([]string{"config", "view", "-o", "json"})...)
	out, err := cmd.Output()
	if err != nil {
		return kubeMeta{}, fmt.Errorf("kubectl config view: %w", err)
	}

	var view kubeconfigView
	if err := json.Unmarshal(out, &view); err != nil {
		return kubeMeta{}, fmt.Errorf("parse kubeconfig json: %w", err)
	}

	meta := kubeMeta{
		Context: strings.TrimSpace(view.CurrentContext),
	}
	if meta.Context == "" {
		return meta, nil
	}

	for _, item := range view.Contexts {
		if item.Name == meta.Context {
			meta.Cluster = strings.TrimSpace(item.Context.Cluster)
			meta.User = strings.TrimSpace(item.Context.User)
			break
		}
	}

	for _, item := range view.Clusters {
		if item.Name == meta.Cluster {
			meta.Server = strings.TrimSpace(item.Cluster.Server)
			break
		}
	}

	return meta, nil
}
