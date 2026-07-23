package logging

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	maxLogSize  = 5 * 1024 * 1024
	maxBackups  = 3
	logDirMode  = 0o700
	logFileMode = 0o600
)

type Logger struct {
	Verbose bool
	path    string
	mu      sync.Mutex
}

func New(dir string, verbose bool) *Logger {
	if dir == "" {
		return &Logger{Verbose: verbose}
	}
	return &Logger{Verbose: verbose, path: filepath.Join(dir, "ozsh.log")}
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
	l.mu.Lock()
	defer l.mu.Unlock()
	dir := filepath.Dir(l.path)
	if err := os.MkdirAll(dir, logDirMode); err != nil {
		return err
	}
	if err := os.Chmod(dir, logDirMode); err != nil {
		return err
	}
	if err := rotate(l.path); err != nil {
		return err
	}
	f, err := os.OpenFile(l.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, logFileMode)
	if err != nil {
		return err
	}
	defer f.Close()
	if err := f.Chmod(logFileMode); err != nil {
		return err
	}
	_, err = fmt.Fprintf(f, "%s [%s] %s\n", time.Now().Format(time.RFC3339), level, msg)
	return err
}

func rotate(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if info.Size() < maxLogSize {
		return nil
	}
	for i := maxBackups - 1; i >= 1; i-- {
		oldPath := fmt.Sprintf("%s.%d", path, i)
		newPath := fmt.Sprintf("%s.%d", path, i+1)
		if _, err := os.Stat(oldPath); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return err
		}
		if err := os.Remove(newPath); err != nil && !os.IsNotExist(err) {
			return err
		}
		if err := os.Rename(oldPath, newPath); err != nil {
			return err
		}
	}
	if err := os.Remove(path + ".1"); err != nil && !os.IsNotExist(err) {
		return err
	}
	return os.Rename(path, path+".1")
}
