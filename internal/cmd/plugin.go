package cmd

import (
	"os"
	"path/filepath"
	"strings"
)

// Plugin name and binary conventions for `kubectl-groot` (ROADMAP #64).
//
// kubectl follows the single-source-of-truth plugin spec:
// https://kubernetes.io/docs/tasks/extend-kubectl/kubectl-plugins/
// A plugin is any executable file in $PATH whose name starts with `kubectl-`.
// When the user runs `kubectl <name> <sub-args>` and a plugin named
// `kubectl-<name>` exists, kubectl appends `<sub-args>` to the plugin's argv
// (so the plugin sees itself as argv[0]=`kubectl-<name>`, argv[1] onwards are
// whatever the user typed after `kubectl <name>`).
//
// Both binaries (`groot` and `kubectl-groot`) share the same main.go, so
// command parsing is identical. The basename `kubectl-groot` is what
// triggers plugin dispatch — we do not need different command logic. The
// standalone `groot` binary stays unchanged for users who don't use kubectl
// plugins.
const (
	pluginBinaryBasename = "kubectl-groot"
	// PluginRootCommand is the synthetic subcommand name kubectl strips when
	// it does NOT match a `kubectl-<name>` plugin. When kubectl calls
	// `kubectl groot collect …` and a `kubectl-groot` binary exists, kubectl
	// invokes it WITHOUT passing the `groot` token: argv becomes
	// ["kubectl-groot", "collect", …]. So Cobra's existing command tree
	// ("groot collect") parses correctly with no argv rewriting. We only
	// need to know the basename to label the running command in --version
	// and notify metadata.
	PluginRootCommand = "groot"
)

// IsPluginInvocation reports whether the current binary was launched under
// the kubectl-groot basename (i.e. by kubectl plugin discovery, or directly
// from a shell that has kubectl-groot on $PATH).
//
// Detection rules, in order:
//  1. If GROOT_FORCE_KUBECTL_PLUGIN=1 (env), always treat as a plugin.
//     Useful in tests and for users who symlink the binary to other names.
//  2. Otherwise, only the literal basename "kubectl-groot" counts. We do
//     NOT treat the string `kubectl` anywhere in argv as a plugin call
//     because that is ambiguous (e.g. `groot collect --context kubectl`).
func IsPluginInvocation() bool {
	if v := strings.TrimSpace(strings.ToLower(os.Getenv("GROOT_FORCE_KUBECTL_PLUGIN"))); v == "1" || v == "true" || v == "yes" {
		return true
	}
	return filepath.Base(os.Args[0]) == pluginBinaryBasename
}

// InvocationLabel returns a short tag describing how the binary was invoked
// ("groot", "kubectl-groot", or the basename when launched via a symlink).
// Used by --version and notify metadata so operators can tell whether the
// kubectl plugin dispatch fired or the standalone binary ran.
func InvocationLabel() string {
	if IsPluginInvocation() {
		return "kubectl-" + PluginRootCommand
	}
	name := filepath.Base(os.Args[0])
	if name == "" || name == "." {
		return PluginRootCommand
	}
	return name
}
