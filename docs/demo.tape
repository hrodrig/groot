# groot terminal demo — run from repository root:
#   make build && bash -c "vhs docs/demo.tape"
#
# Uses ./bin/groot (not PATH) so Homebrew/system installs (e.g. v0.7.2) never
# shadow the VERSION-built binary. Use bash -c (not bare vhs from zsh).

Output docs/demo.gif

Require make

Set Shell "bash"
Set FontSize 14
Set Width 960
Set Height 580
Set Padding 20

Enter
Sleep 300ms

# Build local binary (VERSION from Makefile)
Type "make build"
Enter
Sleep 2s

# Root help — 0.9.x subcommands (validate, inspect, completion)
Type "./bin/groot -h"
Enter
Sleep 2s

# Version / build metadata
Type "./bin/groot --version"
Enter
Sleep 1s

# Preflight subcommand (help only; no cluster required)
Type "./bin/groot validate -h"
Enter
Sleep 1500ms

# Shell completion snippet
Type "./bin/groot completion bash 2>/dev/null | head -n 12"
Enter
Sleep 1500ms

# Sample config (truncated for a short GIF)
Type "./bin/groot --print-sample-config 2>&1 | head -n 22"
Enter
Sleep 2s

Type "exit"
Enter
Sleep 500ms
