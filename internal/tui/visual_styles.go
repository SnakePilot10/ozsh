package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var visualPalette = struct {
	Accent      lipgloss.AdaptiveColor
	Text        lipgloss.AdaptiveColor
	Subtle      lipgloss.AdaptiveColor
	Muted       lipgloss.AdaptiveColor
	Surface     lipgloss.AdaptiveColor
	Panel       lipgloss.AdaptiveColor
	Border      lipgloss.AdaptiveColor
	FocusBorder lipgloss.AdaptiveColor
	Success     lipgloss.AdaptiveColor
	Warning     lipgloss.AdaptiveColor
	Danger      lipgloss.AdaptiveColor
}{
	Accent:      lipgloss.AdaptiveColor{Light: "#006A73", Dark: "#27E6E6"},
	Text:        lipgloss.AdaptiveColor{Light: "#20242B", Dark: "#F2F4F8"},
	Subtle:      lipgloss.AdaptiveColor{Light: "#4D5968", Dark: "#A8B0C0"},
	Muted:       lipgloss.AdaptiveColor{Light: "#667085", Dark: "#858FA3"},
	Surface:     lipgloss.AdaptiveColor{Light: "#E9F5F5", Dark: "#121B20"},
	Panel:       lipgloss.AdaptiveColor{Light: "#FAFCFC", Dark: "#09090D"},
	Border:      lipgloss.AdaptiveColor{Light: "#779092", Dark: "#31575C"},
	FocusBorder: lipgloss.AdaptiveColor{Light: "#007F89", Dark: "#27E6E6"},
	Success:     lipgloss.AdaptiveColor{Light: "#16794A", Dark: "#5FD79A"},
	Warning:     lipgloss.AdaptiveColor{Light: "#9A6700", Dark: "#E5C07B"},
	Danger:      lipgloss.AdaptiveColor{Light: "#B42318", Dark: "#FF5C75"},
}

var (
	logoStyle = lipgloss.NewStyle().
			Foreground(visualPalette.Accent).
			Bold(true)

	tabActiveStyle = lipgloss.NewStyle().
			Foreground(visualPalette.Accent).
			Background(visualPalette.Surface).
			Bold(true).
			Underline(true)

	tabInactiveStyle = lipgloss.NewStyle().
				Foreground(visualPalette.Subtle)

	sectionTitleStyle = lipgloss.NewStyle().
				Foreground(visualPalette.Text).
				Bold(true)

	sectionSubtitleStyle = lipgloss.NewStyle().
				Foreground(visualPalette.Subtle)

	groupLabelStyle = lipgloss.NewStyle().
			Foreground(visualPalette.Accent).
			Bold(true)

	keyStyle = lipgloss.NewStyle().
			Foreground(visualPalette.Subtle).
			Width(12)

	valueStyle = lipgloss.NewStyle().
			Foreground(visualPalette.Text)

	hintStyle = lipgloss.NewStyle().
			Foreground(visualPalette.Muted)

	previewBoxStyle = lipgloss.NewStyle().
			Foreground(visualPalette.Text).
			Background(visualPalette.Surface).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(visualPalette.Border).
			Padding(0, 1)

	variantBadgeStyle = lipgloss.NewStyle().
				Foreground(visualPalette.Accent).
				Background(visualPalette.Surface).
				Padding(0, 1)
)

// init upgrades the compatibility styles still used by older render helpers.
// The semantic component styles above are preferred for new UI code.
func init() {
	accentStyle = lipgloss.NewStyle().Foreground(visualPalette.Accent).Bold(true)
	mutedStyle = lipgloss.NewStyle().Foreground(visualPalette.Muted)
	errorStyle = lipgloss.NewStyle().Foreground(visualPalette.Danger).Bold(true)
	panelStyle = lipgloss.NewStyle().
		Background(visualPalette.Panel).
		Foreground(visualPalette.Text).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(visualPalette.FocusBorder).
		Padding(1, 2)
}

func renderTab(label string, active, compact bool) string {
	if active {
		if compact {
			return tabActiveStyle.Render("▸ " + label)
		}
		return tabActiveStyle.Padding(0, 1).Render(label)
	}
	return tabInactiveStyle.Render(label)
}

func renderHeader(active, width int) string {
	compact := width > 0 && width < 64
	items := make([]string, 0, len(tabs))
	for i, label := range tabs {
		items = append(items, renderTab(label, i == active, compact))
	}
	tabStrip := strings.Join(items, " ")
	logo := logoStyle.Copy().MarginRight(2).Render("ozsh")
	header := lipgloss.JoinHorizontal(lipgloss.Top, logo, tabStrip)
	if width <= 0 || lipgloss.Width(header) <= width {
		return header
	}

	// On very narrow terminals the brand remains visually separate while the
	// navigation moves to a second line instead of being clipped into the logo.
	indent := strings.Repeat(" ", lipgloss.Width("ozsh  "))
	return logo + "\n" + indent + tabStrip
}

func renderSectionHeader(title, subtitle string) string {
	if strings.TrimSpace(subtitle) == "" {
		return sectionTitleStyle.Render(title)
	}
	return sectionTitleStyle.Render(title) + "\n" + sectionSubtitleStyle.Render(subtitle)
}

func renderGroupLabel(label string) string {
	return groupLabelStyle.Render(label)
}

func renderKeyValue(label, value string) string {
	return lipgloss.JoinHorizontal(lipgloss.Top, keyStyle.Render(label), valueStyle.Render(value))
}

func renderHint(text string) string {
	return hintStyle.Render(text)
}

func renderStatus(text string, failed bool) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	if failed {
		return lipgloss.NewStyle().Foreground(visualPalette.Danger).Bold(true).Render("✕ " + text)
	}
	return lipgloss.NewStyle().Foreground(visualPalette.Success).Bold(true).Render("✓ " + text)
}

func renderPreviewBox(label, content string, width int) string {
	if width < 12 {
		width = 12
	}
	frame := previewBoxStyle.GetHorizontalFrameSize()
	inner := width - frame
	if inner < 1 {
		inner = 1
	}
	box := previewBoxStyle.Copy().Width(inner).Render(content)
	return renderGroupLabel(label) + "\n" + box
}

func renderVariantBadge(variant string) string {
	variant = strings.TrimSpace(variant)
	if variant == "" {
		return ""
	}
	return variantBadgeStyle.Render("variant: " + variant)
}
