# groot terminal demo — from repository root, after installing groot on PATH:
#   make install && bash -c "vhs docs/demo.tape"
# Use bash -c (not bare `vhs` from zsh) so the recorder does not inherit zsh/Oh My Zsh prompts.
# Recorded output reflects the binary built with VERSION via make install / make build.
# After a VERSION bump or CLI UX change (0.9.x: validate, inspect, completion), regenerate docs/demo.gif.

Output docs/demo.gif

Require groot

Set Shell "bash"
Set FontSize 14
Set Width 960
Set Height 560
Set Padding 20

Enter
Sleep 300ms

# Root help — 0.9.x subcommands (validate, inspect, completion)
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

# Shell completion snippet
Type "groot completion bash 2>/dev/null | head -n 12"
Enter
Sleep 1500ms

# Sample config (truncated for a short GIF)
Type "groot --print-sample-config 2>&1 | head -n 22"
Enter
Sleep 2s

Type "exit"
Enter
Sleep 500ms
