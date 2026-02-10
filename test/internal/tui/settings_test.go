package tui_test

import (
	"testing"

	"github.com/jackuait/ghost-tab/internal/tui"
)

func TestSettingsMenuItems(t *testing.T) {
	items := tui.GetSettingsMenuItems()

	expectedCount := 5 // add-project, delete-project, select-ai-tool, manage-features, quit

	if len(items) != expectedCount {
		t.Errorf("Expected %d menu items, got %d", expectedCount, len(items))
	}

	// Check first item
	if items[0].Action != "add-project" {
		t.Errorf("Expected first action 'add-project', got %q", items[0].Action)
	}
}
