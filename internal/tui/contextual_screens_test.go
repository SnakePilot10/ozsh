package tui

import (
	"strings"
	"testing"

	"github.com/snakepilot10/ozsh/internal/config"
)

func TestHomeWorkspaceRegions(t *testing.T) {
	assertScreenRegions(t, tabHome, []string{"System summary", "Quick actions"})
}

func TestThemesWorkspaceRegions(t *testing.T) {
	assertScreenRegions(t, tabThemes, []string{"Theme library", "Description", "Palette", "Live preview"})
}

func TestPluginsWorkspaceRegions(t *testing.T) {
	assertScreenRegions(t, tabPlugins, []string{"Recommended plugins", "Selected plugin"})
}

func TestPreviewWorkspaceRegions(t *testing.T) {
	assertScreenRegions(t, tabPreview, []string{"Scenarios", "Context", "Live preview"})
}

func assertScreenRegions(t *testing.T, tab int, labels []string) {
	t.Helper()
	for _, dimensions := range [][2]int{{100, 34}, {58, 28}} {
		model := NewModel(config.Default())
		model.width, model.height = dimensions[0], dimensions[1]
		model.setTab(tab)
		plain := plainText(model.View())
		for _, label := range labels {
			if !strings.Contains(plain, label) {
				t.Fatalf("tab %d at %dx%d lost %q:\n%s", tab, dimensions[0], dimensions[1], label, plain)
			}
		}
		assertViewBounds(t, model.View(), model.width, model.height)
	}
}
