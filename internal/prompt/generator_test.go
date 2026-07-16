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
		"ozsh_prompt_text() {",
		"  local ozsh_raw_separator='  '",
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

	if !strings.Contains(output, `local ozsh_raw_separator=' | '`) {
		t.Errorf("Generate() output missing custom separator:\n%s", output)
	}
}

func TestGenerate_ValidatesConfig(t *testing.T) {
	for _, tc := range []struct {
		name string
		mut  func(*config.Config)
	}{
		{name: "invalid style", mut: func(cfg *config.Config) { cfg.Prompt.Style = "../bad" }},
		{name: "invalid separator control", mut: func(cfg *config.Config) { cfg.Prompt.Separator = "bad\nsep" }},
		{name: "invalid color", mut: func(cfg *config.Config) {
			cfg.Prompt.Segments["user"] = config.SegmentConfig{Enabled: true, FG: "hotpink"}
		}},
		{name: "unknown segment", mut: func(cfg *config.Config) { cfg.Prompt.Order = append(cfg.Prompt.Order, "missing") }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := config.Default()
			tc.mut(cfg)
			if _, err := Generate(cfg); err == nil {
				t.Fatal("Generate() error = nil, want invalid config error")
			}
		})
	}
}

func TestGenerate_PluginSources(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	cfg := config.Default()
	pluginRoot := filepath.Join(home, ".config", "ozsh", "plugins")
	cfg.Plugins.Items = []config.PluginItem{
		{Name: "enabled", Enabled: true, Trusted: true, Source: filepath.Join(pluginRoot, "enabled"), Load: "plugin.zsh"},
		{Name: "disabled", Enabled: false, Source: filepath.Join(pluginRoot, "disabled"), Load: "plugin.zsh"},
		{Name: "untrusted", Enabled: true, Trusted: false, Source: filepath.Join(pluginRoot, "untrusted"), Load: "plugin.zsh"},
		{Name: "hostile", Enabled: true, Trusted: true, Source: filepath.Join(pluginRoot, "hostile"), Load: "$(touch plugin-pwned).zsh"},
	}

	output, err := Generate(cfg)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	if !strings.Contains(output, `ozsh_source_plugin '`+filepath.Join(pluginRoot, "enabled", "plugin.zsh")+`'`) {
		t.Errorf("Generate() output missing enabled plugin source:\n%s", output)
	}
	if strings.Contains(output, filepath.Join(pluginRoot, "disabled")) {
		t.Errorf("Generate() output contains disabled plugin source:\n%s", output)
	}
	if strings.Contains(output, filepath.Join(pluginRoot, "untrusted")) {
		t.Errorf("Generate() output contains untrusted plugin source:\n%s", output)
	}
	if strings.Contains(output, "\""+filepath.Join(pluginRoot, "hostile", "$(touch plugin-pwned).zsh")+"\"") {
		t.Fatalf("Generate() double-quoted hostile plugin path:\n%s", output)
	}
	if !strings.Contains(output, `ozsh_source_plugin '`+filepath.Join(pluginRoot, "hostile", "$(touch plugin-pwned).zsh")+`'`) {
		t.Fatalf("Generate() missing single-quoted hostile plugin path:\n%s", output)
	}
}

func TestGenerate_ZshJoinPreservesEmptyParts(t *testing.T) {
	if _, err := exec.LookPath("zsh"); err != nil {
		t.Skip("zsh not available")
	}
	tmp := t.TempDir()
	output, err := Generate(config.Default())
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	script := filepath.Join(tmp, "join.zsh")
	body := output + `
print -r -- "$(ozsh_join ' | ' '' 'second')"
print -r -- "$(ozsh_join ' | ' 'first' '' 'third')"
print -r -- "$(ozsh_join ' | ' '')"
`
	if err := os.WriteFile(script, []byte(body), 0600); err != nil {
		t.Fatal(err)
	}
	out, err := exec.Command("zsh", "-f", script).CombinedOutput()
	if err != nil {
		t.Fatalf("zsh join script failed: %v\n%s", err, out)
	}
	if got, want := string(out), " | second\nfirst |  | third\n\n"; got != want {
		t.Fatalf("ozsh_join output = %q, want %q", got, want)
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

func TestGenerate_HostileSeparatorDoesNotExecute(t *testing.T) {
	if _, err := exec.LookPath("zsh"); err != nil {
		t.Skip("zsh not available")
	}
	tmp := t.TempDir()
	separatorSentinel := filepath.Join(tmp, "separator-pwned")

	cfg := config.Default()
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
	if _, err := os.Stat(separatorSentinel); err == nil {
		t.Fatalf("hostile generated shell executed sentinel %s\n%s", separatorSentinel, output)
	} else if !os.IsNotExist(err) {
		t.Fatalf("stat sentinel %s: %v", separatorSentinel, err)
	}
}

func TestGenerate_PromptSubstRendersUntrustedDataLiterally(t *testing.T) {
	if _, err := exec.LookPath("zsh"); err != nil {
		t.Skip("zsh not available")
	}
	tmp := t.TempDir()
	fakeBin := filepath.Join(tmp, "bin")
	if err := os.Mkdir(fakeBin, 0755); err != nil {
		t.Fatal(err)
	}
	gitScript := filepath.Join(fakeBin, "git")
	if err := os.WriteFile(gitScript, []byte(`#!/bin/sh
case "$1 $2" in
  "rev-parse --is-inside-work-tree") exit 0 ;;
esac
case "$1 $2 $3" in
  "symbolic-ref --short HEAD") cat "$OZSH_FAKE_GIT_BRANCH_FILE"; exit 0 ;;
esac
exit 1
`), 0755); err != nil {
		t.Fatal(err)
	}

	sepPayload := hostilePromptPayload("sep")
	gitPayload := hostilePromptPayload("git")
	venvPayload := hostilePromptPayload("venv")
	gitPayloadFile := filepath.Join(tmp, "git-payload.txt")
	if err := os.WriteFile(gitPayloadFile, []byte(gitPayload+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	cfg := config.Default()
	cfg.Prompt.Separator = sepPayload
	cfg.Prompt.Order = []string{"user", "cwd", "git"}
	cfg.Prompt.RightPrompt = true
	cfg.Prompt.RightOrder = []string{"venv", "time"}
	for _, name := range append(cfg.Prompt.Order, cfg.Prompt.RightOrder...) {
		segment := cfg.Prompt.Segments[name]
		segment.Enabled = true
		cfg.Prompt.Segments[name] = segment
	}
	output, err := Generate(cfg)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	omega := filepath.Join(tmp, "omega.zsh")
	if err := os.WriteFile(omega, []byte(output), 0600); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command("zsh", "-n", omega).CombinedOutput(); err != nil {
		t.Fatalf("zsh -n failed: %v\n%s\n%s", err, out, output)
	}

	harness := filepath.Join(tmp, "harness.zsh")
	harnessBody := `source ./omega.zsh
emulate -L zsh
setopt PROMPT_SUBST PROMPT_PERCENT PROMPT_BANG
ozsh_prompt
print -P -- "$PROMPT" > prompt-all-on.txt
print -P -- "$RPROMPT" > rprompt-all-on.txt
unsetopt PROMPT_BANG
ozsh_prompt
print -P -- "$PROMPT" > prompt-no-bang.txt
print -P -- "$RPROMPT" > rprompt-no-bang.txt
unsetopt PROMPT_SUBST
setopt PROMPT_PERCENT
ozsh_prompt
print -P -- "$PROMPT" > prompt-no-subst.txt
print -P -- "$RPROMPT" > rprompt-no-subst.txt
setopt PROMPT_SUBST
unsetopt PROMPT_PERCENT PROMPT_BANG
ozsh_prompt
eval "print -r -- \"$PROMPT\"" > prompt-no-percent.txt
eval "print -r -- \"$RPROMPT\"" > rprompt-no-percent.txt
`
	if err := os.WriteFile(harness, []byte(harnessBody), 0600); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("zsh", "-f", harness)
	cmd.Dir = tmp
	cmd.Env = append(os.Environ(),
		"HOME="+tmp,
		"PATH="+fakeBin+string(os.PathListSeparator)+os.Getenv("PATH"),
		"LC_ALL=C",
		"OZSH_FAKE_GIT_BRANCH_FILE="+gitPayloadFile,
		"VIRTUAL_ENV="+filepath.Join(tmp, venvPayload),
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("zsh harness failed: %v\n%s\n%s", err, out, output)
	}
	for _, sentinel := range []string{"sep-cmd", "sep-bt", "git-cmd", "git-bt", "venv-cmd", "venv-bt"} {
		if _, err := os.Stat(filepath.Join(tmp, sentinel)); err == nil {
			t.Fatalf("hostile prompt payload executed sentinel %s\n%s", sentinel, output)
		} else if !os.IsNotExist(err) {
			t.Fatalf("stat sentinel %s: %v", sentinel, err)
		}
	}
	for _, file := range []string{"prompt-all-on.txt", "prompt-no-bang.txt", "prompt-no-subst.txt", "prompt-no-percent.txt"} {
		contents := readTestFile(t, filepath.Join(tmp, file))
		assertHostilePayloadRendered(t, file, contents, "sep", 2)
		assertHostilePayloadRendered(t, file, contents, "git", 1)
		if file != "prompt-no-percent.txt" && (strings.Contains(contents, "%F{") || strings.Contains(contents, "%f")) {
			t.Fatalf("%s still contains unexpanded ozsh color escapes:\n%s", file, contents)
		}
		if file != "prompt-no-percent.txt" && !strings.Contains(contents, "❯") {
			t.Fatalf("%s missing controlled prompt marker:\n%s", file, contents)
		}
	}
	for _, file := range []string{"rprompt-all-on.txt", "rprompt-no-bang.txt", "rprompt-no-subst.txt", "rprompt-no-percent.txt"} {
		contents := readTestFile(t, filepath.Join(tmp, file))
		assertHostilePayloadRendered(t, file, contents, "sep", 1)
		assertHostilePayloadRendered(t, file, contents, "venv", 1)
		if file != "rprompt-no-percent.txt" && (strings.Contains(contents, "%F{") || strings.Contains(contents, "%f") || strings.Contains(contents, "%*")) {
			t.Fatalf("%s still contains unexpanded ozsh prompt escapes:\n%s", file, contents)
		}
	}
}

func TestGenerate_PromptTextReadsAllInput(t *testing.T) {
	if _, err := exec.LookPath("zsh"); err != nil {
		t.Skip("zsh not available")
	}
	tmp := t.TempDir()
	output, err := Generate(config.Default())
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	script := filepath.Join(tmp, "stdin.zsh")
	body := output + `
setopt PROMPT_SUBST PROMPT_PERCENT PROMPT_BANG
printf 'first\nsecond\n' | ozsh_prompt_text_from_stdin > out.txt
`
	if err := os.WriteFile(script, []byte(body), 0600); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("zsh", "-f", script)
	cmd.Dir = tmp
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("zsh stdin harness failed: %v\n%s", err, out)
	}
	if got, want := readTestFile(t, filepath.Join(tmp, "out.txt")), "firstsecond\n"; got != want {
		t.Fatalf("ozsh_prompt_text_from_stdin output = %q, want %q", got, want)
	}
}

func TestGenerate_DynamicCommandSegmentsUsePromptText(t *testing.T) {
	if _, err := exec.LookPath("zsh"); err != nil {
		t.Skip("zsh not available")
	}
	tmp := t.TempDir()
	cfg := config.Default()
	cfg.Prompt.Order = []string{"node", "go", "battery"}
	for _, name := range cfg.Prompt.Order {
		segment := cfg.Prompt.Segments[name]
		segment.Enabled = true
		cfg.Prompt.Segments[name] = segment
	}
	output, err := Generate(cfg)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	omega := filepath.Join(tmp, "omega.zsh")
	if err := os.WriteFile(omega, []byte(output), 0600); err != nil {
		t.Fatal(err)
	}
	harness := filepath.Join(tmp, "dynamic.zsh")
	harnessBody := `setopt PROMPT_SUBST PROMPT_PERCENT PROMPT_BANG
source ./omega.zsh
ozsh_node_version() { print -r -- "$OZSH_FAKE_NODE" }
ozsh_go_version() { print -r -- "$OZSH_FAKE_GO" }
ozsh_battery_level() { print -r -- "$OZSH_FAKE_BATTERY" }
ozsh_prompt
print -P -- "$PROMPT" > dynamic.txt
`
	if err := os.WriteFile(harness, []byte(harnessBody), 0600); err != nil {
		t.Fatal(err)
	}
	nodePayload := hostilePromptPayload("node")
	goPayload := hostilePromptPayload("go")
	batteryPayload := hostilePromptPayload("battery")
	cmd := exec.Command("zsh", "-f", harness)
	cmd.Dir = tmp
	cmd.Env = append(os.Environ(), "HOME="+tmp, "LC_ALL=C", "OZSH_FAKE_NODE="+nodePayload, "OZSH_FAKE_GO="+goPayload, "OZSH_FAKE_BATTERY="+batteryPayload)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("zsh dynamic harness failed: %v\n%s", err, out)
	}
	for _, sentinel := range []string{"node-cmd", "node-bt", "go-cmd", "go-bt", "battery-cmd", "battery-bt"} {
		if _, err := os.Stat(filepath.Join(tmp, sentinel)); err == nil {
			t.Fatalf("dynamic prompt payload executed sentinel %s\n%s", sentinel, output)
		} else if !os.IsNotExist(err) {
			t.Fatalf("stat sentinel %s: %v", sentinel, err)
		}
	}
	contents := readTestFile(t, filepath.Join(tmp, "dynamic.txt"))
	for _, prefix := range []string{"node", "go", "battery"} {
		assertHostilePayloadRendered(t, "dynamic.txt", contents, prefix, 1)
	}
}

func hostilePromptPayload(prefix string) string {
	return "<BEGIN-" + prefix + ">$(touch " + prefix + "-cmd) `touch " + prefix + "-bt` ${HOME} $[1+1] $((1+1)) %n %~ ! \\ literal<END-" + prefix + ">"
}

func assertHostilePayloadRendered(t *testing.T, label, contents, prefix string, wantCount int) {
	t.Helper()
	want := hostilePromptPayload(prefix)
	if count := strings.Count(contents, want); count != wantCount {
		t.Fatalf("%s contains payload %q %d times, want %d:\n%s", label, prefix, count, wantCount, contents)
	}
}

func readTestFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s) error = %v", path, err)
	}
	return string(data)
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
