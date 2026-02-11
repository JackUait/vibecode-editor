package tui_test

import (
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/jackuait/ghost-tab/internal/tui"
)

func TestThemeForTool_Claude(t *testing.T) {
	theme := tui.ThemeForTool("claude")

	if theme.Name != "claude" {
		t.Errorf("Expected Name to be 'claude', got %q", theme.Name)
	}
	if theme.Primary != lipgloss.Color("209") {
		t.Errorf("Expected Primary to be '209', got %q", theme.Primary)
	}
}

func TestThemeForTool_Codex(t *testing.T) {
	theme := tui.ThemeForTool("codex")

	if theme.Primary != lipgloss.Color("114") {
		t.Errorf("Expected Primary to be '114', got %q", theme.Primary)
	}
}

func TestThemeForTool_Copilot(t *testing.T) {
	theme := tui.ThemeForTool("copilot")

	if theme.Primary != lipgloss.Color("141") {
		t.Errorf("Expected Primary to be '141', got %q", theme.Primary)
	}
}

func TestThemeForTool_OpenCode(t *testing.T) {
	theme := tui.ThemeForTool("opencode")

	if theme.Primary != lipgloss.Color("250") {
		t.Errorf("Expected Primary to be '250', got %q", theme.Primary)
	}
}

func TestThemeForTool_Unknown(t *testing.T) {
	theme := tui.ThemeForTool("unknown-tool")

	if theme.Name != "claude" {
		t.Errorf("Expected unknown tool to fall back to claude theme, got Name=%q", theme.Name)
	}
	if theme.Primary != lipgloss.Color("209") {
		t.Errorf("Expected unknown tool Primary to be '209' (claude), got %q", theme.Primary)
	}
}

func TestAllThemes(t *testing.T) {
	expectedNames := map[string]string{
		"claude":   "claude",
		"codex":    "codex",
		"copilot":  "copilot",
		"opencode": "opencode",
	}

	for tool, expectedName := range expectedNames {
		theme := tui.ThemeForTool(tool)
		if theme.Name != expectedName {
			t.Errorf("ThemeForTool(%q): expected Name=%q, got %q", tool, expectedName, theme.Name)
		}
	}
}
