package collector

import (
	"context"
	"fmt"
	"strings"

	"github.com/hrodrig/groot/internal/kubeloader"
)

type kubeMeta struct {
	Context string
	Cluster string
	User    string
	Server  string
}

// ReadKubeMetadata extracts context, cluster, and server metadata from kubeconfig (no cluster call).
func (s *Service) ReadKubeMetadata(ctx context.Context) (kubeMeta, error) {
	_ = ctx
	raw, err := kubeloader.APIConfig(s.cfg.Kubeconfig)
	if err != nil {
		return kubeMeta{}, fmt.Errorf("load kubeconfig: %w", err)
	}

	meta := kubeMeta{
		Context: strings.TrimSpace(raw.CurrentContext),
	}
	if meta.Context == "" {
		return meta, nil
	}

	ctxCfg, ok := raw.Contexts[meta.Context]
	if !ok {
		return meta, nil
	}
	meta.Cluster = strings.TrimSpace(ctxCfg.Cluster)
	meta.User = strings.TrimSpace(ctxCfg.AuthInfo)

	if cl, ok := raw.Clusters[meta.Cluster]; ok {
		meta.Server = strings.TrimSpace(cl.Server)
	}

	return meta, nil
}
