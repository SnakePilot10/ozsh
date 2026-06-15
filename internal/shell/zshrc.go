package shell

import (
	"fmt"
	"os"
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
	after := injectBlockContent(before)
	return before, after, nil
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

	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := os.WriteFile(path, []byte{}, 0644); err != nil {
			return fmt.Errorf("failed to create .zshrc: %w", err)
		}
	}

	if _, err := Backup(); err != nil {
		return fmt.Errorf("backup failed: %w", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read .zshrc: %w", err)
	}
	content := injectBlockContent(string(data))

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write .zshrc: %w", err)
	}
	return nil
}

func RemoveBlock() error {
	path := ZshrcPath()
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf(".zshrc does not exist")
	}

	if _, err := Backup(); err != nil {
		return fmt.Errorf("backup failed: %w", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read .zshrc: %w", err)
	}

	content := removeBlock(string(data))
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write .zshrc: %w", err)
	}
	return nil
}

func EnsureOzshDir() error {
	return os.MkdirAll(OmegaDir(), 0755)
}

func injectBlockContent(content string) string {
	content = removeBlock(content)
	if content != "" {
		content += "\n"
	}
	content += ManagedBlock()
	return content
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
	var out []string
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
	result := strings.Join(out, "\n")
	result = strings.TrimRight(result, "\n")
	if result != "" {
		result += "\n"
	}
	return result
}
