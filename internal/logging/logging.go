package logging

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const (
	maxLogSize = 5 * 1024 * 1024
	maxBackups = 3
)

type Logger struct {
	Verbose bool
	path    string
}

func New(dir string, verbose bool) *Logger {
	return &Logger{
		Verbose: verbose,
		path:    filepath.Join(dir, "ozsh.log"),
	}
}

func (l *Logger) Debug(format string, args ...any) {
	if l == nil {
		return
	}
	msg := fmt.Sprintf(format, args...)
	if l.Verbose {
		fmt.Fprintf(os.Stderr, "debug: %s\n", msg)
	}
	_ = l.write("DEBUG", msg)
}

func (l *Logger) Info(format string, args ...any) {
	if l == nil {
		return
	}
	_ = l.write("INFO", fmt.Sprintf(format, args...))
}

func (l *Logger) Error(format string, args ...any) {
	if l == nil {
		return
	}
	_ = l.write("ERROR", fmt.Sprintf(format, args...))
}

func (l *Logger) write(level, msg string) error {
	if l.path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(l.path), 0755); err != nil {
		return err
	}
	if err := rotate(l.path); err != nil {
		return err
	}
	f, err := os.OpenFile(l.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = fmt.Fprintf(f, "%s [%s] %s\n", time.Now().Format(time.RFC3339), level, msg)
	return err
}

func rotate(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return nil
	}
	if info.Size() < maxLogSize {
		return nil
	}
	for i := maxBackups - 1; i >= 1; i-- {
		oldPath := fmt.Sprintf("%s.%d", path, i)
		newPath := fmt.Sprintf("%s.%d", path, i+1)
		if _, err := os.Stat(oldPath); err == nil {
			_ = os.Rename(oldPath, newPath)
		}
	}
	return os.Rename(path, path+".1")
}
