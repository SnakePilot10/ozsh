package prompt

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/snakepilot10/ozsh/internal/config"
)

func TestSimulatedUsesDisplayNameIconModeLayoutAndSymbol(t *testing.T) {
	cfg := config.Default()
	cfg.Prompt.DisplayName = "pilot"
	cfg.Prompt.IconMode = config.IconModeNerd
	cfg.Prompt.Layout = config.PromptLayoutOneLine
	cfg.Prompt.Newline = false
	cfg.Prompt.Symbol = "λ"
	user := cfg.Prompt.Segments["user"]
	user.CompatibleIcon = "USR"
	user.NerdIcon = "NF"
	cfg.Prompt.Segments["user"] = user

	preview := SimulatedWithContext(cfg, PreviewContext{Username: "system-user", Cwd: "~/repo", GitBranch: "main"})
	for _, want := range []string{"pilot", "NF", "λ"} {
		if !strings.Contains(preview, want) {
			t.Fatalf("preview missing %q: %q", want, preview)
		}
	}
	if strings.Contains(preview, "system-user") || strings.Contains(preview, "\n") {
		t.Fatalf("preview ignored display name or one-line layout: %q", preview)
	}
}

func TestGenerateUsesCustomDisplayNameAndSymbol(t *testing.T) {
	cfg := config.Default()
	cfg.Prompt.DisplayName = "pilot"
	cfg.Prompt.Symbol = "$"
	generated, err := Generate(cfg)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if !strings.Contains(generated, "pilot") || !strings.Contains(generated, "$'\\n$ '") {
		t.Fatalf("generated prompt missing display name or symbol:\n%s", generated)
	}
}

func TestGenerateOrdersCuratedPluginSourcesSafely(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	root := filepath.Join(home, ".config", "ozsh", "plugins")
	cfg := config.Default()
	cfg.Plugins.Items = []config.PluginItem{
		{Name: "zsh-syntax-highlighting", Enabled: true, Trusted: true, Source: filepath.Join(root, "zsh-syntax-highlighting"), Load: "zsh-syntax-highlighting.zsh"},
		{Name: "custom", Enabled: true, Trusted: true, Source: filepath.Join(root, "custom"), Load: "custom.zsh"},
		{Name: "fzf-tab", Enabled: true, Trusted: true, Source: filepath.Join(root, "fzf-tab"), Load: "fzf-tab.plugin.zsh"},
		{Name: "zsh-autosuggestions", Enabled: true, Trusted: true, Source: filepath.Join(root, "zsh-autosuggestions"), Load: "zsh-autosuggestions.zsh"},
	}
	generated, err := Generate(cfg)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	positions := []int{
		strings.Index(generated, "zsh-autosuggestions.zsh"),
		strings.Index(generated, "fzf-tab.plugin.zsh"),
		strings.Index(generated, "custom.zsh"),
		strings.Index(generated, "zsh-syntax-highlighting.zsh"),
	}
	for i, position := range positions {
		if position < 0 || i > 0 && position <= positions[i-1] {
			t.Fatalf("plugin source order positions = %#v", positions)
		}
	}
	if !strings.Contains(generated, "autoload -Uz compinit && compinit") {
		t.Fatal("generated prompt does not initialize completion before fzf-tab")
	}
}
