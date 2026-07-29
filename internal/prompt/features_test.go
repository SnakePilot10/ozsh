package prompt

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/snakepilot10/ozsh/internal/config"
)

func TestGenerateAdvancedShellFeatures(t *testing.T) {
	cfg := config.Default()
	cfg.Prompt.TransientPrompt = true
	cfg.Prompt.OSC7 = true
	cfg.Prompt.OSC133 = true
	cfg.Prompt.Order = []string{"status", "execution_time", "python", "rust"}
	for _, name := range cfg.Prompt.Order {
		segment := cfg.Prompt.Segments[name]
		segment.Enabled = true
		cfg.Prompt.Segments[name] = segment
	}
	python := cfg.Prompt.Segments["python"]
	python.When = "virtualenv"
	python.WhenEnv = "VIRTUAL_ENV"
	python.Icon = "py"
	python.LeadingSymbol = "["
	python.TrailingSymbol = "]"
	python.PaddingLeft = 1
	python.Underline = true
	cfg.Prompt.Segments["python"] = python

	output, err := Generate(cfg)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	for _, want := range []string{
		"ozsh_python_version()", "ozsh_rust_version()", "ozsh_preexec()",
		"ozsh_cached", "OZSH_TRANSIENT_PROMPT", "add-zle-hook-widget line-finish",
		"ozsh_uri_encode()", "\\e]133;D;%d", `[[ -n "${VIRTUAL_ENV:-}" ]]`,
	} {
		if !strings.Contains(output, want) {
			t.Errorf("Generate() output missing %q", want)
		}
	}
	if _, err := exec.LookPath("zsh"); err == nil {
		script := filepath.Join(t.TempDir(), "omega.zsh")
		if err := os.WriteFile(script, []byte(output), 0o600); err != nil {
			t.Fatal(err)
		}
		if syntax, err := exec.Command("zsh", "-n", script).CombinedOutput(); err != nil {
			t.Fatalf("generated feature script syntax error: %v\n%s", err, syntax)
		}
	}
}

func TestGeneratedPromptPreservesFailureStatus(t *testing.T) {
	if _, err := exec.LookPath("zsh"); err != nil {
		t.Skip("zsh not available")
	}
	cfg := config.Default()
	cfg.Prompt.Newline = false
	cfg.Prompt.Order = []string{"status"}
	status := cfg.Prompt.Segments["status"]
	status.Enabled = true
	status.ShowSuccess = false
	cfg.Prompt.Segments["status"] = status
	output, err := Generate(cfg)
	if err != nil {
		t.Fatal(err)
	}
	script := filepath.Join(t.TempDir(), "status.zsh")
	harness := output + "\nfalse\nozsh_prompt\nprint -r -- \"$PROMPT\"\n"
	if err := os.WriteFile(script, []byte(harness), 0o600); err != nil {
		t.Fatal(err)
	}
	result, err := exec.Command("zsh", "-f", script).CombinedOutput()
	if err != nil {
		t.Fatalf("zsh status harness error: %v\n%s", err, result)
	}
	if !strings.Contains(string(result), "✘ 1") {
		t.Fatalf("prompt did not preserve failure status: %q", result)
	}
}

func TestGeneratedCachePersistsInParentShell(t *testing.T) {
	if _, err := exec.LookPath("zsh"); err != nil {
		t.Skip("zsh not available")
	}
	cfg := config.Default()
	cfg.Prompt.Order = []string{"python"}
	python := cfg.Prompt.Segments["python"]
	python.Enabled = true
	python.CacheTTL = 60
	cfg.Prompt.Segments["python"] = python
	output, err := Generate(cfg)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	countFile := filepath.Join(dir, "count")
	script := filepath.Join(dir, "cache.zsh")
	harness := output + `
cache_probe() {
  print x >> "$COUNT_FILE"
  print -r -- value
}
ozsh_cached key 60 cache_probe
first="$REPLY"
ozsh_cached key 60 cache_probe
print -r -- "$first:$REPLY"
`
	if err := os.WriteFile(script, []byte(harness), 0o600); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("zsh", "-f", script)
	cmd.Env = append(os.Environ(), "COUNT_FILE="+countFile)
	result, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("zsh cache harness error: %v\n%s", err, result)
	}
	if !strings.Contains(string(result), "value:value") {
		t.Fatalf("cache result = %q", result)
	}
	count, err := os.ReadFile(countFile)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(count), "x") != 1 {
		t.Fatalf("cache probe executed more than once: %q", count)
	}
}

func TestSimulatedAdvancedStylesAndConditions(t *testing.T) {
	cfg := config.Default()
	cfg.Prompt.Order = []string{"python"}
	segment := cfg.Prompt.Segments["python"]
	segment.Enabled = true
	segment.When = "virtualenv"
	segment.Icon = "py"
	segment.LeadingSymbol = "["
	segment.TrailingSymbol = "]"
	segment.PaddingLeft = 1
	segment.PaddingRight = 1
	segment.Italic = true
	segment.Underline = true
	cfg.Prompt.Segments["python"] = segment
	ctx := DefaultPreviewContext()
	ctx.Venv = "venv"
	output := SimulatedWithContext(cfg, ctx)
	if !strings.Contains(output, "[py 3.12.4]") || !strings.Contains(output, "\x1b[3m\x1b[4m") {
		t.Fatalf("advanced preview style missing: %q", output)
	}
	ctx.Venv = ""
	if output := SimulatedWithContext(cfg, ctx); strings.Contains(output, "3.12.4") {
		t.Fatalf("conditional segment rendered without virtualenv: %q", output)
	}
}
