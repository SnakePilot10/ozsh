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
	logical, src, err := ResolveZshrcTarget()
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(src); err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf(".zshrc does not exist")
		}
		return "", fmt.Errorf("failed to inspect .zshrc%s: %w", zshrcTargetDescription(logical, src), err)
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
		return "", fmt.Errorf("failed to read .zshrc%s: %w", zshrcTargetDescription(logical, src), err)
	}

	stamp := time.Now().Format("2006-01-02-1504")
	for suffix := 0; ; suffix++ {
		name := fmt.Sprintf("zshrc-%s.bak", stamp)
		if suffix > 0 {
			name = fmt.Sprintf("zshrc-%s-%d.bak", stamp, suffix)
		}
		path := filepath.Join(backupDir, name)
		file, openErr := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, backupFileMode)
		if os.IsExist(openErr) {
			continue
		}
		if openErr != nil {
			return "", fmt.Errorf("failed to create backup: %w", openErr)
		}
		if _, err := file.Write(data); err != nil {
			file.Close()
			_ = os.Remove(path)
			return "", fmt.Errorf("failed to write backup: %w", err)
		}
		if err := file.Sync(); err != nil {
			file.Close()
			_ = os.Remove(path)
			return "", fmt.Errorf("failed to flush backup: %w", err)
		}
		if err := file.Close(); err != nil {
			_ = os.Remove(path)
			return "", fmt.Errorf("failed to close backup: %w", err)
		}
		return path, nil
	}
}

func Backups() ([]string, error) {
	if BackupsDir() == "" {
		return nil, fmt.Errorf("cannot determine HOME")
	}
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
