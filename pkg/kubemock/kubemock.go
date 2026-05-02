// Package kubemock provides a PATH kubectl shim for tests (not used in production binaries).
package kubemock

import (
	"os"
	"path/filepath"
	"testing"
)

// Script is POSIX sh; patterns ordered with specific branches before broad *get*ns* matches.
const Script = `#!/bin/sh
j="$*"
case "$j" in
*config*view*json*)
  printf '%s' '{"current-context":"ctx1","contexts":[{"name":"ctx1","context":{"cluster":"my-cluster","user":"u1"}}],"clusters":[{"name":"my-cluster","cluster":{"server":"https://127.0.0.1"}}]}'
  ;;
*config*current-context*)
  echo ctx1
  ;;
*custom-columns*)
  echo "default pod-a node1"
  ;;
*get*ns*|*get*namespaces*)
  echo "namespace/default"
  ;;
*cluster-info*)
  echo "Kubernetes control plane is running"
  ;;
*get*nodes*wide*)
  echo "NAME STATUS ROLES AGE VERSION"
  echo "node1 Ready control-plane 1d v1.30.0"
  ;;
*get*events*)
  echo "LAST SEEN   TYPE     REASON   OBJECT     MESSAGE"
  ;;
*get*pods*-A*wide*)
  echo "NAMESPACE NAME READY STATUS RESTARTS AGE IP NODE"
  echo "default pod-a 1/1 Running 0 1d 10.0.0.1 node1"
  ;;
*get*pods*-A*-o*json*)
  printf '%s' '{"items":[{"metadata":{"namespace":"default","name":"pod-a","labels":{"app.kubernetes.io/name":"api"}},"spec":{"nodeName":"node1"}}]}'
  ;;
*tier=control-plane*)
  printf '%s\t%s\n' "kube-apiserver-node1" "node1"
  ;;
*get*nodes*-o*name*)
  echo "node/node1"
  ;;
*describe*node*)
  echo "Name: node1"
  ;;
*top*node*)
  echo "NAME CPU(cores) CPU% MEMORY(bytes) MEMORY%"
  echo "node1 100m 1% 1000Mi 5%"
  ;;
*get*all*-n*)
  echo "NAME READY UP-TO-DATE AVAILABLE AGE"
  ;;
*logs*--previous*)
  echo "previous log line"
  ;;
*logs*)
  echo "log line"
  ;;
*__GROOT_FAIL__*)
  echo "fail" >&2
  exit 1
  ;;
*)
  echo "ok"
  ;;
esac
exit 0
`

// Install prepends a fake kubectl to PATH; call the returned cleanup to restore PATH.
func Install(t *testing.T) (cleanup func()) {
	t.Helper()
	dir := t.TempDir()
	bin := filepath.Join(dir, "kubectl")
	if err := os.WriteFile(bin, []byte(Script), 0o755); err != nil {
		t.Fatal(err)
	}
	old := os.Getenv("PATH")
	_ = os.Setenv("PATH", dir+string(filepath.ListSeparator)+old)
	return func() { _ = os.Setenv("PATH", old) }
}
