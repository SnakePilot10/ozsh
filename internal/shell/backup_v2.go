package shell

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

const (
	backupDirMode  = 0o700
	backupFileMode = 0o600
)

func Backup() (string, error) {
	src := ZshrcPath()
	if _, err := os.Stat(src); err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf(".zshrc does not exist")
		}
		return "", fmt.Errorf("failed to inspect .zshrc: %w", err)
	}

	backupDir := BackupsDir()
	if err := os.MkdirAll(backupDir, backupDirMode); err != nil {
		return "", fmt.Errorf("failed to create backup dir: %w", err)
	}
	if err := os.Chmod(backupDir, backupDirMode); err != nil {
		return "", fmt.Errorf("failed to secure backup dir: %w", err)
	}
	data, err := os.ReadFile(src)
	if err != nil {
		return "", fmt.Errorf("failed to read .zshrc: %w", err)
	}

	stamp := time.Now().Format("2006-01-02-1504")
	path := filepath.Join(backupDir, fmt.Sprintf("zshrc-%s.bak", stamp))
	for suffix := 1; ; suffix++ {
		if _, err := os.Lstat(path); os.IsNotExist(err) {
			break
		} else if err != nil {
			return "", fmt.Errorf("failed to inspect backup destination: %w", err)
		}
		path = filepath.Join(backupDir, fmt.Sprintf("zshrc-%s-%d.bak", stamp, suffix))
	}
	if err := os.WriteFile(path, data, backupFileMode); err != nil {
		return "", fmt.Errorf("failed to write backup: %w", err)
	}
	if err := os.Chmod(path, backupFileMode); err != nil {
		return "", fmt.Errorf("failed to secure backup: %w", err)
	}
	return path, nil
}

func Backups() ([]string, error) {
	entries, err := os.ReadDir(BackupsDir())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read backups: %w", err)
	}
	backups := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		backups = append(backups, filepath.Join(BackupsDir(), entry.Name()))
	}
	sort.Strings(backups)
	return backups, nil
}
