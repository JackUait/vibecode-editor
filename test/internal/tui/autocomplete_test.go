package tui_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

func TestPathSuggestionProvider_BarePrefix_SearchesHome(t *testing.T) {
	// Create a temp "home" dir with a known subdirectory
	home := t.TempDir()
	os.MkdirAll(filepath.Join(home, "Projects"), 0755)
	os.MkdirAll(filepath.Join(home, "Documents"), 0755)
	t.Setenv("HOME", home)

	provider := tui.PathSuggestionProvider(8)

	// Bare prefix (no / or ~/) should search home directory
	suggestions := provider("Proj")
	found := false
	for _, s := range suggestions {
		if s == "~/Projects/" {
			found = true
		}
	}
	if !found {
		t.Errorf("bare prefix 'Proj' should find ~/Projects/ in home dir, got %v", suggestions)
	}
}

func TestPathSuggestionProvider_AbsolutePrefix_IncludesHomeResults(t *testing.T) {
	home := t.TempDir()
	os.MkdirAll(filepath.Join(home, "Packages"), 0755)
	t.Setenv("HOME", home)

	provider := tui.PathSuggestionProvider(20)

	// "/p" matches root dirs AND should also include ~/Packages/
	suggestions := provider("/p")
	foundHome := false
	foundRoot := false
	for _, s := range suggestions {
		if s == "~/Packages/" {
			foundHome = true
		}
		if strings.HasPrefix(s, "/") {
			foundRoot = true
		}
	}
	if !foundHome {
		t.Errorf("'/p' should include ~/Packages/ from home, got %v", suggestions)
	}
	if !foundRoot {
		t.Errorf("'/p' should also include root matches, got %v", suggestions)
	}
}

func TestPathSuggestionProvider_NonexistentDir(t *testing.T) {
	provider := tui.PathSuggestionProvider(8)
	suggestions := provider("/nonexistent/path/xyz")
	if len(suggestions) != 0 {
		t.Errorf("expected 0 for nonexistent dir, got %d", len(suggestions))
	}
}

func TestAutocompleteModel_InitialState(t *testing.T) {
	provider := func(input string) []string {
		return []string{"a/", "b/", "c/"}
	}
	m := tui.NewAutocomplete(provider, 8)
	if m.ShowSuggestions() {
		t.Error("suggestions should not show initially")
	}
	if m.Selected() != 0 {
		t.Errorf("initial selection should be 0, got %d", m.Selected())
	}
}

func TestAutocompleteModel_SuggestionsAppear(t *testing.T) {
	provider := func(input string) []string {
		if input == "foo" {
			return []string{"foo-bar/", "foo-baz/"}
		}
		return nil
	}
	m := tui.NewAutocomplete(provider, 8)

	m.SetInput("foo")
	m.RefreshSuggestions()

	if !m.ShowSuggestions() {
		t.Error("suggestions should show after setting input")
	}
	if len(m.Suggestions()) != 2 {
		t.Errorf("expected 2 suggestions, got %d", len(m.Suggestions()))
	}
}

func TestAutocompleteModel_NavigateDown(t *testing.T) {
	provider := func(input string) []string {
		return []string{"a/", "b/", "c/"}
	}
	m := tui.NewAutocomplete(provider, 8)
	m.SetInput("x")
	m.RefreshSuggestions()

	m.MoveDown()
	if m.Selected() != 1 {
		t.Errorf("after MoveDown: expected 1, got %d", m.Selected())
	}
}

func TestAutocompleteModel_NavigateUpWraps(t *testing.T) {
	provider := func(input string) []string {
		return []string{"a/", "b/", "c/"}
	}
	m := tui.NewAutocomplete(provider, 8)
	m.SetInput("x")
	m.RefreshSuggestions()

	m.MoveUp()
	if m.Selected() != 2 {
		t.Errorf("after MoveUp from 0: expected 2 (wrap), got %d", m.Selected())
	}
}

func TestAutocompleteModel_AcceptSuggestion(t *testing.T) {
	provider := func(input string) []string {
		if input == "~/co" {
			return []string{"~/code/", "~/config/"}
		}
		return nil
	}
	m := tui.NewAutocomplete(provider, 8)
	m.SetInput("~/co")
	m.RefreshSuggestions()
	m.MoveDown() // select "~/config/"

	accepted := m.AcceptSelected()
	if accepted != "~/config/" {
		t.Errorf("expected ~/config/, got %q", accepted)
	}
}

func TestAutocompleteModel_DismissClearsSuggestions(t *testing.T) {
	provider := func(input string) []string {
		return []string{"a/", "b/"}
	}
	m := tui.NewAutocomplete(provider, 8)
	m.SetInput("x")
	m.RefreshSuggestions()

	m.Dismiss()
	if m.ShowSuggestions() {
		t.Error("suggestions should be hidden after dismiss")
	}
}
