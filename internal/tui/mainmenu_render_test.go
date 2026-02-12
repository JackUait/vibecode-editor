package tui

import (
	"strings"
	"testing"

	"github.com/jackuait/ghost-tab/internal/models"
)

func newTestMenu() *MainMenuModel {
	projects := []models.Project{
		{Name: "test-proj", Path: "/tmp/test-proj"},
	}
	m := NewMainMenu(projects, []string{"claude", "codex"}, "claude", "animated")
	m.width = 100
	m.height = 40
	return m
}

func TestMenuBox_AIToolRightAligned(t *testing.T) {
	m := newTestMenu()
	box := m.renderMenuBox()
	lines := strings.Split(box, "\n")

	// Title row is the second line (index 1), after top border
	if len(lines) < 2 {
		t.Fatal("renderMenuBox produced fewer than 2 lines")
	}
	titleRow := lines[1]

	// The AI tool display name should appear after Ghost Tab, not immediately adjacent
	// With right-alignment, there should be spaces between "Ghost Tab" and the AI tool
	if !strings.Contains(titleRow, "Ghost Tab") {
		t.Error("title row missing 'Ghost Tab'")
	}
	if !strings.Contains(titleRow, "Claude Code") {
		t.Error("title row missing 'Claude Code'")
	}

	// Verify right-alignment: there should be multiple spaces between Ghost Tab and the ◂ arrow
	// Strip ANSI codes to check raw layout
	raw := stripAnsi(titleRow)
	ghostIdx := strings.Index(raw, "Ghost Tab")
	arrowIdx := strings.Index(raw, "◂")
	if ghostIdx < 0 || arrowIdx < 0 {
		t.Fatal("could not find Ghost Tab or ◂ in stripped title row")
	}
	// With right-alignment, there should be significant padding between the end of
	// "Ghost Tab" and "◂" (more than just a single space)
	gap := raw[ghostIdx+len("Ghost Tab") : arrowIdx]
	if len(strings.TrimSpace(gap)) != 0 {
		t.Errorf("expected only whitespace between Ghost Tab and ◂, got %q", gap)
	}
	if len(gap) < 5 {
		t.Errorf("expected at least 5 chars padding for right-alignment, got %d: %q", len(gap), gap)
	}
}

func TestMenuBox_HelpTextPresent(t *testing.T) {
	m := newTestMenu()
	box := m.renderMenuBox()

	raw := stripAnsi(box)
	// Help text should contain navigation hints
	if !strings.Contains(raw, "navigate") {
		t.Error("help text missing 'navigate'")
	}
	if !strings.Contains(raw, "AI tool") {
		t.Error("help text missing 'AI tool' (expected when multiple AI tools available)")
	}
	if !strings.Contains(raw, "select") {
		t.Error("help text missing 'select'")
	}
}

func TestSettingsBox_StateRightAligned(t *testing.T) {
	m := newTestMenu()
	m.settingsMode = true
	m.tabTitle = "full"
	box := m.renderSettingsBox()
	raw := stripAnsi(box)
	lines := strings.Split(raw, "\n")

	// Find lines containing "Ghost Display" and "Tab Title"
	for _, line := range lines {
		if strings.Contains(line, "Ghost Display") && strings.Contains(line, "[Animated]") {
			// State text should be right-aligned: ends near the right border
			// The line should end with the state text followed by the border character
			trimmed := strings.TrimRight(line, " ")
			idx := strings.Index(trimmed, "[Animated]")
			if idx < 0 {
				t.Fatal("could not find [Animated] in Ghost Display line")
			}
			afterState := trimmed[idx+len("[Animated]"):]
			// After state text, only a small gap + border char should remain
			cleaned := strings.TrimSpace(afterState)
			if cleaned != "│" {
				t.Errorf("expected only border after [Animated], got %q", afterState)
			}
			// Between label and state there should be significant padding
			labelEnd := strings.Index(line, "Ghost Display") + len("Ghost Display")
			gap := line[labelEnd:idx]
			if len(strings.TrimSpace(gap)) != 0 {
				t.Errorf("expected only whitespace between label and state, got %q", gap)
			}
			if len(gap) < 5 {
				t.Errorf("expected at least 5 chars gap for right-alignment, got %d", len(gap))
			}
		}
	}
}

// stripAnsi removes ANSI escape sequences from a string.
func stripAnsi(s string) string {
	var result strings.Builder
	i := 0
	for i < len(s) {
		if s[i] == '\x1b' {
			// Skip until 'm' (end of ANSI sequence)
			for i < len(s) && s[i] != 'm' {
				i++
			}
			if i < len(s) {
				i++ // skip the 'm'
			}
			continue
		}
		result.WriteByte(s[i])
		i++
	}
	return result.String()
}
