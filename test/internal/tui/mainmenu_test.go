package tui_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jackuait/ghost-tab/internal/models"
	"github.com/jackuait/ghost-tab/internal/tui"
	"github.com/muesli/termenv"
)

func testProjects() []models.Project {
	return []models.Project{
		{Name: "ghost-tab", Path: "/Users/jack/ghost-tab"},
		{Name: "my-app", Path: "/Users/jack/my-app"},
		{Name: "website", Path: "/Users/jack/website"},
	}
}

func testAITools() []string {
	return []string{"claude", "codex", "copilot", "opencode"}
}

func TestMainMenu_Navigation(t *testing.T) {
	projects := testProjects()
	m := tui.NewMainMenu(projects, testAITools(), "claude", "animated")

	// Initial selection is 0
	if m.SelectedItem() != 0 {
		t.Errorf("Initial SelectedItem: expected 0, got %d", m.SelectedItem())
	}

	// MoveDown increments
	m.MoveDown()
	if m.SelectedItem() != 1 {
		t.Errorf("After MoveDown: expected 1, got %d", m.SelectedItem())
	}

	// MoveDown again
	m.MoveDown()
	if m.SelectedItem() != 2 {
		t.Errorf("After second MoveDown: expected 2, got %d", m.SelectedItem())
	}

	// MoveUp decrements
	m.MoveUp()
	if m.SelectedItem() != 1 {
		t.Errorf("After MoveUp: expected 1, got %d", m.SelectedItem())
	}

	// MoveUp back to 0
	m.MoveUp()
	if m.SelectedItem() != 0 {
		t.Errorf("After second MoveUp: expected 0, got %d", m.SelectedItem())
	}

	// MoveUp wraps to last item
	m.MoveUp()
	expected := m.TotalItems() - 1
	if m.SelectedItem() != expected {
		t.Errorf("Wrap up: expected %d, got %d", expected, m.SelectedItem())
	}

	// MoveDown wraps to first item
	m.MoveDown()
	if m.SelectedItem() != 0 {
		t.Errorf("Wrap down: expected 0, got %d", m.SelectedItem())
	}
}

func TestMainMenu_TotalItems(t *testing.T) {
	// 3 projects + 4 actions = 7
	projects := testProjects()
	m := tui.NewMainMenu(projects, testAITools(), "claude", "animated")
	if m.TotalItems() != 7 {
		t.Errorf("TotalItems with 3 projects: expected 7, got %d", m.TotalItems())
	}

	// 0 projects + 4 actions = 4
	m2 := tui.NewMainMenu(nil, testAITools(), "claude", "animated")
	if m2.TotalItems() != 4 {
		t.Errorf("TotalItems with 0 projects: expected 4, got %d", m2.TotalItems())
	}

	// 1 project + 4 actions = 5
	m3 := tui.NewMainMenu([]models.Project{{Name: "solo", Path: "/solo"}}, testAITools(), "claude", "animated")
	if m3.TotalItems() != 5 {
		t.Errorf("TotalItems with 1 project: expected 5, got %d", m3.TotalItems())
	}
}

func TestMainMenu_AIToolCycling(t *testing.T) {
	tools := testAITools()
	m := tui.NewMainMenu(testProjects(), tools, "claude", "animated")

	// Initial AI tool is claude
	if m.CurrentAITool() != "claude" {
		t.Errorf("Initial AI tool: expected 'claude', got %q", m.CurrentAITool())
	}

	// Cycle next: claude -> codex
	m.CycleAITool("next")
	if m.CurrentAITool() != "codex" {
		t.Errorf("After next: expected 'codex', got %q", m.CurrentAITool())
	}

	// Cycle next: codex -> copilot
	m.CycleAITool("next")
	if m.CurrentAITool() != "copilot" {
		t.Errorf("After next: expected 'copilot', got %q", m.CurrentAITool())
	}

	// Cycle next: copilot -> opencode
	m.CycleAITool("next")
	if m.CurrentAITool() != "opencode" {
		t.Errorf("After next: expected 'opencode', got %q", m.CurrentAITool())
	}

	// Cycle next: opencode wraps to claude
	m.CycleAITool("next")
	if m.CurrentAITool() != "claude" {
		t.Errorf("After next wrap: expected 'claude', got %q", m.CurrentAITool())
	}

	// Cycle prev: claude wraps to opencode
	m.CycleAITool("prev")
	if m.CurrentAITool() != "opencode" {
		t.Errorf("After prev wrap: expected 'opencode', got %q", m.CurrentAITool())
	}

	// Cycle prev: opencode -> copilot
	m.CycleAITool("prev")
	if m.CurrentAITool() != "copilot" {
		t.Errorf("After prev: expected 'copilot', got %q", m.CurrentAITool())
	}
}

func TestMainMenu_CycleAITool_PersistsToFile(t *testing.T) {
	dir := t.TempDir()
	aiToolFile := filepath.Join(dir, "config", "ghost-tab", "ai-tool")

	m := tui.NewMainMenu(testProjects(), testAITools(), "claude", "animated")
	m.SetAIToolFile(aiToolFile)

	// Cycle to codex
	m.CycleAITool("next")

	// File should be written with "codex"
	data, err := os.ReadFile(aiToolFile)
	if err != nil {
		t.Fatalf("ai-tool file not found after cycle: %v", err)
	}
	if strings.TrimSpace(string(data)) != "codex" {
		t.Errorf("ai-tool file should be 'codex', got %q", strings.TrimSpace(string(data)))
	}

	// Cycle again to copilot
	m.CycleAITool("next")
	data, _ = os.ReadFile(aiToolFile)
	if strings.TrimSpace(string(data)) != "copilot" {
		t.Errorf("ai-tool file should be 'copilot' after second cycle, got %q", strings.TrimSpace(string(data)))
	}
}

func TestMainMenu_CycleAITool_DoesNotPersistWithoutFile(t *testing.T) {
	dir := t.TempDir()
	aiToolFile := filepath.Join(dir, "config", "ghost-tab", "ai-tool")

	m := tui.NewMainMenu(testProjects(), testAITools(), "claude", "animated")
	// Do NOT call SetAIToolFile

	m.CycleAITool("next")

	// File should NOT exist
	if _, err := os.Stat(aiToolFile); err == nil {
		t.Error("ai-tool file should not be created when no file path set")
	}
}

func TestMainMenu_AIToolCycling_SingleTool(t *testing.T) {
	m := tui.NewMainMenu(testProjects(), []string{"claude"}, "claude", "animated")

	// With single tool, cycling should stay on same tool
	m.CycleAITool("next")
	if m.CurrentAITool() != "claude" {
		t.Errorf("Single tool next: expected 'claude', got %q", m.CurrentAITool())
	}

	m.CycleAITool("prev")
	if m.CurrentAITool() != "claude" {
		t.Errorf("Single tool prev: expected 'claude', got %q", m.CurrentAITool())
	}
}

func TestMainMenu_AIToolCycling_StartNonFirst(t *testing.T) {
	m := tui.NewMainMenu(testProjects(), testAITools(), "copilot", "animated")

	if m.CurrentAITool() != "copilot" {
		t.Errorf("Initial: expected 'copilot', got %q", m.CurrentAITool())
	}

	m.CycleAITool("next")
	if m.CurrentAITool() != "opencode" {
		t.Errorf("After next from copilot: expected 'opencode', got %q", m.CurrentAITool())
	}
}

func TestMainMenu_AIToolCycling_UnknownTool(t *testing.T) {
	// Unknown current tool should default to index 0
	m := tui.NewMainMenu(testProjects(), testAITools(), "unknown", "animated")

	if m.CurrentAITool() != "claude" {
		t.Errorf("Unknown tool should default to first: expected 'claude', got %q", m.CurrentAITool())
	}
}

func TestMainMenu_LayoutCalculation(t *testing.T) {
	projects := testProjects()
	m := tui.NewMainMenu(projects, testAITools(), "claude", "animated")

	// Side layout: width >= 82 (48 + 3 + 28 + 3)
	layout := m.CalculateLayout(100, 40)
	if layout.GhostPosition != "side" {
		t.Errorf("Side layout at 100x40: expected 'side', got %q", layout.GhostPosition)
	}
	if layout.MenuWidth != 48 {
		t.Errorf("MenuWidth: expected 48, got %d", layout.MenuWidth)
	}

	// Above layout: width < 82 but height sufficient
	// MenuHeight = 7 + (7 * 2) + 1 = 22 (7 items, 1 separator)
	// Need height >= 22 + 15 + 2 = 39
	layout = m.CalculateLayout(60, 45)
	if layout.GhostPosition != "above" {
		t.Errorf("Above layout at 60x45: expected 'above', got %q", layout.GhostPosition)
	}

	// Hidden layout: neither condition met
	layout = m.CalculateLayout(40, 20)
	if layout.GhostPosition != "hidden" {
		t.Errorf("Hidden layout at 40x20: expected 'hidden', got %q", layout.GhostPosition)
	}

	// Exact boundary for side layout
	layout = m.CalculateLayout(82, 40)
	if layout.GhostPosition != "side" {
		t.Errorf("Exact side boundary 82x40: expected 'side', got %q", layout.GhostPosition)
	}

	// Just below side boundary
	layout = m.CalculateLayout(81, 45)
	if layout.GhostPosition != "above" {
		t.Errorf("Below side boundary 81x45: expected 'above', got %q", layout.GhostPosition)
	}
}

func TestMainMenu_LayoutCalculation_MenuHeight(t *testing.T) {
	// 3 projects + 4 actions = 7 items, 1 separator
	projects := testProjects()
	m := tui.NewMainMenu(projects, testAITools(), "claude", "animated")
	layout := m.CalculateLayout(100, 40)

	// MenuHeight = 7 + (7 * 2) + 1 = 22
	expectedHeight := 7 + (7 * 2) + 1
	if layout.MenuHeight != expectedHeight {
		t.Errorf("MenuHeight with 3 projects: expected %d, got %d", expectedHeight, layout.MenuHeight)
	}

	// 0 projects + 4 actions = 4 items, 0 separators (no projects means no separator)
	m2 := tui.NewMainMenu(nil, testAITools(), "claude", "animated")
	layout2 := m2.CalculateLayout(100, 40)
	expectedHeight2 := 7 + (4 * 2) + 0
	if layout2.MenuHeight != expectedHeight2 {
		t.Errorf("MenuHeight with 0 projects: expected %d, got %d", expectedHeight2, layout2.MenuHeight)
	}
}

func TestMainMenu_JumpTo(t *testing.T) {
	projects := testProjects()
	m := tui.NewMainMenu(projects, testAITools(), "claude", "animated")

	// JumpTo 1 (first project, 1-indexed)
	m.JumpTo(1)
	if m.SelectedItem() != 0 {
		t.Errorf("JumpTo(1): expected 0, got %d", m.SelectedItem())
	}

	// JumpTo 2 (second project)
	m.JumpTo(2)
	if m.SelectedItem() != 1 {
		t.Errorf("JumpTo(2): expected 1, got %d", m.SelectedItem())
	}

	// JumpTo 3 (third project)
	m.JumpTo(3)
	if m.SelectedItem() != 2 {
		t.Errorf("JumpTo(3): expected 2, got %d", m.SelectedItem())
	}

	// JumpTo beyond project count should not change selection
	m.JumpTo(4)
	if m.SelectedItem() != 2 {
		t.Errorf("JumpTo(4) beyond projects: expected 2 (unchanged), got %d", m.SelectedItem())
	}

	// JumpTo 0 should not change selection
	m.JumpTo(0)
	if m.SelectedItem() != 2 {
		t.Errorf("JumpTo(0): expected 2 (unchanged), got %d", m.SelectedItem())
	}
}

func TestMainMenu_SelectProject(t *testing.T) {
	projects := testProjects()
	m := tui.NewMainMenu(projects, testAITools(), "claude", "animated")

	// Select first project (index 0)
	newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	mm := newModel.(*tui.MainMenuModel)
	result := mm.Result()

	if result == nil {
		t.Fatal("Expected result after Enter on project, got nil")
	}
	if result.Action != "select-project" {
		t.Errorf("Expected action 'select-project', got %q", result.Action)
	}
	if result.Name != "ghost-tab" {
		t.Errorf("Expected name 'ghost-tab', got %q", result.Name)
	}
	if result.Path != "/Users/jack/ghost-tab" {
		t.Errorf("Expected path '/Users/jack/ghost-tab', got %q", result.Path)
	}
	if result.AITool != "claude" {
		t.Errorf("Expected ai_tool 'claude', got %q", result.AITool)
	}

	// cmd should be tea.Quit
	if cmd == nil {
		t.Error("Expected tea.Quit cmd, got nil")
	}

	// Select second project
	m2 := tui.NewMainMenu(projects, testAITools(), "codex", "animated")
	m2.MoveDown()
	newModel2, _ := m2.Update(tea.KeyMsg{Type: tea.KeyEnter})
	mm2 := newModel2.(*tui.MainMenuModel)
	result2 := mm2.Result()

	if result2 == nil {
		t.Fatal("Expected result after Enter on second project, got nil")
	}
	if result2.Name != "my-app" {
		t.Errorf("Expected name 'my-app', got %q", result2.Name)
	}
	if result2.AITool != "codex" {
		t.Errorf("Expected ai_tool 'codex', got %q", result2.AITool)
	}
}

func TestMainMenu_SelectAction(t *testing.T) {
	projects := testProjects()
	// Actions start at index 3 (after 3 projects)
	// Index 3 = Add, 4 = Delete, 5 = Open Once, 6 = Plain Terminal

	t.Run("add-project", func(t *testing.T) {
		m := tui.NewMainMenu(projects, testAITools(), "claude", "animated")
		// Navigate to add action (index 3)
		for i := 0; i < 3; i++ {
			m.MoveDown()
		}
		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
		mm := newModel.(*tui.MainMenuModel)

		if !mm.InInputMode() {
			t.Error("Expected input mode after Enter on add-project")
		}
		if mm.InputMode() != "add-project" {
			t.Errorf("Expected input mode 'add-project', got %q", mm.InputMode())
		}
		if mm.Result() != nil {
			t.Error("Should not produce result when entering input mode")
		}
	})

	t.Run("delete-project", func(t *testing.T) {
		m := tui.NewMainMenu(projects, testAITools(), "claude", "animated")
		for i := 0; i < 4; i++ {
			m.MoveDown()
		}
		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
		mm := newModel.(*tui.MainMenuModel)

		if !mm.InDeleteMode() {
			t.Error("Expected delete mode after Enter on delete-project")
		}
		if mm.Result() != nil {
			t.Error("Should not produce result when entering delete mode")
		}
	})

	t.Run("open-once", func(t *testing.T) {
		m := tui.NewMainMenu(projects, testAITools(), "claude", "animated")
		for i := 0; i < 5; i++ {
			m.MoveDown()
		}
		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
		mm := newModel.(*tui.MainMenuModel)

		if !mm.InInputMode() {
			t.Error("Expected input mode after Enter on open-once")
		}
		if mm.InputMode() != "open-once" {
			t.Errorf("Expected input mode 'open-once', got %q", mm.InputMode())
		}
		if mm.Result() != nil {
			t.Error("Should not produce result when entering input mode")
		}
	})

	t.Run("plain-terminal", func(t *testing.T) {
		m := tui.NewMainMenu(projects, testAITools(), "claude", "animated")
		for i := 0; i < 6; i++ {
			m.MoveDown()
		}
		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
		mm := newModel.(*tui.MainMenuModel)
		result := mm.Result()

		if result == nil {
			t.Fatal("Expected result, got nil")
		}
		if result.Action != "plain-terminal" {
			t.Errorf("Expected action 'plain-terminal', got %q", result.Action)
		}
	})
}

func TestMainMenu_ActionShortcuts(t *testing.T) {
	projects := testProjects()

	t.Run("a_shortcut", func(t *testing.T) {
		m := tui.NewMainMenu(projects, testAITools(), "claude", "animated")
		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
		mm := newModel.(*tui.MainMenuModel)

		if !mm.InInputMode() {
			t.Error("Expected input mode after 'a' shortcut")
		}
		if mm.InputMode() != "add-project" {
			t.Errorf("Expected input mode 'add-project', got %q", mm.InputMode())
		}
		if mm.Result() != nil {
			t.Error("Should not produce result when entering input mode")
		}
	})

	t.Run("A_shortcut", func(t *testing.T) {
		m := tui.NewMainMenu(projects, testAITools(), "claude", "animated")
		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'A'}})
		mm := newModel.(*tui.MainMenuModel)

		if !mm.InInputMode() {
			t.Error("Expected input mode after 'A' shortcut")
		}
		if mm.InputMode() != "add-project" {
			t.Errorf("Expected input mode 'add-project', got %q", mm.InputMode())
		}
		if mm.Result() != nil {
			t.Error("Should not produce result when entering input mode")
		}
	})

	t.Run("d_shortcut", func(t *testing.T) {
		m := tui.NewMainMenu(projects, testAITools(), "claude", "animated")
		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
		mm := newModel.(*tui.MainMenuModel)

		if !mm.InDeleteMode() {
			t.Error("Expected delete mode after 'd' shortcut")
		}
		if mm.Result() != nil {
			t.Error("Should not produce result when entering delete mode")
		}
	})

	t.Run("o_shortcut", func(t *testing.T) {
		m := tui.NewMainMenu(projects, testAITools(), "claude", "animated")
		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}})
		mm := newModel.(*tui.MainMenuModel)

		if !mm.InInputMode() {
			t.Error("Expected input mode after 'o' shortcut")
		}
		if mm.InputMode() != "open-once" {
			t.Errorf("Expected input mode 'open-once', got %q", mm.InputMode())
		}
		if mm.Result() != nil {
			t.Error("Should not produce result when entering input mode")
		}
	})

	t.Run("p_shortcut", func(t *testing.T) {
		m := tui.NewMainMenu(projects, testAITools(), "claude", "animated")
		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
		mm := newModel.(*tui.MainMenuModel)
		result := mm.Result()

		if result == nil {
			t.Fatal("Expected result for 'p' shortcut, got nil")
		}
		if result.Action != "plain-terminal" {
			t.Errorf("Expected 'plain-terminal', got %q", result.Action)
		}
	})

	t.Run("s_shortcut_enters_settings", func(t *testing.T) {
		m := tui.NewMainMenu(projects, testAITools(), "claude", "animated")
		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
		mm := newModel.(*tui.MainMenuModel)

		if !mm.InSettingsMode() {
			t.Error("Expected settings mode after 's' shortcut")
		}
		if mm.Result() != nil {
			t.Error("Should not produce a result when entering settings mode")
		}
	})
}

func TestMainMenu_QuitEsc(t *testing.T) {
	m := tui.NewMainMenu(testProjects(), testAITools(), "claude", "animated")
	newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	mm := newModel.(*tui.MainMenuModel)
	result := mm.Result()

	if result == nil {
		t.Fatal("Expected result for Esc, got nil")
	}
	if result.Action != "quit" {
		t.Errorf("Expected 'quit', got %q", result.Action)
	}
	if cmd == nil {
		t.Error("Expected tea.Quit cmd, got nil")
	}
}

func TestMainMenu_QuitEsc_IncludesCycledAITool(t *testing.T) {
	m := tui.NewMainMenu(testProjects(), testAITools(), "claude", "animated")
	// Cycle to codex
	m.CycleAITool("next")
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	mm := newModel.(*tui.MainMenuModel)
	result := mm.Result()

	if result == nil {
		t.Fatal("Expected result for Esc, got nil")
	}
	if result.AITool != "codex" {
		t.Errorf("Expected ai_tool 'codex' after cycling, got %q", result.AITool)
	}
}

func TestMainMenu_QuitCtrlC(t *testing.T) {
	m := tui.NewMainMenu(testProjects(), testAITools(), "claude", "animated")
	newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	mm := newModel.(*tui.MainMenuModel)
	result := mm.Result()

	if result == nil {
		t.Fatal("Expected result for Ctrl+C, got nil")
	}
	if result.Action != "quit" {
		t.Errorf("Expected 'quit', got %q", result.Action)
	}
	if cmd == nil {
		t.Error("Expected tea.Quit cmd, got nil")
	}
}

func TestMainMenu_KeyBindings_Navigation(t *testing.T) {
	projects := testProjects()

	t.Run("j_moves_down", func(t *testing.T) {
		m := tui.NewMainMenu(projects, testAITools(), "claude", "animated")
		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
		mm := newModel.(*tui.MainMenuModel)
		if mm.SelectedItem() != 1 {
			t.Errorf("After 'j': expected 1, got %d", mm.SelectedItem())
		}
	})

	t.Run("k_moves_up", func(t *testing.T) {
		m := tui.NewMainMenu(projects, testAITools(), "claude", "animated")
		m.MoveDown()
		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
		mm := newModel.(*tui.MainMenuModel)
		if mm.SelectedItem() != 0 {
			t.Errorf("After 'k': expected 0, got %d", mm.SelectedItem())
		}
	})

	t.Run("arrow_down", func(t *testing.T) {
		m := tui.NewMainMenu(projects, testAITools(), "claude", "animated")
		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
		mm := newModel.(*tui.MainMenuModel)
		if mm.SelectedItem() != 1 {
			t.Errorf("After down arrow: expected 1, got %d", mm.SelectedItem())
		}
	})

	t.Run("arrow_up", func(t *testing.T) {
		m := tui.NewMainMenu(projects, testAITools(), "claude", "animated")
		m.MoveDown()
		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyUp})
		mm := newModel.(*tui.MainMenuModel)
		if mm.SelectedItem() != 0 {
			t.Errorf("After up arrow: expected 0, got %d", mm.SelectedItem())
		}
	})
}

func TestMainMenu_KeyBindings_AIToolCycling(t *testing.T) {
	t.Run("right_arrow_cycles_next", func(t *testing.T) {
		m := tui.NewMainMenu(testProjects(), testAITools(), "claude", "animated")
		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRight})
		mm := newModel.(*tui.MainMenuModel)
		if mm.CurrentAITool() != "codex" {
			t.Errorf("After right arrow: expected 'codex', got %q", mm.CurrentAITool())
		}
	})

	t.Run("left_arrow_cycles_prev", func(t *testing.T) {
		m := tui.NewMainMenu(testProjects(), testAITools(), "claude", "animated")
		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyLeft})
		mm := newModel.(*tui.MainMenuModel)
		if mm.CurrentAITool() != "opencode" {
			t.Errorf("After left arrow: expected 'opencode', got %q", mm.CurrentAITool())
		}
	})
}

func TestMainMenu_KeyBindings_NumberJump(t *testing.T) {
	projects := testProjects()
	m := tui.NewMainMenu(projects, testAITools(), "claude", "animated")

	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})
	mm := newModel.(*tui.MainMenuModel)
	if mm.SelectedItem() != 1 {
		t.Errorf("After '2' key: expected 1, got %d", mm.SelectedItem())
	}

	// Number beyond project count should not change selection
	newModel2, _ := mm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'9'}})
	mm2 := newModel2.(*tui.MainMenuModel)
	if mm2.SelectedItem() != 1 {
		t.Errorf("After '9' key (beyond projects): expected 1 (unchanged), got %d", mm2.SelectedItem())
	}
}

func TestMainMenu_GhostDisplay(t *testing.T) {
	tests := []struct {
		display  string
		expected string
	}{
		{"animated", "animated"},
		{"static", "static"},
		{"none", "none"},
	}

	for _, tt := range tests {
		t.Run(tt.display, func(t *testing.T) {
			m := tui.NewMainMenu(testProjects(), testAITools(), "claude", tt.display)
			if m.GhostDisplay() != tt.expected {
				t.Errorf("GhostDisplay: expected %q, got %q", tt.expected, m.GhostDisplay())
			}
		})
	}
}

func TestMainMenu_SetSize(t *testing.T) {
	m := tui.NewMainMenu(testProjects(), testAITools(), "claude", "animated")
	m.SetSize(120, 50)

	// After SetSize, layout calculation should use the set dimensions
	layout := m.CalculateLayout(120, 50)
	if layout.GhostPosition != "side" {
		t.Errorf("After SetSize(120, 50): expected 'side', got %q", layout.GhostPosition)
	}
}

func TestMainMenu_Init(t *testing.T) {
	// Static mode: Init returns nil (no ticks)
	m := tui.NewMainMenu(testProjects(), testAITools(), "claude", "static")
	cmd := m.Init()
	if cmd != nil {
		t.Error("Init() should return nil for static mode")
	}
}

func TestMainMenu_View(t *testing.T) {
	m := tui.NewMainMenu(testProjects(), testAITools(), "claude", "animated")
	view := m.View()
	if view == "" {
		t.Error("View() should return non-empty placeholder string")
	}
}

func TestMainMenu_WindowSizeMsg(t *testing.T) {
	m := tui.NewMainMenu(testProjects(), testAITools(), "claude", "animated")
	newModel, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 50})
	mm := newModel.(*tui.MainMenuModel)

	// After receiving WindowSizeMsg, the layout should reflect new dimensions
	layout := mm.CalculateLayout(100, 50)
	if layout.GhostPosition != "side" {
		t.Errorf("After WindowSizeMsg: expected 'side', got %q", layout.GhostPosition)
	}
}

func TestMainMenu_NoProjects_ActionIndices(t *testing.T) {
	// With 0 projects, actions start at index 0
	m := tui.NewMainMenu(nil, testAITools(), "claude", "animated")

	// Index 0 = Add (now enters input mode instead of quitting with result)
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	mm := newModel.(*tui.MainMenuModel)

	if !mm.InInputMode() {
		t.Error("Expected input mode after Enter on add-project with no projects")
	}
	if mm.InputMode() != "add-project" {
		t.Errorf("Expected input mode 'add-project', got %q", mm.InputMode())
	}
	if mm.Result() != nil {
		t.Error("Should not produce result when entering input mode")
	}
}

func TestMainMenu_ViewContainsBorders(t *testing.T) {
	projects := []models.Project{{Name: "test", Path: "/test"}}
	m := tui.NewMainMenu(projects, []string{"claude"}, "claude", "animated")
	m.SetSize(80, 30)
	view := m.View()
	// Should contain box-drawing borders
	if !strings.Contains(view, "\u250c") || !strings.Contains(view, "\u2518") {
		t.Error("view should contain box-drawing borders")
	}
	if !strings.Contains(view, "Ghost Tab") {
		t.Error("view should contain 'Ghost Tab' title")
	}
	if !strings.Contains(view, "test") {
		t.Error("view should contain project name")
	}
}

func TestMainMenu_ViewShowsAIToolWithArrows(t *testing.T) {
	projects := []models.Project{}
	tools := []string{"claude", "codex"}
	m := tui.NewMainMenu(projects, tools, "claude", "animated")
	m.SetSize(80, 30)
	view := m.View()
	if !strings.Contains(view, "\u25c2") || !strings.Contains(view, "\u25b8") {
		t.Error("view should show AI tool cycling arrows when multiple tools")
	}
	if !strings.Contains(view, "Claude Code") {
		t.Error("view should show AI tool display name")
	}
}

func TestMainMenu_ViewNoArrowsSingleTool(t *testing.T) {
	m := tui.NewMainMenu(nil, []string{"claude"}, "claude", "animated")
	m.SetSize(80, 30)
	view := m.View()
	if strings.Contains(view, "\u25c2") || strings.Contains(view, "\u25b8") {
		t.Error("should not show cycling arrows with single tool")
	}
}

func TestMainMenu_ViewHelpRow(t *testing.T) {
	m := tui.NewMainMenu(nil, []string{"claude", "codex"}, "claude", "animated")
	m.SetSize(80, 30)
	view := m.View()
	if !strings.Contains(view, "navigate") {
		t.Error("help row should mention navigate")
	}
	if !strings.Contains(view, "AI tool") {
		t.Error("help row should mention AI tool when multiple available")
	}
}

func TestMainMenu_ViewActionItems(t *testing.T) {
	m := tui.NewMainMenu(nil, []string{"claude"}, "claude", "animated")
	m.SetSize(80, 30)
	view := m.View()
	if !strings.Contains(view, "Add") {
		t.Error("view should contain Add action")
	}
	if !strings.Contains(view, "Delete") {
		t.Error("view should contain Delete action")
	}
}

func TestMainMenu_ViewSelectedMarker(t *testing.T) {
	projects := []models.Project{{Name: "p1", Path: "/p1"}, {Name: "p2", Path: "/p2"}}
	m := tui.NewMainMenu(projects, []string{"claude"}, "claude", "animated")
	m.SetSize(80, 30)
	view := m.View()
	// Selected item (first) should have marker
	if !strings.Contains(view, "\u258e") {
		t.Error("view should contain selection marker \u258e")
	}
}

func TestMainMenu_ViewWithGhostSide(t *testing.T) {
	projects := []models.Project{{Name: "test", Path: "/test"}}
	m := tui.NewMainMenu(projects, []string{"claude"}, "claude", "animated")
	m.SetSize(100, 40) // Wide enough for side layout
	view := m.View()
	// Ghost art uses block characters
	if !strings.Contains(view, "\u2588") {
		t.Error("view should contain ghost art block characters in side layout")
	}
}

func TestMainMenu_ViewGhostHiddenWhenNone(t *testing.T) {
	m := tui.NewMainMenu(nil, []string{"claude"}, "claude", "none")
	m.SetSize(100, 40)
	view := m.View()
	// Ghost should not appear -- but menu should still render
	if !strings.Contains(view, "Ghost Tab") {
		t.Error("menu should still render when ghost is hidden")
	}
}

func TestMainMenu_AIToolDisplayName(t *testing.T) {
	tests := []struct {
		tool     string
		expected string
	}{
		{"claude", "Claude Code"},
		{"codex", "Codex CLI"},
		{"copilot", "Copilot CLI"},
		{"opencode", "OpenCode"},
		{"unknown", "unknown"},
	}
	for _, tt := range tests {
		t.Run(tt.tool, func(t *testing.T) {
			got := tui.AIToolDisplayName(tt.tool)
			if got != tt.expected {
				t.Errorf("AIToolDisplayName(%q): expected %q, got %q", tt.tool, tt.expected, got)
			}
		})
	}
}

func TestMainMenu_ViewHelpRowSingleTool(t *testing.T) {
	m := tui.NewMainMenu(nil, []string{"claude"}, "claude", "animated")
	m.SetSize(80, 30)
	view := m.View()
	if !strings.Contains(view, "navigate") {
		t.Error("help row should mention navigate")
	}
	// Single tool: no AI tool mention in help
	if strings.Contains(view, "AI tool") {
		t.Error("help row should NOT mention AI tool when single tool")
	}
}

func TestMainMenu_ViewUpdateVersion(t *testing.T) {
	m := tui.NewMainMenu(nil, []string{"claude"}, "claude", "animated")
	m.SetUpdateVersion("v1.2.3")
	m.SetSize(80, 30)
	view := m.View()
	if !strings.Contains(view, "v1.2.3") {
		t.Error("view should show update version when set")
	}
	if !strings.Contains(view, "Update available") {
		t.Error("view should show 'Update available' message")
	}
}

func TestMainMenu_ViewNoUpdateVersion(t *testing.T) {
	m := tui.NewMainMenu(nil, []string{"claude"}, "claude", "animated")
	m.SetSize(80, 30)
	view := m.View()
	if strings.Contains(view, "Update available") {
		t.Error("view should NOT show update message when version is empty")
	}
}

func TestMainMenu_ViewGhostAbove(t *testing.T) {
	projects := []models.Project{{Name: "test", Path: "/test"}}
	m := tui.NewMainMenu(projects, []string{"claude"}, "claude", "animated")
	// Width too narrow for side (< 82), but height enough for above
	m.SetSize(60, 50)
	view := m.View()
	// Should still contain ghost block characters
	if !strings.Contains(view, "\u2588") {
		t.Error("view should contain ghost art block characters in above layout")
	}
	if !strings.Contains(view, "Ghost Tab") {
		t.Error("view should contain menu title in above layout")
	}
}

func TestMainMenu_BobOffset_InitiallyZero(t *testing.T) {
	m := tui.NewMainMenu(nil, []string{"claude"}, "claude", "animated")
	if m.BobOffset() != 0 {
		t.Errorf("initial bob offset should be 0, got %d", m.BobOffset())
	}
}

func TestMainMenu_BobPhase_AdvancesOnTick(t *testing.T) {
	m := tui.NewMainMenu(nil, []string{"claude"}, "claude", "animated")
	initial := m.BobPhase()
	m.Update(tui.NewBobTickMsg())
	if m.BobPhase() <= initial {
		t.Errorf("bob phase should advance on tick, was %f now %f", initial, m.BobPhase())
	}
}

func TestMainMenu_BobOffset_OnlyZeroOrOne(t *testing.T) {
	m := tui.NewMainMenu(nil, []string{"claude"}, "claude", "animated")
	// Run through many ticks covering a full cycle
	for i := 0; i < 200; i++ {
		m.Update(tui.NewBobTickMsg())
		offset := m.BobOffset()
		if offset != 0 && offset != 1 {
			t.Fatalf("bob offset must be 0 or 1, got %d at tick %d", offset, i)
		}
	}
}

func TestMainMenu_BobOffset_TransitionsDuringCycle(t *testing.T) {
	m := tui.NewMainMenu(nil, []string{"claude"}, "claude", "animated")
	saw0, saw1 := false, false
	// Run through enough ticks for a full cycle (~156 ticks at 16ms for 2.5s)
	for i := 0; i < 200; i++ {
		m.Update(tui.NewBobTickMsg())
		switch m.BobOffset() {
		case 0:
			saw0 = true
		case 1:
			saw1 = true
		}
	}
	if !saw0 || !saw1 {
		t.Errorf("bob should transition between 0 and 1 during a full cycle, saw0=%v saw1=%v", saw0, saw1)
	}
}

func TestMainMenu_BobAnimation_VisibleInView(t *testing.T) {
	lipgloss.SetColorProfile(termenv.Ascii)
	projects := []models.Project{{Name: "test", Path: "/test"}}
	m := tui.NewMainMenu(projects, []string{"claude"}, "claude", "animated")
	m.SetSize(100, 40) // Side layout

	// Collect distinct full views across a full bob cycle
	views := make(map[string]bool)
	for i := 0; i < 200; i++ {
		m.Update(tui.NewBobTickMsg())
		views[m.View()] = true
	}
	// If the ghost actually bobs, we should see at least 2 distinct view outputs
	if len(views) < 2 {
		t.Error("ghost bob animation should produce visibly different views during a full cycle, but all views were identical (centering may be absorbing the movement)")
	}
}

func TestMainMenu_BobPhase_Wraps(t *testing.T) {
	m := tui.NewMainMenu(nil, []string{"claude"}, "claude", "animated")
	// Run through enough ticks for multiple full cycles
	for i := 0; i < 500; i++ {
		m.Update(tui.NewBobTickMsg())
	}
	phase := m.BobPhase()
	// Phase should have wrapped (stayed below 2*pi)
	if phase > 6.3 { // slightly above 2*pi
		t.Errorf("bob phase should wrap around 2*pi, got %f", phase)
	}
}

func TestMainMenu_SleepAfterInactivity(t *testing.T) {
	m := tui.NewMainMenu(nil, []string{"claude"}, "claude", "animated")
	m.SetSleepTimer(120)
	if !m.ShouldSleep() {
		t.Error("should sleep after 120 seconds of inactivity")
	}
}

func TestMainMenu_WakeOnKeypress(t *testing.T) {
	m := tui.NewMainMenu(nil, []string{"claude"}, "claude", "animated")
	m.SetSleepTimer(120)
	m.Wake()
	if m.IsSleeping() {
		t.Error("should be awake after Wake()")
	}
	if m.ShouldSleep() {
		t.Error("sleep timer should be reset after Wake()")
	}
}

func TestMainMenu_GhostHiddenWhenNone(t *testing.T) {
	m := tui.NewMainMenu(nil, []string{"claude"}, "claude", "none")
	m.SetSize(100, 40)
	view := m.View()
	if strings.Contains(view, "\u2588\u2588\u2588\u2588") {
		t.Error("ghost should be hidden when display mode is 'none'")
	}
}

func TestMainMenu_NoAnimationWhenStatic(t *testing.T) {
	m := tui.NewMainMenu(nil, []string{"claude"}, "claude", "static")
	cmd := m.Init()
	if cmd != nil {
		t.Error("static mode should not start animation ticks")
	}
}

func TestMainMenu_AnimationStartsWhenAnimated(t *testing.T) {
	m := tui.NewMainMenu(nil, []string{"claude"}, "claude", "animated")
	cmd := m.Init()
	if cmd == nil {
		t.Error("animated mode should start animation ticks")
	}
}

func TestMainMenu_SleepTimerResetOnKeypress(t *testing.T) {
	m := tui.NewMainMenu(nil, []string{"claude"}, "claude", "animated")
	m.SetSleepTimer(100)
	// Simulate a keypress
	msg := tea.KeyMsg{Type: tea.KeyDown}
	m.Update(msg)
	// Sleep timer should be reset
	if m.ShouldSleep() {
		t.Error("sleep timer should be reset after keypress")
	}
}

func TestMainMenu_MapRowToItem_Projects(t *testing.T) {
	projects := []models.Project{
		{Name: "p1", Path: "/p1"},
		{Name: "p2", Path: "/p2"},
	}
	m := tui.NewMainMenu(projects, []string{"claude"}, "claude", "animated")
	m.SetSize(80, 30)

	// First project at rows 4-5 (row 0: border, 1: title, 2: separator, 3: empty)
	if m.MapRowToItem(4) != 0 {
		t.Errorf("click at row 4 should map to item 0, got %d", m.MapRowToItem(4))
	}
	if m.MapRowToItem(5) != 0 {
		t.Errorf("click at row 5 should map to item 0 (path line), got %d", m.MapRowToItem(5))
	}
	// Second project at rows 6-7
	if m.MapRowToItem(6) != 1 {
		t.Errorf("click at row 6 should map to item 1, got %d", m.MapRowToItem(6))
	}
	if m.MapRowToItem(7) != 1 {
		t.Errorf("click at row 7 should map to item 1 (path line), got %d", m.MapRowToItem(7))
	}
}

func TestMainMenu_MapRowToItem_Actions(t *testing.T) {
	projects := []models.Project{
		{Name: "p1", Path: "/p1"},
	}
	m := tui.NewMainMenu(projects, []string{"claude"}, "claude", "animated")
	m.SetSize(80, 30)

	// 1 project at rows 4-5, separator at row 6, actions start at row 7
	if m.MapRowToItem(7) != 1 {
		t.Errorf("click at first action row should map to item 1 (add-project), got %d", m.MapRowToItem(7))
	}
	if m.MapRowToItem(8) != 2 {
		t.Errorf("click at second action row should map to item 2 (delete-project), got %d", m.MapRowToItem(8))
	}
	if m.MapRowToItem(9) != 3 {
		t.Errorf("click at third action row should map to item 3 (open-once), got %d", m.MapRowToItem(9))
	}
	if m.MapRowToItem(10) != 4 {
		t.Errorf("click at fourth action row should map to item 4 (plain-terminal), got %d", m.MapRowToItem(10))
	}
}

func TestMainMenu_MapRowToItem_Invalid(t *testing.T) {
	projects := []models.Project{{Name: "p1", Path: "/p1"}}
	m := tui.NewMainMenu(projects, []string{"claude"}, "claude", "animated")
	m.SetSize(80, 30)

	// Row 0 is border
	if m.MapRowToItem(0) != -1 {
		t.Errorf("click on border should return -1, got %d", m.MapRowToItem(0))
	}
	// Row 1 is title
	if m.MapRowToItem(1) != -1 {
		t.Errorf("click on title should return -1, got %d", m.MapRowToItem(1))
	}
	// Row 2 is separator
	if m.MapRowToItem(2) != -1 {
		t.Errorf("click on separator should return -1, got %d", m.MapRowToItem(2))
	}
	// Row 3 is empty
	if m.MapRowToItem(3) != -1 {
		t.Errorf("click on empty row should return -1, got %d", m.MapRowToItem(3))
	}
	// Row 6 is separator between projects and actions
	if m.MapRowToItem(6) != -1 {
		t.Errorf("click on project/action separator should return -1, got %d", m.MapRowToItem(6))
	}
	// Row way beyond menu should return -1
	if m.MapRowToItem(100) != -1 {
		t.Errorf("click beyond menu should return -1, got %d", m.MapRowToItem(100))
	}
}

func TestMainMenu_MapRowToItem_NoProjects(t *testing.T) {
	m := tui.NewMainMenu(nil, []string{"claude"}, "claude", "animated")
	m.SetSize(80, 30)

	// No projects, no separator. Actions start at row 4
	if m.MapRowToItem(4) != 0 {
		t.Errorf("first action at row 4 should map to item 0, got %d", m.MapRowToItem(4))
	}
	if m.MapRowToItem(5) != 1 {
		t.Errorf("second action at row 5 should map to item 1, got %d", m.MapRowToItem(5))
	}
}

func TestMainMenu_MapRowToItem_WithUpdateVersion(t *testing.T) {
	projects := []models.Project{{Name: "p1", Path: "/p1"}}
	m := tui.NewMainMenu(projects, []string{"claude"}, "claude", "animated")
	m.SetUpdateVersion("v1.2.3")
	m.SetSize(80, 30)

	// With update version, rows shift down by 1:
	// Row 0: border, 1: title, 2: separator, 3: update notification, 4: empty, 5-6: project
	if m.MapRowToItem(5) != 0 {
		t.Errorf("with update version, project should be at row 5, got %d", m.MapRowToItem(5))
	}
	if m.MapRowToItem(6) != 0 {
		t.Errorf("with update version, project path at row 6 should map to 0, got %d", m.MapRowToItem(6))
	}
}

func TestMainMenu_MouseClickSelectsItem(t *testing.T) {
	projects := []models.Project{
		{Name: "p1", Path: "/p1"},
		{Name: "p2", Path: "/p2"},
	}
	m := tui.NewMainMenu(projects, []string{"claude"}, "claude", "animated")
	m.SetSize(80, 30)

	// Click on second project (row 6)
	mouseMsg := tea.MouseMsg{
		X:      10,
		Y:      6,
		Action: tea.MouseActionPress,
		Button: tea.MouseButtonLeft,
	}
	newModel, _ := m.Update(mouseMsg)
	mm := newModel.(*tui.MainMenuModel)
	if mm.SelectedItem() != 1 {
		t.Errorf("clicking on second project should select item 1, got %d", mm.SelectedItem())
	}
	// Should not quit (single click on non-selected item just selects)
	if mm.Result() != nil {
		t.Error("single click on non-selected item should not produce a result")
	}
}

func TestMainMenu_MouseDoubleClickActivates(t *testing.T) {
	projects := []models.Project{
		{Name: "p1", Path: "/p1"},
		{Name: "p2", Path: "/p2"},
	}
	m := tui.NewMainMenu(projects, []string{"claude"}, "claude", "animated")
	m.SetSize(80, 30)

	// First click selects item 0 (it's already selected, so this acts as double-click)
	mouseMsg := tea.MouseMsg{
		X:      10,
		Y:      4,
		Action: tea.MouseActionPress,
		Button: tea.MouseButtonLeft,
	}
	newModel, cmd := m.Update(mouseMsg)
	mm := newModel.(*tui.MainMenuModel)
	result := mm.Result()

	if result == nil {
		t.Fatal("clicking already-selected item should produce a result (double-click)")
	}
	if result.Action != "select-project" {
		t.Errorf("expected action 'select-project', got %q", result.Action)
	}
	if result.Name != "p1" {
		t.Errorf("expected name 'p1', got %q", result.Name)
	}
	if cmd == nil {
		t.Error("expected tea.Quit cmd on activation")
	}
}

func TestMainMenu_MouseClickInvalidRow(t *testing.T) {
	projects := []models.Project{{Name: "p1", Path: "/p1"}}
	m := tui.NewMainMenu(projects, []string{"claude"}, "claude", "animated")
	m.SetSize(80, 30)

	// Click on border row 0 should not change selection
	mouseMsg := tea.MouseMsg{
		X:      10,
		Y:      0,
		Action: tea.MouseActionPress,
		Button: tea.MouseButtonLeft,
	}
	newModel, _ := m.Update(mouseMsg)
	mm := newModel.(*tui.MainMenuModel)
	if mm.SelectedItem() != 0 {
		t.Errorf("clicking on border should not change selection, got %d", mm.SelectedItem())
	}
}

func TestMainMenu_MouseClickResetsSleeTimer(t *testing.T) {
	m := tui.NewMainMenu(nil, []string{"claude"}, "claude", "animated")
	m.SetSize(80, 30)
	m.SetSleepTimer(100)

	// Click anywhere
	mouseMsg := tea.MouseMsg{
		X:      10,
		Y:      4,
		Action: tea.MouseActionPress,
		Button: tea.MouseButtonLeft,
	}
	m.Update(mouseMsg)
	if m.ShouldSleep() {
		t.Error("mouse click should reset sleep timer")
	}
}

func TestMainMenu_MouseRightClickIgnored(t *testing.T) {
	projects := []models.Project{
		{Name: "p1", Path: "/p1"},
		{Name: "p2", Path: "/p2"},
	}
	m := tui.NewMainMenu(projects, []string{"claude"}, "claude", "animated")
	m.SetSize(80, 30)

	// Right click should not change selection
	mouseMsg := tea.MouseMsg{
		X:      10,
		Y:      6,
		Action: tea.MouseActionPress,
		Button: tea.MouseButtonRight,
	}
	newModel, _ := m.Update(mouseMsg)
	mm := newModel.(*tui.MainMenuModel)
	if mm.SelectedItem() != 0 {
		t.Errorf("right click should not change selection, got %d", mm.SelectedItem())
	}
}

func TestMainMenu_MouseReleaseIgnored(t *testing.T) {
	projects := []models.Project{
		{Name: "p1", Path: "/p1"},
		{Name: "p2", Path: "/p2"},
	}
	m := tui.NewMainMenu(projects, []string{"claude"}, "claude", "animated")
	m.SetSize(80, 30)

	// Mouse release should not trigger selection
	mouseMsg := tea.MouseMsg{
		X:      10,
		Y:      6,
		Action: tea.MouseActionRelease,
		Button: tea.MouseButtonLeft,
	}
	newModel, _ := m.Update(mouseMsg)
	mm := newModel.(*tui.MainMenuModel)
	if mm.SelectedItem() != 0 {
		t.Errorf("mouse release should not change selection, got %d", mm.SelectedItem())
	}
}

func TestMainMenu_SettingsMode(t *testing.T) {
	m := tui.NewMainMenu(nil, []string{"claude"}, "claude", "animated")
	m.SetSize(80, 30)

	if m.InSettingsMode() {
		t.Error("should not start in settings mode")
	}

	m.EnterSettings()
	if !m.InSettingsMode() {
		t.Error("should be in settings mode after EnterSettings()")
	}

	m.ExitSettings()
	if m.InSettingsMode() {
		t.Error("should exit settings mode")
	}
}

func TestMainMenu_SettingsCycleGhostDisplay(t *testing.T) {
	m := tui.NewMainMenu(nil, []string{"claude"}, "claude", "animated")

	m.CycleGhostDisplay()
	if m.GhostDisplay() != "static" {
		t.Errorf("expected static after cycling from animated, got %s", m.GhostDisplay())
	}

	m.CycleGhostDisplay()
	if m.GhostDisplay() != "none" {
		t.Errorf("expected none, got %s", m.GhostDisplay())
	}

	m.CycleGhostDisplay()
	if m.GhostDisplay() != "animated" {
		t.Errorf("expected animated, got %s", m.GhostDisplay())
	}
}

func TestMainMenu_SettingsViewShowsPanel(t *testing.T) {
	m := tui.NewMainMenu(nil, []string{"claude"}, "claude", "animated")
	m.SetSize(80, 30)
	m.EnterSettings()
	view := m.View()

	if !strings.Contains(view, "Settings") {
		t.Error("settings view should show 'Settings' title")
	}
	if !strings.Contains(view, "Ghost Display") {
		t.Error("settings view should show 'Ghost Display' option")
	}
	if !strings.Contains(view, "Animated") {
		t.Error("settings view should show current state 'Animated'")
	}
}

func TestMainMenu_SettingsKeyS(t *testing.T) {
	m := tui.NewMainMenu(nil, []string{"claude"}, "claude", "animated")
	m.SetSize(80, 30)

	// Press S
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}}
	newModel, _ := m.Update(msg)
	mm := newModel.(*tui.MainMenuModel)

	if !mm.InSettingsMode() {
		t.Error("pressing S should enter settings mode")
	}
}

func TestMainMenu_SettingsEscReturns(t *testing.T) {
	m := tui.NewMainMenu(nil, []string{"claude"}, "claude", "animated")
	m.SetSize(80, 30)
	m.EnterSettings()

	msg := tea.KeyMsg{Type: tea.KeyEsc}
	newModel, _ := m.Update(msg)
	mm := newModel.(*tui.MainMenuModel)

	if mm.InSettingsMode() {
		t.Error("Esc in settings should return to main menu")
	}
}

func TestMainMenu_SettingsKeyBDoesNotExit(t *testing.T) {
	m := tui.NewMainMenu(nil, []string{"claude"}, "claude", "animated")
	m.SetSize(80, 30)
	m.EnterSettings()

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}}
	newModel, _ := m.Update(msg)
	mm := newModel.(*tui.MainMenuModel)

	if !mm.InSettingsMode() {
		t.Error("B in settings should not exit settings mode")
	}
}

func TestMainMenu_SettingsLeftArrowCyclesPrevious(t *testing.T) {
	m := tui.NewMainMenu(nil, []string{"claude"}, "claude", "animated")
	m.SetSize(80, 30)
	m.EnterSettings()

	// Left = previous: animated → none
	msg := tea.KeyMsg{Type: tea.KeyLeft}
	newModel, _ := m.Update(msg)
	mm := newModel.(*tui.MainMenuModel)

	if mm.GhostDisplay() != "none" {
		t.Errorf("expected none after pressing left arrow from animated, got %s", mm.GhostDisplay())
	}
}

func TestMainMenu_SettingsRightArrowCyclesNext(t *testing.T) {
	m := tui.NewMainMenu(nil, []string{"claude"}, "claude", "animated")
	m.SetSize(80, 30)
	m.EnterSettings()

	// Right = next: animated → static
	msg := tea.KeyMsg{Type: tea.KeyRight}
	newModel, _ := m.Update(msg)
	mm := newModel.(*tui.MainMenuModel)

	if mm.GhostDisplay() != "static" {
		t.Errorf("expected static after pressing right arrow from animated, got %s", mm.GhostDisplay())
	}
}

func TestMainMenu_SettingsEnterCycles(t *testing.T) {
	m := tui.NewMainMenu(nil, []string{"claude"}, "claude", "animated")
	m.SetSize(80, 30)
	m.EnterSettings()

	// Enter on the selected item (ghost display) should cycle
	msg := tea.KeyMsg{Type: tea.KeyEnter}
	newModel, _ := m.Update(msg)
	mm := newModel.(*tui.MainMenuModel)

	if mm.GhostDisplay() != "static" {
		t.Errorf("expected static after pressing Enter on ghost display, got %s", mm.GhostDisplay())
	}
	// Should still be in settings mode
	if !mm.InSettingsMode() {
		t.Error("should remain in settings mode after Enter")
	}
}

func TestMainMenu_SettingsViewUpdatesAfterCycle(t *testing.T) {
	m := tui.NewMainMenu(nil, []string{"claude"}, "claude", "animated")
	m.SetSize(80, 30)
	m.EnterSettings()
	m.CycleGhostDisplay() // animated -> static

	view := m.View()
	if !strings.Contains(view, "Static") {
		t.Error("settings view should show 'Static' after cycling")
	}
}

func TestMainMenu_SettingsGhostDisplayInResult(t *testing.T) {
	m := tui.NewMainMenu(nil, []string{"claude"}, "claude", "animated")
	m.SetSize(80, 30)

	// Enter settings, cycle ghost display, exit settings
	m.EnterSettings()
	m.CycleGhostDisplay() // animated -> static
	m.ExitSettings()

	// Now quit to get a result
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	mm := newModel.(*tui.MainMenuModel)
	result := mm.Result()

	if result == nil {
		t.Fatal("expected result after quit")
	}
	if result.GhostDisplay != "static" {
		t.Errorf("expected ghost_display 'static' in result, got %q", result.GhostDisplay)
	}
}

func TestMainMenu_SettingsNoGhostDisplayInResultWhenUnchanged(t *testing.T) {
	m := tui.NewMainMenu(nil, []string{"claude"}, "claude", "animated")
	m.SetSize(80, 30)

	// Quit without changing ghost display
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	mm := newModel.(*tui.MainMenuModel)
	result := mm.Result()

	if result == nil {
		t.Fatal("expected result after quit")
	}
	if result.GhostDisplay != "" {
		t.Errorf("expected empty ghost_display when unchanged, got %q", result.GhostDisplay)
	}
}

func TestMainMenu_SettingsNavigationKeys(t *testing.T) {
	m := tui.NewMainMenu(nil, []string{"claude"}, "claude", "animated")
	m.SetSize(80, 30)
	m.EnterSettings()

	// j/k/up/down should not exit settings mode
	for _, msg := range []tea.KeyMsg{
		{Type: tea.KeyRunes, Runes: []rune{'j'}},
		{Type: tea.KeyRunes, Runes: []rune{'k'}},
		{Type: tea.KeyUp},
		{Type: tea.KeyDown},
	} {
		newModel, _ := m.Update(msg)
		mm := newModel.(*tui.MainMenuModel)
		if !mm.InSettingsMode() {
			t.Errorf("navigation key %v should not exit settings mode", msg)
		}
	}
}

func TestMainMenu_SettingsDoesNotQuit(t *testing.T) {
	m := tui.NewMainMenu(nil, []string{"claude"}, "claude", "animated")
	m.SetSize(80, 30)
	m.EnterSettings()

	// Pressing left arrow (cycle) should not produce a quit command
	msg := tea.KeyMsg{Type: tea.KeyLeft}
	_, cmd := m.Update(msg)

	if cmd != nil {
		t.Error("pressing left arrow in settings should not produce a quit command")
	}
}

func TestMainMenu_SettingsUpperS(t *testing.T) {
	m := tui.NewMainMenu(nil, []string{"claude"}, "claude", "animated")
	m.SetSize(80, 30)

	// Press uppercase S
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'S'}}
	newModel, _ := m.Update(msg)
	mm := newModel.(*tui.MainMenuModel)

	if !mm.InSettingsMode() {
		t.Error("pressing uppercase S should enter settings mode")
	}
}

func TestMainMenu_SettingsHelpRow(t *testing.T) {
	m := tui.NewMainMenu(nil, []string{"claude"}, "claude", "animated")
	m.SetSize(80, 30)
	m.EnterSettings()
	view := m.View()

	if !strings.Contains(view, "cycle") {
		t.Error("settings help row should mention 'cycle'")
	}
	if !strings.Contains(view, "close") {
		t.Error("settings help row should mention 'close'")
	}
	if strings.Contains(view, "back") {
		t.Error("settings help row should not mention 'back'")
	}
}

func TestMainMenu_WakeResetsZzz(t *testing.T) {
	m := tui.NewMainMenu(nil, []string{"claude"}, "claude", "animated")
	// Simulate sleeping state: set sleep timer high, send sleepTickMsg to trigger sleep
	m.SetSleepTimer(119)
	m.Update(tui.NewSleepTickMsg())
	if !m.IsSleeping() {
		t.Fatal("ghost should be sleeping after timer reaches 120")
	}
	// Tick the bob (which should advance Zzz when sleeping)
	m.Update(tui.NewBobTickMsg())
	m.Update(tui.NewBobTickMsg())
	// Now wake
	m.Wake()
	if m.IsSleeping() {
		t.Error("should be awake after Wake()")
	}
	// Zzz should be reset (frame 0) -- tested via ZzzFrame()
	if m.ZzzFrame() != 0 {
		t.Errorf("Zzz should be reset to frame 0 after Wake(), got %d", m.ZzzFrame())
	}
}

func TestMainMenu_BobTickAdvancesZzzWhenSleeping(t *testing.T) {
	m := tui.NewMainMenu(nil, []string{"claude"}, "claude", "animated")
	// Put ghost to sleep
	m.SetSleepTimer(119)
	m.Update(tui.NewSleepTickMsg())
	if !m.IsSleeping() {
		t.Fatal("ghost should be sleeping")
	}
	// Advance bob ticks -- Zzz advances every ZzzTickEvery bob ticks
	initialFrame := m.ZzzFrame()
	for i := 0; i < tui.ZzzTickEvery; i++ {
		m.Update(tui.NewBobTickMsg())
	}
	if m.ZzzFrame() != initialFrame+1 {
		t.Errorf("Zzz frame should advance after %d bob ticks when sleeping, expected %d got %d", tui.ZzzTickEvery, initialFrame+1, m.ZzzFrame())
	}
}

func TestMainMenu_BobTickDoesNotAdvanceZzzWhenAwake(t *testing.T) {
	m := tui.NewMainMenu(nil, []string{"claude"}, "claude", "animated")
	// Ghost is awake by default
	initialFrame := m.ZzzFrame()
	m.Update(tui.NewBobTickMsg())
	if m.ZzzFrame() != initialFrame {
		t.Errorf("Zzz frame should NOT advance when awake, expected %d got %d", initialFrame, m.ZzzFrame())
	}
}

func TestMainMenu_ViewShowsZzzWhenSleeping_Side(t *testing.T) {
	projects := []models.Project{{Name: "test", Path: "/test"}}
	m := tui.NewMainMenu(projects, []string{"claude"}, "claude", "animated")
	m.SetSize(100, 40) // Wide enough for side layout
	// Put ghost to sleep
	m.SetSleepTimer(119)
	m.Update(tui.NewSleepTickMsg())
	if !m.IsSleeping() {
		t.Fatal("ghost should be sleeping")
	}
	view := m.View()
	// Zzz output contains lowercase z and uppercase Z
	if !strings.Contains(view, "z") || !strings.Contains(view, "Z") {
		t.Error("view should contain Zzz animation when ghost is sleeping (side layout)")
	}
}

func TestMainMenu_ViewShowsZzzWhenSleeping_Above(t *testing.T) {
	projects := []models.Project{{Name: "test", Path: "/test"}}
	m := tui.NewMainMenu(projects, []string{"claude"}, "claude", "animated")
	m.SetSize(60, 50) // Narrow for above layout
	// Put ghost to sleep
	m.SetSleepTimer(119)
	m.Update(tui.NewSleepTickMsg())
	if !m.IsSleeping() {
		t.Fatal("ghost should be sleeping")
	}
	view := m.View()
	if !strings.Contains(view, "z") || !strings.Contains(view, "Z") {
		t.Error("view should contain Zzz animation when ghost is sleeping (above layout)")
	}
}

func TestMainMenu_ViewNoZzzWhenAwake(t *testing.T) {
	projects := []models.Project{{Name: "test", Path: "/test"}}
	m := tui.NewMainMenu(projects, []string{"claude"}, "claude", "animated")
	m.SetSize(100, 40)
	// Ghost is awake by default
	view := m.View()
	// The Zzz animation produces lines with specific spacing patterns
	// When awake, there should be no Zzz text appended after ghost
	// We check that the view does NOT contain the Zzz pattern
	z := tui.NewZzzAnimation()
	zzzView := z.View()
	if strings.Contains(view, zzzView) {
		t.Error("view should NOT contain Zzz animation when ghost is awake")
	}
}

func TestMainMenu_KeypressWakesAndResetsZzz(t *testing.T) {
	m := tui.NewMainMenu(nil, []string{"claude"}, "claude", "animated")
	// Put ghost to sleep
	m.SetSleepTimer(119)
	m.Update(tui.NewSleepTickMsg())
	if !m.IsSleeping() {
		t.Fatal("ghost should be sleeping")
	}
	// Advance Zzz
	m.Update(tui.NewBobTickMsg())
	m.Update(tui.NewBobTickMsg())
	// Now press a key
	m.Update(tea.KeyMsg{Type: tea.KeyDown})
	if m.IsSleeping() {
		t.Error("keypress should wake the ghost")
	}
	if m.ZzzFrame() != 0 {
		t.Errorf("keypress should reset Zzz frame to 0, got %d", m.ZzzFrame())
	}
}

func TestMainMenu_ViewUnselectedProjectHasColor(t *testing.T) {
	// Force color output so lipgloss emits ANSI codes in tests.
	prev := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	defer lipgloss.SetColorProfile(prev)

	projects := []models.Project{
		{Name: "p1", Path: "/p1"},
		{Name: "p2", Path: "/p2"},
	}
	m := tui.NewMainMenu(projects, []string{"claude"}, "claude", "animated")
	m.SetSize(80, 30)
	view := m.View()

	// The unselected project (p2) should have ANSI color codes applied
	// directly around the project name. When styled, lipgloss wraps "p2"
	// in \x1b[38;5;NNNm...p2...\x1b[0m so the character immediately
	// before "p2" is "m" (the end of the ANSI escape sequence).
	// Without styling, the character before "p2" is a space.
	lines := strings.Split(view, "\n")
	found := false
	for _, line := range lines {
		if strings.Contains(line, "p2") && !strings.Contains(line, "/p2") {
			found = true
			idx := strings.Index(line, "p2")
			if idx == 0 || line[idx-1] != 'm' {
				t.Error("unselected project name 'p2' should have ANSI color codes applied directly (expected 'm' before name)")
			}
			break
		}
	}
	if !found {
		t.Error("could not find line containing unselected project name 'p2'")
	}
}

func TestMainMenu_ViewUnselectedActionHasColor(t *testing.T) {
	// Force color output so lipgloss emits ANSI codes in tests.
	prev := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	defer lipgloss.SetColorProfile(prev)

	// Use no projects so actions are items 0-3, and select item 0 (Add).
	// That makes Delete (item 1) unselected.
	m := tui.NewMainMenu(nil, []string{"claude"}, "claude", "animated")
	m.SetSize(80, 30)
	view := m.View()

	// Find the line containing "Delete" (unselected action label).
	// When styled with dimStyle, lipgloss wraps the label in
	// \x1b[...m...Delete...\x1b[0m so the character immediately
	// before "Delete" is 'm'. Without styling, it's a space.
	lines := strings.Split(view, "\n")
	found := false
	for _, line := range lines {
		if strings.Contains(line, "Delete") {
			found = true
			idx := strings.Index(line, "Delete")
			if idx == 0 || line[idx-1] != 'm' {
				t.Error("unselected action label 'Delete' should have ANSI color codes applied directly (expected 'm' before label)")
			}
			break
		}
	}
	if !found {
		t.Error("could not find line containing unselected action 'Delete'")
	}
}

func TestMainMenu_ViewUnselectedProjectUsesTextColor(t *testing.T) {
	prev := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	defer lipgloss.SetColorProfile(prev)

	projects := []models.Project{
		{Name: "p1", Path: "/p1"},
		{Name: "p2", Path: "/p2"},
	}
	m := tui.NewMainMenu(projects, []string{"claude"}, "claude", "animated")
	m.SetSize(80, 30)
	view := m.View()

	// Unselected project name "p2" should use theme.Text color (223 for claude)
	// not theme.Bright (208). ANSI 256 color format: \x1b[38;5;223m
	lines := strings.Split(view, "\n")
	for _, line := range lines {
		if strings.Contains(line, "p2") && !strings.Contains(line, "/p2") {
			if !strings.Contains(line, "\x1b[38;5;223m") {
				t.Errorf("unselected project name 'p2' should use Text color (223), line: %q", line)
			}
			return
		}
	}
	t.Error("could not find line containing unselected project name 'p2'")
}

func TestMainMenu_ViewSelectedProjectUsesPrimaryColor(t *testing.T) {
	prev := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	defer lipgloss.SetColorProfile(prev)

	projects := []models.Project{
		{Name: "selected-proj", Path: "/selected"},
	}
	m := tui.NewMainMenu(projects, []string{"claude"}, "claude", "animated")
	m.SetSize(80, 30)
	view := m.View()

	// Selected project name should use theme.Primary (209 for claude).
	// Bold styling produces \x1b[1;38;5;209m (with 1; prefix).
	lines := strings.Split(view, "\n")
	for _, line := range lines {
		if strings.Contains(line, "selected-proj") {
			if !strings.Contains(line, "38;5;209m") {
				t.Errorf("selected project name should use Primary color (209), line: %q", line)
			}
			return
		}
	}
	t.Error("could not find line containing selected project name")
}

func TestMainMenu_ViewSelectedPathUsesPrimaryColor(t *testing.T) {
	prev := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	defer lipgloss.SetColorProfile(prev)

	projects := []models.Project{
		{Name: "proj", Path: "/some/selected/path"},
	}
	m := tui.NewMainMenu(projects, []string{"claude"}, "claude", "animated")
	m.SetSize(80, 30)
	view := m.View()

	// Selected project path should use theme.Primary (209), not Dim (166)
	lines := strings.Split(view, "\n")
	for _, line := range lines {
		if strings.Contains(line, "/some/selected/path") {
			if !strings.Contains(line, "\x1b[38;5;209m") {
				t.Errorf("selected project path should use Primary color (209), line: %q", line)
			}
			return
		}
	}
	t.Error("could not find line containing selected project path")
}

func TestMainMenu_ViewUnselectedActionUsesTextColor(t *testing.T) {
	prev := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	defer lipgloss.SetColorProfile(prev)

	// No projects so actions start at item 0. Item 0 (Add) is selected,
	// so Delete (item 1) is unselected.
	m := tui.NewMainMenu(nil, []string{"claude"}, "claude", "animated")
	m.SetSize(80, 30)
	view := m.View()

	// Unselected action label "Delete" should use theme.Text color (223)
	lines := strings.Split(view, "\n")
	for _, line := range lines {
		if strings.Contains(line, "Delete") {
			if !strings.Contains(line, "\x1b[38;5;223m") {
				t.Errorf("unselected action label 'Delete' should use Text color (223), line: %q", line)
			}
			return
		}
	}
	t.Error("could not find line containing unselected action 'Delete'")
}

func TestMainMenu_MouseClickWakesAndResetsZzz(t *testing.T) {
	m := tui.NewMainMenu(nil, []string{"claude"}, "claude", "animated")
	m.SetSize(80, 30)
	// Put ghost to sleep
	m.SetSleepTimer(119)
	m.Update(tui.NewSleepTickMsg())
	if !m.IsSleeping() {
		t.Fatal("ghost should be sleeping")
	}
	// Advance Zzz
	m.Update(tui.NewBobTickMsg())
	m.Update(tui.NewBobTickMsg())
	// Click
	mouseMsg := tea.MouseMsg{
		X:      10,
		Y:      4,
		Action: tea.MouseActionPress,
		Button: tea.MouseButtonLeft,
	}
	m.Update(mouseMsg)
	if m.IsSleeping() {
		t.Error("mouse click should wake the ghost")
	}
	if m.ZzzFrame() != 0 {
		t.Errorf("mouse click should reset Zzz frame to 0, got %d", m.ZzzFrame())
	}
}

func TestMainMenu_ZzzAppearsAboveGhost(t *testing.T) {
	t.Run("side_layout", func(t *testing.T) {
		projects := []models.Project{{Name: "test", Path: "/test"}}
		m := tui.NewMainMenu(projects, []string{"claude"}, "claude", "animated")
		m.SetSize(100, 40) // Wide enough for side layout
		// Put ghost to sleep
		m.SetSleepTimer(119)
		m.Update(tui.NewSleepTickMsg())
		if !m.IsSleeping() {
			t.Fatal("ghost should be sleeping")
		}
		view := m.View()

		// Find line indices for zzz content (z/Z letters) and ghost cap art (▄)
		// Menu text contains neither z/Z nor ▄, so these are unambiguous markers
		lines := strings.Split(view, "\n")
		firstZzzLine := -1
		firstGhostCapLine := -1
		for i, line := range lines {
			if firstZzzLine == -1 && strings.ContainsAny(line, "zZ") {
				firstZzzLine = i
			}
			if firstGhostCapLine == -1 && strings.Contains(line, "\u2584") {
				firstGhostCapLine = i
			}
		}

		if firstZzzLine == -1 {
			t.Fatal("could not find zzz content in view")
		}
		if firstGhostCapLine == -1 {
			t.Fatal("could not find ghost cap art in view")
		}
		if firstZzzLine >= firstGhostCapLine {
			t.Errorf("zzz should appear above ghost: zzz at line %d, ghost cap at line %d", firstZzzLine, firstGhostCapLine)
		}
	})

	t.Run("above_layout", func(t *testing.T) {
		projects := []models.Project{{Name: "test", Path: "/test"}}
		m := tui.NewMainMenu(projects, []string{"claude"}, "claude", "animated")
		m.SetSize(60, 50) // Narrow for above layout
		// Put ghost to sleep
		m.SetSleepTimer(119)
		m.Update(tui.NewSleepTickMsg())
		if !m.IsSleeping() {
			t.Fatal("ghost should be sleeping")
		}
		view := m.View()

		lines := strings.Split(view, "\n")
		firstZzzLine := -1
		firstGhostCapLine := -1
		for i, line := range lines {
			if firstZzzLine == -1 && strings.ContainsAny(line, "zZ") {
				firstZzzLine = i
			}
			if firstGhostCapLine == -1 && strings.Contains(line, "\u2584") {
				firstGhostCapLine = i
			}
		}

		if firstZzzLine == -1 {
			t.Fatal("could not find zzz content in view")
		}
		if firstGhostCapLine == -1 {
			t.Fatal("could not find ghost cap art in view")
		}
		if firstZzzLine >= firstGhostCapLine {
			t.Errorf("zzz should appear above ghost: zzz at line %d, ghost cap at line %d", firstZzzLine, firstGhostCapLine)
		}
	})
}

func TestMainMenu_ViewIsCentered(t *testing.T) {
	m := tui.NewMainMenu(nil, []string{"claude"}, "claude", "none")
	m.SetSize(80, 40)
	view := m.View()

	lines := strings.Split(view, "\n")

	if len(lines) < 2 {
		t.Fatal("expected multiple lines in centered view")
	}

	// First few lines should be blank or whitespace-only (vertical centering)
	firstContentLine := -1
	for i, line := range lines {
		if strings.TrimSpace(line) != "" {
			firstContentLine = i
			break
		}
	}
	if firstContentLine == 0 {
		t.Error("expected vertical centering: first non-blank line should not be at row 0")
	}
	if firstContentLine < 0 {
		t.Fatal("no content found in view")
	}

	// The content line should have leading spaces (horizontal centering)
	contentLine := lines[firstContentLine]
	trimmed := strings.TrimLeft(contentLine, " ")
	leadingSpaces := len(contentLine) - len(trimmed)
	if leadingSpaces < 5 {
		t.Errorf("expected horizontal centering with significant leading spaces, got %d", leadingSpaces)
	}
}

func TestMainMenu_MouseClickWorksWithCentering(t *testing.T) {
	projects := []models.Project{
		{Name: "p1", Path: "/p1"},
		{Name: "p2", Path: "/p2"},
	}
	m := tui.NewMainMenu(projects, []string{"claude"}, "claude", "none")
	m.SetSize(80, 40) // Large terminal -> centering will offset content

	// Need to call View() first so centerOffsetY is calculated
	m.View()

	// With ghost=none, 2 projects + 4 actions = 6 items, 1 separator
	// Menu box height (lines): top border + title + sep + empty + 2*2 proj + sep + 4 actions + sep + help + bottom = ~16 lines
	// Vertical offset = (40 - menuLines) / 2
	// Second project name is at menu row 6, so absolute row = offset + 6
	offset := m.CenterOffsetY()
	if offset <= 0 {
		t.Fatalf("expected positive centering offset with 80x40 and ghost=none, got %d", offset)
	}

	mouseMsg := tea.MouseMsg{
		X:      40,
		Y:      offset + 6,
		Action: tea.MouseActionPress,
		Button: tea.MouseButtonLeft,
	}
	newModel, _ := m.Update(mouseMsg)
	mm := newModel.(*tui.MainMenuModel)
	if mm.SelectedItem() != 1 {
		t.Errorf("clicking centered second project should select item 1, got %d", mm.SelectedItem())
	}
}

func TestMainMenu_TabTitle(t *testing.T) {
	m := tui.NewMainMenu(nil, []string{"claude"}, "claude", "animated")
	m.SetTabTitle("full")

	if m.TabTitle() != "full" {
		t.Errorf("expected 'full', got %q", m.TabTitle())
	}
}

func TestMainMenu_CycleTabTitle(t *testing.T) {
	m := tui.NewMainMenu(nil, []string{"claude"}, "claude", "animated")
	m.SetTabTitle("full")

	m.CycleTabTitle()
	if m.TabTitle() != "project" {
		t.Errorf("expected 'project' after cycling from full, got %q", m.TabTitle())
	}

	m.CycleTabTitle()
	if m.TabTitle() != "full" {
		t.Errorf("expected 'full' after cycling from project, got %q", m.TabTitle())
	}
}

func TestMainMenu_TabTitleInResult(t *testing.T) {
	m := tui.NewMainMenu(nil, []string{"claude"}, "claude", "animated")
	m.SetTabTitle("full")
	m.SetSize(80, 30)

	// Enter settings, cycle tab title, exit settings
	m.EnterSettings()
	m.CycleTabTitle()
	m.ExitSettings()

	// Quit to get result
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	mm := newModel.(*tui.MainMenuModel)
	result := mm.Result()

	if result == nil {
		t.Fatal("expected result after quit")
	}
	if result.TabTitle != "project" {
		t.Errorf("expected tab_title 'project' in result, got %q", result.TabTitle)
	}
}

func TestMainMenu_NoTabTitleInResultWhenUnchanged(t *testing.T) {
	m := tui.NewMainMenu(nil, []string{"claude"}, "claude", "animated")
	m.SetTabTitle("full")
	m.SetSize(80, 30)

	// Quit without changing tab title
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	mm := newModel.(*tui.MainMenuModel)
	result := mm.Result()

	if result == nil {
		t.Fatal("expected result after quit")
	}
	if result.TabTitle != "" {
		t.Errorf("expected empty tab_title when unchanged, got %q", result.TabTitle)
	}
}

func TestMainMenu_SettingsViewShowsTabTitle(t *testing.T) {
	m := tui.NewMainMenu(nil, []string{"claude"}, "claude", "animated")
	m.SetTabTitle("full")
	m.SetSize(80, 30)
	m.EnterSettings()
	view := m.View()

	if !strings.Contains(view, "Tab Title") {
		t.Error("settings view should show 'Tab Title' option")
	}
	if !strings.Contains(view, "Project") {
		t.Error("settings view should show current state label")
	}
}

func TestMainMenu_SideLayoutVisualCentering(t *testing.T) {
	// The visible content (menu + spacer + ghost) should be centered on
	// screen as a unit. The ghost should NOT be right-padded, so the left
	// and right margins of the visible content are approximately equal.
	//
	// Visible content = 48 (menu) + 3 (spacer) + ~28 (ghost) = ~79
	// For width=120: expected left margin ≈ (120 - 79) / 2 ≈ 20
	projects := []models.Project{{Name: "test", Path: "/test"}}
	m := tui.NewMainMenu(projects, []string{"claude"}, "claude", "static")
	width := 120
	m.SetSize(width, 40)
	view := m.View()

	lines := strings.Split(view, "\n")
	for _, line := range lines {
		if strings.Contains(line, "\u250c") { // ┌ (top border)
			trimmedLeft := strings.TrimLeft(line, " ")
			leftMargin := len(line) - len(trimmedLeft)
			// Without ghost padding, content is ~79 chars wide
			// Left margin should be roughly (120 - 79) / 2 ≈ 20
			if leftMargin < 16 {
				t.Errorf("visible content not horizontally centered: left margin %d too small (expected >= 16 for width %d)", leftMargin, width)
			}
			break
		}
	}
}

func TestMainMenu_SideLayoutGhostVerticallyCentered(t *testing.T) {
	// With many projects, the menu box is taller than the ghost (15 lines).
	// The ghost should be vertically centered relative to the menu, not
	// top-aligned. We verify by checking that ghost art (▄ cap) does NOT
	// start on the same line as the menu's top border.
	projects := []models.Project{
		{Name: "p1", Path: "/p1"},
		{Name: "p2", Path: "/p2"},
		{Name: "p3", Path: "/p3"},
		{Name: "p4", Path: "/p4"},
		{Name: "p5", Path: "/p5"},
		{Name: "p6", Path: "/p6"},
	}
	m := tui.NewMainMenu(projects, []string{"claude"}, "claude", "static")
	m.SetSize(120, 50)
	view := m.View()

	lines := strings.Split(view, "\n")
	menuTopLine := -1
	ghostCapLine := -1
	menuBottomLine := -1
	for i, line := range lines {
		if menuTopLine == -1 && strings.Contains(line, "\u250c") { // ┌
			menuTopLine = i
		}
		if strings.Contains(line, "\u2518") { // ┘
			menuBottomLine = i
		}
		if ghostCapLine == -1 && strings.Contains(line, "\u2584") { // ▄ (ghost cap)
			ghostCapLine = i
		}
	}

	if menuTopLine == -1 || ghostCapLine == -1 || menuBottomLine == -1 {
		t.Fatalf("could not find menu borders or ghost cap: top=%d, bottom=%d, ghost=%d", menuTopLine, menuBottomLine, ghostCapLine)
	}

	// Ghost should start below the menu top (vertically centered, not top-aligned)
	if ghostCapLine <= menuTopLine {
		t.Errorf("ghost should be vertically centered, not top-aligned: ghost cap at line %d, menu top at line %d", ghostCapLine, menuTopLine)
	}

	// Ghost should start at least a few lines below menu top when menu is much taller
	menuHeight := menuBottomLine - menuTopLine + 1
	ghostOffset := ghostCapLine - menuTopLine
	// Ghost is ~15 lines, so expected offset ≈ (menuHeight - 15) / 2
	expectedOffset := (menuHeight - 15) / 2
	diff := ghostOffset - expectedOffset
	if diff < 0 {
		diff = -diff
	}
	if diff > 2 {
		t.Errorf("ghost not vertically centered: offset %d from menu top, expected ~%d (menu height %d)", ghostOffset, expectedOffset, menuHeight)
	}
}

func TestTruncateMiddle_Short(t *testing.T) {
	got := tui.TruncateMiddle("hello", 10)
	if got != "hello" {
		t.Errorf("short string should pass through unchanged, got %q", got)
	}
}

func TestTruncateMiddle_Exact(t *testing.T) {
	got := tui.TruncateMiddle("hello", 5)
	if got != "hello" {
		t.Errorf("exact-length string should pass through unchanged, got %q", got)
	}
}

func TestTruncateMiddle_Long(t *testing.T) {
	got := tui.TruncateMiddle("abcdefghij", 7)
	// 7 chars: 3 left + … + 3 right = "abc…hij"
	if got != "abc\u2026hij" {
		t.Errorf("expected %q, got %q", "abc\u2026hij", got)
	}
	if lipgloss.Width(got) != 7 {
		t.Errorf("visual width should be 7, got %d", lipgloss.Width(got))
	}
}

func TestTruncateMiddle_VerySmallMax(t *testing.T) {
	got := tui.TruncateMiddle("abcdefghij", 1)
	if got != "\u2026" {
		t.Errorf("maxWidth=1 should return just ellipsis, got %q", got)
	}
}

func TestMainMenu_ViewTruncatesLongPath(t *testing.T) {
	lipgloss.SetDefaultRenderer(lipgloss.NewRenderer(termenv.NewOutput(termenv.DefaultOutput().TTY(), termenv.WithProfile(termenv.Ascii))))
	longPath := "/Users/jack/Packages/shiftmanager-frontend/microfrontends/backoffice-elements"
	projects := []models.Project{{Name: "proj", Path: longPath}}
	m := tui.NewMainMenu(projects, []string{"claude"}, "claude", "none")
	m.SetSize(80, 30)
	view := m.View()
	found := false
	for _, line := range strings.Split(view, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.Contains(trimmed, "shiftmanager") || strings.Contains(trimmed, "backoffice") {
			found = true
			if !strings.Contains(trimmed, "\u2026") {
				t.Error("long path should be truncated with ellipsis")
			}
			if lipgloss.Width(trimmed) > 48 {
				t.Errorf("path line content should not exceed box width 48, got %d: %q", lipgloss.Width(trimmed), trimmed)
			}
		}
	}
	if !found {
		t.Error("view should contain the project path (truncated)")
	}
}

func TestMainMenu_ViewTruncatesLongName(t *testing.T) {
	lipgloss.SetDefaultRenderer(lipgloss.NewRenderer(termenv.NewOutput(termenv.DefaultOutput().TTY(), termenv.WithProfile(termenv.Ascii))))
	longName := "my-extremely-long-project-name-that-overflows-the-box"
	projects := []models.Project{{Name: longName, Path: "/short"}}
	m := tui.NewMainMenu(projects, []string{"claude"}, "claude", "none")
	m.SetSize(80, 30)
	view := m.View()
	found := false
	for _, line := range strings.Split(view, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.Contains(trimmed, "extremely") || strings.Contains(trimmed, "overflows") {
			found = true
			if !strings.Contains(trimmed, "\u2026") {
				t.Error("long name should be truncated with ellipsis")
			}
			if lipgloss.Width(trimmed) > 48 {
				t.Errorf("name line content should not exceed box width 48, got %d: %q", lipgloss.Width(trimmed), trimmed)
			}
		}
	}
	if !found {
		t.Error("view should contain the project name (truncated)")
	}
}

func TestMainMenu_ViewFillsFullTerminalHeight(t *testing.T) {
	// The view output should fill the full terminal height so that
	// lipgloss.Place produces proper centering. The number of lines
	// in the view string must equal the terminal height.
	projects := []models.Project{
		{Name: "p1", Path: "/p1"},
		{Name: "p2", Path: "/p2"},
		{Name: "p3", Path: "/p3"},
	}
	termHeight := 50
	m := tui.NewMainMenu(projects, []string{"claude", "codex"}, "codex", "animated")
	m.SetSize(120, termHeight)

	view := m.View()
	lines := strings.Split(view, "\n")

	if len(lines) != termHeight {
		t.Errorf("view should have exactly %d lines to fill terminal, got %d", termHeight, len(lines))
	}
}

func TestMainMenu_ViewVerticalCenteringIsSymmetric(t *testing.T) {
	// Top blank rows and bottom blank rows should differ by at most 1.
	projects := []models.Project{
		{Name: "p1", Path: "/p1"},
		{Name: "p2", Path: "/p2"},
		{Name: "p3", Path: "/p3"},
		{Name: "p4", Path: "/p4"},
		{Name: "p5", Path: "/p5"},
		{Name: "p6", Path: "/p6"},
	}
	termHeight := 50
	m := tui.NewMainMenu(projects, []string{"claude", "codex"}, "codex", "animated")
	m.SetSize(120, termHeight)

	view := m.View()
	lines := strings.Split(view, "\n")

	// Count top blank rows
	topBlank := 0
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			topBlank++
		} else {
			break
		}
	}

	// Count bottom blank rows
	bottomBlank := 0
	for i := len(lines) - 1; i >= 0; i-- {
		if strings.TrimSpace(lines[i]) == "" {
			bottomBlank++
		} else {
			break
		}
	}

	diff := topBlank - bottomBlank
	if diff < 0 {
		diff = -diff
	}
	if diff > 1 {
		t.Errorf("vertical centering is not symmetric: %d top blank rows, %d bottom blank rows (diff %d > 1)", topBlank, bottomBlank, diff)
	}
}

func TestMainMenu_InputMode(t *testing.T) {
	m := tui.NewMainMenu(testProjects(), testAITools(), "claude", "animated")
	if m.InputMode() != "" {
		t.Errorf("Initial InputMode should be empty, got %q", m.InputMode())
	}
	if m.InInputMode() {
		t.Error("Should not be in input mode initially")
	}
}

func TestMainMenu_DeleteMode(t *testing.T) {
	m := tui.NewMainMenu(testProjects(), testAITools(), "claude", "animated")
	if m.InDeleteMode() {
		t.Error("Should not be in delete mode initially")
	}
}

func TestMainMenu_FeedbackMsg(t *testing.T) {
	m := tui.NewMainMenu(testProjects(), testAITools(), "claude", "animated")
	if m.FeedbackMsg() != "" {
		t.Errorf("Initial FeedbackMsg should be empty, got %q", m.FeedbackMsg())
	}
}

func TestMainMenu_SetProjectsFile(t *testing.T) {
	m := tui.NewMainMenu(testProjects(), testAITools(), "claude", "animated")
	m.SetProjectsFile("/tmp/test-projects")
	if m.ProjectsFile() != "/tmp/test-projects" {
		t.Errorf("ProjectsFile: expected '/tmp/test-projects', got %q", m.ProjectsFile())
	}
}

func TestMainMenu_AddProject_EntersInputMode(t *testing.T) {
	m := tui.NewMainMenu(testProjects(), testAITools(), "claude", "animated")
	newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	mm := newModel.(*tui.MainMenuModel)

	if !mm.InInputMode() {
		t.Error("Expected input mode after 'a' press")
	}
	if mm.InputMode() != "add-project" {
		t.Errorf("Expected input mode 'add-project', got %q", mm.InputMode())
	}
	if mm.Result() != nil {
		t.Error("Should not produce result when entering input mode")
	}
	if cmd == nil {
		t.Error("Expected a cmd (textinput.Blink) when entering input mode")
	}
}

func TestMainMenu_AddProject_EscCancelsInputMode(t *testing.T) {
	m := tui.NewMainMenu(testProjects(), testAITools(), "claude", "animated")
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	mm := newModel.(*tui.MainMenuModel)

	newModel2, _ := mm.Update(tea.KeyMsg{Type: tea.KeyEsc})
	mm2 := newModel2.(*tui.MainMenuModel)

	if mm2.InInputMode() {
		t.Error("Input mode should be cancelled after Esc")
	}
	if mm2.Result() != nil {
		t.Error("Should not produce result on cancel")
	}
}

func TestMainMenu_AddProject_EmptyEnterCancels(t *testing.T) {
	m := tui.NewMainMenu(testProjects(), testAITools(), "claude", "animated")
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	mm := newModel.(*tui.MainMenuModel)

	newModel2, _ := mm.Update(tea.KeyMsg{Type: tea.KeyEnter})
	mm2 := newModel2.(*tui.MainMenuModel)

	if mm2.InInputMode() {
		t.Error("Input mode should be cancelled on empty Enter")
	}
}

func TestMainMenu_AddProject_SubmitValid(t *testing.T) {
	dir := t.TempDir()
	projFile := filepath.Join(dir, "projects")
	os.WriteFile(projFile, []byte("existing:/tmp/existing\n"), 0644)

	targetDir := filepath.Join(dir, "new-project")
	os.MkdirAll(targetDir, 0755)

	m := tui.NewMainMenu(
		[]models.Project{{Name: "existing", Path: "/tmp/existing"}},
		testAITools(), "claude", "animated",
	)
	m.SetProjectsFile(projFile)

	// Enter add mode
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	mm := newModel.(*tui.MainMenuModel)

	// Type the path character by character
	for _, r := range targetDir {
		mm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}

	// First Enter accepts autocomplete suggestion (adds trailing /),
	// second Enter submits (no suggestions for empty directory)
	mm.Update(tea.KeyMsg{Type: tea.KeyEnter})
	newModel3, _ := mm.Update(tea.KeyMsg{Type: tea.KeyEnter})
	mm3 := newModel3.(*tui.MainMenuModel)

	if mm3.InInputMode() {
		t.Error("Should exit input mode after valid submit")
	}
	if mm3.FeedbackMsg() == "" {
		t.Error("Expected feedback message after adding project")
	}

	data, _ := os.ReadFile(projFile)
	if !strings.Contains(string(data), "new-project:"+targetDir) {
		t.Errorf("Projects file should contain new entry, got: %q", string(data))
	}
}

func TestMainMenu_AddProject_DuplicateShowsError(t *testing.T) {
	dir := t.TempDir()
	projFile := filepath.Join(dir, "projects")
	targetDir := filepath.Join(dir, "existing")
	os.MkdirAll(targetDir, 0755)
	os.WriteFile(projFile, []byte("existing:"+targetDir+"\n"), 0644)

	m := tui.NewMainMenu(
		[]models.Project{{Name: "existing", Path: targetDir}},
		testAITools(), "claude", "animated",
	)
	m.SetProjectsFile(projFile)

	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	mm := newModel.(*tui.MainMenuModel)
	for _, r := range targetDir {
		mm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	// First Enter accepts autocomplete suggestion, second Enter submits
	mm.Update(tea.KeyMsg{Type: tea.KeyEnter})
	newModel2, _ := mm.Update(tea.KeyMsg{Type: tea.KeyEnter})
	mm2 := newModel2.(*tui.MainMenuModel)

	if !mm2.InInputMode() {
		t.Error("Should stay in input mode on duplicate")
	}
}

func TestMainMenu_FeedbackTimer_Dismisses(t *testing.T) {
	dir := t.TempDir()
	projFile := filepath.Join(dir, "projects")
	os.WriteFile(projFile, []byte(""), 0644)

	targetDir := filepath.Join(dir, "feedback-test")
	os.MkdirAll(targetDir, 0755)

	m := tui.NewMainMenu(nil, testAITools(), "claude", "animated")
	m.SetProjectsFile(projFile)

	// Enter add mode
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	mm := newModel.(*tui.MainMenuModel)

	// Type the path
	for _, r := range targetDir {
		mm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}

	// First Enter accepts autocomplete suggestion, second Enter submits
	mm.Update(tea.KeyMsg{Type: tea.KeyEnter})
	newModel2, _ := mm.Update(tea.KeyMsg{Type: tea.KeyEnter})
	mm2 := newModel2.(*tui.MainMenuModel)

	if mm2.FeedbackMsg() == "" {
		t.Error("Expected feedback message")
	}

	// Tick enough times to dismiss
	for i := 0; i < tui.FeedbackDismissTicks+1; i++ {
		mm2.Update(tui.NewBobTickMsg())
	}

	if mm2.FeedbackMsg() != "" {
		t.Errorf("Feedback should be dismissed after %d ticks, got %q", tui.FeedbackDismissTicks, mm2.FeedbackMsg())
	}
}

// Delete mode tests
func TestMainMenu_DeleteProject_EntersDeleteMode(t *testing.T) {
	m := tui.NewMainMenu(testProjects(), testAITools(), "claude", "animated")
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	mm := newModel.(*tui.MainMenuModel)

	if !mm.InDeleteMode() {
		t.Error("Expected delete mode after 'd' press")
	}
	if mm.Result() != nil {
		t.Error("Should not produce result when entering delete mode")
	}
}

func TestMainMenu_DeleteProject_NoProjectsShowsFeedback(t *testing.T) {
	m := tui.NewMainMenu(nil, testAITools(), "claude", "animated")
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	mm := newModel.(*tui.MainMenuModel)

	if mm.InDeleteMode() {
		t.Error("Should not enter delete mode with no projects")
	}
	if mm.FeedbackMsg() == "" {
		t.Error("Should show feedback when no projects to delete")
	}
}

func TestMainMenu_DeleteProject_QCancels(t *testing.T) {
	m := tui.NewMainMenu(testProjects(), testAITools(), "claude", "animated")
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	mm := newModel.(*tui.MainMenuModel)

	newModel2, _ := mm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	mm2 := newModel2.(*tui.MainMenuModel)

	if mm2.InDeleteMode() {
		t.Error("Delete mode should be cancelled after Q")
	}
}

func TestMainMenu_DeleteProject_EscCancels(t *testing.T) {
	m := tui.NewMainMenu(testProjects(), testAITools(), "claude", "animated")
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	mm := newModel.(*tui.MainMenuModel)

	newModel2, _ := mm.Update(tea.KeyMsg{Type: tea.KeyEsc})
	mm2 := newModel2.(*tui.MainMenuModel)

	if mm2.InDeleteMode() {
		t.Error("Delete mode should be cancelled after Esc")
	}
}

func TestMainMenu_DeleteProject_NavigatesProjects(t *testing.T) {
	m := tui.NewMainMenu(testProjects(), testAITools(), "claude", "animated")
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	mm := newModel.(*tui.MainMenuModel)

	if mm.DeleteSelected() != 0 {
		t.Error("Should start at first project")
	}

	newModel2, _ := mm.Update(tea.KeyMsg{Type: tea.KeyDown})
	mm2 := newModel2.(*tui.MainMenuModel)
	if mm2.DeleteSelected() != 1 {
		t.Errorf("After down: expected 1, got %d", mm2.DeleteSelected())
	}

	// Wrap around
	newModel3, _ := mm2.Update(tea.KeyMsg{Type: tea.KeyDown})
	mm3 := newModel3.(*tui.MainMenuModel)
	newModel4, _ := mm3.Update(tea.KeyMsg{Type: tea.KeyDown})
	mm4 := newModel4.(*tui.MainMenuModel)
	if mm4.DeleteSelected() != 0 {
		t.Errorf("After wrapping down: expected 0, got %d", mm4.DeleteSelected())
	}

	// Up wraps
	newModel5, _ := mm4.Update(tea.KeyMsg{Type: tea.KeyUp})
	mm5 := newModel5.(*tui.MainMenuModel)
	if mm5.DeleteSelected() != 2 {
		t.Errorf("After wrapping up from 0: expected 2, got %d", mm5.DeleteSelected())
	}
}

func TestMainMenu_DeleteProject_NumberJumps(t *testing.T) {
	m := tui.NewMainMenu(testProjects(), testAITools(), "claude", "animated")
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	mm := newModel.(*tui.MainMenuModel)

	newModel2, _ := mm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	mm2 := newModel2.(*tui.MainMenuModel)
	if mm2.DeleteSelected() != 2 {
		t.Errorf("After '3' key: expected 2, got %d", mm2.DeleteSelected())
	}

	// Number beyond range does nothing
	newModel3, _ := mm2.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'9'}})
	mm3 := newModel3.(*tui.MainMenuModel)
	if mm3.DeleteSelected() != 2 {
		t.Errorf("After '9' key (beyond range): expected 2 (unchanged), got %d", mm3.DeleteSelected())
	}
}

func TestMainMenu_DeleteProject_JKNavigation(t *testing.T) {
	m := tui.NewMainMenu(testProjects(), testAITools(), "claude", "animated")
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	mm := newModel.(*tui.MainMenuModel)

	// j moves down
	newModel2, _ := mm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	mm2 := newModel2.(*tui.MainMenuModel)
	if mm2.DeleteSelected() != 1 {
		t.Errorf("After 'j': expected 1, got %d", mm2.DeleteSelected())
	}

	// k moves up
	newModel3, _ := mm2.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	mm3 := newModel3.(*tui.MainMenuModel)
	if mm3.DeleteSelected() != 0 {
		t.Errorf("After 'k': expected 0, got %d", mm3.DeleteSelected())
	}
}

func TestMainMenu_DeleteProject_ConfirmDeletes(t *testing.T) {
	dir := t.TempDir()
	projFile := filepath.Join(dir, "projects")
	os.WriteFile(projFile, []byte("ghost-tab:/Users/jack/ghost-tab\nmy-app:/Users/jack/my-app\nwebsite:/Users/jack/website\n"), 0644)

	m := tui.NewMainMenu(testProjects(), testAITools(), "claude", "animated")
	m.SetProjectsFile(projFile)

	// Enter delete mode
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	mm := newModel.(*tui.MainMenuModel)

	// Move to second project (my-app)
	newModel2, _ := mm.Update(tea.KeyMsg{Type: tea.KeyDown})
	mm2 := newModel2.(*tui.MainMenuModel)

	// Press Enter to delete
	newModel3, _ := mm2.Update(tea.KeyMsg{Type: tea.KeyEnter})
	mm3 := newModel3.(*tui.MainMenuModel)

	if mm3.InDeleteMode() {
		t.Error("Should exit delete mode after deletion")
	}
	if mm3.FeedbackMsg() == "" {
		t.Error("Expected feedback after deletion")
	}
	if !strings.Contains(mm3.FeedbackMsg(), "my-app") {
		t.Errorf("Feedback should mention deleted project name, got %q", mm3.FeedbackMsg())
	}

	data, _ := os.ReadFile(projFile)
	if strings.Contains(string(data), "my-app") {
		t.Error("Deleted project should be removed from file")
	}
	if !strings.Contains(string(data), "ghost-tab") {
		t.Error("Other projects should remain")
	}
	if !strings.Contains(string(data), "website") {
		t.Error("Other projects should remain")
	}
}

// Open-once tests
func TestMainMenu_OpenOnce_EntersInputMode(t *testing.T) {
	m := tui.NewMainMenu(testProjects(), testAITools(), "claude", "animated")
	newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}})
	mm := newModel.(*tui.MainMenuModel)

	if !mm.InInputMode() {
		t.Error("Expected input mode after 'o' press")
	}
	if mm.InputMode() != "open-once" {
		t.Errorf("Expected input mode 'open-once', got %q", mm.InputMode())
	}
	if mm.Result() != nil {
		t.Error("Should not produce result when entering input mode")
	}
	if cmd == nil {
		t.Error("Expected a cmd when entering input mode")
	}
}

func TestMainMenu_OpenOnce_SubmitValid(t *testing.T) {
	dir := t.TempDir()
	targetDir := filepath.Join(dir, "temp-project")
	os.MkdirAll(targetDir, 0755)

	m := tui.NewMainMenu(testProjects(), testAITools(), "claude", "animated")

	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}})
	mm := newModel.(*tui.MainMenuModel)

	for _, r := range targetDir {
		mm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}

	// First Enter accepts autocomplete suggestion, second Enter submits
	mm.Update(tea.KeyMsg{Type: tea.KeyEnter})
	newModel2, cmd := mm.Update(tea.KeyMsg{Type: tea.KeyEnter})
	mm2 := newModel2.(*tui.MainMenuModel)

	result := mm2.Result()
	if result == nil {
		t.Fatal("Expected result after valid open-once submit")
	}
	if result.Action != "open-once" {
		t.Errorf("Expected action 'open-once', got %q", result.Action)
	}
	if result.Name != "temp-project" {
		t.Errorf("Expected name 'temp-project', got %q", result.Name)
	}
	if result.Path != targetDir {
		t.Errorf("Expected path %q, got %q", targetDir, result.Path)
	}
	if cmd == nil {
		t.Error("Expected tea.Quit cmd")
	}
}

func TestMainMenu_OpenOnce_InvalidPathShowsError(t *testing.T) {
	m := tui.NewMainMenu(testProjects(), testAITools(), "claude", "animated")
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}})
	mm := newModel.(*tui.MainMenuModel)

	for _, r := range "/nonexistent/path/xyz" {
		mm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}

	newModel2, _ := mm.Update(tea.KeyMsg{Type: tea.KeyEnter})
	mm2 := newModel2.(*tui.MainMenuModel)

	if !mm2.InInputMode() {
		t.Error("Should stay in input mode on invalid path")
	}
	if mm2.Result() != nil {
		t.Error("Should not produce result on invalid path")
	}
}

func TestMainMenu_OpenOnce_EscCancels(t *testing.T) {
	m := tui.NewMainMenu(testProjects(), testAITools(), "claude", "animated")
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}})
	mm := newModel.(*tui.MainMenuModel)

	newModel2, _ := mm.Update(tea.KeyMsg{Type: tea.KeyEsc})
	mm2 := newModel2.(*tui.MainMenuModel)

	if mm2.InInputMode() {
		t.Error("Input mode should be cancelled after Esc")
	}
}

func TestMainMenu_View_InputMode(t *testing.T) {
	lipgloss.SetColorProfile(termenv.Ascii)
	m := tui.NewMainMenu(testProjects(), testAITools(), "claude", "animated")
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	mm := newModel.(*tui.MainMenuModel)

	view := mm.View()
	if !strings.Contains(view, "Path") {
		t.Error("Input mode view should contain 'Path'")
	}
	if !strings.Contains(view, "Add Project") {
		t.Error("Input mode view should show 'Add Project' label")
	}
}

func TestMainMenu_View_InputMode_OpenOnce(t *testing.T) {
	lipgloss.SetColorProfile(termenv.Ascii)
	m := tui.NewMainMenu(testProjects(), testAITools(), "claude", "animated")
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}})
	mm := newModel.(*tui.MainMenuModel)

	view := mm.View()
	if !strings.Contains(view, "Open Once") {
		t.Error("Input mode view should show 'Open Once' label")
	}
}

func assertBoxLinesConsistentWidth(t *testing.T, view string) {
	t.Helper()
	lines := strings.Split(view, "\n")
	boxWidth := -1
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if !strings.ContainsAny(trimmed, "┌├└│") {
			continue
		}
		w := lipgloss.Width(line)
		if boxWidth < 0 {
			boxWidth = w
		} else if w != boxWidth {
			t.Errorf("line width %d differs from expected %d:\n  line: %q", w, boxWidth, line)
		}
	}
	if boxWidth < 0 {
		t.Fatal("no box lines found in view")
	}
}

func TestMainMenu_View_InputBoxLinesHaveConsistentWidth(t *testing.T) {
	lipgloss.SetColorProfile(termenv.Ascii)

	for _, mode := range []struct {
		name string
		key  rune
	}{
		{"add-project", 'a'},
		{"open-once", 'o'},
	} {
		t.Run(mode.name+"/placeholder", func(t *testing.T) {
			m := tui.NewMainMenu(testProjects(), testAITools(), "claude", "animated")
			newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{mode.key}})
			mm := newModel.(*tui.MainMenuModel)
			assertBoxLinesConsistentWidth(t, mm.View())
		})

		t.Run(mode.name+"/with-text", func(t *testing.T) {
			m := tui.NewMainMenu(testProjects(), testAITools(), "claude", "animated")
			newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{mode.key}})
			mm := newModel.(*tui.MainMenuModel)
			// Type "/" to trigger text mode (cursor at end)
			newModel2, _ := mm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
			mm2 := newModel2.(*tui.MainMenuModel)
			assertBoxLinesConsistentWidth(t, mm2.View())
		})
	}
}

func TestMainMenu_View_InputBoxSuggestionsInsideBox(t *testing.T) {
	lipgloss.SetColorProfile(termenv.Ascii)

	m := tui.NewMainMenu(testProjects(), testAITools(), "claude", "animated")
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	mm := newModel.(*tui.MainMenuModel)
	// Type "/" to trigger autocomplete suggestions
	newModel2, _ := mm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	mm2 := newModel2.(*tui.MainMenuModel)

	view := mm2.View()

	// Suggestions should appear inside the box (between │ borders)
	lines := strings.Split(view, "\n")
	foundSuggestion := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		// A suggestion line should be inside │...│ borders and contain a path
		if strings.Contains(trimmed, "/") && strings.HasPrefix(trimmed, "│") && strings.HasSuffix(trimmed, "│") {
			// Skip the input row itself (contains "Path:")
			if !strings.Contains(trimmed, "Path:") {
				foundSuggestion = true
			}
		}
	}
	if !foundSuggestion {
		t.Error("autocomplete suggestions should render inside the box borders")
	}

	// All box lines should have consistent width (including suggestion rows)
	assertBoxLinesConsistentWidth(t, view)
}

func TestMainMenu_View_DeleteMode(t *testing.T) {
	lipgloss.SetColorProfile(termenv.Ascii)
	m := tui.NewMainMenu(testProjects(), testAITools(), "claude", "animated")
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	mm := newModel.(*tui.MainMenuModel)

	view := mm.View()
	if !strings.Contains(view, "delete") && !strings.Contains(view, "Delete") {
		t.Error("Delete mode view should contain 'delete' or 'Delete'")
	}
}

func TestMainMenu_View_FeedbackMessage(t *testing.T) {
	lipgloss.SetColorProfile(termenv.Ascii)
	dir := t.TempDir()
	projFile := filepath.Join(dir, "projects")
	os.WriteFile(projFile, []byte("ghost-tab:/Users/jack/ghost-tab\nmy-app:/Users/jack/my-app\nwebsite:/Users/jack/website\n"), 0644)

	m := tui.NewMainMenu(testProjects(), testAITools(), "claude", "animated")
	m.SetProjectsFile(projFile)

	// Delete to trigger feedback
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	mm := newModel.(*tui.MainMenuModel)
	newModel2, _ := mm.Update(tea.KeyMsg{Type: tea.KeyEnter})
	mm2 := newModel2.(*tui.MainMenuModel)

	view := mm2.View()
	if !strings.Contains(view, "Deleted") {
		t.Error("View should show 'Deleted' feedback")
	}
}

func TestMainMenu_NumberKeyInstantSelect(t *testing.T) {
	projects := testProjects()
	m := tui.NewMainMenu(projects, testAITools(), "claude", "animated")

	// Press '2' — should instantly select "my-app" and quit
	newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})
	if cmd == nil {
		t.Fatal("Number key should return quit command")
	}
	mm := newModel.(*tui.MainMenuModel)
	result := mm.Result()
	if result == nil {
		t.Fatal("Number key should produce a result")
	}
	if result.Action != "select-project" {
		t.Errorf("Expected action 'select-project', got %q", result.Action)
	}
	if result.Name != "my-app" {
		t.Errorf("Expected name 'my-app', got %q", result.Name)
	}
	if result.Path != "/Users/jack/my-app" {
		t.Errorf("Expected path '/Users/jack/my-app', got %q", result.Path)
	}
}

func TestMainMenu_NumberKeyOutOfRange(t *testing.T) {
	projects := testProjects() // 3 projects
	m := tui.NewMainMenu(projects, testAITools(), "claude", "animated")

	// Press '9' — out of range, should do nothing (no quit, no result)
	newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'9'}})
	if cmd != nil {
		t.Error("Out-of-range number key should not return quit command")
	}
	mm := newModel.(*tui.MainMenuModel)
	if mm.Result() != nil {
		t.Error("Out-of-range number key should not produce a result")
	}
}

func TestMainMenu_SetSoundName(t *testing.T) {
	m := tui.NewMainMenu(nil, []string{"claude"}, "claude", "animated")
	m.SetSoundName("Bottle")
	if m.SoundName() != "Bottle" {
		t.Errorf("expected 'Bottle', got %q", m.SoundName())
	}
}

func TestMainMenu_SetSoundName_empty_means_off(t *testing.T) {
	m := tui.NewMainMenu(nil, []string{"claude"}, "claude", "animated")
	m.SetSoundName("")
	if m.SoundName() != "" {
		t.Errorf("expected empty string, got %q", m.SoundName())
	}
}

func TestMainMenu_CycleSoundName_forward(t *testing.T) {
	m := tui.NewMainMenu(nil, []string{"claude"}, "claude", "animated")
	m.SetSoundName("Bottle")
	m.CycleSoundName()
	name := m.SoundName()
	if name == "Bottle" {
		t.Error("expected sound to change after cycling forward")
	}
	if name == "" {
		t.Error("first forward cycle from Bottle should not jump to Off")
	}
}

func TestMainMenu_CycleSoundNameReverse(t *testing.T) {
	m := tui.NewMainMenu(nil, []string{"claude"}, "claude", "animated")
	m.SetSoundName("Bottle")
	m.CycleSoundNameReverse()
	name := m.SoundName()
	if name == "Bottle" {
		t.Error("expected sound to change after cycling backward")
	}
}

func TestMainMenu_CycleSoundName_wraps_through_off(t *testing.T) {
	m := tui.NewMainMenu(nil, []string{"claude"}, "claude", "animated")
	m.SetSoundName("Tink")
	m.CycleSoundName()
	if m.SoundName() != "" {
		t.Errorf("expected Off (empty) after last sound, got %q", m.SoundName())
	}
	m.CycleSoundName()
	if m.SoundName() == "" {
		t.Error("expected to wrap to first sound after Off")
	}
}

func TestMainMenu_CycleSoundNameReverse_wraps_through_off(t *testing.T) {
	m := tui.NewMainMenu(nil, []string{"claude"}, "claude", "animated")
	m.SetSoundName("Basso")
	m.CycleSoundNameReverse()
	if m.SoundName() != "" {
		t.Errorf("expected Off (empty) after first sound reversed, got %q", m.SoundName())
	}
	m.CycleSoundNameReverse()
	if m.SoundName() != "Tink" {
		t.Errorf("expected 'Tink' after Off reversed, got %q", m.SoundName())
	}
}

func TestMainMenu_SoundNameInResult(t *testing.T) {
	projects := testProjects()
	m := tui.NewMainMenu(projects, []string{"claude"}, "claude", "animated")
	m.SetSize(80, 30)
	m.SetSoundName("Bottle")
	m.EnterSettings()
	m.CycleSoundName()
	m.ExitSettings()
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	result := m.Result()
	if result == nil {
		t.Fatal("expected a result")
	}
	if result.SoundName == nil {
		t.Fatal("expected sound_name to be set when changed")
	}
}

func TestMainMenu_NoSoundNameInResultWhenUnchanged(t *testing.T) {
	projects := testProjects()
	m := tui.NewMainMenu(projects, []string{"claude"}, "claude", "animated")
	m.SetSize(80, 30)
	m.SetSoundName("Bottle")
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	result := m.Result()
	if result == nil {
		t.Fatal("expected a result")
	}
	if result.SoundName != nil {
		t.Errorf("expected nil sound_name when unchanged, got %q", *result.SoundName)
	}
}

func TestMainMenu_SoundNameInResultOnQuit(t *testing.T) {
	projects := testProjects()
	m := tui.NewMainMenu(projects, []string{"claude"}, "claude", "animated")
	m.SetSize(80, 30)
	m.SetSoundName("Bottle")
	m.EnterSettings()
	m.CycleSoundName()
	m.ExitSettings()
	// Quit via Esc instead of selecting a project
	m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	result := m.Result()
	if result == nil {
		t.Fatal("expected a result on quit")
	}
	if result.Action != "quit" {
		t.Fatalf("expected action=quit, got %q", result.Action)
	}
	if result.SoundName == nil {
		t.Fatal("expected sound_name to be set when changed and quit via Esc")
	}
}

func TestMainMenu_SetSettingsFile(t *testing.T) {
	m := tui.NewMainMenu(nil, []string{"claude"}, "claude", "animated")
	m.SetSettingsFile("/tmp/test-settings")
	if m.SettingsFile() != "/tmp/test-settings" {
		t.Errorf("expected '/tmp/test-settings', got %q", m.SettingsFile())
	}
}

func TestMainMenu_SetSoundFile(t *testing.T) {
	m := tui.NewMainMenu(nil, []string{"claude"}, "claude", "animated")
	m.SetSoundFile("/tmp/test-features.json")
	if m.SoundFile() != "/tmp/test-features.json" {
		t.Errorf("expected '/tmp/test-features.json', got %q", m.SoundFile())
	}
}

func TestMainMenu_SettingsViewShowsSoundName(t *testing.T) {
	m := tui.NewMainMenu(nil, []string{"claude"}, "claude", "animated")
	m.SetSize(80, 30)
	m.SetSoundName("Glass")
	m.EnterSettings()
	view := m.View()
	if !strings.Contains(view, "Glass") {
		t.Error("settings view should show sound name 'Glass'")
	}
	if !strings.Contains(view, "Sound") {
		t.Error("settings view should show 'Sound' label")
	}
}

func TestMainMenu_SettingsViewShowsOff(t *testing.T) {
	m := tui.NewMainMenu(nil, []string{"claude"}, "claude", "animated")
	m.SetSize(80, 30)
	m.SetSoundName("")
	m.EnterSettings()
	view := m.View()
	if !strings.Contains(view, "[Off]") {
		t.Error("settings view should show '[Off]' when sound is disabled")
	}
}

func TestMainMenu_CycleGhostDisplay_PersistsToFile(t *testing.T) {
	dir := t.TempDir()
	settingsFile := filepath.Join(dir, "settings")
	os.WriteFile(settingsFile, []byte("ghost_display=animated\n"), 0644)

	m := tui.NewMainMenu(nil, []string{"claude"}, "claude", "animated")
	m.SetSettingsFile(settingsFile)
	m.CycleGhostDisplay() // animated -> static

	data, err := os.ReadFile(settingsFile)
	if err != nil {
		t.Fatalf("failed to read settings file: %v", err)
	}
	if !strings.Contains(string(data), "ghost_display=static") {
		t.Errorf("expected ghost_display=static in file, got %q", string(data))
	}
}

func TestMainMenu_CycleGhostDisplay_CreatesFileIfMissing(t *testing.T) {
	dir := t.TempDir()
	settingsFile := filepath.Join(dir, "settings")

	m := tui.NewMainMenu(nil, []string{"claude"}, "claude", "animated")
	m.SetSettingsFile(settingsFile)
	m.CycleGhostDisplay() // animated -> static

	data, err := os.ReadFile(settingsFile)
	if err != nil {
		t.Fatalf("settings file not created: %v", err)
	}
	if !strings.Contains(string(data), "ghost_display=static") {
		t.Errorf("expected ghost_display=static, got %q", string(data))
	}
}

func TestMainMenu_CycleGhostDisplayReverse_PersistsToFile(t *testing.T) {
	dir := t.TempDir()
	settingsFile := filepath.Join(dir, "settings")
	os.WriteFile(settingsFile, []byte("ghost_display=animated\n"), 0644)

	m := tui.NewMainMenu(nil, []string{"claude"}, "claude", "animated")
	m.SetSettingsFile(settingsFile)
	m.CycleGhostDisplayReverse() // animated -> none

	data, err := os.ReadFile(settingsFile)
	if err != nil {
		t.Fatalf("failed to read settings file: %v", err)
	}
	if !strings.Contains(string(data), "ghost_display=none") {
		t.Errorf("expected ghost_display=none in file, got %q", string(data))
	}
}

func TestMainMenu_CycleGhostDisplay_PreservesOtherSettings(t *testing.T) {
	dir := t.TempDir()
	settingsFile := filepath.Join(dir, "settings")
	os.WriteFile(settingsFile, []byte("ghost_display=animated\ntab_title=full\n"), 0644)

	m := tui.NewMainMenu(nil, []string{"claude"}, "claude", "animated")
	m.SetSettingsFile(settingsFile)
	m.CycleGhostDisplay() // animated -> static

	data, err := os.ReadFile(settingsFile)
	if err != nil {
		t.Fatalf("failed to read settings file: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "ghost_display=static") {
		t.Errorf("expected ghost_display=static in file, got %q", content)
	}
	if !strings.Contains(content, "tab_title=full") {
		t.Errorf("expected tab_title=full preserved in file, got %q", content)
	}
}

func TestMainMenu_CycleGhostDisplay_DoesNotPersistWithoutFile(t *testing.T) {
	dir := t.TempDir()
	settingsFile := filepath.Join(dir, "settings")

	m := tui.NewMainMenu(nil, []string{"claude"}, "claude", "animated")
	// Do NOT call SetSettingsFile
	m.CycleGhostDisplay()

	// File should NOT exist
	if _, err := os.Stat(settingsFile); err == nil {
		t.Error("settings file should not be created when no file path set")
	}
}

func TestMainMenu_CycleTabTitle_PersistsToFile(t *testing.T) {
	dir := t.TempDir()
	settingsFile := filepath.Join(dir, "settings")
	os.WriteFile(settingsFile, []byte("tab_title=full\n"), 0644)

	m := tui.NewMainMenu(nil, []string{"claude"}, "claude", "animated")
	m.SetSettingsFile(settingsFile)
	m.SetTabTitle("full")
	m.CycleTabTitle() // full -> project

	data, err := os.ReadFile(settingsFile)
	if err != nil {
		t.Fatalf("failed to read settings file: %v", err)
	}
	if !strings.Contains(string(data), "tab_title=project") {
		t.Errorf("expected tab_title=project in file, got %q", string(data))
	}
}

func TestMainMenu_CycleGhostDisplay_CreatesParentDirs(t *testing.T) {
	dir := t.TempDir()
	settingsFile := filepath.Join(dir, "config", "ghost-tab", "settings")

	m := tui.NewMainMenu(nil, []string{"claude"}, "claude", "animated")
	m.SetSettingsFile(settingsFile)
	m.CycleGhostDisplay() // animated -> static

	data, err := os.ReadFile(settingsFile)
	if err != nil {
		t.Fatalf("settings file not created with parent dirs: %v", err)
	}
	if !strings.Contains(string(data), "ghost_display=static") {
		t.Errorf("expected ghost_display=static, got %q", string(data))
	}
}

func TestMainMenu_CycleSoundName_persists_to_file(t *testing.T) {
	dir := t.TempDir()
	soundFile := filepath.Join(dir, "claude-features.json")
	os.WriteFile(soundFile, []byte(`{"sound":true,"sound_name":"Bottle"}`+"\n"), 0644)

	m := tui.NewMainMenu(nil, []string{"claude"}, "claude", "animated")
	m.SetSoundFile(soundFile)
	m.SetSoundName("Bottle")
	m.CycleSoundName() // Bottle -> Frog

	data, err := os.ReadFile(soundFile)
	if err != nil {
		t.Fatalf("failed to read sound file: %v", err)
	}
	content := string(data)
	if strings.Contains(content, "Bottle") {
		t.Error("expected sound to change from Bottle")
	}
	if !strings.Contains(content, `"sound":true`) && !strings.Contains(content, `"sound": true`) {
		t.Error("expected sound to be enabled")
	}
}

func TestMainMenu_CycleSoundName_to_off_persists(t *testing.T) {
	dir := t.TempDir()
	soundFile := filepath.Join(dir, "claude-features.json")
	os.WriteFile(soundFile, []byte(`{"sound":true,"sound_name":"Tink"}`+"\n"), 0644)

	m := tui.NewMainMenu(nil, []string{"claude"}, "claude", "animated")
	m.SetSoundFile(soundFile)
	m.SetSoundName("Tink")
	m.CycleSoundName() // Tink -> Off

	data, err := os.ReadFile(soundFile)
	if err != nil {
		t.Fatalf("failed to read sound file: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, `"sound":false`) && !strings.Contains(content, `"sound": false`) {
		t.Errorf("expected sound:false when cycled to Off, got %q", content)
	}
}

func TestMainMenu_CycleSoundName_creates_file_if_missing(t *testing.T) {
	dir := t.TempDir()
	soundFile := filepath.Join(dir, "claude-features.json")

	m := tui.NewMainMenu(nil, []string{"claude"}, "claude", "animated")
	m.SetSoundFile(soundFile)
	m.SetSoundName("")
	m.CycleSoundName() // Off -> Basso

	data, err := os.ReadFile(soundFile)
	if err != nil {
		t.Fatalf("sound file not created: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "Basso") {
		t.Errorf("expected Basso in file, got %q", content)
	}
}

func TestMainMenu_CycleSoundNameReverse_persists_to_file(t *testing.T) {
	dir := t.TempDir()
	soundFile := filepath.Join(dir, "claude-features.json")
	os.WriteFile(soundFile, []byte(`{"sound":true,"sound_name":"Blow"}`+"\n"), 0644)

	m := tui.NewMainMenu(nil, []string{"claude"}, "claude", "animated")
	m.SetSoundFile(soundFile)
	m.SetSoundName("Blow")
	m.CycleSoundNameReverse() // Blow -> Basso

	data, err := os.ReadFile(soundFile)
	if err != nil {
		t.Fatalf("failed to read sound file: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "Basso") {
		t.Errorf("expected Basso in file, got %q", content)
	}
}

func TestMainMenu_CycleSoundName_preserves_other_keys(t *testing.T) {
	dir := t.TempDir()
	soundFile := filepath.Join(dir, "claude-features.json")
	os.WriteFile(soundFile, []byte(`{"sound":true,"sound_name":"Bottle","other_key":"value"}`+"\n"), 0644)

	m := tui.NewMainMenu(nil, []string{"claude"}, "claude", "animated")
	m.SetSoundFile(soundFile)
	m.SetSoundName("Bottle")
	m.CycleSoundName() // Bottle -> Frog

	data, err := os.ReadFile(soundFile)
	if err != nil {
		t.Fatalf("failed to read sound file: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, `"other_key"`) {
		t.Errorf("expected other_key preserved in file, got %q", content)
	}
}

func TestMainMenu_CycleSoundName_does_not_persist_without_file(t *testing.T) {
	dir := t.TempDir()
	soundFile := filepath.Join(dir, "claude-features.json")

	m := tui.NewMainMenu(nil, []string{"claude"}, "claude", "animated")
	// Do NOT call SetSoundFile
	m.SetSoundName("Bottle")
	m.CycleSoundName()

	// File should NOT exist
	if _, err := os.Stat(soundFile); err == nil {
		t.Error("sound file should not be created when no file path set")
	}
}

func TestMainMenu_CycleSoundName_creates_parent_dirs(t *testing.T) {
	dir := t.TempDir()
	soundFile := filepath.Join(dir, "config", "ghost-tab", "claude-features.json")

	m := tui.NewMainMenu(nil, []string{"claude"}, "claude", "animated")
	m.SetSoundFile(soundFile)
	m.SetSoundName("")
	m.CycleSoundName() // Off -> Basso

	data, err := os.ReadFile(soundFile)
	if err != nil {
		t.Fatalf("sound file not created with parent dirs: %v", err)
	}
	if !strings.Contains(string(data), "Basso") {
		t.Errorf("expected Basso in file, got %q", string(data))
	}
}

func testProjectsWithWorktrees() []models.Project {
	return []models.Project{
		{
			Name: "ghost-tab",
			Path: "/Users/jack/ghost-tab",
			Worktrees: []models.Worktree{
				{Path: "/Users/jack/wt/feature-auth", Branch: "feature/auth"},
				{Path: "/Users/jack/wt/fix-cleanup", Branch: "fix/cleanup"},
			},
		},
		{Name: "my-app", Path: "/Users/jack/my-app"},
		{
			Name: "website",
			Path: "/Users/jack/website",
			Worktrees: []models.Worktree{
				{Path: "/Users/jack/wt/redesign", Branch: "redesign"},
			},
		},
	}
}

func TestMainMenu_TotalItemsWithExpanded(t *testing.T) {
	projects := testProjectsWithWorktrees()
	m := tui.NewMainMenu(projects, testAITools(), "claude", "animated")

	// No expansions: 3 projects + 4 actions = 7
	if m.TotalItems() != 7 {
		t.Errorf("unexpanded: expected 7, got %d", m.TotalItems())
	}

	// Expand first project (2 worktrees): 3 + 2 + 4 = 9
	m.ToggleWorktrees(0)
	if m.TotalItems() != 9 {
		t.Errorf("expanded first: expected 9, got %d", m.TotalItems())
	}

	// Expand third project too (1 worktree): 3 + 2 + 1 + 4 = 10
	m.ToggleWorktrees(2)
	if m.TotalItems() != 10 {
		t.Errorf("expanded first+third: expected 10, got %d", m.TotalItems())
	}

	// Collapse first project: 3 + 1 + 4 = 8
	m.ToggleWorktrees(0)
	if m.TotalItems() != 8 {
		t.Errorf("collapsed first, third expanded: expected 8, got %d", m.TotalItems())
	}
}

func TestMainMenu_NavigationWithWorktrees(t *testing.T) {
	projects := testProjectsWithWorktrees()
	m := tui.NewMainMenu(projects, testAITools(), "claude", "animated")

	// Expand first project
	m.ToggleWorktrees(0)
	// Items: [proj0, wt0, wt1, proj1, proj2, add, delete, open, plain]

	// Start at 0 (project 0)
	if m.SelectedItem() != 0 {
		t.Errorf("start: expected 0, got %d", m.SelectedItem())
	}

	// Move down into worktree entries
	m.MoveDown()
	if m.SelectedItem() != 1 {
		t.Errorf("after 1 down: expected 1 (wt0), got %d", m.SelectedItem())
	}

	m.MoveDown()
	if m.SelectedItem() != 2 {
		t.Errorf("after 2 down: expected 2 (wt1), got %d", m.SelectedItem())
	}

	// Next is project 1 (index 3)
	m.MoveDown()
	if m.SelectedItem() != 3 {
		t.Errorf("after 3 down: expected 3 (proj1), got %d", m.SelectedItem())
	}
}

func TestMainMenu_CollapseMovesSelectionToProject(t *testing.T) {
	projects := testProjectsWithWorktrees()
	m := tui.NewMainMenu(projects, testAITools(), "claude", "animated")

	// Expand first project, select worktree entry
	m.ToggleWorktrees(0)
	m.MoveDown() // wt0 (item 1)
	if m.SelectedItem() != 1 {
		t.Fatalf("expected on wt0 (item 1), got %d", m.SelectedItem())
	}

	// Collapse — selection should snap back to project 0
	m.ToggleWorktrees(0)
	if m.SelectedItem() != 0 {
		t.Errorf("after collapse: expected 0 (proj0), got %d", m.SelectedItem())
	}
}

func TestMainMenu_ToggleNoWorktrees(t *testing.T) {
	projects := testProjectsWithWorktrees()
	m := tui.NewMainMenu(projects, testAITools(), "claude", "animated")

	// Project 1 has no worktrees — toggle should be a no-op
	before := m.TotalItems()
	m.ToggleWorktrees(1)
	after := m.TotalItems()
	if before != after {
		t.Errorf("toggle on no-worktree project changed total: %d -> %d", before, after)
	}
}
