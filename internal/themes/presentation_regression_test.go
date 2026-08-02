package themes

import (
	"testing"

	"github.com/snakepilot10/ozsh/internal/config"
)

func TestPowerlineAppliesConnectedSegmentStyling(t *testing.T) {
	preset, ok := Get("powerline", "")
	if !ok {
		t.Fatal("powerline preset not found")
	}
	cfg := Apply(config.Default(), preset)

	for _, name := range []string{"user", "cwd", "git"} {
		segment := cfg.Prompt.Segments[name]
		if segment.BG == "" {
			t.Fatalf("powerline segment %q has no background", name)
		}
		if !segment.Bold {
			t.Fatalf("powerline segment %q is not bold", name)
		}
		if segment.NerdIcon == "" || segment.NerdIcon == segment.CompatibleIcon {
			t.Fatalf("powerline segment %q has no distinct Nerd icon: %+v", name, segment)
		}
	}
}

func TestDraculaUsesRightPromptAndPerSegmentPalette(t *testing.T) {
	preset, ok := Get("dracula", "")
	if !ok {
		t.Fatal("dracula preset not found")
	}
	cfg := Apply(config.Default(), preset)

	if len(cfg.Prompt.RightOrder) != 1 || cfg.Prompt.RightOrder[0] != "time" || !cfg.Prompt.RightPrompt {
		t.Fatalf("Dracula right prompt = enabled:%t order:%v, want time", cfg.Prompt.RightPrompt, cfg.Prompt.RightOrder)
	}
	want := map[string]string{
		"user": "#ff79c6",
		"cwd":  "#bd93f9",
		"git":  "#50fa7b",
		"time": "#8be9fd",
	}
	for name, color := range want {
		segment := cfg.Prompt.Segments[name]
		if segment.FG != color {
			t.Fatalf("Dracula %s color = %q, want %q", name, segment.FG, color)
		}
		if !segment.Enabled {
			t.Fatalf("Dracula %s segment is disabled", name)
		}
	}
}

func TestMinimalClearsDecorationsInsteadOfInheritingPreviousTheme(t *testing.T) {
	powerline, _ := Get("powerline", "")
	minimal, _ := Get("minimal", "")
	base := Apply(config.Default(), powerline)
	cfg := Apply(base, minimal)

	for _, name := range []string{"user", "cwd", "git", "status", "time", "host"} {
		segment := cfg.Prompt.Segments[name]
		if segment.BG != "" {
			t.Fatalf("minimal inherited background on %q: %q", name, segment.BG)
		}
		if segment.NerdIcon != "" || segment.CompatibleIcon != "" {
			t.Fatalf("minimal inherited icon on %q: compatible=%q nerd=%q", name, segment.CompatibleIcon, segment.NerdIcon)
		}
	}
}
