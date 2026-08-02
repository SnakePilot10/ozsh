package shell

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
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

// RestoreBackup atomically replaces the resolved .zshrc target with one of the
// regular backup files owned by ozsh. The current target is backed up first.
func RestoreBackup(path string) error {
	root := filepath.Clean(BackupsDir())
	if root == "." || root == "" {
		return fmt.Errorf("cannot determine HOME")
	}
	selected, err := filepath.Abs(filepath.Clean(path))
	if err != nil {
		return fmt.Errorf("resolve backup path: %w", err)
	}
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return fmt.Errorf("resolve backup directory: %w", err)
	}
	if selected == rootAbs || !strings.HasPrefix(selected, rootAbs+string(filepath.Separator)) {
		return fmt.Errorf("backup path must stay inside %s", rootAbs)
	}
	info, err := os.Lstat(selected)
	if err != nil {
		return fmt.Errorf("inspect backup: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return fmt.Errorf("backup must be a regular file")
	}
	data, err := os.ReadFile(selected)
	if err != nil {
		return fmt.Errorf("read backup: %w", err)
	}
	_, target, err := ResolveZshrcTarget()
	if err != nil {
		return err
	}
	mode := os.FileMode(0o600)
	if targetInfo, statErr := os.Stat(target); statErr == nil {
		if !targetInfo.Mode().IsRegular() {
			return fmt.Errorf(".zshrc target must be a regular file")
		}
		mode = targetInfo.Mode().Perm()
		if _, err := Backup(); err != nil {
			return fmt.Errorf("preserve current .zshrc: %w", err)
		}
	} else if !os.IsNotExist(statErr) {
		return fmt.Errorf("inspect .zshrc target: %w", statErr)
	}
	if err := atomicWrite(target, data, mode); err != nil {
		return fmt.Errorf("restore .zshrc backup: %w", err)
	}
	return nil
}
