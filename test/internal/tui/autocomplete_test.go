package tui_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/jackuait/ghost-tab/internal/tui"
)

func TestPathSuggestionProvider_EmptyDefaultsToHome(t *testing.T) {
	provider := tui.PathSuggestionProvider(8)
	suggestions := provider("")
	if len(suggestions) == 0 {
		t.Error("expected suggestions for empty input (should default to ~/)")
	}
	for _, s := range suggestions {
		if s[0] != '~' {
			t.Errorf("suggestion %q should start with ~", s)
		}
	}
}

func TestPathSuggestionProvider_SortedAlphabetically(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "zebra"), 0755)
	os.MkdirAll(filepath.Join(dir, "alpha"), 0755)
	os.MkdirAll(filepath.Join(dir, "mango"), 0755)

	provider := tui.PathSuggestionProvider(8)
	suggestions := provider(dir + "/")

	if len(suggestions) < 3 {
		t.Fatalf("expected at least 3 suggestions, got %d", len(suggestions))
	}

	for i := 1; i < len(suggestions); i++ {
		if suggestions[i] < suggestions[i-1] {
			t.Errorf("suggestions not sorted: %q comes after %q", suggestions[i], suggestions[i-1])
		}
	}
}

func TestPathSuggestionProvider_GlobMatching(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "my-project"), 0755)
	os.MkdirAll(filepath.Join(dir, "your-project"), 0755)
	os.MkdirAll(filepath.Join(dir, "my-app"), 0755)

	provider := tui.PathSuggestionProvider(8)

	suggestions := provider(dir + "/project")
	if len(suggestions) < 2 {
		t.Errorf("expected at least 2 glob matches for 'project', got %d: %v",
			len(suggestions), suggestions)
	}
}

func TestPathSuggestionProvider_MaxResults(t *testing.T) {
	dir := t.TempDir()
	for i := 0; i < 12; i++ {
		os.MkdirAll(filepath.Join(dir, fmt.Sprintf("dir%02d", i)), 0755)
	}

	provider := tui.PathSuggestionProvider(5)
	suggestions := provider(dir + "/")
	if len(suggestions) > 5 {
		t.Errorf("expected max 5 suggestions, got %d", len(suggestions))
	}
}

func TestPathSuggestionProvider_HiddenDirsFiltered(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".hidden"), 0755)
	os.MkdirAll(filepath.Join(dir, "visible"), 0755)

	provider := tui.PathSuggestionProvider(8)
	suggestions := provider(dir + "/")

	for _, s := range suggestions {
		base := filepath.Base(s[:len(s)-1])
		if base[0] == '.' {
			t.Errorf("hidden dir %q should be filtered", s)
		}
	}
}

func TestPathSuggestionProvider_CaseInsensitive(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "MyProject"), 0755)

	provider := tui.PathSuggestionProvider(8)
	suggestions := provider(dir + "/myp")

	if len(suggestions) != 1 {
		t.Errorf("expected 1 case-insensitive match, got %d: %v",
			len(suggestions), suggestions)
	}
}

func TestPathSuggestionProvider_TrailingSlash(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "subdir"), 0755)

	provider := tui.PathSuggestionProvider(8)
	suggestions := provider(dir + "/")

	for _, s := range suggestions {
		if s[len(s)-1] != '/' {
			t.Errorf("suggestion %q should end with /", s)
		}
	}
}

func TestPathSuggestionProvider_NonexistentDir(t *testing.T) {
	provider := tui.PathSuggestionProvider(8)
	suggestions := provider("/nonexistent/path/xyz")
	if len(suggestions) != 0 {
		t.Errorf("expected 0 for nonexistent dir, got %d", len(suggestions))
	}
}
