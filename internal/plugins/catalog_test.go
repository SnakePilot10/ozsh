package plugins

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/snakepilot10/ozsh/internal/config"
)

func TestCatalogReturnsSafeRecommendedOrder(t *testing.T) {
	catalog := Catalog()
	got := make([]string, len(catalog))
	for i, definition := range catalog {
		got[i] = definition.ID
	}
	want := []string{"zsh-autosuggestions", "fzf-tab", "zsh-syntax-highlighting"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Catalog IDs = %#v, want %#v", got, want)
	}
	catalog[0].Name = "changed"
	if Catalog()[0].Name == "changed" {
		t.Fatal("Catalog() returned aliased definitions")
	}
}

func TestFindDefinitionUsesCuratedID(t *testing.T) {
	definition, ok := FindDefinition("fzf-tab")
	if !ok || definition.Load != "fzf-tab.plugin.zsh" {
		t.Fatalf("FindDefinition(fzf-tab) = %#v, %t", definition, ok)
	}
	if _, ok := FindDefinition("custom"); ok {
		t.Fatal("FindDefinition(custom) unexpectedly succeeded")
	}
}

func TestStatusForReportsSelectedInstalledHealthyAndActive(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	definition, _ := FindDefinition("zsh-autosuggestions")
	root := filepath.Join(home, ".config", "ozsh", "plugins", definition.ID)
	if err := os.MkdirAll(root, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, definition.Load), []byte("# plugin\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg := config.Default()
	cfg.Plugins.Items = []config.PluginItem{{Name: definition.ID, Source: root, Load: definition.Load, Enabled: true, Trusted: true}}
	status := StatusFor(cfg, definition)
	if !status.Selected || !status.Installed || !status.Healthy || !status.Trusted || !status.Active {
		t.Fatalf("StatusFor() = %+v", status)
	}
}

func TestOrderItemsKeepsSyntaxHighlightingLast(t *testing.T) {
	items := []config.PluginItem{
		{Name: "zsh-syntax-highlighting"},
		{Name: "custom"},
		{Name: "fzf-tab"},
		{Name: "zsh-autosuggestions"},
	}
	got := orderItems(items)
	want := []string{"zsh-autosuggestions", "fzf-tab", "custom", "zsh-syntax-highlighting"}
	names := make([]string, len(got))
	for i, item := range got {
		names[i] = item.Name
	}
	if !reflect.DeepEqual(names, want) {
		t.Fatalf("orderItems() = %#v, want %#v", names, want)
	}
}
