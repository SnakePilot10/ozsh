package plugins

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/snakepilot10/ozsh/internal/config"
)

const cloneTimeout = 45 * time.Second

var pluginNamePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,95}$`)

func Dir() string {
	home := os.Getenv("HOME")
	if home == "" {
		return ""
	}
	return filepath.Join(home, ".config", "ozsh", "plugins")
}

func Add(cfg *config.Config, rawURL, load string) (string, error) {
	rawURL = strings.TrimSpace(rawURL)
	load = filepath.Clean(strings.TrimSpace(load))
	if load == "." {
		load = ""
	}
	if err := validateLoad(load); err != nil {
		return "", err
	}

	pluginURL, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("invalid plugin URL: %w", err)
	}
	if pluginURL.Scheme != "https" || pluginURL.Host == "" || pluginURL.User != nil {
		return "", fmt.Errorf("plugin URL must be an https repository URL without credentials")
	}
	if pluginURL.Fragment != "" {
		return "", fmt.Errorf("plugin URL must not include a fragment")
	}

	name := strings.TrimSuffix(filepath.Base(strings.TrimSuffix(pluginURL.Path, "/")), ".git")
	if err := validateName(name); err != nil {
		return "", err
	}
	for _, item := range cfg.Plugins.Items {
		if item.Name == name {
			return "", fmt.Errorf("plugin %q already exists", name)
		}
	}

	pluginsDir := Dir()
	if pluginsDir == "" {
		return "", fmt.Errorf("cannot determine plugins directory")
	}
	if err := os.MkdirAll(pluginsDir, 0o700); err != nil {
		return "", fmt.Errorf("failed to create plugins dir: %w", err)
	}
	if err := os.Chmod(pluginsDir, 0o700); err != nil {
		return "", fmt.Errorf("failed to secure plugins dir: %w", err)
	}

	dst := filepath.Join(pluginsDir, name)
	if _, err := os.Lstat(dst); err == nil {
		return "", fmt.Errorf("plugin directory already exists: %s", dst)
	} else if !os.IsNotExist(err) {
		return "", fmt.Errorf("failed to inspect plugin destination: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), cloneTimeout)
	defer cancel()
	out, err := exec.CommandContext(ctx, "git", "clone", "--depth", "1", rawURL, dst).CombinedOutput()
	if err != nil {
		_ = os.RemoveAll(dst)
		if ctx.Err() == context.DeadlineExceeded {
			return "", fmt.Errorf("plugin clone timed out after %s", cloneTimeout)
		}
		message := strings.TrimSpace(string(out))
		if message == "" {
			return "", fmt.Errorf("failed to clone plugin: %w", err)
		}
		return "", fmt.Errorf("failed to clone plugin: %s", message)
	}

	cfg.Plugins.Items = append(cfg.Plugins.Items, config.PluginItem{
		Name:    name,
		Enabled: true,
		Trusted: false,
		Source:  dst,
		Load:    load,
	})
	return name, nil
}

func Remove(cfg *config.Config, name string) error {
	if err := validateName(name); err != nil {
		return err
	}
	for i, item := range cfg.Plugins.Items {
		if item.Name != name {
			continue
		}
		if err := validatePluginSource(item); err != nil {
			return fmt.Errorf("plugin %q has an unsafe source path: %w", name, err)
		}
		if err := os.RemoveAll(item.Source); err != nil {
			return fmt.Errorf("failed to remove plugin directory: %w", err)
		}
		cfg.Plugins.Items = append(cfg.Plugins.Items[:i], cfg.Plugins.Items[i+1:]...)
		return nil
	}
	return fmt.Errorf("plugin %q not found", name)
}

func SetEnabled(cfg *config.Config, name string, enabled bool) error {
	if err := validateName(name); err != nil {
		return err
	}
	for i := range cfg.Plugins.Items {
		if cfg.Plugins.Items[i].Name == name {
			cfg.Plugins.Items[i].Enabled = enabled
			return nil
		}
	}
	return fmt.Errorf("plugin %q not found", name)
}

func SetTrusted(cfg *config.Config, name string, trusted bool) error {
	if err := validateName(name); err != nil {
		return err
	}
	for i := range cfg.Plugins.Items {
		if cfg.Plugins.Items[i].Name != name {
			continue
		}
		if trusted {
			if err := validateTrustTarget(cfg.Plugins.Items[i]); err != nil {
				return fmt.Errorf("plugin %q cannot be trusted: %w", name, err)
			}
		}
		cfg.Plugins.Items[i].Trusted = trusted
		return nil
	}
	return fmt.Errorf("plugin %q not found", name)
}

func validateName(name string) error {
	if !pluginNamePattern.MatchString(name) {
		return fmt.Errorf("invalid plugin name %q", name)
	}
	return nil
}

func validateLoad(load string) error {
	if load == "" {
		return nil
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

func validatePluginSource(item config.PluginItem) error {
	if err := validateName(item.Name); err != nil {
		return err
	}
	root := Dir()
	if root == "" {
		return fmt.Errorf("cannot determine plugins directory")
	}
	expected := filepath.Clean(filepath.Join(root, item.Name))
	if filepath.Clean(item.Source) != expected {
		return fmt.Errorf("source must be %s", expected)
	}
	return nil
}

func validateTrustTarget(item config.PluginItem) error {
	if err := validatePluginSource(item); err != nil {
		return err
	}
	load := filepath.Clean(strings.TrimSpace(item.Load))
	if load == "." {
		load = ""
	}
	if err := validateLoad(load); err != nil {
		return err
	}
	if load == "" {
		return fmt.Errorf("plugin load file is required before trusting a plugin")
	}

	sourceInfo, err := os.Lstat(item.Source)
	if err != nil {
		return fmt.Errorf("plugin directory unavailable: %w", err)
	}
	if sourceInfo.Mode()&os.ModeSymlink != 0 || !sourceInfo.IsDir() {
		return fmt.Errorf("plugin source must be a real directory")
	}

	target := filepath.Join(item.Source, load)
	info, err := os.Lstat(target)
	if err != nil {
		return fmt.Errorf("load file unavailable: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("load file must not be a symlink")
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("load file must be a regular file")
	}
	file, err := os.Open(target)
	if err != nil {
		return fmt.Errorf("load file must be readable: %w", err)
	}
	return file.Close()
}