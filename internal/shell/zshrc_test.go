package shell

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func withTempHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	return home
}

func readZshrc(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile(ZshrcPath())
	if err != nil {
		t.Fatalf("ReadFile(.zshrc) error = %v", err)
	}
	return string(data)
}

func TestInjectBlockEmptyZshrc(t *testing.T) {
	withTempHome(t)
	if err := os.WriteFile(ZshrcPath(), []byte{}, 0o644); err != nil {
		t.Fatalf("WriteFile(.zshrc) error = %v", err)
	}
	if err := InjectBlock(); err != nil {
		t.Fatalf("InjectBlock() error = %v", err)
	}
	content := readZshrc(t)
	if !strings.Contains(content, blockStart) || !strings.Contains(content, blockBody) || !strings.Contains(content, blockEnd) {
		t.Fatalf("InjectBlock() did not write ozsh block:\n%s", content)
	}
}

func TestInjectBlockReplacesExistingBlockWithoutDuplicate(t *testing.T) {
	withTempHome(t)
	original := "export EDITOR=vim\n" + blockStart + "\nstale\n" + blockEnd + "\nalias ll='ls -la'\n"
	if err := os.WriteFile(ZshrcPath(), []byte(original), 0o644); err != nil {
		t.Fatalf("WriteFile(.zshrc) error = %v", err)
	}
	if err := InjectBlock(); err != nil {
		t.Fatalf("InjectBlock() error = %v", err)
	}
	content := readZshrc(t)
	if strings.Count(content, blockStart) != 1 || strings.Count(content, blockEnd) != 1 {
		t.Fatalf("InjectBlock() duplicated ozsh block:\n%s", content)
	}
	if strings.Contains(content, "stale") {
		t.Fatalf("InjectBlock() preserved stale block content:\n%s", content)
	}
	if !strings.Contains(content, "export EDITOR=vim") || !strings.Contains(content, "alias ll='ls -la'") {
		t.Fatalf("InjectBlock() removed original content:\n%s", content)
	}
}

func TestPreviewInjectBlockMatchesInjectBlock(t *testing.T) {
	withTempHome(t)
	original := "export EDITOR=vim\n"
	if err := os.WriteFile(ZshrcPath(), []byte(original), 0o644); err != nil {
		t.Fatalf("WriteFile(.zshrc) error = %v", err)
	}
	_, preview, err := PreviewInjectBlock()
	if err != nil {
		t.Fatalf("PreviewInjectBlock() error = %v", err)
	}
	if err := InjectBlock(); err != nil {
		t.Fatalf("InjectBlock() error = %v", err)
	}
	if got := readZshrc(t); got != preview {
		t.Fatalf("InjectBlock() content differs from preview:\ngot:\n%s\nwant:\n%s", got, preview)
	}
}

func TestDiffLinesShowsManagedBlockChanges(t *testing.T) {
	before := "export EDITOR=vim\n"
	after := before + "\n" + ManagedBlock()
	diff := DiffLines(before, after)
	for _, want := range []string{"+ " + blockStart, "+ " + blockBody, "+ " + blockEnd} {
		if !strings.Contains(diff, want) {
			t.Fatalf("DiffLines() missing %q in:\n%s", want, diff)
		}
	}
}

func TestRemoveBlockRemovesOnlyOzshBlock(t *testing.T) {
	withTempHome(t)
	original := "before\n" + blockStart + "\n" + blockBody + "\n" + blockEnd + "\nafter\n"
	if err := os.WriteFile(ZshrcPath(), []byte(original), 0o644); err != nil {
		t.Fatalf("WriteFile(.zshrc) error = %v", err)
	}
	if err := RemoveBlock(); err != nil {
		t.Fatalf("RemoveBlock() error = %v", err)
	}
	content := readZshrc(t)
	if strings.Contains(content, blockStart) || strings.Contains(content, blockBody) || strings.Contains(content, blockEnd) {
		t.Fatalf("RemoveBlock() left ozsh block behind:\n%s", content)
	}
	if content != "before\nafter\n" {
		t.Fatalf("RemoveBlock() did not preserve original content, got %q", content)
	}
}

func TestMalformedBlockDoesNotDeleteZshrcTail(t *testing.T) {
	home := withTempHome(t)
	original := "export EDITOR=vim\n" + blockStart + "\nuser aliases stay here\n"
	if err := os.WriteFile(filepath.Join(home, ".zshrc"), []byte(original), 0o644); err != nil {
		t.Fatalf("WriteFile(.zshrc) error = %v", err)
	}
	if _, _, err := PreviewInjectBlock(); err == nil {
		t.Fatal("PreviewInjectBlock() error = nil, want malformed block error")
	}
	if err := InjectBlock(); err == nil {
		t.Fatal("InjectBlock() error = nil, want malformed block error")
	}
	if err := RemoveBlock(); err == nil {
		t.Fatal("RemoveBlock() error = nil, want malformed block error")
	}
	if got := readZshrc(t); got != original {
		t.Fatalf("malformed block modified zshrc: got %q, want %q", got, original)
	}
}

func TestRemoveBlockPreservesMissingFinalNewline(t *testing.T) {
	original := "export EDITOR=vim"
	content := injectBlockContent(original)
	if got := removeBlock(content); got != original {
		t.Fatalf("removeBlock() = %q, want %q", got, original)
	}
}

func TestHasBlock(t *testing.T) {
	withTempHome(t)
	if HasBlock() {
		t.Fatal("HasBlock() = true with missing .zshrc")
	}
	if err := os.WriteFile(ZshrcPath(), []byte("plain\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(.zshrc) error = %v", err)
	}
	if HasBlock() {
		t.Fatal("HasBlock() = true without ozsh block")
	}
	if err := os.WriteFile(ZshrcPath(), []byte(blockStart+"\n"+blockEnd+"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(.zshrc) error = %v", err)
	}
	if !HasBlock() {
		t.Fatal("HasBlock() = false with ozsh block")
	}
}

func TestBackupCreatesTimestampedBackup(t *testing.T) {
	home := withTempHome(t)
	if err := os.WriteFile(ZshrcPath(), []byte("original\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(.zshrc) error = %v", err)
	}
	backupPath, err := Backup()
	if err != nil {
		t.Fatalf("Backup() error = %v", err)
	}
	wantDir := filepath.Join(home, ".config", "ozsh", "backups")
	if filepath.Dir(backupPath) != wantDir {
		t.Fatalf("Backup() dir = %q, want %q", filepath.Dir(backupPath), wantDir)
	}
	if matched := regexp.MustCompile(`zshrc-\d{4}-\d{2}-\d{2}-\d{4}(?:-\d+)?\.bak$`).MatchString(filepath.Base(backupPath)); !matched {
		t.Fatalf("Backup() filename %q does not include expected timestamp", filepath.Base(backupPath))
	}
	data, err := os.ReadFile(backupPath)
	if err != nil {
		t.Fatalf("ReadFile(backup) error = %v", err)
	}
	if string(data) != "original\n" {
		t.Fatalf("Backup() content = %q", data)
	}
}

func TestInjectAndRemovePreserveRelativeZshrcSymlink(t *testing.T) {
	home := withTempHome(t)
	target := filepath.Join(home, "dotfiles", "zshrc")
	if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
		t.Fatalf("MkdirAll(target dir) error = %v", err)
	}
	if err := os.WriteFile(target, []byte("export EDITOR=vim\n"), 0o600); err != nil {
		t.Fatalf("WriteFile(target) error = %v", err)
	}
	if err := os.Symlink(filepath.Join("dotfiles", "zshrc"), ZshrcPath()); err != nil {
		t.Fatalf("Symlink(.zshrc) error = %v", err)
	}

	logical, resolved, err := ResolveZshrcTarget()
	if err != nil {
		t.Fatalf("ResolveZshrcTarget() error = %v", err)
	}
	if logical != ZshrcPath() || resolved != target {
		t.Fatalf("ResolveZshrcTarget() = (%q, %q), want (%q, %q)", logical, resolved, ZshrcPath(), target)
	}
	if err := InjectBlock(); err != nil {
		t.Fatalf("InjectBlock() error = %v", err)
	}
	info, err := os.Lstat(ZshrcPath())
	if err != nil {
		t.Fatalf("Lstat(.zshrc) error = %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatal("InjectBlock replaced .zshrc symlink")
	}
	if !strings.Contains(string(readFileBytes(t, target)), blockStart) {
		t.Fatal("InjectBlock did not update symlink target")
	}
	if err := RemoveBlock(); err != nil {
		t.Fatalf("RemoveBlock() error = %v", err)
	}
	info, err = os.Lstat(ZshrcPath())
	if err != nil {
		t.Fatalf("Lstat(.zshrc after reset) error = %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatal("RemoveBlock replaced .zshrc symlink")
	}
	if got := string(readFileBytes(t, target)); got != "export EDITOR=vim\n" {
		t.Fatalf("RemoveBlock target content = %q", got)
	}
}

func TestResolveZshrcTargetSymlinkCases(t *testing.T) {
	home := withTempHome(t)
	target := filepath.Join(home, "real.zshrc")
	if err := os.WriteFile(target, []byte("real\n"), 0o600); err != nil {
		t.Fatalf("WriteFile(target) error = %v", err)
	}
	absLink := filepath.Join(home, "abs-link")
	if err := os.Symlink(target, absLink); err != nil {
		t.Fatalf("Symlink(abs-link) error = %v", err)
	}
	if err := os.Symlink(absLink, ZshrcPath()); err != nil {
		t.Fatalf("Symlink(.zshrc) error = %v", err)
	}
	_, resolved, err := ResolveZshrcTarget()
	if err != nil {
		t.Fatalf("ResolveZshrcTarget(chain) error = %v", err)
	}
	if resolved != target {
		t.Fatalf("ResolveZshrcTarget(chain) target = %q, want %q", resolved, target)
	}
}

func TestResolveZshrcTargetRejectsUnsafeSymlinks(t *testing.T) {
	tests := []struct {
		name  string
		setup func(t *testing.T, home string)
	}{
		{
			name: "broken",
			setup: func(t *testing.T, home string) {
				if err := os.Symlink("missing", ZshrcPath()); err != nil {
					t.Fatalf("Symlink(broken) error = %v", err)
				}
			},
		},
		{
			name: "loop",
			setup: func(t *testing.T, home string) {
				if err := os.Symlink(".zshrc-loop", ZshrcPath()); err != nil {
					t.Fatalf("Symlink(loop a) error = %v", err)
				}
				if err := os.Symlink(".zshrc", filepath.Join(home, ".zshrc-loop")); err != nil {
					t.Fatalf("Symlink(loop b) error = %v", err)
				}
			},
		},
		{
			name: "directory",
			setup: func(t *testing.T, home string) {
				dir := filepath.Join(home, "zshrc-dir")
				if err := os.Mkdir(dir, 0o700); err != nil {
					t.Fatalf("Mkdir(dir) error = %v", err)
				}
				if err := os.Symlink(dir, ZshrcPath()); err != nil {
					t.Fatalf("Symlink(dir) error = %v", err)
				}
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			home := withTempHome(t)
			tc.setup(t, home)
			if _, _, err := ResolveZshrcTarget(); err == nil {
				t.Fatal("ResolveZshrcTarget() error = nil, want rejection")
			}
		})
	}
}

func readFileBytes(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s) error = %v", path, err)
	}
	return data
}
