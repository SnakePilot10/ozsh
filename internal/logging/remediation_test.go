package logging

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoggerUsesPrivateModes(t *testing.T) {
	dir := t.TempDir()
	logger := New(filepath.Join(dir, "private"), false)
	logger.Info("hello")

	logPath := filepath.Join(dir, "private", "ozsh.log")
	info, err := os.Stat(logPath)
	if err != nil {
		t.Fatalf("Stat(log) error = %v", err)
	}
	if info.Mode().Perm() != logFileMode {
		t.Fatalf("log mode = %#o, want %#o", info.Mode().Perm(), logFileMode)
	}
	dirInfo, err := os.Stat(filepath.Dir(logPath))
	if err != nil {
		t.Fatalf("Stat(log dir) error = %v", err)
	}
	if dirInfo.Mode().Perm() != logDirMode {
		t.Fatalf("log dir mode = %#o, want %#o", dirInfo.Mode().Perm(), logDirMode)
	}
}

func TestRotateMissingFileIsNoop(t *testing.T) {
	if err := rotate(filepath.Join(t.TempDir(), "missing.log")); err != nil {
		t.Fatalf("rotate(missing) error = %v", err)
	}
}
