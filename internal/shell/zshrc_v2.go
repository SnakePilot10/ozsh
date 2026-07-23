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
	_, target, err := ResolveZshrcTarget()
	if err != nil {
		return false
	}
	data, err := os.ReadFile(target)
	if err != nil {
		return false
	}
	content := string(data)
	return strings.Contains(content, blockStart) && strings.Contains(content, blockEnd)
}

func ManagedBlock() string {
	return fmt.Sprintf("%s\n%s\n%s\n", blockStart, blockBody, blockEnd)
}

func PreviewInjectBlock() (string, string, error) {
	logical, target, err := ResolveZshrcTarget()
	if err != nil {
		return "", "", err
	}
	data, err := os.ReadFile(target)
	if err != nil {
		if os.IsNotExist(err) {
			data = []byte{}
		} else {
			return "", "", fmt.Errorf("failed to read .zshrc%s: %w", zshrcTargetDescription(logical, target), err)
		}
	}
	before := string(data)
	if hasMalformedBlock(before) {
		return "", "", fmt.Errorf("managed ozsh block is malformed; restore or repair .zshrc before applying")
	}
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
	logical, target, err := ResolveZshrcTarget()
	if err != nil {
		return err
	}
	mode, err := ensureZshrc(target)
	if err != nil {
		return err
	}
	data, err := os.ReadFile(target)
	if err != nil {
		return fmt.Errorf("failed to read .zshrc%s: %w", zshrcTargetDescription(logical, target), err)
	}
	content := string(data)
	if hasMalformedBlock(content) {
		return fmt.Errorf("managed ozsh block is malformed; refusing to modify .zshrc")
	}
	if _, err := Backup(); err != nil {
		return fmt.Errorf("backup failed: %w", err)
	}
	if err := atomicWrite(target, []byte(injectBlockContent(content)), mode); err != nil {
		return fmt.Errorf("failed to write .zshrc%s: %w", zshrcTargetDescription(logical, target), err)
	}
	return nil
}

func RemoveBlock() error {
	logical, target, err := ResolveZshrcTarget()
	if err != nil {
		return err
	}
	info, err := os.Stat(target)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf(".zshrc does not exist")
		}
		return fmt.Errorf("failed to inspect .zshrc%s: %w", zshrcTargetDescription(logical, target), err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf(".zshrc%s must be a regular file", zshrcTargetDescription(logical, target))
	}
	data, err := os.ReadFile(target)
	if err != nil {
		return fmt.Errorf("failed to read .zshrc%s: %w", zshrcTargetDescription(logical, target), err)
	}
	content := string(data)
	if hasMalformedBlock(content) {
		return fmt.Errorf("managed ozsh block is malformed; refusing to modify .zshrc")
	}
	if _, err := Backup(); err != nil {
		return fmt.Errorf("backup failed: %w", err)
	}
	if err := atomicWrite(target, []byte(removeBlock(content)), info.Mode().Perm()); err != nil {
		return fmt.Errorf("failed to write .zshrc%s: %w", zshrcTargetDescription(logical, target), err)
	}
	return nil
}

func ResolveZshrcTarget() (logicalPath, targetPath string, err error) {
	if ZshrcPath() == "" {
		return "", "", fmt.Errorf("cannot determine HOME")
	}
	logicalPath = filepath.Clean(ZshrcPath())
	if abs, absErr := filepath.Abs(logicalPath); absErr == nil {
		logicalPath = filepath.Clean(abs)
	}
	info, err := os.Lstat(logicalPath)
	if err != nil {
		if os.IsNotExist(err) {
			return logicalPath, logicalPath, nil
		}
		return logicalPath, "", fmt.Errorf("failed to inspect .zshrc: %w", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		if !info.Mode().IsRegular() {
			return logicalPath, logicalPath, fmt.Errorf(".zshrc must be a regular file")
		}
		return logicalPath, logicalPath, nil
	}

	targetPath, err = filepath.EvalSymlinks(logicalPath)
	if err != nil {
		return logicalPath, "", fmt.Errorf("failed to resolve .zshrc symlink %s: %w", logicalPath, err)
	}
	if abs, absErr := filepath.Abs(targetPath); absErr == nil {
		targetPath = abs
	}
	targetPath = filepath.Clean(targetPath)
	info, err = os.Stat(targetPath)
	if err != nil {
		return logicalPath, targetPath, fmt.Errorf("failed to inspect .zshrc target %s: %w", targetPath, err)
	}
	if !info.Mode().IsRegular() {
		return logicalPath, targetPath, fmt.Errorf(".zshrc target %s must be a regular file", targetPath)
	}
	return logicalPath, targetPath, nil
}

func EnsureOzshDir() error {
	if OmegaDir() == "" {
		return fmt.Errorf("cannot determine HOME")
	}
	if err := os.MkdirAll(OmegaDir(), 0o700); err != nil {
		return err
	}
	return os.Chmod(OmegaDir(), 0o700)
}

func ensureZshrc(path string) (os.FileMode, error) {
	info, err := os.Stat(path)
	if err == nil {
		if !info.Mode().IsRegular() {
			return 0, fmt.Errorf(".zshrc target must be a regular file")
		}
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

func zshrcTargetDescription(logicalPath, targetPath string) string {
	if targetPath == "" || logicalPath == targetPath {
		return ""
	}
	return fmt.Sprintf(" (%s -> %s)", logicalPath, targetPath)
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

func hasMalformedBlock(content string) bool {
	inBlock := false
	for _, line := range strings.Split(content, "\n") {
		switch strings.TrimSpace(line) {
		case blockStart:
			if inBlock {
				return true
			}
			inBlock = true
		case blockEnd:
			if !inBlock {
				return true
			}
			inBlock = false
		}
	}
	return inBlock
}

func removeBlock(content string) string {
	singleSuffix := "\n" + ManagedBlock()
	if strings.HasSuffix(content, singleSuffix) {
		return strings.TrimSuffix(content, singleSuffix)
	}
	if strings.HasPrefix(content, ManagedBlock()) {
		return content[len(ManagedBlock()):]
	}
	if hasMalformedBlock(content) {
		return content
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
