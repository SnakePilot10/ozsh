package plugins

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/snakepilot10/ozsh/internal/config"
)

func TestUpdateRevokesTrustWhenRevisionChanges(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	remote := filepath.Join(t.TempDir(), "remote.git")
	seed := filepath.Join(t.TempDir(), "seed")
	runGitTest(t, "", "init", "--bare", "--initial-branch=main", remote)
	runGitTest(t, "", "init", "--initial-branch=main", seed)
	runGitTest(t, seed, "config", "user.email", "test@example.com")
	runGitTest(t, seed, "config", "user.name", "Test")
	writePluginTestFile(t, filepath.Join(seed, "plugin.zsh"), "# v1\n")
	runGitTest(t, seed, "add", "plugin.zsh")
	runGitTest(t, seed, "commit", "-m", "v1")
	runGitTest(t, seed, "remote", "add", "origin", remote)
	runGitTest(t, seed, "push", "-u", "origin", "main")

	managed := filepath.Join(Dir(), "demo")
	if err := os.MkdirAll(filepath.Dir(managed), 0o700); err != nil {
		t.Fatal(err)
	}
	runGitTest(t, "", "clone", remote, managed)
	oldRevision := strings.TrimSpace(runGitTest(t, managed, "rev-parse", "HEAD"))
	cfg := config.Default()
	cfg.Plugins.Items = []config.PluginItem{{
		Name: "demo", Enabled: true, Trusted: true, Source: managed, Load: "plugin.zsh", Revision: oldRevision,
	}}

	changed, err := Update(cfg, "demo")
	if err != nil || changed {
		t.Fatalf("Update(no change) = %t, %v", changed, err)
	}
	if !cfg.Plugins.Items[0].Trusted {
		t.Fatal("Update(no change) revoked trust")
	}

	writePluginTestFile(t, filepath.Join(seed, "plugin.zsh"), "# v2\n")
	runGitTest(t, seed, "add", "plugin.zsh")
	runGitTest(t, seed, "commit", "-m", "v2")
	runGitTest(t, seed, "push", "origin", "main")
	changed, err = Update(cfg, "demo")
	if err != nil || !changed {
		t.Fatalf("Update(changed) = %t, %v", changed, err)
	}
	if cfg.Plugins.Items[0].Trusted {
		t.Fatal("Update(changed) did not revoke trust")
	}
	if cfg.Plugins.Items[0].Revision == oldRevision {
		t.Fatal("Update(changed) did not record new revision")
	}
	inspected, err := Inspect(cfg, "demo")
	if err != nil || inspected.Revision != cfg.Plugins.Items[0].Revision {
		t.Fatalf("Inspect() = %+v, %v", inspected, err)
	}
}

func runGitTest(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v error: %v\n%s", args, err, output)
	}
	return string(output)
}

func writePluginTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}
