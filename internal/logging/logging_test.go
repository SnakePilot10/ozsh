package logging

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoggerWritesLevelsToLogFile(t *testing.T) {
	dir := t.TempDir()
	logger := New(dir, false)

	logger.Debug("debug %s", "message")
	logger.Info("info %d", 7)
	logger.Error("error %s", "message")

	data, err := os.ReadFile(filepath.Join(dir, "ozsh.log"))
	if err != nil {
		t.Fatalf("ReadFile(ozsh.log) error = %v", err)
	}
	log := string(data)
	for _, want := range []string{
		"[DEBUG] debug message",
		"[INFO] info 7",
		"[ERROR] error message",
	} {
		if !strings.Contains(log, want) {
			t.Fatalf("log missing %q:\n%s", want, log)
		}
	}
}

func TestLoggerNilReceiverIsNoop(t *testing.T) {
	var logger *Logger

	logger.Debug("debug")
	logger.Info("info")
	logger.Error("error")
}

func TestLoggerWithoutDirectoryDoesNotWriteToWorkingDirectory(t *testing.T) {
	dir := t.TempDir()
	old, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("Chdir(temp) error = %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(old) })

	New("", false).Info("must not be written")
	if _, err := os.Stat(filepath.Join(dir, "ozsh.log")); !os.IsNotExist(err) {
		t.Fatalf("logger created ozsh.log without a config directory: %v", err)
	}
}

func TestLoggerRotatesLargeLogFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ozsh.log")
	large := strings.Repeat("x", maxLogSize)
	if err := os.WriteFile(path, []byte(large), 0644); err != nil {
		t.Fatalf("WriteFile(large log) error = %v", err)
	}

	New(dir, false).Info("after rotation")

	rotated, err := os.ReadFile(path + ".1")
	if err != nil {
		t.Fatalf("ReadFile(rotated log) error = %v", err)
	}
	if len(rotated) != maxLogSize {
		t.Fatalf("rotated log size = %d, want %d", len(rotated), maxLogSize)
	}
	current, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(current log) error = %v", err)
	}
	if !strings.Contains(string(current), "[INFO] after rotation") {
		t.Fatalf("current log missing post-rotation entry:\n%s", current)
	}
}
