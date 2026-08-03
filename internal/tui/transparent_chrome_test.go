package tui

import (
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
	"github.com/snakepilot10/ozsh/internal/config"
)

var sgrSequence = regexp.MustCompile(`\x1b\[([0-9;]*)m`)

func firstBackgroundSGR(value string) string {
	for _, match := range sgrSequence.FindAllStringSubmatch(value, -1) {
		parameters := strings.Split(match[1], ";")
		for index, parameter := range parameters {
			code, err := strconv.Atoi(parameter)
			if err != nil {
				continue
			}
			if (code >= 40 && code <= 49) || (code >= 100 && code <= 107) {
				return match[0]
			}
			if code == 48 && index+1 < len(parameters) {
				mode, modeErr := strconv.Atoi(parameters[index+1])
				if modeErr == nil && (mode == 2 || mode == 5) {
					return match[0]
				}
			}
		}
	}
	return ""
}

func configWithoutPromptBackgrounds() *config.Config {
	cfg := config.Default()
	for name, segment := range cfg.Prompt.Segments {
		segment.BG = ""
		cfg.Prompt.Segments[name] = segment
	}
	return cfg
}

func useTrueColor(t *testing.T) {
	t.Helper()
	previous := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() { lipgloss.SetColorProfile(previous) })
}

func TestTUIChromeDoesNotEmitBackgroundColors(t *testing.T) {
	useTrueColor(t)
	for tab := range tabs {
		model := NewModel(configWithoutPromptBackgrounds())
		model.width, model.height = 100, 34
		model.setTab(tab)
		if match := firstBackgroundSGR(model.View()); match != "" {
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
