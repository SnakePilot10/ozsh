package plugins

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// CloneRunner isolates repository cloning so the TUI and tests do not depend
// directly on a process implementation.
type CloneRunner interface {
	Clone(ctx context.Context, repositoryURL, destination string) error
}

// ExecCloneRunner performs a shallow clone with the system Git executable.
type ExecCloneRunner struct{}

func (ExecCloneRunner) Clone(ctx context.Context, repositoryURL, destination string) error {
	output, err := exec.CommandContext(ctx, "git", "clone", "--depth", "1", repositoryURL, destination).CombinedOutput()
	if err == nil {
		return nil
	}
	if ctxErr := ctx.Err(); ctxErr != nil {
		return ctxErr
	}
	message := strings.TrimSpace(string(output))
	if message == "" {
		if errors.Is(err, exec.ErrNotFound) {
			return fmt.Errorf("git is not available: %w", err)
		}
		return fmt.Errorf("git clone failed: %w", err)
	}
	return fmt.Errorf("git clone failed: %s: %w", message, err)
}

// StagedRepository is an inactive checkout awaiting candidate selection and
// explicit trust confirmation.
type StagedRepository struct {
	Repository Repository
	StagingDir string
	FinalDir   string
	Candidates []Candidate
}

// StageRepository validates and clones a repository into a private temporary
// checkout. Every failure removes the staging directory.
func StageRepository(ctx context.Context, cfg *config.Config, rawURL string, runner CloneRunner) (stage StagedRepository, err error) {
	repository, err := ParseRepository(rawURL)
	if err != nil {
		return StagedRepository{}, err
	}
	if err := ValidateNewRepository(cfg, repository); err != nil {
		return StagedRepository{}, err
	}
	pluginsDir := Dir()
	if pluginsDir == "" {
		return StagedRepository{}, fmt.Errorf("cannot determine plugins directory")
	}
	if err := os.MkdirAll(pluginsDir, 0o700); err != nil {
		return StagedRepository{}, fmt.Errorf("create plugins directory: %w", err)
	}
	if err := os.Chmod(pluginsDir, 0o700); err != nil {
		return StagedRepository{}, fmt.Errorf("secure plugins directory: %w", err)
	}

	finalDir := filepath.Join(pluginsDir, repository.Name)
	if _, statErr := os.Lstat(finalDir); statErr == nil {
		return StagedRepository{}, fmt.Errorf("plugin directory already exists: %s", finalDir)
	} else if !os.IsNotExist(statErr) {
		return StagedRepository{}, fmt.Errorf("inspect plugin destination: %w", statErr)
	}

	stagingDir, err := os.MkdirTemp(pluginsDir, ".staging-")
	if err != nil {
		return StagedRepository{}, fmt.Errorf("create plugin staging directory: %w", err)
	}
	stage = StagedRepository{
		Repository: repository,
		StagingDir: stagingDir,
		FinalDir:   finalDir,
	}
	cleanup := true
	defer func() {
		if cleanup {
			_ = stage.Cleanup()
		}
	}()
	if err := os.Chmod(stagingDir, 0o700); err != nil {
		return StagedRepository{}, fmt.Errorf("secure plugin staging directory: %w", err)
	}
	if runner == nil {
		runner = ExecCloneRunner{}
	}

	cloneCtx, cancel := context.WithTimeout(ctx, cloneTimeout)
	defer cancel()
	if err := runner.Clone(cloneCtx, repository.URL, stagingDir); err != nil {
		if ctxErr := cloneCtx.Err(); ctxErr != nil {
			return StagedRepository{}, ctxErr
		}
		return StagedRepository{}, fmt.Errorf("clone plugin %q: %w", repository.Name, err)
	}
	if ctxErr := cloneCtx.Err(); ctxErr != nil {
		return StagedRepository{}, ctxErr
	}

	candidates, err := DiscoverCandidates(stagingDir, repository.Name, DefaultScanLimits)
	if err != nil {
		return StagedRepository{}, fmt.Errorf("discover plugin load files: %w", err)
	}
	stage.Candidates = candidates
	cleanup = false
	return stage, nil
}

// Cleanup removes only a staging directory owned by the managed plugin root.
func (stage StagedRepository) Cleanup() error {
	if strings.TrimSpace(stage.StagingDir) == "" {
		return nil
	}
	pluginsDir := Dir()
	if pluginsDir == "" {
		return fmt.Errorf("cannot determine plugins directory")
	}
	root := filepath.Clean(pluginsDir)
	staging := filepath.Clean(stage.StagingDir)
	relative, err := filepath.Rel(root, staging)
	if err != nil {
		return fmt.Errorf("resolve staging path: %w", err)
	}
	if filepath.Dir(relative) != "." || !strings.HasPrefix(filepath.Base(relative), ".staging-") {
		return fmt.Errorf("refusing to remove unmanaged staging path: %s", staging)
	}
	if err := os.RemoveAll(staging); err != nil {
		return fmt.Errorf("remove plugin staging directory: %w", err)
	}
	return nil
}
