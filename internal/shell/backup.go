package shell

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

func Backup() (string, error) {
	src := ZshrcPath()
	if _, err := os.Stat(src); os.IsNotExist(err) {
		return "", fmt.Errorf(".zshrc does not exist")
	}

	backupDir := BackupsDir()
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create backup dir: %w", err)
	}

	ts := time.Now().Format("2006-01-02-1504")
	dst := filepath.Join(backupDir, fmt.Sprintf("zshrc-%s.bak", ts))

	data, err := os.ReadFile(src)
	if err != nil {
		return "", fmt.Errorf("failed to read .zshrc: %w", err)
	}
	if err := os.WriteFile(dst, data, 0644); err != nil {
		return "", fmt.Errorf("failed to write backup: %w", err)
	}
	return dst, nil
}

func Backups() ([]string, error) {
	entries, err := os.ReadDir(BackupsDir())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read backups: %w", err)
	}
	var backups []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		backups = append(backups, filepath.Join(BackupsDir(), entry.Name()))
	}
	sort.Strings(backups)
	return backups, nil
}
