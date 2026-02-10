package tui_test

import (
	"testing"

	"github.com/jackuait/ghost-tab/internal/tui"
)

func TestConfirmModel(t *testing.T) {
	model := tui.NewConfirmDialog("Delete project?")

	if model.Message != "Delete project?" {
		t.Errorf("Expected message 'Delete project?', got %q", model.Message)
	}

	if model.Confirmed {
		t.Error("Expected confirmed to be false initially")
	}
}
