package config

import (
	"fmt"
	"strings"
)

// ValidateExtraKubectl ensures each collection.extra_kubectl line uses an allowlisted,
// read-oriented subcommand (or config view / auth can-i). Groot runs these via the
// Kubernetes client with argv slices (no shell), but limiting verbs reduces risk from mis-edited config.
func ValidateExtraKubectl(cmds []string) error {
	for i, raw := range cmds {
		cmd := strings.TrimSpace(raw)
		if cmd == "" {
			continue
		}
		parts := strings.Fields(cmd)
		if len(parts) == 0 {
			continue
		}
		sub := strings.ToLower(parts[0])
		if extraKubectlSubcommandAllowed(sub, parts) {
			continue
		}
		return fmt.Errorf(
			"collection.extra_kubectl[%d]: %q: subcommand %q is not allowed (read-only diagnostics only; see README, section Config)",
			i, cmd, parts[0],
		)
	}
	return nil
}

func extraKubectlSubcommandAllowed(sub string, parts []string) bool {
	switch sub {
	case "get", "describe", "top", "logs",
		"api-resources", "api-versions", "version", "cluster-info":
		return true
	case "config":
		return len(parts) >= 2 && strings.EqualFold(parts[1], "view")
	case "auth":
		return len(parts) >= 2 && strings.EqualFold(parts[1], "can-i")
	default:
		return false
	}
}
