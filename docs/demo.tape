# groot terminal demo — run from repository root:
#   make install && PATH="$(go env GOPATH)/bin:$PATH" bash -c "vhs docs/demo.tape"
#
# Prepend GOPATH/bin when invoking vhs so a stale Homebrew groot (e.g. v0.7.x)
# does not shadow the VERSION-built binary from `make install`.
# Use bash -c (not bare vhs from zsh).

Output docs/demo.gif

Require make

Set Shell "bash"
Set FontSize 14
Set Width 960
Set Height 580
Set Padding 20

Enter
Sleep 300ms

# Install VERSION-built binary to GOPATH/bin
Type "make install"
Enter
Sleep 2s

# Root help — 1.0.x subcommands (validate, inspect, notify test, collect --output json)
Type "groot -h"
Enter
Sleep 2s

# Version / build metadata
Type "groot --version"
Enter
Sleep 1s

# Preflight subcommand (help only; no cluster required)
Type "groot validate -h"
Enter
Sleep 1500ms

# Notify smoke subcommand (1.0.3 — no cluster required)
Type "groot notify test -h"
Enter
Sleep 1500ms

# Sample config (truncated for a short GIF)
Type "groot --print-sample-config 2>&1 | head -n 22"
Enter
Sleep 2s

Type "exit"
Enter
Sleep 500ms
