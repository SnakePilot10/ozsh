package tui

import (
	"reflect"
	"strings"
	"testing"

	"github.com/snakepilot10/ozsh/internal/config"
)

func TestThemeGalleryExposesCircuitVariantsAsDirectChoices(t *testing.T) {
	model := NewModel(config.Default())
	model.setTab(tabThemes)

	var got []string
	for cursor := 0; cursor < model.selectionCount(); cursor++ {
		model.cursor = cursor
		preset, ok := model.selectedTheme()
		if !ok {
			t.Fatalf("selectedTheme() failed at cursor %d", cursor)
		}
		if preset.ID == "circuit" {
			got = append(got, preset.Variant)
		}
	}

	want := []string{"blue", "green", "amber", "red", "mono", "neon"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("direct Circuit choices = %#v, want %#v", got, want)
	}
}

func TestThemeGalleryRendersEveryCircuitVariantName(t *testing.T) {
	model := NewModel(config.Default())
	model.setTab(tabThemes)
	model.width = 100
	model.height = 42

	view := model.themes()
	for _, name := range []string{"Circuit Blue", "Circuit Green", "Circuit Amber", "Circuit Red", "Circuit Mono", "Circuit Neon"} {
		if !strings.Contains(view, name) {
			t.Fatalf("theme gallery missing %q:\n%s", name, view)
		}
	}
}

func TestThemeGalleryMarksOnlyTheAppliedCircuitVariant(t *testing.T) {
	cfg := config.Default()
	cfg.Theme.ID = "circuit"
	cfg.Theme.Variant = "amber"
	cfg.Theme.Name = "Circuit Amber"
	model := NewModel(cfg)
	model.setTab(tabThemes)
	model.width = 100
	model.height = 42
	for cursor := 0; cursor < model.selectionCount(); cursor++ {
		model.cursor = cursor
		preset, ok := model.selectedTheme()
		if ok && preset.ID == "circuit" && preset.Variant == "amber" {
			break
		}
	}

	view := model.themes()
	if !strings.Contains(view, "✓ Circuit Amber") {
		t.Fatalf("applied amber variant is not marked:\n%s", view)
	}
	for _, other := range []string{"✓ Circuit Blue", "✓ Circuit Green", "✓ Circuit Red", "✓ Circuit Mono", "✓ Circuit Neon"} {
		if strings.Contains(view, other) {
			t.Fatalf("non-applied variant %q is marked:\n%s", other, view)
		}
	}
}
