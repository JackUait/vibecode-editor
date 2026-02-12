package tui_test

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jackuait/ghost-tab/internal/tui"
)

func TestSettingsMenuItems(t *testing.T) {
	items := tui.GetSettingsMenuItems()

	expectedCount := 4 // add-project, delete-project, select-ai-tool, quit

	if len(items) != expectedCount {
		t.Errorf("Expected %d menu items, got %d", expectedCount, len(items))
	}

	// Check first item
	if items[0].Action != "add-project" {
		t.Errorf("Expected first action 'add-project', got %q", items[0].Action)
	}
}

func TestSettingsMenu_New(t *testing.T) {
	m := tui.NewSettingsMenu()
	if m.Selected() != nil {
		t.Error("Selected should be nil initially")
	}
}

func TestSettingsMenu_InitReturnsNil(t *testing.T) {
	m := tui.NewSettingsMenu()
	if m.Init() != nil {
		t.Error("Init should return nil")
	}
}

func TestSettingsMenu_EnterSelectsItem(t *testing.T) {
	m := tui.NewSettingsMenu()
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Error("Enter should return quit command")
	}
	result := updated.(tui.SettingsMenuModel)
	if result.Selected() == nil {
		t.Fatal("Enter should select current item")
	}
	if result.Selected().Action != "add-project" {
		t.Errorf("Expected first item 'add-project', got %q", result.Selected().Action)
	}
}

func TestSettingsMenu_EscSelectsQuit(t *testing.T) {
	m := tui.NewSettingsMenu()
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd == nil {
		t.Error("Esc should return quit command")
	}
	result := updated.(tui.SettingsMenuModel)
	if result.Selected() == nil {
		t.Fatal("Esc should set selected to quit action")
	}
	if result.Selected().Action != "quit" {
		t.Errorf("Expected quit action, got %q", result.Selected().Action)
	}
}

func TestSettingsMenu_CtrlCSelectsQuit(t *testing.T) {
	m := tui.NewSettingsMenu()
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if cmd == nil {
		t.Error("Ctrl+C should return quit command")
	}
	result := updated.(tui.SettingsMenuModel)
	if result.Selected() == nil {
		t.Fatal("Ctrl+C should set selected to quit action")
	}
	if result.Selected().Action != "quit" {
		t.Errorf("Expected quit action, got %q", result.Selected().Action)
	}
}

func TestSettingsMenu_WindowSizeMsg(t *testing.T) {
	m := tui.NewSettingsMenu()
	updated, cmd := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	if cmd != nil {
		t.Error("WindowSizeMsg should return nil cmd")
	}
	_ = updated
}

func TestSettingsMenu_ViewNonEmpty(t *testing.T) {
	m := tui.NewSettingsMenu()
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	view := updated.(tui.SettingsMenuModel).View()
	if view == "" {
		t.Error("View should not be empty before quitting")
	}
}

func TestSettingsMenu_ViewEmptyAfterQuit(t *testing.T) {
	m := tui.NewSettingsMenu()
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	result := updated.(tui.SettingsMenuModel)
	if result.View() != "" {
		t.Error("View should be empty after quitting")
	}
}
