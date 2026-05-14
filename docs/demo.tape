# groot terminal demo — from repository root, after installing groot on PATH:
#   make install && bash -c "vhs docs/demo.tape"
# Use bash -c (not bare `vhs` from zsh) so the recorder does not inherit zsh/Oh My Zsh prompts.
# Recorded output reflects the binary built with VERSION via make install / make build.
# After a VERSION bump, regenerate docs/demo.gif so the embedded recording matches the release.

Output docs/demo.gif

Require groot

Set Shell "bash"
Set FontSize 14
Set Width 920
Set Height 520
Set Padding 20

Enter
Sleep 300ms

# Root help
Type "groot -h"
Enter
Sleep 2s

# Version / build metadata
Type "groot --version"
Enter
Sleep 1s

# Sample config (truncated for a short GIF)
Type "groot --print-sample-config 2>&1 | head -n 28"
Enter
Sleep 2s

Type "exit"
Enter
Sleep 500ms
