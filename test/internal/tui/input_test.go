package tui_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

func TestPathSuggestions_DirectoryCompletion(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "project-a"), 0755)
	os.MkdirAll(filepath.Join(dir, "project-b"), 0755)
	os.MkdirAll(filepath.Join(dir, "other"), 0755)

	suggestions := tui.GetPathSuggestions(dir + "/proj")
	if len(suggestions) != 2 {
		t.Errorf("expected 2 suggestions for 'proj', got %d: %v", len(suggestions), suggestions)
	}
}

func TestPathSuggestions_MaxEight(t *testing.T) {
	dir := t.TempDir()
	for i := 0; i < 12; i++ {
		os.MkdirAll(filepath.Join(dir, fmt.Sprintf("dir%02d", i)), 0755)
	}

	suggestions := tui.GetPathSuggestions(dir + "/")
	if len(suggestions) > 8 {
		t.Errorf("expected max 8 suggestions, got %d", len(suggestions))
	}
}

func TestPathSuggestions_CaseInsensitive(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "MyProject"), 0755)

	suggestions := tui.GetPathSuggestions(dir + "/myp")
	if len(suggestions) != 1 {
		t.Errorf("expected 1 case-insensitive match, got %d: %v", len(suggestions), suggestions)
	}
}

func TestPathSuggestions_EmptyInput(t *testing.T) {
	suggestions := tui.GetPathSuggestions("")
	if len(suggestions) != 0 {
		t.Errorf("expected 0 suggestions for empty input, got %d", len(suggestions))
	}
}

func TestPathSuggestions_NonexistentDir(t *testing.T) {
	suggestions := tui.GetPathSuggestions("/nonexistent/path/foo")
	if len(suggestions) != 0 {
		t.Errorf("expected 0 suggestions for nonexistent dir, got %d", len(suggestions))
	}
}

func TestPathSuggestions_TrailingSlash(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "subdir"), 0755)

	suggestions := tui.GetPathSuggestions(dir + "/")
	if len(suggestions) < 1 {
		t.Errorf("expected at least 1 suggestion for trailing slash, got %d", len(suggestions))
	}
	// All suggestions should end with /
	for _, s := range suggestions {
		if !strings.HasSuffix(s, "/") {
			t.Errorf("suggestion %q should end with /", s)
		}
	}
}
