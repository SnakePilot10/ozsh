package shell

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteOmegaCreatesPrivateGeneratedFile(t *testing.T) {
	home := withTempHome(t)
	if err := WriteOmega([]byte("# generated\n")); err != nil {
		t.Fatalf("WriteOmega() error = %v", err)
	}
	path := filepath.Join(home, ".config", "ozsh", "omega.zsh")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(omega) error = %v", err)
	}
	if string(data) != "# generated\n" {
		t.Fatalf("WriteOmega() content = %q", data)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat(omega) error = %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("omega mode = %#o, want 0600", info.Mode().Perm())
	}
}

func TestBackupCollisionCreatesDistinctFiles(t *testing.T) {
	withTempHome(t)
	if err := os.WriteFile(ZshrcPath(), []byte("first\n"), 0o600); err != nil {
		t.Fatalf("WriteFile(.zshrc) error = %v", err)
	}
	first, err := Backup()
	if err != nil {
		t.Fatalf("first Backup() error = %v", err)
	}
	second, err := Backup()
	if err != nil {
		t.Fatalf("second Backup() error = %v", err)
	}
	if first == second {
		t.Fatalf("backup collision reused %q", first)
	}
	backups, err := Backups()
	if err != nil {
		t.Fatalf("Backups() error = %v", err)
	}
	if len(backups) != 2 {
		t.Fatalf("Backups() count = %d, want 2", len(backups))
	}
	for _, path := range backups {
		if !strings.HasPrefix(filepath.Base(path), "zshrc-") {
			t.Fatalf("unexpected backup path %q", path)
		}
	}
}

func TestRemoveBlockRejectsMissingZshrcAndMalformedEnd(t *testing.T) {
	home := withTempHome(t)
	if err := RemoveBlock(); err == nil {
		t.Fatal("RemoveBlock(missing) error = nil")
	}
	malformed := "alias ll='ls -la'\n" + blockEnd + "\n"
	if err := os.WriteFile(filepath.Join(home, ".zshrc"), []byte(malformed), 0o600); err != nil {
		t.Fatalf("WriteFile(.zshrc) error = %v", err)
	}
	if err := RemoveBlock(); err == nil {
		t.Fatal("RemoveBlock(unmatched end) error = nil")
	}
	if got := readZshrc(t); got != malformed {
		t.Fatalf("RemoveBlock changed malformed zshrc: %q", got)
	}
}

func TestPreviewInjectBlockCreatesPlanForMissingZshrc(t *testing.T) {
	withTempHome(t)
	before, after, err := PreviewInjectBlock()
	if err != nil {
		t.Fatalf("PreviewInjectBlock() error = %v", err)
	}
	if before != "" || !strings.Contains(after, blockStart) {
		t.Fatalf("PreviewInjectBlock() before=%q after=%q", before, after)
	}
}
