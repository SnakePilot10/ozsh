package main

import "testing"

func TestThemeSelectionWithoutVariant(t *testing.T) {
	name, variant, err := parseThemeSelection([]string{"dracula"})
	if err != nil {
		t.Fatalf("parseThemeSelection() error = %v", err)
	}
	if name != "dracula" || variant != "" {
		t.Fatalf("parseThemeSelection() = %q/%q, want dracula with no variant", name, variant)
	}
}
