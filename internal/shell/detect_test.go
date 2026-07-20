package shell

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectPathsAndExistingFiles(t *testing.T) {
	home := withTempHome(t)

	if got := Home(); got != home {
		t.Fatalf("Home() = %q, want %q", got, home)
	}
	if got := ZshrcPath(); got != filepath.Join(home, ".zshrc") {
		t.Fatalf("ZshrcPath() = %q, want home .zshrc", got)
	}
	if got := OmegaZshPath(); got != filepath.Join(home, ".config", "ozsh", "omega.zsh") {
		t.Fatalf("OmegaZshPath() = %q, want omega path under home", got)
	}
	if ZshrcExists() || ConfigExists() {
		t.Fatal("file existence checks returned true before files were created")
	}

	if err := os.MkdirAll(OmegaDir(), 0755); err != nil {
		t.Fatalf("MkdirAll(OmegaDir) error = %v", err)
	}
	if err := os.WriteFile(ZshrcPath(), []byte{}, 0644); err != nil {
		t.Fatalf("WriteFile(.zshrc) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(OmegaDir(), "config.toml"), []byte{}, 0644); err != nil {
		t.Fatalf("WriteFile(config.toml) error = %v", err)
	}
	if !ZshrcExists() || !ConfigExists() {
		t.Fatal("file existence checks returned false after files were created")
	}
}

func TestTermuxDetection(t *testing.T) {
	t.Setenv("TERMUX_VERSION", "1")
	t.Setenv("PREFIX", "/data/data/com.termux/files/usr")
	t.Setenv("PATH", "/data/data/com.termux/files/usr/bin")

	if !IsTermux() {
		t.Fatal("IsTermux() = false, want true")
	}
	if IsTermuxChroot() {
		t.Fatal("IsTermuxChroot() = true for normal Termux PATH")
	}
	if got := TermuxPrefix(); got != "/data/data/com.termux/files/usr" {
		t.Fatalf("TermuxPrefix() = %q, want Termux prefix", got)
	}
}

func TestTermuxChrootDetection(t *testing.T) {
	t.Setenv("TERMUX_VERSION", "1")
	t.Setenv("PREFIX", "/data/data/com.termux/files/usr")
	t.Setenv("PATH", "/data/data/com.termux/files/usr/bin:/usr/bin")
	if !IsTermuxChroot() {
		t.Fatal("IsTermuxChroot() = false with standalone /usr/bin")
	}
}

func TestShellDetection(t *testing.T) {
	t.Setenv("SHELL", "/usr/bin/zsh")
	if !ZshIsDefaultShell() {
		t.Fatal("ZshIsDefaultShell() = false for zsh")
	}

	t.Setenv("SHELL", "/bin/bash")
	if ZshIsDefaultShell() {
		t.Fatal("ZshIsDefaultShell() = true for bash")
	}
}

func TestHasZshMatchesExecutableLookup(t *testing.T) {
	if _, err := os.Stat("/usr/bin/zsh"); err == nil && !HasZsh() {
		t.Fatal("HasZsh() = false even though /usr/bin/zsh exists")
	}
}
