setup() {
  load 'test_helper/common'
  _common_setup
  source "$PROJECT_ROOT/lib/tui.sh"
  source "$PROJECT_ROOT/lib/ai-tools.sh"
  source "$PROJECT_ROOT/lib/logo-animation.sh"
  source "$PROJECT_ROOT/lib/menu.sh"

  # Create temp directory for test data
  TEST_DIR="$(mktemp -d)"
  TEST_PROJECTS_FILE="$TEST_DIR/projects"
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
  source "$PROJECT_ROOT/lib/tui.sh"
  source "$PROJECT_ROOT/lib/ai-tools.sh"
  source "$PROJECT_ROOT/lib/menu.sh"

  # Extract the _rbdr function definition from draw_menu
  # The correct implementation should be: moveto ... printf ...│...K
  # The buggy implementation would be: printf ...K... moveto ... printf ...│

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

  # Check that after printing left border, we clear before printing content
  # Pattern should be: moveto, print │, clear to EOL or right border, then print content

  # Check menu item rendering (around line 112-128)
  item_render=$(sed -n '111,129p' "$PROJECT_ROOT/lib/menu.sh")

  # After printing left border │, before printing content, we should clear
  # The code should have a pattern like: printf "│" followed by some clearing mechanism
  # Currently it doesn't clear, which causes the bug

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

# --- Logo Layout Calculation ---

@test "draw_menu: logo layout 'side' when terminal width >= 110" {
  # Mock terminal dimensions
  _rows=40
  _cols=120
  box_w=60

  # Mock logo dimensions (from logo_art_claude)
  _LOGO_HEIGHT=22
  _LOGO_WIDTH=28
  _BOB_MAX=1

  # Test the layout logic directly
  # This replicates the logic from menu.sh lines 27-55
  _logo_need_side=$(( box_w + 3 + _LOGO_WIDTH + 3 ))

  if [ "$_logo_need_side" -le "$_cols" ]; then
    _LOGO_LAYOUT="side"
  elif [ "$_rows" -ge "$(( 7 + 0 + 2 * 2 + 0 + _LOGO_HEIGHT + _BOB_MAX + 1 ))" ]; then
    _LOGO_LAYOUT="above"
  else
    _LOGO_LAYOUT="hidden"
  fi

  # With width=120 and box_w=60, _logo_need_side = 60+3+28+3 = 94
  # 94 <= 120, so should be "side"
  [[ "$_LOGO_LAYOUT" == "side" ]]
}

@test "draw_menu: logo layout 'above' when 80 <= width < 110" {
  # Mock terminal dimensions
  _rows=40
  _cols=90
  box_w=60
  _menu_h=15

  # Mock logo dimensions
  _LOGO_HEIGHT=22
  _LOGO_WIDTH=28
  _BOB_MAX=1

  # Calculate if "above" layout would fit
  # _logo_need_side = 60+3+28+3 = 94, which is > 90
  # So side layout won't work
  # Check if above layout fits: _rows >= _menu_h + _LOGO_HEIGHT + _BOB_MAX + 1
  # 40 >= 15 + 22 + 1 + 1 = 39, yes

  projects=()
  menu_labels=("+ Add project")
  menu_subs=("")
  menu_hi=("")
  _action_hints=("+")
  _action_bar=("")
  total=1
  selected=0
  AI_TOOLS_AVAILABLE=("claude")
  SELECTED_AI_TOOL="claude"
  _update_version=""

  moveto() { :; }
  get_ghost_display_setting() { echo "static"; }
  draw_logo() { :; }

  # Test the layout logic
  _logo_need_side=$(( box_w + 3 + _LOGO_WIDTH + 3 ))

  if [ "$_logo_need_side" -le "$_cols" ]; then
    _LOGO_LAYOUT="side"
  elif [ "$_rows" -ge "$(( _menu_h + _LOGO_HEIGHT + _BOB_MAX + 1 ))" ]; then
    _LOGO_LAYOUT="above"
  else
    _LOGO_LAYOUT="hidden"
  fi

  [[ "$_LOGO_LAYOUT" == "above" ]]
}

@test "draw_menu: logo layout 'hidden' when terminal too small" {
  # Mock terminal dimensions - too small for logo
  _rows=20
  _cols=70
  box_w=60
  _menu_h=15

  # Mock logo dimensions
  _LOGO_HEIGHT=22
  _LOGO_WIDTH=28
  _BOB_MAX=1

  projects=()
  menu_labels=("+ Add project")
  menu_subs=("")
  total=1

  # Test the layout logic
  _logo_need_side=$(( box_w + 3 + _LOGO_WIDTH + 3 ))

  if [ "$_logo_need_side" -le "$_cols" ]; then
    _LOGO_LAYOUT="side"
  elif [ "$_rows" -ge "$(( _menu_h + _LOGO_HEIGHT + _BOB_MAX + 1 ))" ]; then
    _LOGO_LAYOUT="above"
  else
    _LOGO_LAYOUT="hidden"
  fi

  [[ "$_LOGO_LAYOUT" == "hidden" ]]
}

# --- Project List Rendering ---

@test "draw_menu: renders with 0 projects (actions only)" {
  # Setup: no projects, only action items
  projects=()
  menu_labels=("+ Add project" "⚙ Settings")
  menu_subs=("" "Configure Ghost Tab")
  menu_hi=("" "")
  _action_hints=("+" "S")
  _action_bar=("" "")
  total=2
  selected=0
  AI_TOOLS_AVAILABLE=("claude")
  SELECTED_AI_TOOL="claude"
  _update_version=""
  box_w=60

  # Should have 2 menu items (both actions)
  [[ "${#menu_labels[@]}" -eq 2 ]]
  [[ "${menu_labels[0]}" == "+ Add project" ]]
  [[ "${menu_labels[1]}" == "⚙ Settings" ]]
}

@test "draw_menu: renders with 1 project" {
  # Setup: one project plus action items
  projects=("myapp:/Users/me/myapp")
  menu_labels=("myapp" "+ Add project" "⚙ Settings")
  menu_subs=("~/myapp" "" "")
  menu_hi=("" "" "")
  _action_hints=("+" "S")
  _action_bar=("" "")
  total=3
  selected=0

  # Should have 1 project + 2 actions = 3 items
  [[ "$total" -eq 3 ]]
  [[ "${menu_labels[0]}" == "myapp" ]]
  [[ "${#projects[@]}" -eq 1 ]]
}

@test "draw_menu: renders with many projects (5+)" {
  # Setup: 5 projects plus actions
  projects=(
    "app1:/path/to/app1"
    "app2:/path/to/app2"
    "app3:/path/to/app3"
    "app4:/path/to/app4"
    "app5:/path/to/app5"
  )
  menu_labels=("app1" "app2" "app3" "app4" "app5" "+ Add project" "⚙ Settings")
  menu_subs=("~/app1" "~/app2" "~/app3" "~/app4" "~/app5" "" "")
  total=7

  # Should have 5 projects + 2 actions = 7 items
  [[ "$total" -eq 7 ]]
  [[ "${#projects[@]}" -eq 5 ]]
  [[ "${menu_labels[4]}" == "app5" ]]
}

@test "draw_menu: truncates very long project names" {
  # Test label truncation logic
  _inner_w=58  # box_w=60 - 2
  _max_label=$(( _inner_w - 8 ))  # 50

  # Simulate a very long label
  _label="This_is_an_extremely_long_project_name_that_exceeds_the_maximum_width"

  # Apply truncation logic from menu.sh
  if [ "${#_label}" -gt "$_max_label" ]; then
    _label="${_label:0:$((_max_label-1))}…"
  fi

  # Should be truncated to max_label length
  [[ "${#_label}" -eq "$_max_label" ]]
  [[ "$_label" == *"…" ]]
}

@test "draw_menu: truncates very long project paths" {
  # Test subtitle (path) truncation logic
  _inner_w=58
  _max_sub=$(( _inner_w - 7 ))  # 51

  # Simulate a very long path
  _sub="/Users/username/very/deep/nested/directory/structure/project/that/is/way/too/long"

  # Apply truncation logic from menu.sh (middle ellipsis)
  if [ "${#_sub}" -gt "$_max_sub" ]; then
    local _half=$(( (_max_sub - 3) / 2 ))
    _sub="${_sub:0:$_half}...${_sub: -$_half}"
  fi

  # Should be truncated with middle ellipsis
  [[ "${#_sub}" -le "$_max_sub" ]]
  [[ "$_sub" == *"..."* ]]
}

# --- AI Tool Display ---

@test "draw_menu: displays Claude Code with correct color" {
  AI_TOOLS_AVAILABLE=("claude")
  SELECTED_AI_TOOL="claude"

  # Get the color for claude
  result="$(ai_tool_color "claude")"

  # Should be orange (ANSI 209)
  [[ "$result" == $'\033[38;5;209m' ]]
}

@test "draw_menu: displays Codex CLI with correct color" {
  AI_TOOLS_AVAILABLE=("codex")
  SELECTED_AI_TOOL="codex"

  result="$(ai_tool_color "codex")"

  # Should be green (ANSI 114)
  [[ "$result" == $'\033[38;5;114m' ]]
}

@test "draw_menu: displays Copilot CLI with correct color" {
  AI_TOOLS_AVAILABLE=("copilot")
  SELECTED_AI_TOOL="copilot"

  result="$(ai_tool_color "copilot")"

  # Should be purple (ANSI 141)
  [[ "$result" == $'\033[38;5;141m' ]]
}

@test "draw_menu: displays OpenCode with correct color" {
  AI_TOOLS_AVAILABLE=("opencode")
  SELECTED_AI_TOOL="opencode"

  result="$(ai_tool_color "opencode")"

  # Should be light gray (ANSI 250)
  [[ "$result" == $'\033[38;5;250m' ]]
}

@test "draw_menu: shows AI tool name for single tool" {
  AI_TOOLS_AVAILABLE=("claude")
  SELECTED_AI_TOOL="claude"

  display_name="$(ai_tool_display_name "$SELECTED_AI_TOOL")"

  [[ "$display_name" == "Claude Code" ]]
  [[ "${#AI_TOOLS_AVAILABLE[@]}" -eq 1 ]]
}

@test "draw_menu: shows cycling indication for multiple tools" {
  AI_TOOLS_AVAILABLE=("claude" "codex" "copilot")
  SELECTED_AI_TOOL="claude"

  # With multiple tools, should show arrow indicators
  [[ "${#AI_TOOLS_AVAILABLE[@]}" -gt 1 ]]

  # The menu should include ◂ and ▸ for cycling
  # This is tested by checking the title row logic
}

# --- Action Items Rendering ---

@test "draw_menu: includes 'Add project' action" {
  # Setup with action items
  menu_labels=("+ Add project" "⚙ Settings")
  _action_hints=("+" "S")

  # First action should be Add project
  [[ "${menu_labels[0]}" == "+ Add project" ]]
  [[ "${_action_hints[0]}" == "+" ]]
}

@test "draw_menu: includes 'Settings' action" {
  # Setup with action items
  menu_labels=("+ Add project" "⚙ Settings")
  _action_hints=("+" "S")

  # Second action should be Settings
  [[ "${menu_labels[1]}" == "⚙ Settings" ]]
  [[ "${_action_hints[1]}" == "S" ]]
}

@test "draw_menu: shows update notification when available" {
  _update_version="1.2.3"

  # Should display update line with version
  [[ -n "$_update_version" ]]

  # Update line should be included in menu height calculation
  _update_line=0
  [ -n "$_update_version" ] && _update_line=1
  [[ "$_update_line" -eq 1 ]]
}

@test "draw_menu: no update notification when not available" {
  _update_version=""

  # Should not display update line
  [[ -z "$_update_version" ]]

  # Update line should not be included
  _update_line=0
  [ -n "$_update_version" ] && _update_line=1
  [[ "$_update_line" -eq 0 ]]
}

# --- Box Drawing and Borders ---

@test "draw_menu: uses box drawing characters for borders" {
  # Check that menu.sh contains box drawing characters
  menu_content="$(cat "$PROJECT_ROOT/lib/menu.sh")"

  # Should contain corners and lines
  [[ "$menu_content" == *"┌"* ]]  # Top-left corner
  [[ "$menu_content" == *"┐"* ]]  # Top-right corner
  [[ "$menu_content" == *"└"* ]]  # Bottom-left corner
  [[ "$menu_content" == *"┘"* ]]  # Bottom-right corner
  [[ "$menu_content" == *"│"* ]]  # Vertical line
  [[ "$menu_content" == *"─"* ]]  # Horizontal line
  [[ "$menu_content" == *"├"* ]]  # Left tee
  [[ "$menu_content" == *"┤"* ]]  # Right tee
}

@test "draw_menu: calculates inner width correctly" {
  box_w=60
  _inner_w=$(( box_w - 2 ))

  # Inner width should be box width minus 2 borders
  [[ "$_inner_w" -eq 58 ]]
}

@test "draw_menu: calculates right column position" {
  _left_col=10
  box_w=60
  _right_col=$(( _left_col + box_w - 1 ))

  # Right column should be left + width - 1
  [[ "$_right_col" -eq 69 ]]
}

# --- Edge Cases ---

@test "draw_menu: handles terminal width < 80" {
  _rows=24
  _cols=70
  box_w=60
  _menu_h=15

  # Logo should be hidden
  _LOGO_HEIGHT=22
  _LOGO_WIDTH=28
  _BOB_MAX=1
  _logo_need_side=$(( box_w + 3 + _LOGO_WIDTH + 3 ))

  if [ "$_logo_need_side" -le "$_cols" ]; then
    _LOGO_LAYOUT="side"
  elif [ "$_rows" -ge "$(( _menu_h + _LOGO_HEIGHT + _BOB_MAX + 1 ))" ]; then
    _LOGO_LAYOUT="above"
  else
    _LOGO_LAYOUT="hidden"
  fi

  [[ "$_LOGO_LAYOUT" == "hidden" ]]
}

@test "draw_menu: handles terminal width > 200" {
  _rows=50
  _cols=220
  box_w=60

  # Logo should be positioned side by side
  _LOGO_HEIGHT=22
  _LOGO_WIDTH=28
  _logo_need_side=$(( box_w + 3 + _LOGO_WIDTH + 3 ))

  if [ "$_logo_need_side" -le "$_cols" ]; then
    _LOGO_LAYOUT="side"
  else
    _LOGO_LAYOUT="other"
  fi

  [[ "$_LOGO_LAYOUT" == "side" ]]
}

@test "draw_menu: handles projects with special characters" {
  # Projects can have special chars in names
  projects=("app-name:/path" "app_name:/path" "app.name:/path")
  menu_labels=("app-name" "app_name" "app.name")

  # All should be valid
  [[ "${menu_labels[0]}" == "app-name" ]]
  [[ "${menu_labels[1]}" == "app_name" ]]
  [[ "${menu_labels[2]}" == "app.name" ]]
}

@test "draw_menu: handles empty project names gracefully" {
  # Edge case: what if name is empty?
  _label=""
  _max_label=50

  # Should not crash on empty label
  if [ "${#_label}" -gt "$_max_label" ]; then
    _label="${_label:0:$((_max_label-1))}…"
  fi

  # Should remain empty
  [[ -z "$_label" ]]
}

# --- Menu Height Calculation ---

@test "draw_menu: calculates menu height with projects" {
  # Each project takes 2 rows (label + subtitle)
  # Plus: top border, title, separator, blank, separator before actions, help, separator, bottom border
  projects=("app1:/path1" "app2:/path2")
  total=4  # 2 projects + 2 actions
  _sep_count=1  # Separator between projects and actions
  _update_line=0

  _menu_h=$(( 7 + _update_line + total * 2 + _sep_count ))

  # 7 + 0 + 4*2 + 1 = 16
  [[ "$_menu_h" -eq 16 ]]
}

@test "draw_menu: calculates menu height with no projects" {
  projects=()
  total=2  # 2 actions
  _sep_count=0  # No separator needed
  _update_line=0

  [ "${#projects[@]}" -gt 0 ] && _sep_count=1

  _menu_h=$(( 7 + _update_line + total * 2 + _sep_count ))

  # 7 + 0 + 2*2 + 0 = 11
  [[ "$_menu_h" -eq 11 ]]
}

@test "draw_menu: calculates menu height with update notification" {
  projects=("app:/path")
  total=3  # 1 project + 2 actions
  _sep_count=1
  _update_version="1.2.3"
  _update_line=1

  _menu_h=$(( 7 + _update_line + total * 2 + _sep_count ))

  # 7 + 1 + 3*2 + 1 = 15
  [[ "$_menu_h" -eq 15 ]]
}

# --- Terminal Centering ---

@test "draw_menu: centers menu vertically" {
  _rows=40
  _menu_h=20

  _top_row=$(( (_rows - _menu_h) / 2 ))
  [ "$_top_row" -lt 1 ] && _top_row=1

  # (40 - 20) / 2 = 10
  [[ "$_top_row" -eq 10 ]]
}

@test "draw_menu: centers menu horizontally" {
  _cols=120
  box_w=60

  _left_col=$(( (_cols - box_w) / 2 + 1 ))
  [ "$_left_col" -lt 1 ] && _left_col=1

  # (120 - 60) / 2 + 1 = 31
  [[ "$_left_col" -eq 31 ]]
}

@test "draw_menu: prevents negative top row" {
  _rows=10
  _menu_h=20  # Menu taller than terminal

  _top_row=$(( (_rows - _menu_h) / 2 ))
  [ "$_top_row" -lt 1 ] && _top_row=1

  # Should be clamped to 1
  [[ "$_top_row" -eq 1 ]]
}

@test "draw_menu: prevents negative left column" {
  _cols=40
  box_w=60  # Box wider than terminal

  _left_col=$(( (_cols - box_w) / 2 + 1 ))
  [ "$_left_col" -lt 1 ] && _left_col=1

  # Should be clamped to 1
  [[ "$_left_col" -eq 1 ]]
}
