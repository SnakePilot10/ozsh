package tui

import "strings"

func (m Model) pluginRemoveConfirmation(spec layoutSpec) string {
	var body strings.Builder
	body.WriteString(renderSectionHeader("Remove custom plugin", "Queue a reversible removal"))
	body.WriteString("\n\n")
	body.WriteString(renderGroupLabel("Pending removal"))
	body.WriteString("\n")
	body.WriteString(accentStyle.Render(m.pluginRemoveName))
	body.WriteString("\n\n")
	body.WriteString("The checkout will remain on disk until Review & Apply succeeds.")
	body.WriteString("\n")
	body.WriteString("Cancelling Apply leaves the plugin and its files unchanged.")
	body.WriteString("\n\n")
	body.WriteString(renderHint("y/Enter queue removal  ·  n/Esc cancel"))
	return fitHeight(body.String(), spec.contentWidth, spec.workspaceHeight)
}
