package apply

import (
	"fmt"
	"os"
	"path/filepath"
)

type fileSnapshot struct {
	path      string
	existed   bool
	directory bool
	mode      os.FileMode
	data      []byte
}

func captureFile(path string) (fileSnapshot, error) {
	path = filepath.Clean(path)
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return fileSnapshot{path: path}, nil
	}
	if err != nil {
		return fileSnapshot{}, fmt.Errorf("inspect %s: %w", path, err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fileSnapshot{}, fmt.Errorf("snapshot path must not be a symlink: %s", path)
	}
	snapshot := fileSnapshot{
		path:      path,
		existed:   true,
		directory: info.IsDir(),
		mode:      info.Mode().Perm(),
	}
	if snapshot.directory {
		return snapshot, nil
	}
	if !info.Mode().IsRegular() {
		return fileSnapshot{}, fmt.Errorf("snapshot path must be a regular file or directory: %s", path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return fileSnapshot{}, fmt.Errorf("read snapshot %s: %w", path, err)
	}
	snapshot.data = data
	return snapshot, nil
}

func (snapshot fileSnapshot) Restore() error {
	if snapshot.path == "" {
		return fmt.Errorf("snapshot path is empty")
	}
	if !snapshot.existed {
		if err := os.RemoveAll(snapshot.path); err != nil {
			return fmt.Errorf("remove newly created path %s: %w", snapshot.path, err)
		}
		return nil
	}
	if snapshot.directory {
		return snapshot.restoreDirectory()
	}
	return snapshot.restoreRegularFile()
}

func (snapshot fileSnapshot) restoreDirectory() error {
	info, err := os.Lstat(snapshot.path)
	if err == nil && info.IsDir() && info.Mode()&os.ModeSymlink == 0 {
		if err := os.Chmod(snapshot.path, snapshot.mode); err != nil {
			return fmt.Errorf("restore directory mode %s: %w", snapshot.path, err)
		}
		return nil
	}
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("inspect restore path %s: %w", snapshot.path, err)
	}
	if err := os.RemoveAll(snapshot.path); err != nil {
		return fmt.Errorf("remove replacement path %s: %w", snapshot.path, err)
	}
	if err := os.MkdirAll(snapshot.path, snapshot.mode); err != nil {
		return fmt.Errorf("restore directory %s: %w", snapshot.path, err)
	}
	return nil
}

func (snapshot fileSnapshot) restoreRegularFile() error {
	parent := filepath.Dir(snapshot.path)
	if err := os.MkdirAll(parent, 0o700); err != nil {
		return fmt.Errorf("create snapshot parent %s: %w", parent, err)
	}
	if info, err := os.Lstat(snapshot.path); err == nil {
		if info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			if err := os.RemoveAll(snapshot.path); err != nil {
				return fmt.Errorf("remove replacement path %s: %w", snapshot.path, err)
			}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("inspect restore target %s: %w", snapshot.path, err)
	}

	temporary, err := os.CreateTemp(parent, ".ozsh-restore-*")
	if err != nil {
		return fmt.Errorf("create restore file for %s: %w", snapshot.path, err)
	}
	temporaryPath := temporary.Name()
	cleanup := func() {
		_ = temporary.Close()
		_ = os.Remove(temporaryPath)
	}
	if err := temporary.Chmod(snapshot.mode); err != nil {
		cleanup()
		return fmt.Errorf("restore mode for %s: %w", snapshot.path, err)
	}
	if _, err := temporary.Write(snapshot.data); err != nil {
		cleanup()
		return fmt.Errorf("restore data for %s: %w", snapshot.path, err)
	}
	if err := temporary.Sync(); err != nil {
		cleanup()
		return fmt.Errorf("sync restore file for %s: %w", snapshot.path, err)
	}
	if err := temporary.Close(); err != nil {
		_ = os.Remove(temporaryPath)
		return fmt.Errorf("close restore file for %s: %w", snapshot.path, err)
	}
	if err := os.Rename(temporaryPath, snapshot.path); err != nil {
		_ = os.Remove(temporaryPath)
		return fmt.Errorf("replace restored file %s: %w", snapshot.path, err)
	}
	return nil
}
