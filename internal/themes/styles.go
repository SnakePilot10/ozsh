package themes

import "github.com/snakepilot10/ozsh/internal/config"

type segmentStyle struct {
	fg, bg, compatible, nerd string
	bold, showSuccess        bool
}

type promptStyle struct {
	order, right             []string
	separator, nerdSeparator string
	segments                 map[string]segmentStyle
}

func seg(fg, bg, compatible, nerd string, bold bool) segmentStyle {
	return segmentStyle{fg: fg, bg: bg, compatible: compatible, nerd: nerd, bold: bold}
}

func okStatus(s segmentStyle) segmentStyle { s.showSuccess = true; return s }

func styleFor(p Preset) promptStyle {
	switch p.ID + variantSuffix(p.Variant) {
	case "minimal":
		return promptStyle{[]string{"cwd", "git", "status"}, []string{}, " ", "", map[string]segmentStyle{
			"cwd": seg("#d8dee9", "", "", "", false), "git": seg("#8fbc8f", "", "", "", false), "status": seg("#e06c75", "", "", "", false),
		}}
	case "pure":
		return promptStyle{[]string{"cwd", "git", "status"}, []string{"time"}, "  ", "", map[string]segmentStyle{
			"cwd": seg("#5fd7ff", "", "", "", true), "git": seg("#af87ff", "", "", "", false), "status": seg("#ff5f5f", "", "", "", true), "time": seg("#707a8c", "", "", "", false),
		}}
	case "powerline":
		return promptStyle{[]string{"user", "cwd", "git", "status"}, []string{"time"}, " ▶ ", "", map[string]segmentStyle{
			"user": seg("#1e222a", "#61afef", "@", "", true), "cwd": seg("#1e222a", "#98c379", "dir", "", true), "git": seg("#1e222a", "#c678dd", "git", "", true), "status": seg("#ffffff", "#e06c75", "!", "", true), "time": seg("#61afef", "", "time", "", true),
		}}
	case "cyberpunk":
		return promptStyle{[]string{"user", "cwd", "git", "status"}, []string{"time"}, " // ", "", map[string]segmentStyle{
			"user": seg("#ff2a6d", "#26101d", "usr", "", true), "cwd": seg("#00f5ff", "#082a30", "path", "", true), "git": seg("#ffe600", "", "git", "", true), "status": seg("#ff2a6d", "", "err", "", true), "time": seg("#00ff9f", "", "time", "", false),
		}}
	case "matrix":
		return promptStyle{[]string{"user", "host", "cwd", "git", "status"}, []string{"time"}, " :: ", "", map[string]segmentStyle{
			"user": seg("#7cff6b", "", "ID", "", true), "host": seg("#3f7050", "", "NODE", "", false), "cwd": seg("#00ff41", "", "PATH", "", true), "git": seg("#c6ff00", "", "REV", "", false), "status": okStatus(seg("#ff3b3b", "", "RC", "", true)), "time": seg("#3f7050", "", "T", "", false),
		}}
	case "dracula":
		return promptStyle{[]string{"user", "cwd", "git", "status"}, []string{"time"}, " • ", "", map[string]segmentStyle{
			"user": seg("#ff79c6", "", "@", "", true), "cwd": seg("#bd93f9", "", "~", "", true), "git": seg("#50fa7b", "", "git", "", false), "status": seg("#ff5555", "", "!", "", true), "time": seg("#8be9fd", "", "time", "", false),
		}}
	case "nord":
		return promptStyle{[]string{"user", "host", "cwd", "git", "status"}, []string{"time"}, " › ", "", map[string]segmentStyle{
			"user": seg("#81a1c1", "", "@", "", true), "host": seg("#5e81ac", "", "host", "", false), "cwd": seg("#88c0d0", "", "dir", "", true), "git": seg("#a3be8c", "", "git", "", false), "status": seg("#bf616a", "", "!", "", true), "time": seg("#b48ead", "", "time", "", false),
		}}
	case "gruvbox":
		return promptStyle{[]string{"user", "cwd", "git", "venv", "status"}, []string{"time"}, " · ", "", map[string]segmentStyle{
			"user": seg("#fabd2f", "", "usr", "", true), "cwd": seg("#fe8019", "", "dir", "", true), "git": seg("#8ec07c", "", "git", "", true), "venv": seg("#b8bb26", "", "py", "", false), "status": seg("#fb4934", "", "err", "", true), "time": seg("#928374", "", "time", "", false),
		}}
	case "catppuccin":
		return promptStyle{[]string{"user", "cwd", "git", "node", "status"}, []string{"time"}, " ◇ ", "", map[string]segmentStyle{
			"user": seg("#f5c2e7", "", "@", "", true), "cwd": seg("#cba6f7", "", "~", "", true), "git": seg("#a6e3a1", "", "git", "", false), "node": seg("#89b4fa", "", "node", "", false), "status": seg("#f38ba8", "", "!", "", true), "time": seg("#89dceb", "", "time", "", false),
		}}
	case "termux":
		return promptStyle{[]string{"cwd", "git", "status"}, []string{}, " | ", "", map[string]segmentStyle{
			"cwd": seg("#00bcd4", "", "~", "", true), "git": seg("#8bc34a", "", "git", "", false), "status": seg("#ff5252", "", "!", "", true),
		}}
	case "circuit:blue":
		return circuitStyle([]string{"user", "host", "cwd", "git", "status"}, []string{"time"}, " ┆ ", "#37b6ff", "#f6c85f", "#36d399", "#ff5c72", "USR", "DEV", "DIAG")
	case "circuit:green":
		return circuitStyle([]string{"host", "cwd", "git", "status"}, []string{"jobs"}, " ─ ", "#36d399", "#77ff9c", "#d7df5f", "#ff6174", "PCB", "BOARD", "TEST")
	case "circuit:amber":
		return circuitStyle([]string{"user", "cwd", "git", "status"}, []string{"time"}, " ┆ ", "#ffb000", "#ffd166", "#b6d957", "#ff5c5c", "OP", "METER", "WARN")
	case "circuit:red":
		return circuitStyle([]string{"host", "cwd", "git", "status"}, []string{"time"}, " / ", "#ff4d5a", "#ffbd59", "#4de09a", "#ffffff", "UNIT", "FAULT", "ERR")
	case "circuit:mono":
		return promptStyle{[]string{"cwd", "git", "status"}, []string{}, "  ", "", map[string]segmentStyle{
			"cwd": seg("#e5e7eb", "", "", "", true), "git": seg("#aeb4bd", "", "", "", false), "status": seg("#f3f4f6", "", "", "", true),
		}}
	case "circuit:neon":
		return promptStyle{[]string{"user", "cwd", "git", "node", "status"}, []string{"time"}, " ◆ ", "", map[string]segmentStyle{
			"user": seg("#ff2a6d", "#24102d", "ID", "", true), "cwd": seg("#00f5ff", "#07272d", "NODE", "", true), "git": seg("#39ff14", "", "REV", "", true), "node": seg("#fff01f", "", "JS", "", false), "status": okStatus(seg("#ff2a6d", "", "SCAN", "", true)), "time": seg("#765a91", "", "TIME", "", false),
		}}
	case "retro":
		return promptStyle{[]string{"user", "host", "cwd", "git", "status"}, []string{"time"}, " | ", "", map[string]segmentStyle{
			"user": seg("#111006", "#ffcc00", "USR", "", true), "host": seg("#ffcc00", "", "SYS", "", false), "cwd": seg("#111006", "#a8d840", "DIR", "", true), "git": seg("#8ec07c", "", "GIT", "", true), "status": seg("#ff5f5f", "", "ERR", "", true), "time": seg("#8b815d", "", "CLK", "", false),
		}}
	}
	return promptStyle{p.Order, p.RightOrder, p.Separator, "", map[string]segmentStyle{
		"user": seg(p.Theme.Accent, "", p.UserIcon, p.UserIcon, false), "cwd": seg(p.Theme.Warning, "", p.CwdIcon, p.CwdIcon, false), "git": seg(p.Theme.Success, "", p.GitIcon, p.GitIcon, false), "status": seg(p.Theme.Error, "", p.StatusIcon, p.StatusIcon, true),
	}}
}

func circuitStyle(order, right []string, separator, accent, cwd, git, status, userLabel, hostLabel, statusLabel string) promptStyle {
	return promptStyle{order, right, separator, "", map[string]segmentStyle{
		"user": seg(accent, "", userLabel, "", true), "host": seg(accent, "", hostLabel, "", false), "cwd": seg(cwd, "", "PATH", "", true), "git": seg(git, "", "REV", "", false), "status": okStatus(seg(status, "", statusLabel, "", true)), "time": seg(accent, "", "TIME", "", false), "jobs": seg(accent, "", "JOBS", "", false),
	}}
}

func variantSuffix(v string) string {
	if v == "" {
		return ""
	}
	return ":" + v
}

func applyPresentation(cfg *config.Config, p Preset) {
	style := styleFor(p)
	cfg.Prompt.Order = append([]string(nil), style.order...)
	cfg.Prompt.RightOrder = append([]string(nil), style.right...)
	cfg.Prompt.RightPrompt = len(style.right) > 0
	cfg.Prompt.Layout = p.Layout
	cfg.Prompt.Newline = p.Layout == config.PromptLayoutTwoLine
	cfg.Prompt.Symbol = p.Symbol
	cfg.Prompt.Separator = style.separator
	if cfg.Prompt.IconMode == config.IconModeNerd && style.nerdSeparator != "" {
		cfg.Prompt.Separator = style.nerdSeparator
	}

	active := map[string]bool{}
	for _, name := range style.order {
		active[name] = true
	}
	for _, name := range style.right {
		active[name] = true
	}
	for name, current := range cfg.Prompt.Segments {
		current.Enabled, current.FG, current.BG, current.Bold = active[name], p.Theme.Muted, "", false
		current.Icon, current.CompatibleIcon, current.NerdIcon, current.ShowSuccess = "", "", "", false
		if s, ok := style.segments[name]; ok {
			current.FG, current.BG, current.Bold = s.fg, s.bg, s.bold
			current.CompatibleIcon, current.NerdIcon, current.ShowSuccess = s.compatible, s.nerd, s.showSuccess
		}
		cfg.Prompt.Segments[name] = current
	}
}
