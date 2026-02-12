# Go TUI Feature Restoration Design

Restore features lost during the bash-to-Go TUI migration: sleeping ghost animation, reusable autocomplete component, and comprehensive Go test coverage.

## Background

During the migration from bash TUI modules to Go/Bubbletea, several features were lost or left incomplete:

- **Autocomplete module** (`lib/autocomplete.sh`) was deleted. Go `add-project` has basic path suggestions but lacks: empty-input default (`~/`), alphabetical sorting, glob-style matching, and reusability.
- **Sleeping ghost art** exists for all 4 tools but the Zzz animation is static, ghost colors are hardcoded (bypassing the theme system), and the main menu doesn't render Zzz alongside the sleeping ghost.
- **Go TUI tests** were never written to replace the deleted bash test files (`menu.bats`, `ai-select.bats`, `settings-menu.bats`, `logo-animation.bats`, `autocomplete.bats`).

## Phase 1: Reusable Autocomplete Component

### New file: `internal/tui/autocomplete.go`

Extract path autocomplete from `input.go` into a standalone Bubbletea component.

```go
type AutocompleteModel struct {
    Input        textinput.Model
    Suggestions  []string
    Selected     int
    Provider     func(input string) []string
    MaxResults   int  // default 8
}
```

**Pluggable provider pattern** — suggestion source is a function, not hardcoded to filesystem. Path completion is one provider; project name completion could follow.

**Built-in `PathSuggestionProvider`:**
- Empty input defaults to `~/`
- Case-insensitive glob matching (not just prefix)
- Results sorted alphabetically
- Directories only, hidden dirs filtered
- Tilde expansion via `util.ExpandPath()`

**Keyboard:** Up/Down navigate, Tab accepts, Escape dismisses, typing filters live.

**Rendering:** Bordered dropdown below input, reverse-video highlight on selected.

### Changes to `input.go`

- Remove ~120 lines of inline suggestion logic (suggestion generation, rendering, navigation)
- Replace with embedded `AutocompleteModel`
- Keep two-phase form (name then path) — only path phase uses autocomplete
- `add-project` subcommand behavior stays identical from user perspective

### Test file: `internal/tui/autocomplete_test.go`

- Provider returns suggestions for valid input
- Empty input defaults to `~/`
- Results are sorted alphabetically
- Glob matching works (case-insensitive)
- Selection navigation (up/down wrap)
- Tab accepts selected suggestion
- Escape dismisses suggestions
- MaxResults limit respected

## Phase 2: Sleeping Ghost Completion

### Refactor ghost colors: `internal/tui/ghost.go`

Ghost art functions currently hardcode ANSI 256-color values via `c(n)`. Meanwhile, `theme.go` defines `SleepPrimary`/`SleepAccent` as lipgloss colors separately. These are disconnected.

**Change:** Ghost art functions accept an `AIToolTheme` parameter and derive their color codes from it. This makes themes the single source of truth.

```go
// Before:
func ghostClaude() []string {
    O := c(209)  // hardcoded
    ...
}

// After:
func ghostClaude(theme AIToolTheme) []string {
    O := colorFromTheme(theme.Primary)
    ...
}
```

This requires adding theme color fields that map to the ghost art's color slots (Primary, Dim, Bright, Accent, Cap, DarkFeet, EyeWhite, EyePupil — already in `AIToolTheme`). The sleeping variants use `SleepPrimary` and `SleepAccent` instead.

### Animated Zzz: `internal/tui/zzz.go`

New component for animated sleeping indicator.

- 3-4 frame animation cycle with Z characters floating upward
- Reuses existing 200ms tick from main menu
- Renders as overlay text positioned relative to ghost's top-right corner
- Uses theme's `SleepAccent` color
- Exposes `Update(msg)` and `View()` for embedding

### Main menu integration: `internal/tui/mainmenu.go`

- Render Zzz overlay alongside sleeping ghost art
- Apply `SleepPrimary`/`SleepAccent` to surrounding UI elements (borders, title text) when ghost is sleeping
- Wake event clears Zzz and restores awake colors

### Test files

**`internal/tui/ghost_test.go`:**
- All 8 variants return exactly 15 lines
- Line widths are consistent across variants
- `GhostForTool` dispatches correctly for all tools
- Sleeping vs awake art differs (different color codes)
- Theme-derived colors match expected values

**`internal/tui/zzz_test.go`:**
- Frame cycling advances on tick
- Frames differ from each other
- Reset returns to frame 0

## Phase 3: Remaining Test Coverage

Test the Bubbletea `Update()` state machine — send messages in, assert state changes. Don't test visual rendering (string output) except for critical formatting like ghost art dimensions.

### `internal/tui/mainmenu_test.go`
- Project navigation (up/down/jump to index)
- AI tool cycling (left/right arrows)
- Sleep timer triggers `ghostSleeping` after 120s of inactivity
- Wake on any input resets sleep state
- Ghost display mode toggle (animated/static/none)
- Action routing: select-project, add-project, delete-project, open-once, plain-terminal, quit

### `internal/tui/aitools_test.go`
- Tool detection populates list correctly
- Uninstalled tool selection is blocked
- List navigation works
- Correct JSON output structure

### `internal/tui/input_test.go`
- Two-phase form transitions (name -> path)
- Validation errors on empty fields
- Validation errors on invalid paths
- Autocomplete integration (suggestions appear on path input)

### `internal/tui/confirm_test.go`
- Y/y returns confirmed
- N/n returns denied
- Escape returns denied

### `internal/tui/settings_test.go`
- Menu navigation selects items
- Action strings match expected values

### `internal/tui/multiselect_test.go`
- Toggle selection with space
- At least one required validation
- Pre-checked state for installed tools
- Enter confirms selection

### `internal/tui/theme_test.go`
- All 4 tools have themes defined
- `ThemeForTool` unknown tool falls back to claude
- `ApplyTheme` sets package-level styles

### `internal/models/project_test.go`
- Parse name:path format correctly
- Skip comment lines (# prefix)
- Skip blank lines
- Handle malformed lines gracefully

### `internal/models/aitool_test.go`
- Display names are correct for all tools
- Detection logic uses correct commands

## Implementation Order

- **Phases 1 and 2 can run in parallel** — they touch different files
- **Phase 3 depends on both** — tests validate the final state after refactoring

## Files Summary

### New files (~9)
- `internal/tui/autocomplete.go`
- `internal/tui/autocomplete_test.go`
- `internal/tui/zzz.go`
- `internal/tui/zzz_test.go`
- `internal/tui/ghost_test.go`
- `internal/tui/mainmenu_test.go`
- `internal/tui/aitools_test.go`
- `internal/tui/confirm_test.go`
- `internal/tui/settings_test.go`
- `internal/tui/multiselect_test.go`
- `internal/tui/theme_test.go`
- `internal/tui/input_test.go`
- `internal/models/project_test.go`
- `internal/models/aitool_test.go`

### Modified files (~4)
- `internal/tui/ghost.go` — theme-derived colors
- `internal/tui/input.go` — embed AutocompleteModel
- `internal/tui/mainmenu.go` — Zzz rendering, sleep theme colors
- `internal/tui/theme.go` — helper to extract ANSI code from lipgloss color (if needed)
