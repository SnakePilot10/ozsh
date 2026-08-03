package plugins

import (
	"fmt"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/snakepilot10/ozsh/internal/config"
)

// Repository is a validated HTTPS plugin repository and its managed name.
type Repository struct {
	URL  string
	Name string
}

// ParseRepository validates an HTTPS repository URL and derives its safe name.
func ParseRepository(raw string) (Repository, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return Repository{}, fmt.Errorf("plugin URL is required")
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return Repository{}, fmt.Errorf("invalid plugin URL: %w", err)
	}
	if parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil {
		return Repository{}, fmt.Errorf("plugin URL must be an https repository URL without credentials")
	}
	if parsed.RawQuery != "" || parsed.ForceQuery {
		return Repository{}, fmt.Errorf("plugin URL must not include a query string")
	}
	if parsed.Fragment != "" {
		return Repository{}, fmt.Errorf("plugin URL must not include a fragment")
	}
	name := strings.TrimSuffix(filepath.Base(strings.TrimSuffix(parsed.Path, "/")), ".git")
	if err := validateName(name); err != nil {
		return Repository{}, err
	}
	return Repository{URL: raw, Name: name}, nil
}

// ValidateNewRepository rejects names already managed or reserved by ozsh.
func ValidateNewRepository(cfg *config.Config, repository Repository) error {
	if cfg == nil {
		return fmt.Errorf("config is nil")
	}
	if err := validateName(repository.Name); err != nil {
		return err
	}
	if _, curated := FindDefinition(repository.Name); curated {
		return fmt.Errorf("plugin %q already exists in the recommended catalog", repository.Name)
	}
	for _, item := range cfg.Plugins.Items {
		if item.Name == repository.Name {
			return fmt.Errorf("plugin %q already exists", repository.Name)
		}
	}
	return nil
}

// ValidateLoadPath validates a relative shell file inside a managed checkout.
func ValidateLoadPath(load string) error {
	load = strings.TrimSpace(load)
	if load == "" {
		return fmt.Errorf("plugin load file is required")
	}
	if filepath.IsAbs(load) {
		return fmt.Errorf("plugin load path must be relative")
	}
	clean := filepath.Clean(load)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return fmt.Errorf("plugin load path must stay inside plugin")
	}
	ext := strings.ToLower(filepath.Ext(clean))
	if ext != ".zsh" && ext != ".sh" {
		return fmt.Errorf("plugin load file must be .zsh or .sh")
	}
	return nil
}
