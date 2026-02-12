package tui_test

import (
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
		result := mm.Result()

		if result == nil {
			t.Fatal("Expected result, got nil")
		}
		if result.Action != "add-project" {
			t.Errorf("Expected action 'add-project', got %q", result.Action)
		}
		if result.AITool != "claude" {
			t.Errorf("Expected ai_tool 'claude', got %q", result.AITool)
		}
	})

	t.Run("delete-project", func(t *testing.T) {
		m := tui.NewMainMenu(projects, testAITools(), "claude", "animated")
		for i := 0; i < 4; i++ {
			m.MoveDown()
		}
		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
		mm := newModel.(*tui.MainMenuModel)
		result := mm.Result()

		if result == nil {
			t.Fatal("Expected result, got nil")
		}
		if result.Action != "delete-project" {
			t.Errorf("Expected action 'delete-project', got %q", result.Action)
		}
	})

	t.Run("open-once", func(t *testing.T) {
		m := tui.NewMainMenu(projects, testAITools(), "claude", "animated")
		for i := 0; i < 5; i++ {
			m.MoveDown()
		}
		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
		mm := newModel.(*tui.MainMenuModel)
		result := mm.Result()

		if result == nil {
			t.Fatal("Expected result, got nil")
		}
		if result.Action != "open-once" {
			t.Errorf("Expected action 'open-once', got %q", result.Action)
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
		result := mm.Result()

		if result == nil {
			t.Fatal("Expected result for 'a' shortcut, got nil")
		}
		if result.Action != "add-project" {
			t.Errorf("Expected 'add-project', got %q", result.Action)
		}
	})

	t.Run("A_shortcut", func(t *testing.T) {
		m := tui.NewMainMenu(projects, testAITools(), "claude", "animated")
		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'A'}})
		mm := newModel.(*tui.MainMenuModel)
		result := mm.Result()

		if result == nil {
			t.Fatal("Expected result for 'A' shortcut, got nil")
		}
		if result.Action != "add-project" {
			t.Errorf("Expected 'add-project', got %q", result.Action)
		}
	})

	t.Run("d_shortcut", func(t *testing.T) {
		m := tui.NewMainMenu(projects, testAITools(), "claude", "animated")
		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
		mm := newModel.(*tui.MainMenuModel)
		result := mm.Result()

		if result == nil {
			t.Fatal("Expected result for 'd' shortcut, got nil")
		}
		if result.Action != "delete-project" {
			t.Errorf("Expected 'delete-project', got %q", result.Action)
		}
	})

	t.Run("o_shortcut", func(t *testing.T) {
		m := tui.NewMainMenu(projects, testAITools(), "claude", "animated")
		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}})
		mm := newModel.(*tui.MainMenuModel)
		result := mm.Result()

		if result == nil {
			t.Fatal("Expected result for 'o' shortcut, got nil")
		}
		if result.Action != "open-once" {
			t.Errorf("Expected 'open-once', got %q", result.Action)
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

	// Index 0 = Add, 1 = Delete, 2 = Open Once, 3 = Plain Terminal
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	mm := newModel.(*tui.MainMenuModel)
	result := mm.Result()

	if result == nil {
		t.Fatal("Expected result, got nil")
	}
	if result.Action != "add-project" {
		t.Errorf("Expected 'add-project' at index 0 with no projects, got %q", result.Action)
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

func TestMainMenu_BobOffsets(t *testing.T) {
	offsets := tui.BobOffsets()
	if len(offsets) != 14 {
		t.Errorf("expected 14 bob offsets, got %d", len(offsets))
	}
	expected := []int{0, 0, 0, 0, 1, 1, 1, 1, 1, 1, 0, 0, 0, 0}
	for i, v := range offsets {
		if v != expected[i] {
			t.Errorf("bob offset[%d]: expected %d, got %d", i, expected[i], v)
		}
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

func TestMainMenu_SettingsKeyBReturns(t *testing.T) {
	m := tui.NewMainMenu(nil, []string{"claude"}, "claude", "animated")
	m.SetSize(80, 30)
	m.EnterSettings()

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}}
	newModel, _ := m.Update(msg)
	mm := newModel.(*tui.MainMenuModel)

	if mm.InSettingsMode() {
		t.Error("B in settings should return to main menu")
	}
}

func TestMainMenu_SettingsKeyACycles(t *testing.T) {
	m := tui.NewMainMenu(nil, []string{"claude"}, "claude", "animated")
	m.SetSize(80, 30)
	m.EnterSettings()

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}}
	newModel, _ := m.Update(msg)
	mm := newModel.(*tui.MainMenuModel)

	if mm.GhostDisplay() != "static" {
		t.Errorf("expected static after pressing A, got %s", mm.GhostDisplay())
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

	// Pressing A (cycle) should not produce a quit command
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}}
	_, cmd := m.Update(msg)

	if cmd != nil {
		t.Error("pressing A in settings should not produce a quit command")
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
	if !strings.Contains(view, "back") {
		t.Error("settings help row should mention 'back'")
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
	// Advance bob tick -- should also advance Zzz
	initialFrame := m.ZzzFrame()
	m.Update(tui.NewBobTickMsg())
	if m.ZzzFrame() != initialFrame+1 {
		t.Errorf("Zzz frame should advance on bob tick when sleeping, expected %d got %d", initialFrame+1, m.ZzzFrame())
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
