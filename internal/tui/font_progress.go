package tui

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/snakepilot10/ozsh/internal/config"
	"github.com/snakepilot10/ozsh/internal/fonts"
	"github.com/snakepilot10/ozsh/internal/shell"
)

type termuxUsabilityState struct {
	previewEditing bool

	fontName       string
	fontDownloaded int64
	fontTotal      int64
	fontEvents     <-chan fontInstallEvent
}

type fontInstallEvent struct {
	downloaded int64
	total      int64
	result     *fontInstallResult
}

type fontProgressMsg struct {
	Downloaded int64
	Total      int64
}

func startFontInstall(cfg *config.Config, font fonts.Font) <-chan fontInstallEvent {
	events := make(chan fontInstallEvent, 1)
	snapshot := cloneConfig(cfg)
	home := os.Getenv("HOME")
	termux := shell.IsTermux()

	go func() {
		defer close(events)
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
		defer cancel()

		manager := fonts.NewManager(home, termux)
		err := manager.Install(ctx, font, func(downloaded, total int64) {
			publishFontProgress(events, fontInstallEvent{downloaded: downloaded, total: total})
		})
		result := fontInstallResult{cfg: snapshot, font: font, err: err}
		if err == nil {
			snapshot.Prompt.IconMode = config.IconModeNerd
			if saveErr := config.Save(snapshot); saveErr != nil {
				result.err = fmt.Errorf("font installed but icon mode could not be saved: %w", saveErr)
			}
		}
		events <- fontInstallEvent{result: &result}
	}()

	return events
}

func publishFontProgress(events chan fontInstallEvent, event fontInstallEvent) {
	select {
	case events <- event:
		return
	default:
	}

	select {
	case <-events:
	default:
	}
	select {
	case events <- event:
	default:
	}
}

func waitForFontInstall(events <-chan fontInstallEvent) tea.Cmd {
	return func() tea.Msg {
		if events == nil {
			return fontInstallResult{err: errors.New("font install progress stream is unavailable")}
		}
		event, ok := <-events
		if !ok {
			return fontInstallResult{err: errors.New("font install progress stream closed unexpectedly")}
		}
		if event.result != nil {
			return *event.result
		}
		return fontProgressMsg{Downloaded: event.downloaded, Total: event.total}
	}
}

func (m Model) fontInstallWorkspace(spec layoutSpec) string {
	var body strings.Builder
	body.WriteString(renderSectionHeader("Installing Nerd Font", m.fontName))
	body.WriteString("\n\n")

	percentage := 0
	if m.fontTotal > 0 {
		percentage = int(m.fontDownloaded * 100 / m.fontTotal)
		if percentage < 0 {
			percentage = 0
		}
		if percentage > 100 {
			percentage = 100
		}
	}
	barWidth := spec.contentWidth - 14
	if barWidth < 12 {
		barWidth = 12
	}
	if barWidth > 32 {
		barWidth = 32
	}
	filled := barWidth * percentage / 100
	bar := accentStyle.Render(strings.Repeat("█", filled)) + mutedStyle.Render(strings.Repeat("░", barWidth-filled))
	fmt.Fprintf(&body, "%s %3d%%\n", bar, percentage)
	if m.fontTotal > 0 {
		fmt.Fprintf(&body, "%s / %s\n", formatByteCount(m.fontDownloaded), formatByteCount(m.fontTotal))
	} else if m.fontDownloaded > 0 {
		fmt.Fprintf(&body, "%s downloaded\n", formatByteCount(m.fontDownloaded))
	} else {
		body.WriteString("Waiting for download data…\n")
	}

	body.WriteString("\n")
	if percentage >= 100 {
		body.WriteString("Download complete. Verifying SHA-256 and activating the font.")
	} else {
		body.WriteString("Downloading the pinned archive. SHA-256 verification runs before installation.")
	}
	return fitHeight(body.String(), spec.contentWidth, spec.workspaceHeight)
}

func formatByteCount(value int64) string {
	if value < 1024 {
		return fmt.Sprintf("%d B", value)
	}
	const unit = 1024
	units := []string{"KiB", "MiB", "GiB"}
	amount := float64(value)
	index := -1
	for amount >= unit && index+1 < len(units) {
		amount /= unit
		index++
	}
	return fmt.Sprintf("%.1f %s", amount, units[index])
}
