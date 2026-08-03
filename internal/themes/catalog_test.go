package themes

import (
	"reflect"
	"testing"

	"github.com/snakepilot10/ozsh/internal/config"
)

func TestListReturnsEverySelectablePreset(t *testing.T) {
	presets := List()
	if len(presets) != 17 {
		t.Fatalf("List() length = %d, want 17", len(presets))
	}
	var circuit []string
	for _, preset := range presets {
		if preset.ID == "circuit" {
			circuit = append(circuit, preset.Variant)
		}
	}
	wantCircuit := []string{"blue", "green", "amber", "red", "mono", "neon"}
	if !reflect.DeepEqual(circuit, wantCircuit) {
		t.Fatalf("List() Circuit variants = %#v, want %#v", circuit, wantCircuit)
	}
}

func TestFamiliesReturnsTwelveUniqueThemes(t *testing.T) {
	presets := Families()
	if len(presets) != 12 {
		t.Fatalf("Families() length = %d, want 12", len(presets))
	}
	seen := map[string]bool{}
	for _, preset := range presets {
		if seen[preset.ID] {
			t.Fatalf("Families() returned duplicate ID %q", preset.ID)
		}
		seen[preset.ID] = true
	}
}

func TestCircuitVariantsAndLookup(t *testing.T) {
	want := []string{"blue", "green", "amber", "red", "mono", "neon"}
	if got := Variants("circuit"); !reflect.DeepEqual(got, want) {
		t.Fatalf("Variants(circuit) = %#v, want %#v", got, want)
	}
	preset, ok := Get("circuit", "amber")
	if !ok || preset.Name != "Circuit Amber" || preset.Variant != "amber" {
		t.Fatalf("Get(circuit, amber) = %#v, %t", preset, ok)
	}
	first, ok := Get("circuit", "")
	if !ok || first.Variant != "blue" {
		t.Fatalf("Get(circuit, empty) = %#v, %t, want blue", first, ok)
	}
}

func TestListAndGetReturnIndependentCopies(t *testing.T) {
	list := List()
	list[0].Order[0] = "changed"
	fresh := List()
	if fresh[0].Order[0] == "changed" {
		t.Fatal("List() returned aliased order slices")
	}
	preset, _ := Get("cyberpunk", "")
	preset.Order[0] = "changed"
	freshPreset, _ := Get("cyberpunk", "")
	if freshPreset.Order[0] == "changed" {
		t.Fatal("Get() returned aliased order slices")
	}
}

func TestApplyClonesConfigurationAndAppliesPresentation(t *testing.T) {
	base := config.Default()
	base.Prompt.DisplayName = "pilot"
	preset, ok := Get("minimal", "")
	if !ok {
		t.Fatal("minimal preset not found")
	}
	applied := Apply(base, preset)
	if applied == base {
		t.Fatal("Apply() returned the input pointer")
	}
	if applied.Theme.ID != "minimal" || applied.Prompt.Layout != config.PromptLayoutOneLine || applied.Prompt.Newline {
		t.Fatalf("applied presentation = theme %q layout %q newline=%t", applied.Theme.ID, applied.Prompt.Layout, applied.Prompt.Newline)
	}
	if applied.Prompt.Symbol != "$" || applied.Prompt.DisplayName != "pilot" {
		t.Fatalf("applied symbol/display = %q/%q", applied.Prompt.Symbol, applied.Prompt.DisplayName)
	}
	applied.Prompt.Order[0] = "changed"
	if base.Prompt.Order[0] == "changed" {
		t.Fatal("Apply() aliased prompt order")
	}
}
