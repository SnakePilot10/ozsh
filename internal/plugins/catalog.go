package plugins

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/snakepilot10/ozsh/internal/config"
)

type Definition struct {
	ID          string
	Name        string
	Description string
	URL         string
	Load        string
}

type Status struct {
	Selected  bool
	Installed bool
	Healthy   bool
	Trusted   bool
	Active    bool
}

var curatedCatalog = []Definition{
	{
		ID: "zsh-autosuggestions", Name: "Autosuggestions",
		Description: "Suggest commands from history while you type.",
		URL: "https://github.com/zsh-users/zsh-autosuggestions.git", Load: "zsh-autosuggestions.zsh",
	},
	{
		ID: "fzf-tab", Name: "fzf-tab",
		Description: "Replace standard completion selection with an fzf menu.",
		URL: "https://github.com/Aloxaf/fzf-tab.git", Load: "fzf-tab.plugin.zsh",
	},
	{
		ID: "zsh-syntax-highlighting", Name: "Syntax highlighting",
		Description: "Highlight commands before execution and load last.",
		URL: "https://github.com/zsh-users/zsh-syntax-highlighting.git", Load: "zsh-syntax-highlighting.zsh",
	},
}

func Catalog() []Definition {
	return append([]Definition(nil), curatedCatalog...)
}

func FindDefinition(id string) (Definition, bool) {
	for _, definition := range curatedCatalog {
		if definition.ID == id {
			return definition, true
		}
	}
	return Definition{}, false
}

func StatusFor(cfg *config.Config, definition Definition) Status {
	if cfg == nil {
		return Status{}
	}
	status := Status{Selected: contains(cfg.Plugins.Selected, definition.ID)}
	for _, item := range cfg.Plugins.Items {
		if item.Name != definition.ID {
			continue
		}
		status.Installed = true
		status.Trusted = item.Trusted
		status.Healthy = healthyItem(item, definition)
		status.Active = status.Healthy && item.Enabled && item.Trusted
		break
	}
	return status
}

func InstallRecommended(ctx context.Context, cfg *config.Config, selected []string) error {
	if cfg == nil {
		return fmt.Errorf("config is nil")
	}
	cfg.Plugins.Selected = uniqueSelected(selected)
	for _, definition := range curatedCatalog {
		if !contains(cfg.Plugins.Selected, definition.ID) {
			continue
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		index := itemIndex(cfg.Plugins.Items, definition.ID)
		if index < 0 {
			name, err := Add(cfg, definition.URL, definition.Load)
			if err != nil {
				return fmt.Errorf("install %s: %w", definition.Name, err)
			}
			index = itemIndex(cfg.Plugins.Items, name)
		}
		cfg.Plugins.Items[index].Enabled = true
		if err := SetTrusted(cfg, definition.ID, true); err != nil {
			return fmt.Errorf("activate %s: %w", definition.Name, err)
		}
	}
	cfg.Plugins.Items = orderItems(cfg.Plugins.Items)
	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("save recommended plugins: %w", err)
	}
	return nil
}

func healthyItem(item config.PluginItem, definition Definition) bool {
	if item.Load != definition.Load {
		return false
	}
	rootInfo, err := os.Lstat(item.Source)
	if err != nil || rootInfo.Mode()&os.ModeSymlink != 0 || !rootInfo.IsDir() {
		return false
	}
	path := filepath.Join(item.Source, item.Load)
	info, err := os.Lstat(path)
	return err == nil && info.Mode()&os.ModeSymlink == 0 && info.Mode().IsRegular()
}

func orderItems(items []config.PluginItem) []config.PluginItem {
	priorityFor := func(name string) int {
		switch name {
		case "zsh-autosuggestions":
			return 0
		case "fzf-tab":
			return 1
		case "zsh-syntax-highlighting":
			return 3
		default:
			return 2
		}
	}
	result := append([]config.PluginItem(nil), items...)
	for i := 1; i < len(result); i++ {
		current := result[i]
		currentPriority := priorityFor(current.Name)
		j := i - 1
		for j >= 0 && priorityFor(result[j].Name) > currentPriority {
			result[j+1] = result[j]
			j--
		}
		result[j+1] = current
	}
	return result
}

func itemIndex(items []config.PluginItem, name string) int {
	for i, item := range items {
		if item.Name == name {
			return i
		}
	}
	return -1
}

func contains(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}

func uniqueSelected(values []string) []string {
	seen := map[string]struct{}{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		if _, ok := FindDefinition(value); !ok {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}
