package tui

import (
	"regexp"
	"strings"
	"testing"

	"github.com/snakepilot10/ozsh/internal/config"
)

var backgroundSGR = regexp.MustCompile(`\x1b\[(?:4[0-9]|10[0-7]|48;(?:2|5);[0-9;]+)m`)

func configWithoutPromptBackgrounds() *config.Config {
	cfg := config.Default()
	for name, segment := range cfg.Prompt.Segments {
		segment.BG = ""
		cfg.Prompt.Segments[name] = segment
	}
	return cfg
}

func TestTUIChromeDoesNotEmitBackgroundColors(t *testing.T) {
	for tab := range tabs {
		model := NewModel(configWithoutPromptBackgrounds())
		model.width, model.height = 100, 34
		model.setTab(tab)
		if match := backgroundSGR.FindString(model.View()); match != "" {
			t.Fatalf("tab %d emitted background SGR %q", tab, match)
		}
	}
}

func TestSelectionRemainsIdentifiableWithoutColor(t *testing.T) {
	model := NewModel(config.Default())
	model.width, model.height = 58, 28
	model.setTab(tabPlugins)
	plain := plainText(model.View())
	if !strings.Contains(plain, "› [") {
		t.Fatalf("selected row lost marker:\n%s", plain)
	}
	if renderTab("Plugins", true, false) == renderTab("Plugins", false, false) {
		t.Fatal("active and inactive tabs render identically")
	}
}
