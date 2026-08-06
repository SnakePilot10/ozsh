package plugins

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/snakepilot10/ozsh/internal/config"
)

type cloneFunc func(context.Context, string, string) error

func (fn cloneFunc) Clone(ctx context.Context, repositoryURL, destination string) error {
	return fn(ctx, repositoryURL, destination)
}

func TestStageRepositoryClonesAndFindsCandidates(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	runner := cloneFunc(func(ctx context.Context, repositoryURL, destination string) error {
		if repositoryURL != "https://example.com/demo.git" {
			t.Fatalf("repository URL = %q", repositoryURL)
		}
		if _, ok := ctx.Deadline(); !ok {
			t.Fatal("clone context has no timeout")
		}
		return os.WriteFile(filepath.Join(destination, "demo.plugin.zsh"), []byte("# plugin\n"), 0o600)
	})

	stage, err := StageRepository(context.Background(), config.Default(), "https://example.com/demo.git", runner)
	if err != nil {
		t.Fatalf("StageRepository() error = %v", err)
	}
	if !strings.HasPrefix(filepath.Base(stage.StagingDir), ".staging-") {
		t.Fatalf("staging directory = %q", stage.StagingDir)
	}
	if stage.FinalDir != filepath.Join(Dir(), "demo") {
		t.Fatalf("final directory = %q", stage.FinalDir)
	}
	if len(stage.Candidates) != 1 || stage.Candidates[0].RelativePath != "demo.plugin.zsh" {
		t.Fatalf("candidates = %#v", stage.Candidates)
	}
	info, err := os.Stat(stage.StagingDir)
	if err != nil {
		t.Fatalf("Stat(staging) error = %v", err)
	}
	if info.Mode().Perm() != 0o700 {
		t.Fatalf("staging mode = %o, want 700", info.Mode().Perm())
	}
	if err := stage.Cleanup(); err != nil {
		t.Fatalf("Cleanup() error = %v", err)
	}
	if _, err := os.Stat(stage.StagingDir); !os.IsNotExist(err) {
		t.Fatalf("staging directory still exists: %v", err)
	}
}

func TestStageRepositoryCancellationRemovesStagingDirectory(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	runner := cloneFunc(func(ctx context.Context, _, _ string) error { return ctx.Err() })
	_, err := StageRepository(ctx, config.Default(), "https://example.com/demo.git", runner)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("StageRepository() error = %v, want context cancellation", err)
	}
	matches, globErr := filepath.Glob(filepath.Join(Dir(), ".staging-*"))
	if globErr != nil {
		t.Fatal(globErr)
	}
	if len(matches) != 0 {
		t.Fatalf("cancel left staging directories: %v", matches)
	}
}

func TestStageRepositoryCloneFailureCleansUp(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	cloneErr := errors.New("network unavailable")
	runner := cloneFunc(func(_ context.Context, _, _ string) error { return cloneErr })
	_, err := StageRepository(context.Background(), config.Default(), "https://example.com/demo.git", runner)
	if !errors.Is(err, cloneErr) {
		t.Fatalf("StageRepository() error = %v, want clone failure", err)
	}
	matches, _ := filepath.Glob(filepath.Join(Dir(), ".staging-*"))
	if len(matches) != 0 {
		t.Fatalf("failure left staging directories: %v", matches)
	}
}

func TestStageRepositoryRejectsExistingDestinationBeforeClone(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	if err := os.MkdirAll(filepath.Join(Dir(), "demo"), 0o700); err != nil {
		t.Fatal(err)
	}
	called := false
	runner := cloneFunc(func(_ context.Context, _, _ string) error {
		called = true
		return nil
	})
	_, err := StageRepository(context.Background(), config.Default(), "https://example.com/demo.git", runner)
	if err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("StageRepository() error = %v, want destination conflict", err)
	}
	if called {
		t.Fatal("clone runner was called despite destination conflict")
	}
}

func TestExecCloneRunnerUsesShallowClone(t *testing.T) {
	root := t.TempDir()
	argsPath := filepath.Join(root, "args.txt")
	gitPath := filepath.Join(root, "git")
	script := "#!/bin/sh\nprintf '%s\\n' \"$@\" > \"$OZSH_TEST_ARGS\"\n"
	if err := os.WriteFile(gitPath, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("OZSH_TEST_ARGS", argsPath)
	t.Setenv("PATH", root)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := (ExecCloneRunner{}).Clone(ctx, "https://example.com/demo.git", filepath.Join(root, "checkout")); err != nil {
		t.Fatalf("Clone() error = %v", err)
	}
	data, err := os.ReadFile(argsPath)
	if err != nil {
		t.Fatal(err)
	}
	got := strings.Fields(string(data))
	want := []string{"clone", "--depth", "1", "https://example.com/demo.git", filepath.Join(root, "checkout")}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("git args = %#v, want %#v", got, want)
	}
}
