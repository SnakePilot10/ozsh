package tui

import (
	"regexp"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/snakepilot10/ozsh/internal/config"
)

var ansiSequence = regexp.MustCompile(`\x1b\[[0-?]*[ -/]*[@-~]`)

func plainText(value string) string {
	return ansiSequence.ReplaceAllString(value, "")
}

func TestHeaderSeparatesBrandFromTabs(t *testing.T) {
	model := NewModel(config.Default())
	model.width, model.height = 72, 30
	var header string
	for _, line := range strings.Split(plainText(model.View()), "\n") {
		if strings.Contains(line, "ozsh") && strings.Contains(line, "Home") {
			header = line
			break
		}
	}
	brand := strings.Index(header, "ozsh")
	home := strings.Index(header, "Home")
	if brand < 0 || home < 0 {
		t.Fatalf("header lost brand or first tab: %q", header)
	}
	gap := home - (brand + len("ozsh"))
	if gap < 2 {
		t.Fatalf("brand-to-tabs gap = %d, want at least 2 cells: %q", gap, header)
	}
}

func TestActiveAndInactiveTabsRenderDifferently(t *testing.T) {
	active := renderTab("Home", true, false)
	inactive := renderTab("Home", false, false)
	if active == inactive {
		t.Fatalf("tab states render identically: active=%q inactive=%q", active, inactive)
	}
	if lipgloss.Width(active) == 0 || lipgloss.Width(inactive) == 0 {
		t.Fatalf("tab state rendered an empty label: active=%q inactive=%q", active, inactive)
	}
}

func TestThemeDetailsSeparatePaletteFromPreview(t *testing.T) {
	model := NewModel(config.Default())
	model.setTab(tabThemes)
	model.width, model.height = 100, 34
	view := plainText(model.View())
	palette := strings.Index(view, "Palette")
	preview := strings.Index(view, "Live preview")
	if palette < 0 || preview < 0 || palette >= preview {
		t.Fatalf("theme metadata hierarchy missing:\n%s", view)
	}
	paletteLines := strings.Split(view[palette:preview], "\n")
	for _, line := range paletteLines {
		if strings.Contains(line, "~/dev/ozsh") {
			t.Fatalf("prompt output leaked into palette metadata: %q", line)
		}
	}
}

func TestFiveScreensRemainReadableAtNarrowWidth(t *testing.T) {
	headings := []string{"Home", "Prompt", "Themes", "Plugins", "Preview"}
	for tab, heading := range headings {
		model := NewModel(config.Default())
		model.width, model.height = 58, 28
		model.setTab(tab)
		view := model.View()
		plain := plainText(view)
		if !strings.Contains(plain, heading) {
			t.Fatalf("tab %d lost heading %q:\n%s", tab, heading, plain)
		}
		if !strings.Contains(strings.ToLower(plain), "apply") || !strings.Contains(strings.ToLower(plain), "quit") {
			t.Fatalf("tab %d lost global help:\n%s", tab, plain)
		}
		assertViewBounds(t, view, model.width, model.height)
	}
}

func TestAllScreensRespectResponsiveBounds(t *testing.T) {
	dimensions := [][2]int{{100, 34}, {72, 30}, {58, 28}, {48, 18}}
	for tab := range tabs {
		for _, size := range dimensions {
			model := NewModel(config.Default())
			model.width, model.height = size[0], size[1]
			model.setTab(tab)
			assertViewBounds(t, model.View(), model.width, model.height)
		}
	}
}
