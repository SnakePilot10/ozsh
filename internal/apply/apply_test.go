package apply

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/snakepilot10/ozsh/internal/config"
	"github.com/snakepilot10/ozsh/internal/shell"
)

func TestApplyConfigGenerateFailureChangesNothing(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.WriteFile(shell.ZshrcPath(), []byte("export EDITOR=vim\n"), 0o600); err != nil {
		t.Fatalf("WriteFile(.zshrc) error = %v", err)
	}
	cfg := config.Default()
	cfg.Version = config.CurrentConfigVersion + 1

	err := ApplyConfig(cfg)
	if err == nil || !strings.Contains(err.Error(), "generate prompt") {
		t.Fatalf("ApplyConfig() error = %v, want generate error", err)
	}
	if _, statErr := os.Stat(config.Path()); !os.IsNotExist(statErr) {
		t.Fatalf("config was changed after generate failure: %v", statErr)
	}
	if _, statErr := os.Stat(shell.OmegaZshPath()); !os.IsNotExist(statErr) {
		t.Fatalf("omega was changed after generate failure: %v", statErr)
	}
	if data := readFile(t, shell.ZshrcPath()); data != "export EDITOR=vim\n" {
		t.Fatalf("zshrc changed after generate failure: %q", data)
	}
}

func TestApplyConfigWriteOmegaFailureLeavesSavedConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.WriteFile(shell.ZshrcPath(), []byte("export EDITOR=vim\n"), 0o600); err != nil {
		t.Fatalf("WriteFile(.zshrc) error = %v", err)
	}
	if err := os.MkdirAll(shell.OmegaZshPath(), 0o700); err != nil {
		t.Fatalf("MkdirAll(omega path as dir) error = %v", err)
	}
	cfg := config.Default()
	cfg.Prompt.Separator = " | "

	err := ApplyConfig(cfg)
	if err == nil || !strings.Contains(err.Error(), "config saved, but omega.zsh could not be updated") {
		t.Fatalf("ApplyConfig() error = %v, want write omega failure", err)
	}
	saved, loadErr := config.Load()
	if loadErr != nil {
		t.Fatalf("config.Load() error = %v", loadErr)
	}
	if saved.Prompt.Separator != " | " {
		t.Fatalf("config not saved after omega failure, separator = %q", saved.Prompt.Separator)
	}
	if data := readFile(t, shell.ZshrcPath()); data != "export EDITOR=vim\n" {
		t.Fatalf("zshrc changed after omega failure: %q", data)
	}
}

func TestApplyConfigPreflightFailureChangesNothing(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	malformed := "export EDITOR=vim\n# >>> ozsh >>>\nstale\n"
	if err := os.WriteFile(shell.ZshrcPath(), []byte(malformed), 0o600); err != nil {
		t.Fatalf("WriteFile(.zshrc) error = %v", err)
	}
	cfg := config.Default()
	cfg.Prompt.Separator = " | "

	err := ApplyConfig(cfg)
	if err == nil || !strings.Contains(err.Error(), "preflight .zshrc") {
		t.Fatalf("ApplyConfig() error = %v, want preflight failure", err)
	}
	if _, statErr := os.Stat(config.Path()); !os.IsNotExist(statErr) {
		t.Fatalf("config changed after preflight failure: %v", statErr)
	}
	if _, statErr := os.Stat(shell.OmegaZshPath()); !os.IsNotExist(statErr) {
		t.Fatalf("omega changed after preflight failure: %v", statErr)
	}
	if data := readFile(t, shell.ZshrcPath()); data != malformed {
		t.Fatalf("zshrc changed after inject failure: %q", data)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		t.Fatalf("ReadFile(%s) error = %v", path, err)
	}
	return string(data)
}
