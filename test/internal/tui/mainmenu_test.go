package tui_test

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jackuait/ghost-tab/internal/models"
	"github.com/jackuait/ghost-tab/internal/tui"
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

	t.Run("s_shortcut", func(t *testing.T) {
		m := tui.NewMainMenu(projects, testAITools(), "claude", "animated")
		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
		mm := newModel.(*tui.MainMenuModel)
		result := mm.Result()

		if result == nil {
			t.Fatal("Expected result for 's' shortcut, got nil")
		}
		if result.Action != "settings" {
			t.Errorf("Expected 'settings', got %q", result.Action)
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
	m := tui.NewMainMenu(testProjects(), testAITools(), "claude", "animated")
	cmd := m.Init()
	if cmd != nil {
		t.Error("Init() should return nil for now")
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
