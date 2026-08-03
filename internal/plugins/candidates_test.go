package plugins

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func writeCandidate(t *testing.T, root, relative string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("MkdirAll(%s) error = %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte("# plugin\n"), 0o600); err != nil {
		t.Fatalf("WriteFile(%s) error = %v", path, err)
	}
}

func candidatePaths(candidates []Candidate) []string {
	paths := make([]string, len(candidates))
	for i, candidate := range candidates {
		paths[i] = filepath.ToSlash(candidate.RelativePath)
	}
	return paths
}

func TestDiscoverCandidatesRanksExpectedFiles(t *testing.T) {
	root := t.TempDir()
	for _, path := range []string{
		"demo.plugin.zsh",
		"other.plugin.zsh",
		"plugin.zsh",
		"plugin.sh",
		"lib/nested.plugin.zsh",
		"lib/deeper/plugin.zsh",
	} {
		writeCandidate(t, root, path)
	}

	got, err := DiscoverCandidates(root, "demo", DefaultScanLimits)
	if err != nil {
		t.Fatalf("DiscoverCandidates() error = %v", err)
	}
	want := []string{
		"demo.plugin.zsh",
		"other.plugin.zsh",
		"plugin.zsh",
		"plugin.sh",
		"lib/nested.plugin.zsh",
		"lib/deeper/plugin.zsh",
	}
	if paths := candidatePaths(got); !reflect.DeepEqual(paths, want) {
		t.Fatalf("candidate paths = %#v, want %#v", paths, want)
	}
	if got[0].Kind != "plugin.zsh" || got[0].Depth != 0 {
		t.Fatalf("first candidate metadata = %#v", got[0])
	}
}

func TestDiscoverCandidatesSkipsExcludedAndDeepDirectories(t *testing.T) {
	root := t.TempDir()
	writeCandidate(t, root, ".git/hooks/plugin.sh")
	writeCandidate(t, root, "node_modules/tool/plugin.zsh")
	writeCandidate(t, root, "docs/example.plugin.zsh")
	writeCandidate(t, root, "a/b/c/d/e/too-deep.zsh")
	writeCandidate(t, root, "a/b/c/d/allowed.zsh")

	got, err := DiscoverCandidates(root, "demo", DefaultScanLimits)
	if err != nil {
		t.Fatalf("DiscoverCandidates() error = %v", err)
	}
	want := []string{"a/b/c/d/allowed.zsh"}
	if paths := candidatePaths(got); !reflect.DeepEqual(paths, want) {
		t.Fatalf("candidate paths = %#v, want %#v", paths, want)
	}
}

func TestDiscoverCandidatesEnforcesCandidateLimit(t *testing.T) {
	root := t.TempDir()
	writeCandidate(t, root, "one.zsh")
	writeCandidate(t, root, "two.zsh")
	_, err := DiscoverCandidates(root, "demo", ScanLimits{MaxDepth: 4, MaxCandidates: 1})
	if err == nil || !strings.Contains(err.Error(), "candidate limit") {
		t.Fatalf("DiscoverCandidates() error = %v, want candidate limit", err)
	}
}

func TestDiscoverCandidatesRejectsMatchingSymlink(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "real.txt")
	if err := os.WriteFile(target, []byte("# plugin\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(root, "demo.plugin.zsh")); err != nil {
		t.Fatal(err)
	}
	if _, err := DiscoverCandidates(root, "demo", DefaultScanLimits); err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("DiscoverCandidates() error = %v, want symlink rejection", err)
	}
}

func TestValidateCandidateRejectsTraversalAndIntermediateSymlink(t *testing.T) {
	root := t.TempDir()
	if err := ValidateCandidate(root, "../outside.zsh"); err == nil {
		t.Fatal("ValidateCandidate(traversal) error = nil")
	}

	external := t.TempDir()
	writeCandidate(t, external, "plugin.zsh")
	if err := os.Symlink(external, filepath.Join(root, "linked")); err != nil {
		t.Fatal(err)
	}
	if err := ValidateCandidate(root, "linked/plugin.zsh"); err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("ValidateCandidate(symlink) error = %v", err)
	}
}
