package shell

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	blockStart = "# >>> ozsh >>>"
	blockEnd   = "# <<< ozsh <<<"
	blockBody  = `source "$HOME/.config/ozsh/omega.zsh"`
)

func HasBlock() bool {
	data, err := os.ReadFile(ZshrcPath())
	if err != nil {
		return false
	}
	return strings.Contains(string(data), blockStart)
}

func ManagedBlock() string {
	return fmt.Sprintf("%s\n%s\n%s\n", blockStart, blockBody, blockEnd)
}

func PreviewInjectBlock() (string, string, error) {
	data, err := os.ReadFile(ZshrcPath())
	if err != nil {
		if os.IsNotExist(err) {
			data = []byte{}
		} else {
			return "", "", fmt.Errorf("failed to read .zshrc: %w", err)
		}
	}
	before := string(data)
	return before, injectBlockContent(before), nil
}

func DiffLines(before, after string) string {
	if before == after {
		return "no .zshrc changes"
	}
	beforeLines := strings.Split(strings.TrimRight(before, "\n"), "\n")
	afterLines := strings.Split(strings.TrimRight(after, "\n"), "\n")
	beforeSet := map[string]int{}
	for _, line := range beforeLines {
		if line != "" {
			beforeSet[line]++
		}
	}
	afterSet := map[string]int{}
	for _, line := range afterLines {
		if line != "" {
			afterSet[line]++
		}
	}

	var b strings.Builder
	for _, line := range beforeLines {
		if line == "" {
			continue
		}
		if afterSet[line] > 0 {
			afterSet[line]--
			continue
		}
		fmt.Fprintf(&b, "- %s\n", line)
	}
	for _, line := range afterLines {
		if line == "" {
			continue
		}
		if beforeSet[line] > 0 {
			beforeSet[line]--
			continue
		}
		fmt.Fprintf(&b, "+ %s\n", line)
	}
	return strings.TrimRight(b.String(), "\n")
}

func InjectBlock() error {
	path := ZshrcPath()
	mode, err := ensureZshrc(path)
	if err != nil {
		return err
	}
	if _, err := Backup(); err != nil {
		return fmt.Errorf("backup failed: %w", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read .zshrc: %w", err)
	}
	if err := atomicWrite(path, []byte(injectBlockContent(string(data))), mode); err != nil {
		return fmt.Errorf("failed to write .zshrc: %w", err)
	}
	return nil
}

func RemoveBlock() error {
	path := ZshrcPath()
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf(".zshrc does not exist")
		}
		return fmt.Errorf("failed to inspect .zshrc: %w", err)
	}
	if _, err := Backup(); err != nil {
		return fmt.Errorf("backup failed: %w", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read .zshrc: %w", err)
	}
	if err := atomicWrite(path, []byte(removeBlock(string(data))), info.Mode().Perm()); err != nil {
		return fmt.Errorf("failed to write .zshrc: %w", err)
	}
	return nil
}

func EnsureOzshDir() error {
	if err := os.MkdirAll(OmegaDir(), 0o700); err != nil {
		return err
	}
	return os.Chmod(OmegaDir(), 0o700)
}

func ensureZshrc(path string) (os.FileMode, error) {
	info, err := os.Stat(path)
	if err == nil {
		return info.Mode().Perm(), nil
	}
	if !os.IsNotExist(err) {
		return 0, fmt.Errorf("failed to inspect .zshrc: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return 0, fmt.Errorf("failed to create .zshrc directory: %w", err)
	}
	if err := atomicWrite(path, []byte{}, 0o600); err != nil {
		return 0, fmt.Errorf("failed to create .zshrc: %w", err)
	}
	return 0o600, nil
}

func atomicWrite(path string, data []byte, mode os.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".ozsh-write-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if err := tmp.Chmod(mode); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return err
	}
	return os.Chmod(path, mode)
}

func injectBlockContent(content string) string {
	content = removeBlock(content)
	if content != "" {
		content += "\n"
	}
	return content + ManagedBlock()
}

func removeBlock(content string) string {
	singleSuffix := "\n" + ManagedBlock()
	if strings.HasSuffix(content, singleSuffix) {
		return strings.TrimSuffix(content, singleSuffix)
	}
	if strings.HasPrefix(content, ManagedBlock()) {
		return content[len(ManagedBlock()):]
	}

	lines := strings.Split(content, "\n")
	out := make([]string, 0, len(lines))
	inBlock := false
	foundBlock := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == blockStart {
			inBlock = true
			foundBlock = true
			continue
		}
		if trimmed == blockEnd {
			inBlock = false
			continue
		}
		if !inBlock {
			out = append(out, line)
		}
	}
	if !foundBlock {
		return content
	}
	result := strings.TrimRight(strings.Join(out, "\n"), "\n")
	if result != "" {
		result += "\n"
	}
	return result
}
