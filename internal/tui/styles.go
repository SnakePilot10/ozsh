package tui

import (
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/lipgloss"

	"github.com/snakepilot10/ozsh/internal/config"
)

type messageLevel uint8

const (
	messageInfo messageLevel = iota
	messageSuccess
	messageWarning
	messageError
)

type styles struct {
	accent  lipgloss.Style
	muted   lipgloss.Style
	success lipgloss.Style
	warning lipgloss.Style
	error   lipgloss.Style
	panel   lipgloss.Style
}

func newStyles(theme config.ThemeConfig, ascii, noColor bool) styles {
	if noColor {
		border := lipgloss.RoundedBorder()
		if ascii {
			border = lipgloss.ASCIIBorder()
		}
		plain := lipgloss.NewStyle()
		return styles{
			accent: plain.Bold(true), muted: plain, success: plain.Bold(true),
			warning: plain.Bold(true), error: plain.Bold(true),
			panel: plain.Border(border).Padding(1, 2),
		}
	}
	accent := adaptive("#005f87", theme.Accent, "#00f5ff")
	muted := adaptive("#5f5f67", theme.Muted, "#6b6b80")
	success := adaptive("#006b45", theme.Success, "#00ff9f")
	warning := adaptive("#7a5c00", theme.Warning, "#ffe600")
	errorColor := adaptive("#b00020", theme.Error, "#ff003c")
	background := adaptive("#f7f7f8", theme.Background, "#09090d")
	foreground := lipgloss.AdaptiveColor{Light: "#202024", Dark: "#e0e0e0"}

	border := lipgloss.RoundedBorder()
	if ascii {
		border = lipgloss.ASCIIBorder()
	}
	return styles{
		accent:  lipgloss.NewStyle().Foreground(accent).Bold(true),
		muted:   lipgloss.NewStyle().Foreground(muted),
		success: lipgloss.NewStyle().Foreground(success).Bold(true),
		warning: lipgloss.NewStyle().Foreground(warning).Bold(true),
		error:   lipgloss.NewStyle().Foreground(errorColor).Bold(true),
		panel: lipgloss.NewStyle().
			Background(background).
			Foreground(foreground).
			Border(border).
			BorderForeground(accent).
			Padding(1, 2),
	}
}

func adaptive(light, dark, fallback string) lipgloss.AdaptiveColor {
	if dark == "" {
		dark = fallback
	}
	return lipgloss.AdaptiveColor{Light: light, Dark: dark}
}

func (s styles) helpStyles() help.Styles {
	return help.Styles{
		Ellipsis:       s.muted,
		ShortKey:       s.accent,
		ShortDesc:      s.muted,
		ShortSeparator: s.muted,
		FullKey:        s.accent,
		FullDesc:       s.muted,
		FullSeparator:  s.muted,
	}
}

func (s styles) message(level messageLevel) lipgloss.Style {
	switch level {
	case messageSuccess:
		return s.success
	case messageWarning:
		return s.warning
	case messageError:
		return s.error
	default:
		return s.muted
	}
}
