package prompt

// PreviewScenario returns deterministic contexts used by the TUI gallery.
func PreviewScenario(id string) (PreviewContext, bool) {
	base := DefaultPreviewContext()
	switch id {
	case "clean":
		base.GitDirty = false
		base.ExitStatus = 0
		base.Jobs = 0
		return base, true
	case "git-dirty":
		base.GitDirty = true
		base.ExitStatus = 0
		return base, true
	case "command-failed":
		base.GitDirty = false
		base.ExitStatus = 7
		return base, true
	case "dev-project":
		base.Cwd = "~/dev/ozsh"
		base.GitBranch = "feature/tui-v2"
		base.GitDirty = true
		base.Venv = "ozsh-dev"
		base.Node = "v22.0.0"
		base.Go = "go1.25.0"
		base.Jobs = 2
		return base, true
	case "low-battery":
		base.GitDirty = false
		base.Battery = "12%"
		base.Jobs = 0
		return base, true
	default:
		return PreviewContext{}, false
	}
}
