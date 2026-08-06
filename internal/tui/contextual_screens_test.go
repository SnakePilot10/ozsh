package tui

import (
	"strings"
	"testing"

	"github.com/snakepilot10/ozsh/internal/config"
	"github.com/snakepilot10/ozsh/internal/plugins"
)

func TestHomeWorkspaceRegions(t *testing.T) {
	assertScreenRegions(t, tabHome, []string{"System summary", "Quick actions"})
}

func TestThemesWorkspaceRegions(t *testing.T) {
	assertScreenRegions(t, tabThemes, []string{"Theme library", "Description", "Palette", "Live preview"})
}

func TestPluginsWorkspaceRegions(t *testing.T) {
	assertScreenRegions(t, tabPlugins, []string{
		"Recommended plugins",
		"Custom plugins",
		"No custom plugins yet",
		"[a] Add custom plugin",
		"Selected plugin",
	})
}

func TestPluginsWorkspaceShowsCustomPluginAndDetails(t *testing.T) {
	cfg := config.Default()
	cfg.Plugins.Items = append(cfg.Plugins.Items, config.PluginItem{
		Name:    "my-plugin",
		Enabled: true,
		Trusted: false,
		Source:  "/tmp/my-plugin",
		Load:    "my.plugin.zsh",
	})
	model := NewModel(cfg)
	model.width, model.height = 100, 34
	model.setTab(tabPlugins)
	model.cursor = len(plugins.Catalog())

	plain := plainText(model.View())
	for _, expected := range []string{
		"Custom plugins",
		"my-plugin",
		"[untrusted]",
		"Managed path",
		"/tmp/my-plugin",
		"Load file",
		"my.plugin.zsh",
	} {
		if !strings.Contains(plain, expected) {
			t.Fatalf("custom plugin screen lost %q:\n%s", expected, plain)
		}
	}
}

func TestPluginsCursorSelectsCustomItemAfterCatalog(t *testing.T) {
	cfg := config.Default()
	cfg.Plugins.Items = append(cfg.Plugins.Items, config.PluginItem{
		Name: "my-plugin", Source: "/tmp/my-plugin", Load: "my.plugin.zsh",
	})
	model := NewModel(cfg)
	model.setTab(tabPlugins)
	model.cursor = len(plugins.Catalog())

	item, ok := model.selectedPluginListItem()
	if !ok {
		t.Fatal("selectedPluginListItem() ok = false")
	}
	if item.Kind != pluginItemCustom || item.ConfigIndex != 0 {
		t.Fatalf("selected item = %#v", item)
	}
	if model.selectionCount() != len(plugins.Catalog())+1 {
		t.Fatalf("selectionCount() = %d", model.selectionCount())
	}
}

func TestPluginsFooterOwnsAddCustomKey(t *testing.T) {
	plain := plainText(screenFooter(tabPlugins))
	for _, expected := range []string{"a add custom", "space enable", "t/u trust", "l load file", "d remove", "Ctrl+A apply"} {
		if !strings.Contains(plain, expected) {
			t.Fatalf("plugins footer lost %q: %s", expected, plain)
		}
	}
	if strings.Contains(plain, "x advanced") {
		t.Fatalf("plugins footer still exposes legacy advanced mode: %s", plain)
	}
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
