package collector

import (
	"context"
	"strings"
	"time"
)

// enrichSummary fills identity fields used by notify (Slack attachment, etc.).
func (s *Service) enrichSummary(ctx context.Context, summary *Summary) {
	if summary == nil {
		return
	}
	summary.ClientID = strings.TrimSpace(s.cfg.ClientID)
	summary.Cluster = s.resolveClusterName(ctx)
	if meta, err := s.ReadKubeMetadata(ctx); err == nil {
		summary.Context = strings.TrimSpace(meta.Context)
		summary.User = strings.TrimSpace(meta.User)
		summary.Server = strings.TrimSpace(meta.Server)
	}
	ver := strings.TrimSpace(s.buildInfo.Version)
	if ver == "" {
		ver = "dev"
	}
	summary.Version = ver
	if strings.TrimSpace(summary.Mode) == "" {
		summary.Mode = "collect"
	}
	if summary.FinishedAt.IsZero() {
		summary.FinishedAt = time.Now().UTC()
	}
}
