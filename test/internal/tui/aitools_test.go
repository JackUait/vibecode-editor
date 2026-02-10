package tui_test

import (
	"testing"

	"github.com/jackuait/ghost-tab/internal/models"
	"github.com/jackuait/ghost-tab/internal/tui"
)

func TestNewAIToolSelector(t *testing.T) {
	tools := []models.AITool{
		{Name: "claude", Command: "claude", Installed: true},
		{Name: "codex", Command: "codex", Installed: false},
	}

	model := tui.NewAIToolSelector(tools)

	if model.Selected() != nil {
		t.Error("Expected selected to be nil initially")
	}
}

func TestAIToolSelectorSelected(t *testing.T) {
	tools := []models.AITool{
		{Name: "claude", Command: "claude", Installed: true},
	}

	model := tui.NewAIToolSelector(tools)

	// Simulate selection by accessing through the model
	selected := model.Selected()

	if selected != nil {
		t.Error("Expected selected to be nil before any selection")
	}
}
