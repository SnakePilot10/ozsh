package tui

import (
	"strings"
	"testing"

	"github.com/snakepilot10/ozsh/internal/config"
)

func TestPromptWorkspaceWideRegions(t *testing.T) {
	model := NewModel(config.Default())
	model.width, model.height = 100, 34
	model.setTab(tabPrompt)

	plain := plainText(model.View())
	for _, label := range []string{"Configuration", "Segments", "Live preview", "Selected segment"} {
		if !strings.Contains(plain, label) {
			t.Fatalf("wide Prompt workspace lost %q:\n%s", label, plain)
		}
	}
}

func TestPromptWorkspaceCompactStacksRegions(t *testing.T) {
	model := NewModel(config.Default())
	model.width, model.height = 58, 28
	model.setTab(tabPrompt)

	plain := plainText(model.View())
	last := -1
	for _, label := range []string{"Configuration", "Segments", "Live preview", "Selected segment"} {
		index := strings.Index(plain, label)
		if index < 0 {
			t.Fatalf("compact Prompt workspace lost %q:\n%s", label, plain)
		}
		if index <= last {
			t.Fatalf("compact Prompt regions are not stacked in semantic order:\n%s", plain)
		}
		last = index
	}
}

func TestSelectedSegmentDetailsFollowCursor(t *testing.T) {
	model := NewModel(config.Default())
	model.width, model.height = 100, 34
	model.setTab(tabPrompt)
	model.cursor = 0

	userView := plainText(model.View())
	if !strings.Contains(userView, "Shows the current shell identity") {
		t.Fatalf("user details missing:\n%s", userView)
	}

	model.cursor = 1
	directoryView := plainText(model.View())
	if !strings.Contains(directoryView, "Shows the current working directory") {
		t.Fatalf("directory details did not follow cursor:\n%s", directoryView)
	}
	if strings.Contains(directoryView, "Shows the current shell identity") {
		t.Fatalf("stale user details remained after cursor movement:\n%s", directoryView)
	}
}
