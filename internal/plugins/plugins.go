package plugins

import (
    "fmt"
    "net/url"
    "os"
    "os/exec"
    "path/filepath"
    "strings"

    "github.com/snakepilot10/ozsh/internal/config"
)

func Dir() string {
    home := os.Getenv("HOME")
    if home == "" {
        return ""
    }
    return filepath.Join(home, ".config", "ozsh", "plugins")
}

func Add(cfg *config.Config, rawURL, load string) (string, error) {
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
    if pluginURL.Scheme != "https" {
        return "", fmt.Errorf("plugin URL must use https")
    }
    name := strings.TrimSuffix(filepath.Base(pluginURL.Path), ".git")
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
    dst := filepath.Join(pluginsDir, name)
    if err := os.MkdirAll(pluginsDir, 0755); err != nil {
        return "", fmt.Errorf("failed to create plugins dir: %w", err)
    }
    if err := exec.Command("git", "clone", "--depth", "1", rawURL, dst).Run(); err != nil {
        return "", fmt.Errorf("failed to clone plugin: %w", err)
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
        if item.Name == name {
            // Remove the plugin directory from disk.
            if item.Source != "" {
                if err := os.RemoveAll(item.Source); err != nil {
                    return fmt.Errorf("failed to remove plugin directory: %w", err)
                }
            }
            cfg.Plugins.Items = append(cfg.Plugins.Items[:i], cfg.Plugins.Items[i+1:]...)
            return nil
        }
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
        if cfg.Plugins.Items[i].Name == name {
            if trusted {
                if err := validateTrustTarget(cfg.Plugins.Items[i]); err != nil {
                    return fmt.Errorf("plugin %q cannot be trusted: %w", name, err)
                }
            }
            cfg.Plugins.Items[i].Trusted = trusted
            return nil
        }
    }
    return fmt.Errorf("plugin %q not found", name)
}

func validateName(name string) error {
    if name == "" || name == "." || name == ".." {
        return fmt.Errorf("invalid plugin name %q", name)
    }
    if strings.ContainsAny(name, `/\\`) {
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
    if load == ".." || strings.HasPrefix(load, ".."+string(filepath.Separator)) {
        return fmt.Errorf("plugin load path must stay inside plugin")
    }
    ext := strings.ToLower(filepath.Ext(load))
    if ext != ".zsh" && ext != ".sh" {
        return fmt.Errorf("plugin load file must be .zsh or .sh")
    }
    return nil
}

func validateTrustTarget(item config.PluginItem) error {
    load := filepath.Clean(strings.TrimSpace(item.Load))
    if load == "." {
        load = ""
    }
    if err := validateLoad(load); err != nil {
        return err
    }

    target := item.Source
    if load != "" {
        target = filepath.Join(item.Source, load)
    }
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
