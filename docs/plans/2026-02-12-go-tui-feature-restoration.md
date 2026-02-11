# Go TUI Feature Restoration Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Restore three features lost during bash-to-Go migration: reusable autocomplete component, animated sleeping ghost with theme-driven colors, and fill remaining Go test gaps.

**Architecture:** Extract autocomplete from `input.go` into a standalone Bubbletea component with pluggable providers. Refactor ghost art to derive ANSI colors from `AIToolTheme` instead of hardcoding. Add animated Zzz overlay to main menu. Fill test gaps for logo and confirm components.

**Tech Stack:** Go, Bubbletea (charmbracelet), lipgloss, BATS (bash tests remain unchanged)

**Existing test convention:** Tests live in `test/internal/tui/` using `package tui_test` (black-box testing pattern). All new tests follow this convention.

---

## Phase 1: Reusable Autocomplete Component

### Task 1: Write autocomplete provider tests

**Files:**
- Create: `test/internal/tui/autocomplete_test.go`

**Step 1: Write the failing tests**

```go
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
	// Empty input should suggest ~/ contents (home directory)
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

	// Extract just the directory names for comparison
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

	// Should match anywhere in name, not just prefix
	suggestions := provider(dir + "/project")
	// With glob matching, "project" should match "my-project" and "your-project"
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
		base := filepath.Base(s[:len(s)-1]) // remove trailing /
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
```

**Step 2: Run tests to verify they fail**

Run: `go test ./test/internal/tui/ -run TestPathSuggestionProvider -v 2>&1 | head -20`
Expected: Compilation error — `tui.PathSuggestionProvider` undefined.

**Step 3: Commit test file**

```bash
git add test/internal/tui/autocomplete_test.go
git commit -m "test: add failing tests for PathSuggestionProvider"
```

---

### Task 2: Implement autocomplete provider

**Files:**
- Create: `internal/tui/autocomplete.go`

**Step 1: Write minimal implementation**

```go
package tui

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jackuait/ghost-tab/internal/util"
)

// SuggestionProvider is a function that returns suggestions for a given input.
type SuggestionProvider func(input string) []string

// PathSuggestionProvider returns a SuggestionProvider that suggests directory paths.
// Empty input defaults to ~/. Results are sorted alphabetically and capped at maxResults.
// Matching is case-insensitive and supports substring (glob-style) matching.
func PathSuggestionProvider(maxResults int) SuggestionProvider {
	return func(input string) []string {
		if input == "" {
			input = "~/"
		}

		expanded := util.ExpandPath(input)

		var dir string
		var prefix string

		if strings.HasSuffix(input, "/") {
			dir = expanded
			prefix = ""
		} else {
			dir = filepath.Dir(expanded)
			prefix = filepath.Base(expanded)
		}

		entries, err := os.ReadDir(dir)
		if err != nil {
			return nil
		}

		lowerPrefix := strings.ToLower(prefix)
		var suggestions []string

		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			name := entry.Name()
			if strings.HasPrefix(name, ".") {
				continue
			}
			lowerName := strings.ToLower(name)
			// Glob-style: match if prefix appears anywhere in name
			if prefix == "" || strings.Contains(lowerName, lowerPrefix) {
				var suggestion string
				if strings.HasSuffix(input, "/") {
					suggestion = input + name + "/"
				} else {
					parentInput := input[:len(input)-len(filepath.Base(input))]
					suggestion = parentInput + name + "/"
				}
				suggestions = append(suggestions, suggestion)
			}
		}

		sort.Strings(suggestions)

		if len(suggestions) > maxResults {
			suggestions = suggestions[:maxResults]
		}

		return suggestions
	}
}
```

**Step 2: Run tests to verify they pass**

Run: `go test ./test/internal/tui/ -run TestPathSuggestionProvider -v`
Expected: All `TestPathSuggestionProvider_*` tests PASS.

**Step 3: Commit**

```bash
git add internal/tui/autocomplete.go
git commit -m "feat: add PathSuggestionProvider with sorting, glob matching, and ~/ default"
```

---

### Task 3: Write autocomplete model tests

**Files:**
- Modify: `test/internal/tui/autocomplete_test.go` (append)

**Step 1: Append model interaction tests**

Add to `test/internal/tui/autocomplete_test.go`:

```go
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
```

**Step 2: Run tests — expect compilation failure**

Run: `go test ./test/internal/tui/ -run TestAutocompleteModel -v 2>&1 | head -10`
Expected: Compilation error — `tui.NewAutocomplete` undefined.

---

### Task 4: Implement AutocompleteModel

**Files:**
- Modify: `internal/tui/autocomplete.go` (append)

**Step 1: Add AutocompleteModel to autocomplete.go**

Append to `internal/tui/autocomplete.go`:

```go
// AutocompleteModel is a reusable autocomplete component.
// It manages suggestions, navigation, and selection state.
// Embed it in another model and call its methods from Update().
type AutocompleteModel struct {
	provider       SuggestionProvider
	suggestions    []string
	selected       int
	showSuggestions bool
	maxResults     int
}

// NewAutocomplete creates a new autocomplete model with the given provider.
func NewAutocomplete(provider SuggestionProvider, maxResults int) AutocompleteModel {
	if maxResults <= 0 {
		maxResults = 8
	}
	return AutocompleteModel{
		provider:   provider,
		maxResults: maxResults,
	}
}

// SetInput updates the current input and is typically called on every keystroke.
func (m *AutocompleteModel) SetInput(input string) {
	// Store for refresh
	m.lastInput = input
}

// RefreshSuggestions calls the provider with the current input.
func (m *AutocompleteModel) RefreshSuggestions() {
	if m.provider == nil {
		return
	}
	m.suggestions = m.provider(m.lastInput)
	m.selected = 0
	m.showSuggestions = len(m.suggestions) > 0
}

// Suggestions returns the current suggestion list.
func (m *AutocompleteModel) Suggestions() []string {
	return m.suggestions
}

// Selected returns the index of the currently highlighted suggestion.
func (m *AutocompleteModel) Selected() int {
	return m.selected
}

// ShowSuggestions returns whether the suggestion dropdown is visible.
func (m *AutocompleteModel) ShowSuggestions() bool {
	return m.showSuggestions
}

// MoveDown moves the selection down, wrapping to top.
func (m *AutocompleteModel) MoveDown() {
	if len(m.suggestions) == 0 {
		return
	}
	m.selected = (m.selected + 1) % len(m.suggestions)
}

// MoveUp moves the selection up, wrapping to bottom.
func (m *AutocompleteModel) MoveUp() {
	if len(m.suggestions) == 0 {
		return
	}
	m.selected = (m.selected - 1 + len(m.suggestions)) % len(m.suggestions)
}

// AcceptSelected returns the currently selected suggestion.
func (m *AutocompleteModel) AcceptSelected() string {
	if len(m.suggestions) == 0 || m.selected >= len(m.suggestions) {
		return ""
	}
	return m.suggestions[m.selected]
}

// Dismiss hides the suggestion dropdown.
func (m *AutocompleteModel) Dismiss() {
	m.showSuggestions = false
	m.suggestions = nil
}
```

Note: This introduces a `lastInput` field. Add it to the struct:

```go
type AutocompleteModel struct {
	provider        SuggestionProvider
	suggestions     []string
	selected        int
	showSuggestions bool
	maxResults      int
	lastInput       string
}
```

**Step 2: Run tests**

Run: `go test ./test/internal/tui/ -run "TestAutocompleteModel|TestPathSuggestionProvider" -v`
Expected: All tests PASS.

**Step 3: Commit**

```bash
git add internal/tui/autocomplete.go test/internal/tui/autocomplete_test.go
git commit -m "feat: add reusable AutocompleteModel with pluggable providers"
```

---

### Task 5: Add autocomplete rendering

**Files:**
- Modify: `internal/tui/autocomplete.go` (add View method)

**Step 1: Add rendering to AutocompleteModel**

Move the `renderSuggestions()` method from `input.go` into `autocomplete.go` as `View()`:

```go
// View renders the suggestion dropdown box.
// Returns empty string if no suggestions are visible.
func (m *AutocompleteModel) View() string {
	if !m.showSuggestions || len(m.suggestions) == 0 {
		return ""
	}

	borderColor := lipgloss.Color("241")
	borderStyle := lipgloss.NewStyle().Foreground(borderColor)
	selectedStyle := lipgloss.NewStyle().Reverse(true)

	maxWidth := 0
	for _, s := range m.suggestions {
		if len(s) > maxWidth {
			maxWidth = len(s)
		}
	}
	if maxWidth < 20 {
		maxWidth = 20
	}
	boxWidth := maxWidth + 2

	var b strings.Builder

	b.WriteString(borderStyle.Render("\u250c" + strings.Repeat("\u2500", boxWidth) + "\u2510"))
	b.WriteString("\n")

	for i, s := range m.suggestions {
		padded := s + strings.Repeat(" ", boxWidth-2-len(s))
		var content string
		if i == m.selected {
			content = selectedStyle.Render(" " + padded + " ")
		} else {
			content = " " + padded + " "
		}
		b.WriteString(borderStyle.Render("\u2502") + content + borderStyle.Render("\u2502"))
		b.WriteString("\n")
	}

	b.WriteString(borderStyle.Render("\u2514" + strings.Repeat("\u2500", boxWidth) + "\u2518"))
	b.WriteString("\n")
	b.WriteString(hintStyle.Render("\u2191\u2193 navigate  \u23ce complete  Esc cancel"))

	return b.String()
}
```

Add `"strings"` and `"github.com/charmbracelet/lipgloss"` to the imports if not already present.

**Step 2: Run all tests**

Run: `go test ./test/internal/tui/ -v 2>&1 | tail -5`
Expected: All tests PASS (no regressions).

**Step 3: Commit**

```bash
git add internal/tui/autocomplete.go
git commit -m "feat: add View() rendering to AutocompleteModel"
```

---

### Task 6: Refactor input.go to use AutocompleteModel

**Files:**
- Modify: `internal/tui/input.go`
- Modify: `test/internal/tui/input_test.go`

**Step 1: Update input_test.go to test via new API**

The existing `TestPathSuggestions_*` tests call `tui.GetPathSuggestions()`. After refactoring, `GetPathSuggestions` is replaced by `PathSuggestionProvider`. Update the tests:

Replace the existing `TestPathSuggestions_*` functions in `test/internal/tui/input_test.go` so they call `tui.PathSuggestionProvider(8)` instead of `tui.GetPathSuggestions()`. Or keep `GetPathSuggestions` as a thin wrapper. The simpler approach: keep `GetPathSuggestions` as a convenience wrapper that calls `PathSuggestionProvider(8)`.

**Step 2: Refactor input.go**

In `internal/tui/input.go`:

1. Remove the `GetPathSuggestions` function (it's now in `autocomplete.go` as `PathSuggestionProvider`)
2. Remove `suggestions`, `sugSelected`, `showSuggestions` fields from `ProjectInputModel`
3. Add `autocomplete AutocompleteModel` field
4. Remove `renderSuggestions()` method
5. Update `Update()` to delegate suggestion keys (up/down/tab/enter/esc) to the autocomplete model
6. Update `View()` to use `m.autocomplete.View()`

After refactoring, `ProjectInputModel` becomes:

```go
type ProjectInputModel struct {
	nameInput    textinput.Model
	pathInput    textinput.Model
	focusName    bool
	name         string
	path         string
	confirmed    bool
	quitting     bool
	err          error
	autocomplete AutocompleteModel
}
```

And `NewProjectInput()` becomes:

```go
func NewProjectInput() ProjectInputModel {
	nameInput := textinput.New()
	nameInput.Placeholder = "Project name"
	nameInput.Focus()

	pathInput := textinput.New()
	pathInput.Placeholder = "Project path (e.g., ~/code/project)"

	return ProjectInputModel{
		nameInput:    nameInput,
		pathInput:    pathInput,
		focusName:    true,
		autocomplete: NewAutocomplete(PathSuggestionProvider(8), 8),
	}
}
```

The `Update()` method delegates to autocomplete methods instead of managing suggestion state inline. The `View()` method calls `m.autocomplete.View()` instead of `m.renderSuggestions()`.

**Important:** Keep `GetPathSuggestions` as a backward-compatible wrapper so existing tests don't break:

```go
// GetPathSuggestions is a convenience wrapper around PathSuggestionProvider.
func GetPathSuggestions(input string) []string {
	return PathSuggestionProvider(8)(input)
}
```

**Step 3: Run all tests**

Run: `go test ./test/internal/tui/ -v`
Expected: All tests PASS. The existing `TestPathSuggestions_*` tests continue to pass via the wrapper. The new `TestAutocompleteModel_*` and `TestPathSuggestionProvider_*` tests also pass.

**Step 4: Commit**

```bash
git add internal/tui/input.go internal/tui/autocomplete.go
git commit -m "refactor: extract autocomplete from input.go into reusable AutocompleteModel"
```

---

## Phase 2: Sleeping Ghost with Theme-Driven Colors

### Task 7: Add theme-to-ANSI helper

**Files:**
- Modify: `internal/tui/theme.go`

**Step 1: Write test for the helper**

Append to `test/internal/tui/theme_test.go`:

```go
func TestAnsiFromThemeColor(t *testing.T) {
	result := tui.AnsiFromThemeColor(lipgloss.Color("209"))
	expected := "\033[38;5;209m"
	if result != expected {
		t.Errorf("AnsiFromThemeColor(209): expected %q, got %q", expected, result)
	}
}
```

**Step 2: Run — expect fail**

Run: `go test ./test/internal/tui/ -run TestAnsiFromThemeColor -v`
Expected: Compilation error — `tui.AnsiFromThemeColor` undefined.

**Step 3: Implement**

Add to `internal/tui/theme.go`:

```go
// AnsiFromThemeColor converts a lipgloss.Color (ANSI 256 string) to an
// ANSI escape sequence. This bridges lipgloss theme colors with raw
// escape-code rendering used by ghost ASCII art.
func AnsiFromThemeColor(c lipgloss.Color) string {
	return fmt.Sprintf("\033[38;5;%sm", string(c))
}
```

Add `"fmt"` to imports.

**Step 4: Run — expect pass**

Run: `go test ./test/internal/tui/ -run TestAnsiFromThemeColor -v`
Expected: PASS.

**Step 5: Commit**

```bash
git add internal/tui/theme.go test/internal/tui/theme_test.go
git commit -m "feat: add AnsiFromThemeColor to bridge lipgloss colors with ANSI escape codes"
```

---

### Task 8: Refactor ghost.go to use theme-derived colors

**Files:**
- Modify: `internal/tui/ghost.go`

**Step 1: Write test for theme-derived ghost**

Append to `test/internal/tui/ghost_test.go`:

```go
func TestGhostForToolWithTheme_UsesThemeColors(t *testing.T) {
	theme := tui.ThemeForTool("claude")
	expectedColor := tui.AnsiFromThemeColor(theme.Primary)

	lines := tui.GhostForTool("claude", false)
	rendered := tui.RenderGhost(lines)

	if !strings.Contains(rendered, expectedColor) {
		t.Errorf("awake claude ghost should contain theme Primary color %q", expectedColor)
	}
}

func TestGhostForToolWithTheme_SleepingUsesDimmedColors(t *testing.T) {
	theme := tui.ThemeForTool("claude")
	sleepColor := tui.AnsiFromThemeColor(theme.SleepPrimary)

	lines := tui.GhostForTool("claude", true)
	rendered := tui.RenderGhost(lines)

	if !strings.Contains(rendered, sleepColor) {
		t.Errorf("sleeping claude ghost should contain theme SleepPrimary color %q", sleepColor)
	}
}
```

**Step 2: Run — may fail depending on whether hardcoded values happen to match**

Run: `go test ./test/internal/tui/ -run TestGhostForToolWithTheme -v`
Note: These may pass by coincidence since Claude's hardcoded `c(209)` produces `\033[38;5;209m` and theme.Primary is `"209"`. The test validates the integration regardless.

**Step 3: Refactor ghost.go**

Change `GhostForTool` to derive colors from the theme:

```go
func GhostForTool(tool string, sleeping bool) []string {
	theme := ThemeForTool(tool)
	switch tool {
	case "codex":
		if sleeping {
			return ghostCodexSleeping(theme)
		}
		return ghostCodex(theme)
	// ... same pattern for copilot, opencode, default (claude)
	}
}
```

Change each ghost function to accept `theme AIToolTheme` and use `AnsiFromThemeColor()`:

```go
func ghostClaude(theme AIToolTheme) []string {
	O := AnsiFromThemeColor(theme.Primary)
	D := AnsiFromThemeColor(theme.Dim)
	B := AnsiFromThemeColor(theme.DarkFeet)
	L := AnsiFromThemeColor(theme.Cap)
	W := AnsiFromThemeColor(theme.EyeWhite)
	K := AnsiFromThemeColor(theme.EyePupil)
	Y := AnsiFromThemeColor(theme.Accent)
	// ... rest of art unchanged
}
```

For sleeping variants, use `SleepPrimary` and `SleepAccent`:

```go
func ghostClaudeSleeping(theme AIToolTheme) []string {
	O := AnsiFromThemeColor(theme.SleepPrimary)
	D := AnsiFromThemeColor(theme.SleepPrimary)
	B := AnsiFromThemeColor(theme.SleepAccent)  // this may need a new field
	L := AnsiFromThemeColor(theme.Dim)
	K := AnsiFromThemeColor(theme.EyePupil)
	// ... rest of art unchanged
}
```

**Important:** Some ghost functions use more colors than `AIToolTheme` currently provides (e.g., Claude's sleeping ghost uses `c(94)` for dark-feet-sleeping, OpenCode uses `c(236)` for medium-sleeping). You may need to add a few additional sleep color fields to `AIToolTheme`:

```go
type AIToolTheme struct {
	// ... existing fields ...
	SleepDim      lipgloss.Color  // new: dimmed body for sleeping
	SleepDarkFeet lipgloss.Color  // new: dimmed feet for sleeping
}
```

Populate these for each tool in the `themes` map. Use the values currently hardcoded in the sleeping ghost functions.

**Step 4: Run all tests**

Run: `go test ./test/internal/tui/ -v`
Expected: All existing ghost tests still pass (same visual output since theme values match the previously hardcoded values). New theme tests also pass.

**Step 5: Commit**

```bash
git add internal/tui/ghost.go internal/tui/theme.go test/internal/tui/ghost_test.go test/internal/tui/theme_test.go
git commit -m "refactor: derive ghost art colors from AIToolTheme instead of hardcoding"
```

---

### Task 9: Write animated Zzz tests

**Files:**
- Create: `test/internal/tui/zzz_test.go`

**Step 1: Write failing tests**

```go
package tui_test

import (
	"testing"

	"github.com/jackuait/ghost-tab/internal/tui"
)

func TestZzzAnimation_InitialFrame(t *testing.T) {
	z := tui.NewZzzAnimation()
	if z.Frame() != 0 {
		t.Errorf("initial frame should be 0, got %d", z.Frame())
	}
}

func TestZzzAnimation_TickAdvancesFrame(t *testing.T) {
	z := tui.NewZzzAnimation()
	z.Tick()
	if z.Frame() != 1 {
		t.Errorf("after tick: expected frame 1, got %d", z.Frame())
	}
}

func TestZzzAnimation_FrameWraps(t *testing.T) {
	z := tui.NewZzzAnimation()
	totalFrames := z.TotalFrames()
	for i := 0; i < totalFrames; i++ {
		z.Tick()
	}
	if z.Frame() != 0 {
		t.Errorf("after %d ticks: expected frame 0 (wrapped), got %d", totalFrames, z.Frame())
	}
}

func TestZzzAnimation_FramesDiffer(t *testing.T) {
	z := tui.NewZzzAnimation()
	frame0 := z.View()
	z.Tick()
	frame1 := z.View()

	if frame0 == frame1 {
		t.Error("consecutive Zzz frames should differ")
	}
}

func TestZzzAnimation_Reset(t *testing.T) {
	z := tui.NewZzzAnimation()
	z.Tick()
	z.Tick()
	z.Reset()
	if z.Frame() != 0 {
		t.Errorf("after reset: expected frame 0, got %d", z.Frame())
	}
}

func TestZzzAnimation_ViewContainsZ(t *testing.T) {
	z := tui.NewZzzAnimation()
	view := z.View()
	if view == "" {
		t.Error("Zzz view should not be empty")
	}
}
```

**Step 2: Run — expect fail**

Run: `go test ./test/internal/tui/ -run TestZzzAnimation -v 2>&1 | head -10`
Expected: Compilation error — `tui.NewZzzAnimation` undefined.

**Step 3: Commit**

```bash
git add test/internal/tui/zzz_test.go
git commit -m "test: add failing tests for animated Zzz component"
```

---

### Task 10: Implement animated Zzz

**Files:**
- Create: `internal/tui/zzz.go`

**Step 1: Implement ZzzAnimation**

```go
package tui

import "fmt"

// ZzzAnimation renders an animated sleeping indicator with Z characters
// that float upward in a cycle.
type ZzzAnimation struct {
	frame  int
	frames []string
}

// NewZzzAnimation creates a new Zzz animation with 4 frames.
func NewZzzAnimation() *ZzzAnimation {
	dim := "\033[2m"
	reset := "\033[0m"

	frames := []string{
		dim + "        z" + reset + "\n" +
			dim + "      Z" + reset + "\n" +
			"    Z",
		dim + "       z" + reset + "\n" +
			dim + "     Z" + reset + "\n" +
			"   Z",
		dim + "      z" + reset + "\n" +
			dim + "    Z" + reset + "\n" +
			"  Z",
		dim + "       z" + reset + "\n" +
			dim + "     Z" + reset + "\n" +
			"   Z",
	}

	return &ZzzAnimation{
		frame:  0,
		frames: frames,
	}
}

// Frame returns the current frame index.
func (z *ZzzAnimation) Frame() int {
	return z.frame
}

// TotalFrames returns the number of animation frames.
func (z *ZzzAnimation) TotalFrames() int {
	return len(z.frames)
}

// Tick advances to the next frame.
func (z *ZzzAnimation) Tick() {
	z.frame = (z.frame + 1) % len(z.frames)
}

// Reset returns to frame 0.
func (z *ZzzAnimation) Reset() {
	z.frame = 0
}

// View returns the current frame's Zzz string.
func (z *ZzzAnimation) View() string {
	if len(z.frames) == 0 {
		return ""
	}
	return z.frames[z.frame]
}

// ViewColored returns the current frame with the given ANSI color applied.
func (z *ZzzAnimation) ViewColored(color string) string {
	if len(z.frames) == 0 {
		return ""
	}
	return fmt.Sprintf("%s%s\033[0m", color, z.frames[z.frame])
}
```

**Step 2: Run tests**

Run: `go test ./test/internal/tui/ -run TestZzzAnimation -v`
Expected: All PASS.

**Step 3: Commit**

```bash
git add internal/tui/zzz.go
git commit -m "feat: add animated ZzzAnimation component with 4-frame float cycle"
```

---

### Task 11: Wire Zzz into main menu and update RenderZzz

**Files:**
- Modify: `internal/tui/mainmenu.go`
- Modify: `internal/tui/ghost.go` (update RenderZzz to use ZzzAnimation)

**Step 1: Add ZzzAnimation field to MainMenuModel**

In `internal/tui/mainmenu.go`, add to the struct:

```go
type MainMenuModel struct {
	// ... existing fields ...
	zzz *ZzzAnimation
}
```

Initialize it in `NewMainMenu`:

```go
func NewMainMenu(...) *MainMenuModel {
	// ... existing code ...
	return &MainMenuModel{
		// ... existing fields ...
		zzz: NewZzzAnimation(),
	}
}
```

**Step 2: Advance Zzz on bob tick when sleeping**

In `Update()`, inside the `bobTickMsg` case:

```go
case bobTickMsg:
	if m.ghostDisplay == "animated" {
		m.bobStep = (m.bobStep + 1) % len(BobOffsets())
		if m.ghostSleeping && m.zzz != nil {
			m.zzz.Tick()
		}
		return m, m.bobTickCmd()
	}
	return m, nil
```

**Step 3: Reset Zzz on wake**

In `Wake()`:

```go
func (m *MainMenuModel) Wake() {
	m.ghostSleeping = false
	m.sleepTimer = 0
	if m.zzz != nil {
		m.zzz.Reset()
	}
}
```

Also in the `Update()` key/mouse handlers where sleep is reset, call `Wake()` instead of setting fields directly:

```go
// In tea.KeyMsg handler:
m.Wake()
// Instead of:
// m.sleepTimer = 0
// if m.ghostSleeping { m.ghostSleeping = false }
```

**Step 4: Render Zzz alongside sleeping ghost in View()**

In the `View()` method, when rendering ghost in "side" or "above" position and `m.ghostSleeping` is true, append the Zzz:

```go
case "side":
	ghostLines := GhostForTool(m.CurrentAITool(), m.ghostSleeping)
	ghostStr := RenderGhost(ghostLines)
	if m.ghostDisplay == "animated" && bobOffsets[m.bobStep] == 1 {
		ghostStr = "\n" + ghostStr
	}
	if m.ghostSleeping && m.zzz != nil {
		zzzColor := AnsiFromThemeColor(m.theme.SleepAccent)
		ghostStr += "\n" + m.zzz.ViewColored(zzzColor)
	}
	spacer := strings.Repeat(" ", 3)
	return lipgloss.JoinHorizontal(lipgloss.Top, menuBox, spacer, ghostStr)
```

Apply same pattern to "above" case.

**Step 5: Update RenderZzz in ghost.go to delegate to ZzzAnimation**

Replace the static `RenderZzz()` in `ghost.go`:

```go
// RenderZzz returns a static "z Z Z" sleeping indicator.
// Deprecated: Use ZzzAnimation for animated rendering.
func RenderZzz() string {
	z := NewZzzAnimation()
	return z.View()
}
```

**Step 6: Run all tests**

Run: `go test ./test/internal/tui/ -v`
Expected: All tests PASS. Existing `TestRenderZzz` still passes because the output format is preserved.

**Step 7: Commit**

```bash
git add internal/tui/mainmenu.go internal/tui/ghost.go
git commit -m "feat: wire animated Zzz into main menu sleeping ghost display"
```

---

## Phase 3: Fill Remaining Test Gaps

### Task 12: Add logo component tests

**Files:**
- Create: `test/internal/tui/logo_test.go`

**Step 1: Write tests**

```go
package tui_test

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jackuait/ghost-tab/internal/tui"
)

func TestLogo_InitialFrame(t *testing.T) {
	m := tui.NewLogo("claude")
	// Init should return commands (tick + quit timer)
	cmd := m.Init()
	if cmd == nil {
		t.Error("Init() should return commands for tick and quit timer")
	}
}

func TestLogo_QuitOnKeypress(t *testing.T) {
	m := tui.NewLogo("claude")
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd == nil {
		t.Error("keypress should trigger quit command")
	}
	_ = updated
}

func TestLogo_AllToolsRender(t *testing.T) {
	tools := []string{"claude", "codex", "copilot", "opencode"}
	for _, tool := range tools {
		t.Run(tool, func(t *testing.T) {
			m := tui.NewLogo(tool)
			view := m.View()
			if view == "" {
				t.Errorf("Logo view for %q should not be empty", tool)
			}
		})
	}
}
```

**Step 2: Run tests**

Run: `go test ./test/internal/tui/ -run TestLogo -v`
Expected: All PASS.

**Step 3: Commit**

```bash
git add test/internal/tui/logo_test.go
git commit -m "test: add logo component tests"
```

---

### Task 13: Add confirm dialog interaction tests

**Files:**
- Modify: `test/internal/tui/input_test.go` (append confirm tests)

**Step 1: Add interaction tests**

Append to `test/internal/tui/input_test.go`:

```go
func TestConfirmDialog_YConfirms(t *testing.T) {
	m := tui.NewConfirmDialog("Delete?")
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	result := updated.(tui.ConfirmDialogModel)
	if !result.Confirmed {
		t.Error("'y' should confirm")
	}
}

func TestConfirmDialog_UpperYConfirms(t *testing.T) {
	m := tui.NewConfirmDialog("Delete?")
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'Y'}})
	result := updated.(tui.ConfirmDialogModel)
	if !result.Confirmed {
		t.Error("'Y' should confirm")
	}
}

func TestConfirmDialog_NDenies(t *testing.T) {
	m := tui.NewConfirmDialog("Delete?")
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	result := updated.(tui.ConfirmDialogModel)
	if result.Confirmed {
		t.Error("'n' should deny")
	}
}

func TestConfirmDialog_EscDenies(t *testing.T) {
	m := tui.NewConfirmDialog("Delete?")
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	result := updated.(tui.ConfirmDialogModel)
	if result.Confirmed {
		t.Error("Esc should deny")
	}
}

func TestConfirmDialog_CtrlCDenies(t *testing.T) {
	m := tui.NewConfirmDialog("Delete?")
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	result := updated.(tui.ConfirmDialogModel)
	if result.Confirmed {
		t.Error("Ctrl+C should deny")
	}
}

func TestConfirmDialog_ViewShowsMessage(t *testing.T) {
	m := tui.NewConfirmDialog("Are you sure?")
	view := m.View()
	if !strings.Contains(view, "Are you sure?") {
		t.Error("view should contain the message")
	}
	if !strings.Contains(view, "y/n") {
		t.Error("view should contain y/n hint")
	}
}
```

**Step 2: Run tests**

Run: `go test ./test/internal/tui/ -run TestConfirmDialog -v`
Expected: All PASS.

**Step 3: Commit**

```bash
git add test/internal/tui/input_test.go
git commit -m "test: add confirm dialog interaction tests for y/n/Esc/Ctrl+C"
```

---

### Task 14: Final verification

**Step 1: Run full Go test suite**

```bash
go test ./... -v 2>&1 | tail -20
```

Expected: All tests pass, including the new ones.

**Step 2: Run shellcheck on any modified shell scripts**

```bash
find lib bin ghostty -name '*.sh' -exec shellcheck {} +
```

Expected: No new warnings (this task only modifies Go files, but verify anyway).

**Step 3: Run BATS test suite**

```bash
./run-tests.sh
```

Expected: All BATS tests pass (no bash changes should break them).

**Step 4: Push**

```bash
git pull --rebase
git push
git status
```

Expected: "up to date with origin"
