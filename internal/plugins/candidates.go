package plugins

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ScanLimits bounds work performed against an untrusted plugin repository.
type ScanLimits struct {
	MaxDepth      int
	MaxCandidates int
}

// DefaultScanLimits keeps discovery useful without walking arbitrary trees.
var DefaultScanLimits = ScanLimits{MaxDepth: 4, MaxCandidates: 128}

// Candidate is a validated shell file that the user may choose to source.
type Candidate struct {
	RelativePath string
	Kind         string
	Depth        int
}

var excludedCandidateDirectories = map[string]struct{}{
	".git":         {},
	"build":        {},
	"dist":         {},
	"docs":         {},
	"examples":     {},
	"node_modules": {},
	"test":         {},
	"tests":        {},
	"vendor":       {},
}

// DiscoverCandidates finds safe plugin entry points and returns them in a
// deterministic preference order.
func DiscoverCandidates(root, repositoryName string, limits ScanLimits) ([]Candidate, error) {
	if limits.MaxDepth < 0 {
		return nil, fmt.Errorf("candidate max depth must not be negative")
	}
	if limits.MaxCandidates <= 0 {
		return nil, fmt.Errorf("candidate limit must be positive")
	}
	root = filepath.Clean(root)
	rootInfo, err := os.Lstat(root)
	if err != nil {
		return nil, fmt.Errorf("inspect plugin root: %w", err)
	}
	if rootInfo.Mode()&os.ModeSymlink != 0 || !rootInfo.IsDir() {
		return nil, fmt.Errorf("plugin root must be a real directory")
	}

	candidates := make([]Candidate, 0)
	err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == root {
			return nil
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return fmt.Errorf("resolve candidate path: %w", err)
		}
		relative = filepath.Clean(relative)
		if relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
			return fmt.Errorf("candidate escaped plugin root: %s", relative)
		}

		if entry.IsDir() {
			if _, excluded := excludedCandidateDirectories[strings.ToLower(entry.Name())]; excluded {
				return filepath.SkipDir
			}
			if pathDepth(relative) > limits.MaxDepth {
				return filepath.SkipDir
			}
			return nil
		}

		kind, accepted := candidateKind(entry.Name())
		if !accepted {
			return nil
		}
		depth := pathDepth(filepath.Dir(relative))
		if depth > limits.MaxDepth {
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("candidate %q must not be a symlink", relative)
		}
		if err := ValidateCandidate(root, relative); err != nil {
			return fmt.Errorf("candidate %q is unsafe: %w", relative, err)
		}
		candidates = append(candidates, Candidate{
			RelativePath: relative,
			Kind:         kind,
			Depth:        depth,
		})
		if len(candidates) > limits.MaxCandidates {
			return fmt.Errorf("candidate limit exceeded: more than %d shell files", limits.MaxCandidates)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Slice(candidates, func(i, j int) bool {
		left := candidateRank(candidates[i], repositoryName)
		right := candidateRank(candidates[j], repositoryName)
		if left != right {
			return left < right
		}
		if candidates[i].Depth != candidates[j].Depth {
			return candidates[i].Depth < candidates[j].Depth
		}
		return filepath.ToSlash(candidates[i].RelativePath) < filepath.ToSlash(candidates[j].RelativePath)
	})
	return candidates, nil
}

// ValidateCandidate proves a selected file stays inside root and contains no
// symlink path components before it can become a trusted load target.
func ValidateCandidate(root, relative string) error {
	root = filepath.Clean(root)
	relative = filepath.Clean(strings.TrimSpace(relative))
	if relative == "" || relative == "." {
		return fmt.Errorf("candidate path is required")
	}
	if filepath.IsAbs(relative) || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return fmt.Errorf("candidate path must stay inside plugin")
	}
	if err := ValidateLoadPath(relative); err != nil {
		return err
	}

	rootInfo, err := os.Lstat(root)
	if err != nil {
		return fmt.Errorf("plugin root unavailable: %w", err)
	}
	if rootInfo.Mode()&os.ModeSymlink != 0 || !rootInfo.IsDir() {
		return fmt.Errorf("plugin root must be a real directory")
	}

	current := root
	for _, component := range strings.Split(relative, string(filepath.Separator)) {
		current = filepath.Join(current, component)
		info, err := os.Lstat(current)
		if err != nil {
			return fmt.Errorf("load path component unavailable: %w", err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("load path must not contain symlinks")
		}
	}
	finalInfo, err := os.Lstat(current)
	if err != nil {
		return fmt.Errorf("load file unavailable: %w", err)
	}
	if !finalInfo.Mode().IsRegular() {
		return fmt.Errorf("load file must be a regular file")
	}
	file, err := os.Open(current)
	if err != nil {
		return fmt.Errorf("load file must be readable: %w", err)
	}
	return file.Close()
}

func candidateKind(name string) (string, bool) {
	lower := strings.ToLower(name)
	switch {
	case strings.HasSuffix(lower, ".plugin.zsh"):
		return "plugin.zsh", true
	case strings.HasSuffix(lower, ".zsh"):
		return "zsh", true
	case strings.HasSuffix(lower, ".sh"):
		return "sh", true
	default:
		return "", false
	}
}

func candidateRank(candidate Candidate, repositoryName string) int {
	root := candidate.Depth == 0
	base := strings.ToLower(filepath.Base(candidate.RelativePath))
	preferred := strings.ToLower(repositoryName + ".plugin.zsh")
	if root && candidate.Kind == "plugin.zsh" && base == preferred {
		return 0
	}
	if root {
		switch candidate.Kind {
		case "plugin.zsh":
			return 1
		case "zsh":
			return 2
		case "sh":
			return 3
		}
	}
	switch candidate.Kind {
	case "plugin.zsh":
		return 4
	case "zsh":
		return 5
	default:
		return 6
	}
}

func pathDepth(relative string) int {
	relative = filepath.Clean(relative)
	if relative == "." || relative == "" {
		return 0
	}
	return len(strings.Split(relative, string(filepath.Separator)))
}
