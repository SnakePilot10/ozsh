Output docs/screencasts/tui-visual-hierarchy.gif

Set Shell "zsh"
Set FontSize 16
Set Width 1100
Set Height 760
Set TypingSpeed 35ms

Type "go run ./cmd/ozsh tui"
Enter
Sleep 2s

# Prompt: wide configuration, segments, live preview, and details.
Type "2"
Sleep 2s
Type "j"
Sleep 1s
Type "j"
Sleep 2s

# Themes: library and contextual preview.
Type "3"
Sleep 2s
Type "]"
Sleep 1s

# Plugins: selected-plugin health and load details.
Type "4"
Sleep 2s
Type "j"
Sleep 1s

# Preview: scenario/context workspace and expanded live output.
Type "5"
Sleep 2s
Type "]"
Sleep 1s

# Resize below the 72-column breakpoint to demonstrate stacked Termux layout.
Set Width 620
Set Height 900
Sleep 2s
Type "2"
Sleep 3s

Ctrl+C
