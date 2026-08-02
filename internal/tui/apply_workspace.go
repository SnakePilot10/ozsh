package tui

import (
	"fmt"
	"strings"
)

func (m Model) technicalApplyWorkspace() string {
	cfg := m.reviewedConfig
	if cfg == nil {
		return "Review & Apply\n\nReview expired. Press n/esc, then open Review & Apply again."
	}

	active, missing := reviewedPluginCounts(cfg)
	var b strings.Builder
	b.WriteString(renderSectionHeader("Review & Apply", "Technical details"))
	b.WriteString("\n\n")
	b.WriteString(renderGroupLabel("Planned .zshrc diff"))
	b.WriteString("\n")
	b.WriteString(m.applyDiff)
	b.WriteString("\n\n")
	fmt.Fprintf(&b, "Theme %s  ·  Layout %s  ·  Icons %s\n", cfg.Theme.Name, cfg.Prompt.Layout, cfg.Prompt.IconMode)
	fmt.Fprintf(&b, "Plugins %d ready  ·  %d need setup\n", active, missing)
	b.WriteString("\nPress y to apply this exact snapshot or n/esc to cancel.")
	return b.String()
}
