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
		for index := 0; index < len(parameters); index++ {
			code, err := strconv.Atoi(parameters[index])
			if err != nil {
				continue
			}
			if code == 38 || code == 48 {
				if index+1 >= len(parameters) {
					continue
				}
				mode, modeErr := strconv.Atoi(parameters[index+1])
				if modeErr != nil {
					continue
				}
				if code == 48 && (mode == 2 || mode == 5) {
					return match[0]
				}
				switch mode {
				case 2:
					index += 4
				case 5:
					index += 2
				}
				continue
			}
			if (code >= 40 && code <= 49) || (code >= 100 && code <= 107) {
				return match[0]
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

func TestBackgroundDetectorDistinguishesForegroundPayload(t *testing.T) {
	foreground := "\x1b[38;2;48;87;92mtext\x1b[0m"
	if match := firstBackgroundSGR(foreground); match != "" {
		t.Fatalf("foreground was misclassified as background: %q", match)
	}
	background := "\x1b[38;2;242;244;248;48;2;9;9;13mtext\x1b[0m"
	if match := firstBackgroundSGR(background); match == "" {
		t.Fatal("truecolor background was not detected")
	}
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
