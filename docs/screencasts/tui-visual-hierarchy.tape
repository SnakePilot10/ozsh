Output docs/screencasts/tui-visual-hierarchy.gif

Require go

Set Shell "zsh"
Set FontSize 16
Set Width 1100
Set Height 760
Set TypingSpeed 35ms

# Build an offline custom-plugin fixture and intercept only the demo clone.
Hide
Type@0ms `rm -rf /tmp/ozsh-vhs-home /tmp/ozsh-vhs-bin /tmp/ozsh-vhs-plugin && mkdir -p /tmp/ozsh-vhs-home /tmp/ozsh-vhs-bin /tmp/ozsh-vhs-plugin && touch /tmp/ozsh-vhs-home/.zshrc && printf '%s\n' '# offline demo plugin' > /tmp/ozsh-vhs-plugin/offline.plugin.zsh && printf '%s\n' '#!/bin/sh' 'for last; do :; done' 'cp -R /tmp/ozsh-vhs-plugin/. "$last"/' > /tmp/ozsh-vhs-bin/git && chmod +x /tmp/ozsh-vhs-bin/git && clear`
Enter
Show

Type "HOME=/tmp/ozsh-vhs-home PATH=/tmp/ozsh-vhs-bin:$PATH go run ./cmd/ozsh tui"
Enter
Wait+Screen /Home/
Sleep 1s

# Prompt selection uses markers and accent text instead of filled gray rows.
Type "2"
Sleep 1s
Type "j"
Sleep 2s

# Open the discoverable custom-plugin workflow.
Type "4"
Wait+Screen /Recommended plugins/
Sleep 1s
Type "a"
Wait+Screen /Repository URL/
Sleep 1s
Type "https://github.com/example/offline-plugin.git"
Enter
Wait+Screen /Choose load file/
Sleep 2s

# Select the locally cloned candidate and reach the explicit trust warning.
Enter
Wait+Screen /Trust review/
Sleep 3s

# Cancel trust, discard the staging checkout, and return to Plugins.
Ctrl+[
Sleep 1s
Ctrl+[
Sleep 1s
Ctrl+[
Wait+Screen /Recommended plugins/
Sleep 2s

Ctrl+C

Hide
Type@0ms `rm -rf /tmp/ozsh-vhs-home /tmp/ozsh-vhs-bin /tmp/ozsh-vhs-plugin`
Enter
Show
