package prompt

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/snakepilot10/ozsh/internal/config"
)

func TestGenerate_Basic(t *testing.T) {
	cfg := &config.Config{
		Prompt: config.PromptConfig{
			Order: []string{"user", "cwd"},
			Segments: map[string]config.SegmentConfig{
				"user": {Enabled: true, FG: "cyan", Bold: true},
				"cwd":  {Enabled: true, FG: "blue", Bold: false},
			},
		},
	}

	output, err := Generate(cfg)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	if !strings.Contains(output, "parts+=(\"%F{cyan}%B%n%b%f\")") {
		t.Errorf("Output missing user segment:\n%s", output)
	}
	if !strings.Contains(output, "parts+=(\"%F{blue}%~%f\")") {
		t.Errorf("Output missing cwd segment:\n%s", output)
	}
}

func TestGenerate_DisabledSegment(t *testing.T) {
	cfg := &config.Config{
		Prompt: config.PromptConfig{
			Order: []string{"user", "time"},
			Segments: map[string]config.SegmentConfig{
				"user": {Enabled: true, FG: "cyan", Bold: true},
				"time": {Enabled: false, FG: "white", Bold: false},
			},
		},
	}

	output, err := Generate(cfg)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	if !strings.Contains(output, "parts+=(\"%F{cyan}%B%n%b%f\")") {
		t.Errorf("Output missing user segment:\n%s", output)
	}
	if strings.Contains(output, "%F{white}%*") {
		t.Errorf("Output contains disabled time segment:\n%s", output)
	}
}

func TestGenerate_Status(t *testing.T) {
	cfg := &config.Config{
		Prompt: config.PromptConfig{
			Order: []string{"status"},
			Segments: map[string]config.SegmentConfig{
				"status": {Enabled: true, FG: "red", Bold: true, ShowSuccess: true},
			},
		},
	}

	output, err := Generate(cfg)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	if !strings.Contains(output, "if [[ \"$last_status\" == \"0\" ]]; then") {
		t.Errorf("Output missing success check:\n%s", output)
	}
	if !strings.Contains(output, "parts+=(\"%F{red}%B✓%b%f\")") {
		t.Errorf("Output missing success icon:\n%s", output)
	}
}

func TestGenerate_StatusNoSuccess(t *testing.T) {
	cfg := &config.Config{
		Prompt: config.PromptConfig{
			Order: []string{"status"},
			Segments: map[string]config.SegmentConfig{
				"status": {Enabled: true, FG: "red", Bold: true, ShowSuccess: false},
			},
		},
	}

	output, err := Generate(cfg)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	if !strings.Contains(output, "if [[ \"$last_status\" != \"0\" ]]; then") {
		t.Errorf("Output missing failure check:\n%s", output)
	}
	if strings.Contains(output, "== \"0\"") {
		t.Errorf("Output should not contain success check:\n%s", output)
	}
}

func TestGenerate_Git(t *testing.T) {
	cfg := &config.Config{
		Prompt: config.PromptConfig{
			Order: []string{"git"},
			Segments: map[string]config.SegmentConfig{
				"git": {Enabled: true, FG: "green", Bold: false},
			},
		},
	}

	output, err := Generate(cfg)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	if !strings.Contains(output, "ozsh_git_branch") {
		t.Errorf("Output missing git function:\n%s", output)
	}
	if !strings.Contains(output, "parts+=(\"%F{green}${git_branch}%f\")") {
		t.Errorf("Output missing git segment in parts:\n%s", output)
	}
}

func TestGenerate_ZshSyntaxShape(t *testing.T) {
	cfg := config.Default()

	output, err := Generate(cfg)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	required := []string{
		"autoload -Uz colors && colors",
		"ozsh_join() {",
		"ozsh_prompt() {",
		"  local last_status=\"$?\"",
		"  local parts=()",
		"  local ozsh_separator='  '",
		"  PROMPT=\"$(ozsh_join \"$ozsh_separator\" \"${parts[@]}\")\"$'\\n❯ '",
		"}",
		"precmd_functions+=(ozsh_prompt)",
	}
	for _, want := range required {
		if !strings.Contains(output, want) {
			t.Errorf("Generate() output missing %q:\n%s", want, output)
		}
	}
}

func TestGenerate_StatusShowSuccessFalseOmitsSuccessIcon(t *testing.T) {
	cfg := &config.Config{
		Prompt: config.PromptConfig{
			Order: []string{"status"},
			Segments: map[string]config.SegmentConfig{
				"status": {Enabled: true, FG: "red", Bold: true, ShowSuccess: false},
			},
		},
	}

	output, err := Generate(cfg)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	if strings.Contains(output, "✓") {
		t.Errorf("Generate() output contains success icon when show_success=false:\n%s", output)
	}
}

func TestGenerate_GitSegmentSkipsOutsideRepo(t *testing.T) {
	cfg := &config.Config{
		Prompt: config.PromptConfig{
			Order: []string{"git"},
			Segments: map[string]config.SegmentConfig{
				"git": {Enabled: true, FG: "green"},
			},
		},
	}

	output, err := Generate(cfg)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	if !strings.Contains(output, "git rev-parse --is-inside-work-tree >/dev/null 2>&1 || return") {
		t.Errorf("Generate() git helper does not skip outside repositories:\n%s", output)
	}
}

func TestGenerate_Idempotent(t *testing.T) {
	cfg := config.Default()

	first, err := Generate(cfg)
	if err != nil {
		t.Fatalf("first Generate() error = %v", err)
	}
	second, err := Generate(cfg)
	if err != nil {
		t.Fatalf("second Generate() error = %v", err)
	}

	if first != second {
		t.Errorf("Generate() is not idempotent:\nfirst:\n%s\nsecond:\n%s", first, second)
	}
}

func TestGenerate_RightPromptAndAdditionalSegments(t *testing.T) {
	cfg := config.Default()
	cfg.Prompt.Order = []string{"host", "venv", "node", "go", "battery", "jobs"}
	cfg.Prompt.RightOrder = []string{"time"}
	cfg.Prompt.RightPrompt = true
	for _, name := range append(cfg.Prompt.Order, cfg.Prompt.RightOrder...) {
		segment := cfg.Prompt.Segments[name]
		segment.Enabled = true
		cfg.Prompt.Segments[name] = segment
	}

	output, err := Generate(cfg)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	required := []string{
		"ozsh_venv_name()",
		"ozsh_node_version()",
		"ozsh_go_version()",
		"ozsh_battery_level()",
		"parts+=(\"%F{cyan}%m%f\")",
		"right_parts+=(\"%F{blue}%*%f\")",
		"RPROMPT=\"$(ozsh_join \"$ozsh_separator\" \"${right_parts[@]}\")\"",
	}
	for _, want := range required {
		if !strings.Contains(output, want) {
			t.Errorf("Generate() output missing %q:\n%s", want, output)
		}
	}
}

func TestGenerate_UsesConfiguredSeparator(t *testing.T) {
	cfg := config.Default()
	cfg.Prompt.Separator = " | "

	output, err := Generate(cfg)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	if !strings.Contains(output, `local ozsh_separator=' | '`) {
		t.Errorf("Generate() output missing custom separator:\n%s", output)
	}
}

func TestGenerate_HeaderAndPluginSources(t *testing.T) {
	cfg := config.Default()
	cfg.Header.Enabled = true
	cfg.Header.Style = "ascii"
	cfg.Header.Text = "hello"
	cfg.Plugins.Items = []config.PluginItem{
		{Name: "enabled", Enabled: true, Trusted: true, Source: "/tmp/plugin", Load: "plugin.zsh"},
		{Name: "disabled", Enabled: false, Source: "/tmp/disabled.zsh"},
		{Name: "untrusted", Enabled: true, Trusted: false, Source: "/tmp/untrusted.zsh"},
	}

	output, err := Generate(cfg)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	if !strings.Contains(output, `print -P -- "$(ozsh_color "#00f5ff" "cyan")"'hello%f'`) {
		t.Errorf("Generate() output missing header:\n%s", output)
	}
	if !strings.Contains(output, `ozsh_source_plugin "/tmp/plugin/plugin.zsh"`) {
		t.Errorf("Generate() output missing enabled plugin source:\n%s", output)
	}
	if strings.Contains(output, "/tmp/disabled.zsh") {
		t.Errorf("Generate() output contains disabled plugin source:\n%s", output)
	}
	if strings.Contains(output, "/tmp/untrusted.zsh") {
		t.Errorf("Generate() output contains untrusted plugin source:\n%s", output)
	}
}

func TestGenerate_DoesNotReadTemplateFromWorkingDirectory(t *testing.T) {
	tmp := t.TempDir()
	if err := os.Mkdir(filepath.Join(tmp, "templates"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmp, "templates", "omega.zsh.tmpl"), []byte("MALICIOUS_TEMPLATE"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Chdir(tmp)

	cfg := config.Default()
	cfg.Prompt.Style = "omega"

	output, err := Generate(cfg)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if strings.Contains(output, "MALICIOUS_TEMPLATE") {
		t.Fatalf("Generate() read template from working directory:\n%s", output)
	}
	if !strings.Contains(output, "Generated by ozsh template") {
		t.Fatalf("Generate() did not use embedded template:\n%s", output)
	}
}

func TestGenerate_HostileHeaderAndSeparatorDoNotExecute(t *testing.T) {
	if _, err := exec.LookPath("zsh"); err != nil {
		t.Skip("zsh not available")
	}
	tmp := t.TempDir()
	headerSentinel := filepath.Join(tmp, "header-pwned")
	separatorSentinel := filepath.Join(tmp, "separator-pwned")

	cfg := config.Default()
	cfg.Header.Enabled = true
	cfg.Header.Style = "ascii"
	cfg.Header.Text = "hello $(touch " + headerSentinel + ") `touch " + headerSentinel + "` \" '"
	cfg.Prompt.Separator = "$(touch " + separatorSentinel + "):`touch " + separatorSentinel + "`:\")}; touch " + separatorSentinel + " #"
	cfg.Prompt.Order = []string{"user", "cwd"}
	cfg.Prompt.RightOrder = nil

	output, err := Generate(cfg)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	script := filepath.Join(tmp, "omega.zsh")
	if err := os.WriteFile(script, []byte(output+"\nozsh_prompt\n"), 0600); err != nil {
		t.Fatal(err)
	}

	if out, err := exec.Command("zsh", "-n", script).CombinedOutput(); err != nil {
		t.Fatalf("zsh -n failed: %v\n%s\n%s", err, out, output)
	}
	if out, err := exec.Command("zsh", "-f", script).CombinedOutput(); err != nil {
		t.Fatalf("zsh execution failed: %v\n%s\n%s", err, out, output)
	}
	for _, sentinel := range []string{headerSentinel, separatorSentinel} {
		if _, err := os.Stat(sentinel); err == nil {
			t.Fatalf("hostile generated shell executed sentinel %s\n%s", sentinel, output)
		} else if !os.IsNotExist(err) {
			t.Fatalf("stat sentinel %s: %v", sentinel, err)
		}
	}
}

func TestGenerate_DisableHeavySegmentsSkipsRuntimeCommands(t *testing.T) {
	cfg := config.Default()
	cfg.Prompt.DisableHeavySegments = true
	cfg.Prompt.Order = []string{"user", "git", "node", "go", "battery"}
	for _, name := range cfg.Prompt.Order {
		segment := cfg.Prompt.Segments[name]
		segment.Enabled = true
		cfg.Prompt.Segments[name] = segment
	}

	output, err := Generate(cfg)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	if !strings.Contains(output, `%n`) {
		t.Fatalf("Generate() output missing light user segment:\n%s", output)
	}
	for _, heavy := range []string{"ozsh_git_branch", "ozsh_node_version", "ozsh_go_version", "ozsh_battery_level"} {
		if strings.Contains(output, heavy) {
			t.Fatalf("Generate() output contains heavy helper %q:\n%s", heavy, output)
		}
	}
}

func TestSimulated_DisableHeavySegmentsSkipsRuntimePreview(t *testing.T) {
	cfg := config.Default()
	cfg.Prompt.DisableHeavySegments = true
	cfg.Prompt.Order = []string{"user", "git"}
	cfg.Prompt.Segments["git"] = config.SegmentConfig{Enabled: true, FG: "green"}

	output := Simulated(cfg)
	if strings.Contains(output, "main") {
		t.Fatalf("Simulated() rendered heavy git segment: %q", output)
	}
}

func TestGenerate_HexColorFallbackHelper(t *testing.T) {
	cfg := config.Default()
	cfg.Prompt.Order = []string{"user"}
	cfg.Prompt.Segments["user"] = config.SegmentConfig{Enabled: true, FG: "#ff003c", Bold: true}

	output, err := Generate(cfg)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	required := []string{
		"ozsh_color()",
		`COLORTERM`,
		`$(ozsh_color "#ff003c" "red")%B%n%b%f`,
	}
	for _, want := range required {
		if !strings.Contains(output, want) {
			t.Errorf("Generate() output missing %q:\n%s", want, output)
		}
	}
}

func TestSimulated_RendersHexColorANSI(t *testing.T) {
	cfg := config.Default()
	cfg.Prompt.Order = []string{"user"}
	cfg.Prompt.Segments["user"] = config.SegmentConfig{Enabled: true, FG: "#00f5ff"}

	output := Simulated(cfg)
	if !strings.Contains(output, "\x1b[38;2;0;245;255m") {
		t.Fatalf("Simulated() output missing truecolor ANSI: %q", output)
	}
}
