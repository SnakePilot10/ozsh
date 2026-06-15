package main

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/snakepilot10/ozsh/internal/config"
	"github.com/snakepilot10/ozsh/internal/shell"
)

func TestRunApplyGeneratesOmegaAndInjectsBlock(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.WriteFile(filepath.Join(home, ".zshrc"), []byte("export EDITOR=vim\n"), 0644); err != nil {
		t.Fatalf("WriteFile(.zshrc) error = %v", err)
	}

	runApply()

	omega, err := os.ReadFile(shell.OmegaZshPath())
	if err != nil {
		t.Fatalf("ReadFile(omega.zsh) error = %v", err)
	}
	if !strings.Contains(string(omega), "ozsh_prompt()") {
		t.Fatalf("omega.zsh missing generated prompt:\n%s", omega)
	}

	zshrc, err := os.ReadFile(shell.ZshrcPath())
	if err != nil {
		t.Fatalf("ReadFile(.zshrc) error = %v", err)
	}
	if !strings.Contains(string(zshrc), `source "$HOME/.config/ozsh/omega.zsh"`) {
		t.Fatalf(".zshrc missing ozsh source block:\n%s", zshrc)
	}
}

func TestRunResetRestoresOriginalZshrcContent(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	original := "export EDITOR=vim\nalias ll='ls -la'\n"
	if err := os.WriteFile(filepath.Join(home, ".zshrc"), []byte(original), 0644); err != nil {
		t.Fatalf("WriteFile(.zshrc) error = %v", err)
	}

	runApply()
	runReset()

	zshrc, err := os.ReadFile(shell.ZshrcPath())
	if err != nil {
		t.Fatalf("ReadFile(.zshrc) error = %v", err)
	}
	if string(zshrc) != original {
		t.Fatalf(".zshrc after reset = %q, want %q", zshrc, original)
	}
}

func TestRunResetRestoresOriginalZshrcWithoutFinalNewline(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	original := "export EDITOR=vim"
	if err := os.WriteFile(filepath.Join(home, ".zshrc"), []byte(original), 0644); err != nil {
		t.Fatalf("WriteFile(.zshrc) error = %v", err)
	}

	runApply()
	runReset()

	zshrc, err := os.ReadFile(shell.ZshrcPath())
	if err != nil {
		t.Fatalf("ReadFile(.zshrc) error = %v", err)
	}
	if string(zshrc) != original {
		t.Fatalf(".zshrc after reset = %q, want %q", zshrc, original)
	}
}

func TestSuggestCommand(t *testing.T) {
	if got := suggestCommand("aplly"); got != "apply" {
		t.Fatalf("suggestCommand(aplly) = %q, want apply", got)
	}
}

func TestParseGlobalFlags(t *testing.T) {
	args, verbose := parseGlobalFlags([]string{"--verbose", "preview", "-v", "--real"})

	if !verbose {
		t.Fatal("parseGlobalFlags() verbose = false, want true")
	}
	if strings.Join(args, " ") != "preview --real" {
		t.Fatalf("parseGlobalFlags() args = %v, want preview --real", args)
	}
}

func TestRunUpdateWithoutCheckoutReportsCurrentVersion(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("OZSH_INSTALL_DIR", filepath.Join(home, "missing"))

	out := captureStdout(t, func() {
		runUpdate([]string{"--check"})
	})

	if !strings.Contains(out, version) || !strings.Contains(out, "no source checkout found") {
		t.Fatalf("runUpdate(no checkout) output = %q, want version and no checkout message", out)
	}
}

func TestUpdateStateReportsUpToDateAndAvailable(t *testing.T) {
	origin := filepath.Join(t.TempDir(), "origin.git")
	seed := filepath.Join(t.TempDir(), "seed")
	installDir := filepath.Join(t.TempDir(), "install")

	runGit(t, "", "init", "--bare", origin)
	runGit(t, "", "clone", origin, seed)
	runGit(t, seed, "config", "user.email", "test@example.com")
	runGit(t, seed, "config", "user.name", "Test User")
	if err := os.WriteFile(filepath.Join(seed, "README.md"), []byte("v1\n"), 0644); err != nil {
		t.Fatalf("WriteFile(seed README) error = %v", err)
	}
	runGit(t, seed, "add", "README.md")
	runGit(t, seed, "commit", "-m", "initial")
	runGit(t, seed, "push", "-u", "origin", "HEAD")
	runGit(t, "", "clone", origin, installDir)

	state, err := updateState(installDir)
	if err != nil {
		t.Fatalf("updateState(up-to-date) error = %v", err)
	}
	if state != "ozsh is up to date" {
		t.Fatalf("updateState(up-to-date) = %q", state)
	}

	if err := os.WriteFile(filepath.Join(seed, "README.md"), []byte("v2\n"), 0644); err != nil {
		t.Fatalf("WriteFile(seed README update) error = %v", err)
	}
	runGit(t, seed, "commit", "-am", "update")
	runGit(t, seed, "push")
	runGit(t, installDir, "fetch", "--quiet")

	state, err = updateState(installDir)
	if err != nil {
		t.Fatalf("updateState(available) error = %v", err)
	}
	if state != "new version available (run ozsh update)" {
		t.Fatalf("updateState(available) = %q", state)
	}
}

func TestRunDoctorReportsCriticalChecks(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("SHELL", "/usr/bin/zsh")
	if err := config.Save(config.Default()); err != nil {
		t.Fatalf("config.Save() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(home, ".zshrc"), []byte("export EDITOR=vim\n"), 0644); err != nil {
		t.Fatalf("WriteFile(.zshrc) error = %v", err)
	}

	out := captureStdout(t, runDoctor)

	for _, want := range []string{"zsh is installed", "config.toml exists", ".zshrc exists", "All critical checks passed."} {
		if !strings.Contains(out, want) {
			t.Fatalf("runDoctor() output missing %q:\n%s", want, out)
		}
	}
}

func TestRunPreviewSimulatedAndReal(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	simulated := captureStdout(t, func() {
		runPreview(nil)
	})
	if !strings.Contains(simulated, "❯") {
		t.Fatalf("runPreview() output = %q, want prompt marker", simulated)
	}

	real := captureStdout(t, func() {
		runPreview([]string{"--real"})
	})
	if !strings.Contains(real, "ozsh_prompt()") {
		t.Fatalf("runPreview(--real) output missing generated prompt:\n%s", real)
	}
}

func TestRunThemeCommands(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	list := captureStdout(t, func() {
		runTheme([]string{"list"})
	})
	if !strings.Contains(list, "cyber-cyan") {
		t.Fatalf("theme list output = %q, want cyber-cyan", list)
	}

	applied := captureStdout(t, func() {
		runTheme([]string{"apply", "neon-red"})
	})
	if !strings.Contains(applied, "theme applied: neon-red") {
		t.Fatalf("theme apply output = %q, want applied message", applied)
	}

	preview := captureStdout(t, func() {
		runTheme([]string{"preview", "neon-red"})
	})
	if !strings.Contains(preview, "\x1b[38;2;255;0;60m") {
		t.Fatalf("theme preview output = %q, want neon red ANSI", preview)
	}
}

func TestRunHeaderCommands(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	list := captureStdout(t, func() {
		runHeader([]string{"list"})
	})
	if !strings.Contains(list, "snake") {
		t.Fatalf("header list output = %q, want snake", list)
	}

	applied := captureStdout(t, func() {
		runHeader([]string{"apply", "snake"})
	})
	if !strings.Contains(applied, "header applied: snake") {
		t.Fatalf("header apply output = %q, want applied message", applied)
	}

	preview := captureStdout(t, func() {
		runHeader([]string{"preview", "snake"})
	})
	if !strings.Contains(preview, "snake") {
		t.Fatalf("header preview output = %q, want snake text", preview)
	}
}

func TestRunPluginCommands(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	emptyList := captureStdout(t, func() {
		runPlugin([]string{"list"})
	})
	if !strings.Contains(emptyList, "no plugins configured") {
		t.Fatalf("plugin list output = %q, want no plugins message", emptyList)
	}

	pluginDir := filepath.Join(home, ".config", "ozsh", "plugins", "demo")
	if err := os.MkdirAll(pluginDir, 0755); err != nil {
		t.Fatalf("MkdirAll(plugin dir) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(pluginDir, "plugin.zsh"), []byte("# plugin\n"), 0644); err != nil {
		t.Fatalf("WriteFile(plugin.zsh) error = %v", err)
	}
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("config.Load() error = %v", err)
	}
	cfg.Plugins.Items = []config.PluginItem{
		{Name: "demo", Enabled: true, Trusted: false, Source: pluginDir, Load: "plugin.zsh"},
	}
	if err := config.Save(cfg); err != nil {
		t.Fatalf("config.Save() error = %v", err)
	}

	for _, tc := range []struct {
		args []string
		want string
	}{
		{args: []string{"disable", "demo"}, want: "plugin disable: demo"},
		{args: []string{"enable", "demo"}, want: "plugin enable: demo"},
		{args: []string{"trust", "demo"}, want: "plugin trust: demo"},
		{args: []string{"untrust", "demo"}, want: "plugin untrust: demo"},
		{args: []string{"list"}, want: "demo"},
	} {
		out := captureStdout(t, func() {
			runPlugin(tc.args)
		})
		if !strings.Contains(out, tc.want) {
			t.Fatalf("plugin %s output = %q, want %q", strings.Join(tc.args, " "), out, tc.want)
		}
	}
}

func TestCommandVersionThemePreviewAndHeaderList(t *testing.T) {
	bin := buildTestBinary(t)
	home := t.TempDir()

	for _, tc := range []struct {
		name string
		args []string
		want string
	}{
		{name: "version", args: []string{"version"}, want: version},
		{name: "update check no checkout", args: []string{"update", "--check"}, want: "no source checkout found"},
		{name: "theme preview", args: []string{"theme", "preview", "neon-red"}, want: "\x1b[38;2;255;0;60m"},
		{name: "header list", args: []string{"header", "list"}, want: "snake"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cmd := exec.Command(bin, tc.args...)
			cmd.Env = append(os.Environ(), "HOME="+home)
			var out bytes.Buffer
			cmd.Stdout = &out
			cmd.Stderr = &out
			if err := cmd.Run(); err != nil {
				t.Fatalf("%s error = %v\n%s", strings.Join(tc.args, " "), err, out.String())
			}
			if !strings.Contains(out.String(), tc.want) {
				t.Fatalf("%s output = %q, want contains %q", strings.Join(tc.args, " "), out.String(), tc.want)
			}
		})
	}
}

func TestUpdateCheckReportsAvailableUpdate(t *testing.T) {
	bin := buildTestBinary(t)
	home := t.TempDir()
	origin := filepath.Join(t.TempDir(), "origin.git")
	seed := filepath.Join(t.TempDir(), "seed")
	installDir := filepath.Join(t.TempDir(), "install")

	runGit(t, "", "init", "--bare", origin)
	runGit(t, "", "clone", origin, seed)
	runGit(t, seed, "config", "user.email", "test@example.com")
	runGit(t, seed, "config", "user.name", "Test User")
	if err := os.WriteFile(filepath.Join(seed, "README.md"), []byte("v1\n"), 0644); err != nil {
		t.Fatalf("WriteFile(seed README) error = %v", err)
	}
	runGit(t, seed, "add", "README.md")
	runGit(t, seed, "commit", "-m", "initial")
	runGit(t, seed, "push", "-u", "origin", "HEAD")
	runGit(t, "", "clone", origin, installDir)

	if err := os.WriteFile(filepath.Join(seed, "README.md"), []byte("v2\n"), 0644); err != nil {
		t.Fatalf("WriteFile(seed README update) error = %v", err)
	}
	runGit(t, seed, "commit", "-am", "update")
	runGit(t, seed, "push")

	cmd := exec.Command(bin, "update", "--check")
	cmd.Env = append(os.Environ(), "HOME="+home, "OZSH_INSTALL_DIR="+installDir)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		t.Fatalf("ozsh update --check error = %v\n%s", err, out.String())
	}
	if !strings.Contains(out.String(), "new version available") {
		t.Fatalf("update check output = %q, want new version notification", out.String())
	}
	if !strings.Contains(out.String(), "ozsh update") {
		t.Fatalf("update check output = %q, want actionable update command", out.String())
	}
}

func BenchmarkRunApply(b *testing.B) {
	home := b.TempDir()
	b.Setenv("HOME", home)
	if err := os.WriteFile(filepath.Join(home, ".zshrc"), []byte("export EDITOR=vim\n"), 0644); err != nil {
		b.Fatalf("WriteFile(.zshrc) error = %v", err)
	}

	devNull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		b.Fatalf("OpenFile(os.DevNull) error = %v", err)
	}
	defer devNull.Close()
	stdout := os.Stdout
	os.Stdout = devNull
	defer func() {
		os.Stdout = stdout
	}()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		runApply()
	}
}

func buildTestBinary(t *testing.T) string {
	t.Helper()

	bin := filepath.Join(t.TempDir(), "ozsh-test")
	cmd := exec.Command("go", "build", "-buildvcs=false", "-o", bin, ".")
	cmd.Env = append(os.Environ(), "GOCACHE=/tmp/ozsh-go-build", "GOMODCACHE=/tmp/ozsh-gomod")
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		t.Fatalf("go build error = %v\n%s", err, out.String())
	}
	return bin
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("Pipe() error = %v", err)
	}
	os.Stdout = w
	fn()
	if err := w.Close(); err != nil {
		t.Fatalf("Close(stdout pipe writer) error = %v", err)
	}
	os.Stdout = old
	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("ReadAll(stdout pipe reader) error = %v", err)
	}
	return string(out)
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()

	cmd := exec.Command("git", args...)
	if dir != "" {
		cmd.Dir = dir
	}
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		t.Fatalf("git %s error = %v\n%s", strings.Join(args, " "), err, out.String())
	}
}
