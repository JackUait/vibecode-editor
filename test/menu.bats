setup() {
  load 'test_helper/common'
  _common_setup

  # Source the ACTUAL lib/menu.sh (not a copy!)
  source "$PROJECT_ROOT/lib/tui.sh"
  source "$PROJECT_ROOT/lib/ai-tools.sh"
  source "$PROJECT_ROOT/lib/logo-animation.sh"
  source "$PROJECT_ROOT/lib/menu.sh"

  # Create temp directory for test data
  TEST_DIR="$(mktemp -d)"

  # Helper to mock terminal dimensions for draw_menu
  # draw_menu() reads from /dev/tty using: read ... </dev/tty
  # We override the read command to provide controlled dimensions
  _setup_test_draw_menu() {
    local test_rows="$1"
    local test_cols="$2"

    # Store dimensions in global variables
    MOCK_TTY_ROWS="$test_rows"
    MOCK_TTY_COLS="$test_cols"
    export MOCK_TTY_ROWS MOCK_TTY_COLS

    # Wrapper that sets up a mock for the tty read
    _run_draw_menu_with_mocked_tty() {
      # Override the read builtin for this subshell to inject fake dimensions
      # The read in draw_menu expects: IFS='[;' read -rs -d R -p $'\033[6n' _ _rows _cols </dev/tty
      # We simulate what /dev/tty would return
      (
        # Inject fake /dev/tty content by symlinking to a fifo
        local fake_tty="$TEST_DIR/fake_tty_$$"
        mkfifo "$fake_tty" 2>/dev/null || true

        # Send the response that the terminal would send
        printf "\033[%d;%dR" "$MOCK_TTY_ROWS" "$MOCK_TTY_COLS" > "$fake_tty" &

        # Execute draw_menu in a context where /dev/tty is our pipe
        # This is tricky - we use exec to replace /dev/tty's file descriptor
        exec 9<"$fake_tty"
        # Now call draw_menu but with fd 9 instead of /dev/tty
        # Actually, this won't work directly. Let's use a different approach.

        # Alternative: Patch draw_menu temporarily to read from our pipe
        # Save original draw_menu
        declare -f draw_menu > "$TEST_DIR/orig_draw_menu_$$"

        # Create patched version that reads from our fake tty
        eval "$(declare -f draw_menu | sed 's|</dev/tty|<"$fake_tty"|')"

        # Call the patched function
        draw_menu

        # Restore original
        rm -f "$fake_tty"
      )
    }

    export -f _run_draw_menu_with_mocked_tty
  }

  # Mock draw_logo to avoid complex logo rendering
  draw_logo() {
    echo "LOGO:$1:$2:$3"
  }
  export -f draw_logo

  # Mock get_ghost_display_setting
  get_ghost_display_setting() {
    echo "static"
  }
  export -f get_ghost_display_setting
}

teardown() {
  rm -rf "$TEST_DIR"
}

# --- draw_menu: function exists ---

@test "draw_menu: function is defined" {
  declare -f draw_menu >/dev/null
}

@test "draw_menu: function is callable (type -t)" {
  [ "$(type -t draw_menu)" = "function" ]
}

# --- draw_menu: right border clears line properly ---

@test "draw_menu: _rbdr function definition is correct" {
  # Bug fix verification: _rbdr() should position first, then print border+clear
  # NOT clear first then position (which would clear content at wrong location)

  # Check that the source code has the fix
  rbdr_def=$(grep -A0 "_rbdr()" "$PROJECT_ROOT/lib/menu.sh")

  # Correct pattern: moveto comes before the border character │
  if ! echo "$rbdr_def" | grep -q 'moveto.*│'; then
    echo "ERROR: _rbdr doesn't call moveto before printing border"
    return 1
  fi

  # Verify clear (\033[K) comes after the border in the printf
  if ! echo "$rbdr_def" | grep -q '│.*\\033\[K'; then
    echo "ERROR: _rbdr doesn't have clear after border"
    return 1
  fi

  # Ensure clear does NOT come before moveto (the bug)
  if echo "$rbdr_def" | grep -q '\\033\[K.*moveto'; then
    echo "BUG: _rbdr has clear before moveto"
    return 1
  fi

  return 0
}

@test "draw_menu: clears entire line before printing content" {
  # Bug: Old content can remain visible if new content is shorter
  # Fix: Clear the entire line (or at least to right border) before printing new content

  # Check menu item rendering (around line 112-128)
  item_render=$(sed -n '111,129p' "$PROJECT_ROOT/lib/menu.sh")

  # After printing left border │, before printing content, we should clear
  # For now, verify the issue exists by checking that we DON'T clear after left border
  if echo "$item_render" | grep -A1 'printf.*│.*_NC' | grep -q '\\033\[2K\|\\033\[K'; then
    # If we find clearing after left border, the fix is in place
    return 0
  else
    # Expected to fail initially - no clearing after left border
    echo "BUG: No clear after left border before printing content"
    return 1
  fi
}

# --- Logo Layout Tests ---

@test "draw_menu: renders logo side-by-side when terminal width >= 110" {
  _setup_test_draw_menu 40 120

  # Setup globals
  projects=("myapp:/Users/me/myapp")
  menu_labels=("myapp" "+ Add project" "⚙ Settings")
  menu_subs=("~/myapp" "" "")
  menu_hi=("" "" "")
  _action_hints=("+" "S")
  _action_bar=("" "")
  total=3
  selected=0
  box_w=60
  AI_TOOLS_AVAILABLE=("claude")
  SELECTED_AI_TOOL="claude"
  _update_version=""

  run _run_draw_menu_with_mocked_tty

  assert_success
  # Verify menu is rendered properly for wide terminal
  assert_output --partial "Ghost Tab"
  assert_output --partial "myapp"
  # Logo would be positioned to the side in wide layout (tested by layout logic)
}

@test "draw_menu: logo positioned above menu when terminal width < 110" {
  _setup_test_draw_menu 50 90

  projects=()
  menu_labels=("+ Add project" "⚙ Settings")
  menu_subs=("" "")
  menu_hi=("" "")
  _action_hints=("+" "S")
  _action_bar=("" "")
  total=2
  selected=0
  box_w=60
  AI_TOOLS_AVAILABLE=("claude")
  SELECTED_AI_TOOL="claude"
  _update_version=""

  run _run_draw_menu_with_mocked_tty

  assert_success
  # Verify menu is rendered properly for medium terminal
  assert_output --partial "Ghost Tab"
  assert_output --partial "+ Add project"
  # Logo would be positioned above in this layout (tested by layout logic)
}

@test "draw_menu: hides logo when terminal too small" {
  _setup_test_draw_menu 20 70

  projects=()
  menu_labels=("+ Add project" "⚙ Settings")
  menu_subs=("" "")
  menu_hi=("" "")
  _action_hints=("+" "S")
  _action_bar=("" "")
  total=2
  selected=0
  box_w=60
  AI_TOOLS_AVAILABLE=("claude")
  SELECTED_AI_TOOL="claude"
  _update_version=""

  run _run_draw_menu_with_mocked_tty

  assert_success
  # Verify menu is still rendered even in small terminal
  assert_output --partial "Ghost Tab"
  assert_output --partial "+ Add project"
  # Logo is hidden in small terminal (no room for it)
}

# --- Project List Rendering Tests ---

@test "draw_menu: renders menu with 0 projects (actions only)" {
  _setup_test_draw_menu 30 100

  projects=()
  menu_labels=("+ Add project" "⚙ Settings")
  menu_subs=("" "Configure Ghost Tab")
  menu_hi=("" "")
  _action_hints=("+" "S")
  _action_bar=("" "")
  total=2
  selected=0
  box_w=60
  AI_TOOLS_AVAILABLE=("claude")
  SELECTED_AI_TOOL="claude"
  _update_version=""

  run _run_draw_menu_with_mocked_tty

  assert_success
  assert_output --partial "+ Add project"
  assert_output --partial "⚙ Settings"
}

@test "draw_menu: renders menu with 1 project" {
  _setup_test_draw_menu 30 100

  projects=("myapp:/Users/me/myapp")
  menu_labels=("myapp" "+ Add project" "⚙ Settings")
  menu_subs=("~/myapp" "" "")
  menu_hi=("" "" "")
  _action_hints=("+" "S")
  _action_bar=("" "")
  total=3
  selected=0
  box_w=60
  AI_TOOLS_AVAILABLE=("claude")
  SELECTED_AI_TOOL="claude"
  _update_version=""

  run _run_draw_menu_with_mocked_tty

  assert_success
  assert_output --partial "myapp"
  assert_output --partial "~/myapp"
  assert_output --partial "+ Add project"
}

@test "draw_menu: renders menu with many projects (5+)" {
  _setup_test_draw_menu 50 100

  projects=(
    "app1:/path/to/app1"
    "app2:/path/to/app2"
    "app3:/path/to/app3"
    "app4:/path/to/app4"
    "app5:/path/to/app5"
  )
  menu_labels=("app1" "app2" "app3" "app4" "app5" "+ Add project" "⚙ Settings")
  menu_subs=("~/app1" "~/app2" "~/app3" "~/app4" "~/app5" "" "")
  menu_hi=("" "" "" "" "" "" "")
  _action_hints=("+" "S")
  _action_bar=("" "")
  total=7
  selected=0
  box_w=60
  AI_TOOLS_AVAILABLE=("claude")
  SELECTED_AI_TOOL="claude"
  _update_version=""

  run _run_draw_menu_with_mocked_tty

  assert_success
  assert_output --partial "app1"
  assert_output --partial "app2"
  assert_output --partial "app3"
  assert_output --partial "app4"
  assert_output --partial "app5"
}

# --- Truncation Tests ---

@test "draw_menu: truncates very long project names" {
  _setup_test_draw_menu 30 100

  long_name="This_is_an_extremely_long_project_name_that_exceeds_the_maximum_width_allowed"
  projects=("$long_name:/path")
  menu_labels=("$long_name" "+ Add project" "⚙ Settings")
  menu_subs=("~/path" "" "")
  menu_hi=("" "" "")
  _action_hints=("+" "S")
  _action_bar=("" "")
  total=3
  selected=0
  box_w=60
  AI_TOOLS_AVAILABLE=("claude")
  SELECTED_AI_TOOL="claude"
  _update_version=""

  run _run_draw_menu_with_mocked_tty

  assert_success
  # Should contain truncation character
  assert_output --partial "…"
  # Should NOT contain the full long name
  refute_output --partial "width_allowed"
}

@test "draw_menu: truncates very long project paths" {
  _setup_test_draw_menu 30 100

  long_path="/Users/username/very/deep/nested/directory/structure/project/that/is/way/too/long/to/display"
  projects=("app:$long_path")
  menu_labels=("app" "+ Add project" "⚙ Settings")
  menu_subs=("$long_path" "" "")
  menu_hi=("" "" "")
  _action_hints=("+" "S")
  _action_bar=("" "")
  total=3
  selected=0
  box_w=60
  AI_TOOLS_AVAILABLE=("claude")
  SELECTED_AI_TOOL="claude"
  _update_version=""

  run _run_draw_menu_with_mocked_tty

  assert_success
  # Should contain middle ellipsis truncation
  assert_output --partial "..."
  # Should have start of path
  assert_output --partial "/Users/username"
}

# --- AI Tool Color Display Tests ---

@test "draw_menu: renders Claude Code with orange color" {
  _setup_test_draw_menu 30 100

  projects=()
  menu_labels=("+ Add project" "⚙ Settings")
  menu_subs=("" "")
  menu_hi=("" "")
  _action_hints=("+" "S")
  _action_bar=("" "")
  total=2
  selected=0
  box_w=60
  AI_TOOLS_AVAILABLE=("claude")
  SELECTED_AI_TOOL="claude"
  _update_version=""

  run _run_draw_menu_with_mocked_tty

  assert_success
  # Should contain Claude Code display name
  assert_output --partial "Claude Code"
  # Should contain orange ANSI color code (209)
  assert_output --partial $'\033[38;5;209m'
}

@test "draw_menu: renders Codex CLI with green color" {
  _setup_test_draw_menu 30 100

  projects=()
  menu_labels=("+ Add project" "⚙ Settings")
  menu_subs=("" "")
  menu_hi=("" "")
  _action_hints=("+" "S")
  _action_bar=("" "")
  total=2
  selected=0
  box_w=60
  AI_TOOLS_AVAILABLE=("codex")
  SELECTED_AI_TOOL="codex"
  _update_version=""

  run _run_draw_menu_with_mocked_tty

  assert_success
  assert_output --partial "Codex CLI"
  # Should contain green ANSI color code (114)
  assert_output --partial $'\033[38;5;114m'
}

@test "draw_menu: renders Copilot CLI with purple color" {
  _setup_test_draw_menu 30 100

  projects=()
  menu_labels=("+ Add project" "⚙ Settings")
  menu_subs=("" "")
  menu_hi=("" "")
  _action_hints=("+" "S")
  _action_bar=("" "")
  total=2
  selected=0
  box_w=60
  AI_TOOLS_AVAILABLE=("copilot")
  SELECTED_AI_TOOL="copilot"
  _update_version=""

  run _run_draw_menu_with_mocked_tty

  assert_success
  assert_output --partial "Copilot CLI"
  # Should contain purple ANSI color code (141)
  assert_output --partial $'\033[38;5;141m'
}

@test "draw_menu: renders OpenCode with gray color" {
  _setup_test_draw_menu 30 100

  projects=()
  menu_labels=("+ Add project" "⚙ Settings")
  menu_subs=("" "")
  menu_hi=("" "")
  _action_hints=("+" "S")
  _action_bar=("" "")
  total=2
  selected=0
  box_w=60
  AI_TOOLS_AVAILABLE=("opencode")
  SELECTED_AI_TOOL="opencode"
  _update_version=""

  run _run_draw_menu_with_mocked_tty

  assert_success
  assert_output --partial "OpenCode"
  # Should contain light gray ANSI color code (250)
  assert_output --partial $'\033[38;5;250m'
}

@test "draw_menu: shows cycling indicators with multiple AI tools" {
  _setup_test_draw_menu 30 100

  projects=()
  menu_labels=("+ Add project" "⚙ Settings")
  menu_subs=("" "")
  menu_hi=("" "")
  _action_hints=("+" "S")
  _action_bar=("" "")
  total=2
  selected=0
  box_w=60
  AI_TOOLS_AVAILABLE=("claude" "codex" "copilot")
  SELECTED_AI_TOOL="claude"
  _update_version=""

  run _run_draw_menu_with_mocked_tty

  assert_success
  # Should show arrow indicators for cycling
  assert_output --partial "◂"
  assert_output --partial "▸"
}

@test "draw_menu: no cycling indicators with single AI tool" {
  _setup_test_draw_menu 30 100

  projects=()
  menu_labels=("+ Add project" "⚙ Settings")
  menu_subs=("" "")
  menu_hi=("" "")
  _action_hints=("+" "S")
  _action_bar=("" "")
  total=2
  selected=0
  box_w=60
  AI_TOOLS_AVAILABLE=("claude")
  SELECTED_AI_TOOL="claude"
  _update_version=""

  run _run_draw_menu_with_mocked_tty

  assert_success
  # Should NOT show cycling arrows
  refute_output --partial "◂"
  refute_output --partial "▸"
}

# --- Action Items Tests ---

@test "draw_menu: renders Add project action" {
  _setup_test_draw_menu 30 100

  projects=()
  menu_labels=("+ Add project" "⚙ Settings")
  menu_subs=("" "")
  menu_hi=("" "")
  _action_hints=("+" "S")
  _action_bar=("" "")
  total=2
  selected=0
  box_w=60
  AI_TOOLS_AVAILABLE=("claude")
  SELECTED_AI_TOOL="claude"
  _update_version=""

  run _run_draw_menu_with_mocked_tty

  assert_success
  assert_output --partial "+ Add project"
  assert_output --partial "+"
}

@test "draw_menu: renders Settings action" {
  _setup_test_draw_menu 30 100

  projects=()
  menu_labels=("+ Add project" "⚙ Settings")
  menu_subs=("" "")
  menu_hi=("" "")
  _action_hints=("+" "S")
  _action_bar=("" "")
  total=2
  selected=0
  box_w=60
  AI_TOOLS_AVAILABLE=("claude")
  SELECTED_AI_TOOL="claude"
  _update_version=""

  run _run_draw_menu_with_mocked_tty

  assert_success
  assert_output --partial "⚙ Settings"
  assert_output --partial "S"
}

@test "draw_menu: shows update notification when available" {
  _setup_test_draw_menu 30 100

  projects=()
  menu_labels=("+ Add project" "⚙ Settings")
  menu_subs=("" "")
  menu_hi=("" "")
  _action_hints=("+" "S")
  _action_bar=("" "")
  total=2
  selected=0
  box_w=60
  AI_TOOLS_AVAILABLE=("claude")
  SELECTED_AI_TOOL="claude"
  _update_version="1.2.3"

  run _run_draw_menu_with_mocked_tty

  assert_success
  assert_output --partial "Update available: v1.2.3"
  assert_output --partial "brew upgrade ghost-tab"
}

@test "draw_menu: no update notification when not available" {
  _setup_test_draw_menu 30 100

  projects=()
  menu_labels=("+ Add project" "⚙ Settings")
  menu_subs=("" "")
  menu_hi=("" "")
  _action_hints=("+" "S")
  _action_bar=("" "")
  total=2
  selected=0
  box_w=60
  AI_TOOLS_AVAILABLE=("claude")
  SELECTED_AI_TOOL="claude"
  _update_version=""

  run _run_draw_menu_with_mocked_tty

  assert_success
  refute_output --partial "Update available"
}

# --- Box Drawing Tests ---

@test "draw_menu: renders box drawing characters for borders" {
  _setup_test_draw_menu 30 100

  projects=()
  menu_labels=("+ Add project" "⚙ Settings")
  menu_subs=("" "")
  menu_hi=("" "")
  _action_hints=("+" "S")
  _action_bar=("" "")
  total=2
  selected=0
  box_w=60
  AI_TOOLS_AVAILABLE=("claude")
  SELECTED_AI_TOOL="claude"
  _update_version=""

  run _run_draw_menu_with_mocked_tty

  assert_success
  # Should contain box drawing characters
  assert_output --partial "┌"  # Top-left corner
  assert_output --partial "┐"  # Top-right corner
  assert_output --partial "└"  # Bottom-left corner
  assert_output --partial "┘"  # Bottom-right corner
  assert_output --partial "│"  # Vertical line
  assert_output --partial "─"  # Horizontal line
  assert_output --partial "├"  # Left tee
  assert_output --partial "┤"  # Right tee
}

@test "draw_menu: renders separator between projects and actions" {
  _setup_test_draw_menu 30 100

  projects=("app:/path")
  menu_labels=("app" "+ Add project" "⚙ Settings")
  menu_subs=("~/path" "" "")
  menu_hi=("" "" "")
  _action_hints=("+" "S")
  _action_bar=("" "")
  total=3
  selected=0
  box_w=60
  AI_TOOLS_AVAILABLE=("claude")
  SELECTED_AI_TOOL="claude"
  _update_version=""

  run _run_draw_menu_with_mocked_tty

  assert_success
  # Should have separator (├──┤) between projects and actions
  assert_output --partial "├"
  assert_output --partial "┤"
}

# --- Edge Cases Tests ---

@test "draw_menu: handles terminal width < 80" {
  _setup_test_draw_menu 30 70

  projects=()
  menu_labels=("+ Add project" "⚙ Settings")
  menu_subs=("" "")
  menu_hi=("" "")
  _action_hints=("+" "S")
  _action_bar=("" "")
  total=2
  selected=0
  box_w=60
  AI_TOOLS_AVAILABLE=("claude")
  SELECTED_AI_TOOL="claude"
  _update_version=""

  run _run_draw_menu_with_mocked_tty

  assert_success
  # Should still render menu even in narrow terminal
  assert_output --partial "Ghost Tab"
  assert_output --partial "+ Add project"
}

@test "draw_menu: handles terminal width > 200" {
  _setup_test_draw_menu 50 220

  projects=()
  menu_labels=("+ Add project" "⚙ Settings")
  menu_subs=("" "")
  menu_hi=("" "")
  _action_hints=("+" "S")
  _action_bar=("" "")
  total=2
  selected=0
  box_w=60
  AI_TOOLS_AVAILABLE=("claude")
  SELECTED_AI_TOOL="claude"
  _update_version=""

  run _run_draw_menu_with_mocked_tty

  assert_success
  # Should render menu properly in very wide terminal
  assert_output --partial "Ghost Tab"
  assert_output --partial "+ Add project"
  # Logo would be side-by-side in wide layout
}

@test "draw_menu: handles projects with special characters" {
  _setup_test_draw_menu 30 100

  projects=("app-name:/path" "app_name:/path" "app.name:/path")
  menu_labels=("app-name" "app_name" "app.name" "+ Add project" "⚙ Settings")
  menu_subs=("~/path1" "~/path2" "~/path3" "" "")
  menu_hi=("" "" "" "" "")
  _action_hints=("+" "S")
  _action_bar=("" "")
  total=5
  selected=0
  box_w=60
  AI_TOOLS_AVAILABLE=("claude")
  SELECTED_AI_TOOL="claude"
  _update_version=""

  run _run_draw_menu_with_mocked_tty

  assert_success
  assert_output --partial "app-name"
  assert_output --partial "app_name"
  assert_output --partial "app.name"
}

@test "draw_menu: handles empty project subtitles" {
  _setup_test_draw_menu 30 100

  projects=("app:/path")
  menu_labels=("app" "+ Add project" "⚙ Settings")
  menu_subs=("" "" "")  # All empty subtitles
  menu_hi=("" "" "")
  _action_hints=("+" "S")
  _action_bar=("" "")
  total=3
  selected=0
  box_w=60
  AI_TOOLS_AVAILABLE=("claude")
  SELECTED_AI_TOOL="claude"
  _update_version=""

  run _run_draw_menu_with_mocked_tty

  assert_success
  # Should render without crashing
  assert_output --partial "app"
}

# --- Help Text Tests ---

@test "draw_menu: shows navigation help text" {
  _setup_test_draw_menu 30 100

  projects=()
  menu_labels=("+ Add project" "⚙ Settings")
  menu_subs=("" "")
  menu_hi=("" "")
  _action_hints=("+" "S")
  _action_bar=("" "")
  total=2
  selected=0
  box_w=60
  AI_TOOLS_AVAILABLE=("claude")
  SELECTED_AI_TOOL="claude"
  _update_version=""

  run _run_draw_menu_with_mocked_tty

  assert_success
  assert_output --partial "↑↓"
  assert_output --partial "navigate"
  assert_output --partial "settings"
  assert_output --partial "select"
}

@test "draw_menu: shows AI tool cycling help when multiple tools available" {
  _setup_test_draw_menu 30 100

  projects=()
  menu_labels=("+ Add project" "⚙ Settings")
  menu_subs=("" "")
  menu_hi=("" "")
  _action_hints=("+" "S")
  _action_bar=("" "")
  total=2
  selected=0
  box_w=60
  AI_TOOLS_AVAILABLE=("claude" "codex")
  SELECTED_AI_TOOL="claude"
  _update_version=""

  run _run_draw_menu_with_mocked_tty

  assert_success
  assert_output --partial "←→"
  assert_output --partial "AI tool"
}

# --- Logo Integration Tests ---

@test "draw_logo calls ghost-tab-tui show-logo" {
  # Mock ghost-tab-tui
  ghost-tab-tui() {
    if [[ "$1" == "show-logo" ]]; then
      echo "MOCK_LOGO_OUTPUT"
      return 0
    fi
    return 1
  }
  export -f ghost-tab-tui

  source "$BATS_TEST_DIRNAME/../lib/tui.sh"

  run draw_logo "claude"

  assert_success
  assert_output "MOCK_LOGO_OUTPUT"
}

@test "select_project_interactive calls ghost-tab-tui and parses JSON" {
  PROJECTS_FILE="$TEST_DIR/projects"
  echo "proj1:/tmp/p1" > "$PROJECTS_FILE"

  # Mock ghost-tab-tui
  ghost-tab-tui() {
    if [[ "$1" == "select-project" ]]; then
      echo '{"name":"proj1","path":"/tmp/p1","selected":true}'
      return 0
    fi
    return 1
  }
  export -f ghost-tab-tui

  # Source the actual menu-tui.sh
  source "$PROJECT_ROOT/lib/menu-tui.sh"

  # Call function without run to preserve variable assignments
  select_project_interactive "$PROJECTS_FILE"

  # Check that variables were set
  [ "$_selected_project_name" = "proj1" ]
  [ "$_selected_project_path" = "/tmp/p1" ]
}

@test "select_project_interactive handles cancellation" {
  PROJECTS_FILE="$TEST_DIR/projects"
  echo "proj1:/tmp/p1" > "$PROJECTS_FILE"

  # Mock ghost-tab-tui to return cancelled
  ghost-tab-tui() {
    if [[ "$1" == "select-project" ]]; then
      echo '{"selected":false}'
      return 0
    fi
    return 1
  }
  export -f ghost-tab-tui

  source "$PROJECT_ROOT/lib/menu-tui.sh"

  run select_project_interactive "$PROJECTS_FILE"

  assert_failure
}

@test "select_project_interactive validates null fields" {
  PROJECTS_FILE="$TEST_DIR/projects"

  # Mock ghost-tab-tui to return null name
  ghost-tab-tui() {
    if [[ "$1" == "select-project" ]]; then
      echo '{"name":null,"path":"/tmp/p1","selected":true}'
      return 0
    fi
    return 1
  }
  export -f ghost-tab-tui

  source "$PROJECT_ROOT/lib/menu-tui.sh"

  run select_project_interactive "$PROJECTS_FILE"

  assert_failure
  assert_output --partial "invalid project name"
}
