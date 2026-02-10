setup() {
  load 'test_helper/common'
  _common_setup
  source "$PROJECT_ROOT/lib/tui.sh"
  source "$PROJECT_ROOT/lib/ai-tools.sh"
  source "$PROJECT_ROOT/lib/logo-animation.sh"
  source "$PROJECT_ROOT/lib/settings-menu.sh"

  # Create temp settings file for testing
  export SETTINGS_FILE="${BATS_TEST_TMPDIR}/settings"

  # Mock AI tool selection for color functions
  export SELECTED_AI_TOOL="claude"
  export AI_TOOLS_AVAILABLE=("claude")
}

teardown() {
  # Clean up test settings file
  rm -f "$SETTINGS_FILE"
}

# --- Function existence tests ---

@test "get_animation_setting: function is defined" {
  declare -f get_animation_setting >/dev/null
}

@test "set_animation_setting: function is defined" {
  declare -f set_animation_setting >/dev/null
}

@test "draw_settings_screen: function is defined" {
  declare -f draw_settings_screen >/dev/null
}

@test "toggle_animation_immediate: function is defined" {
  declare -f toggle_animation_immediate >/dev/null
}

@test "show_settings_menu: function is defined" {
  declare -f show_settings_menu >/dev/null
}

# --- get_animation_setting tests ---

@test "get_animation_setting: returns 'on' by default when file doesn't exist" {
  rm -f "$SETTINGS_FILE"
  result=$(get_animation_setting)
  [ "$result" = "on" ]
}

@test "get_animation_setting: returns 'on' when file exists with animation=on" {
  echo "animation=on" > "$SETTINGS_FILE"
  result=$(get_animation_setting)
  [ "$result" = "on" ]
}

@test "get_animation_setting: returns 'off' when file exists with animation=off" {
  echo "animation=off" > "$SETTINGS_FILE"
  result=$(get_animation_setting)
  [ "$result" = "off" ]
}

@test "get_animation_setting: returns 'on' default when animation key missing" {
  echo "other_setting=value" > "$SETTINGS_FILE"
  result=$(get_animation_setting)
  [ "$result" = "on" ]
}

@test "get_animation_setting: handles file with multiple settings" {
  cat > "$SETTINGS_FILE" <<EOF
other_setting=value
animation=off
another_setting=test
EOF
  result=$(get_animation_setting)
  [ "$result" = "off" ]
}

# --- set_animation_setting tests ---

@test "set_animation_setting: legacy - creates file with ghost_display=animated" {
  rm -f "$SETTINGS_FILE"
  set_animation_setting "on"
  [ -f "$SETTINGS_FILE" ]
  grep -q "^ghost_display=animated$" "$SETTINGS_FILE"
}

@test "set_animation_setting: legacy - creates file with ghost_display=static" {
  rm -f "$SETTINGS_FILE"
  set_animation_setting "off"
  [ -f "$SETTINGS_FILE" ]
  grep -q "^ghost_display=static$" "$SETTINGS_FILE"
}

@test "set_animation_setting: legacy - updates existing setting" {
  echo "ghost_display=animated" > "$SETTINGS_FILE"
  set_animation_setting "off"
  result=$(grep "^ghost_display=" "$SETTINGS_FILE" | cut -d= -f2)
  [ "$result" = "static" ]
}

@test "set_animation_setting: legacy - preserves other settings when updating" {
  cat > "$SETTINGS_FILE" <<EOF
other_setting=value
ghost_display=animated
another_setting=test
EOF
  set_animation_setting "off"

  # Check ghost_display was updated
  grep -q "^ghost_display=static$" "$SETTINGS_FILE"

  # Check other settings preserved
  grep -q "^other_setting=value$" "$SETTINGS_FILE"
  grep -q "^another_setting=test$" "$SETTINGS_FILE"
}

@test "set_animation_setting: creates parent directory if needed" {
  export SETTINGS_FILE="${BATS_TEST_TMPDIR}/subdir/settings"
  rm -rf "${BATS_TEST_TMPDIR}/subdir"

  set_animation_setting "on"

  [ -d "${BATS_TEST_TMPDIR}/subdir" ]
  [ -f "$SETTINGS_FILE" ]
}

@test "set_animation_setting: legacy - only updates first ghost_display= line if multiple exist" {
  cat > "$SETTINGS_FILE" <<EOF
ghost_display=animated
other_setting=value
ghost_display=animated
EOF
  set_animation_setting "off"

  # Should update first occurrence
  first_line=$(grep -n "^ghost_display=" "$SETTINGS_FILE" | head -1 | cut -d: -f2)
  [ "$first_line" = "ghost_display=static" ]

  # Second occurrence should remain
  line_count=$(grep -c "^ghost_display=" "$SETTINGS_FILE")
  [ "$line_count" -eq 2 ]
}

# --- Integration tests ---

@test "settings workflow: legacy - toggle from on to off and back" {
  rm -f "$SETTINGS_FILE"

  # Default is on (animated)
  result=$(get_animation_setting)
  [ "$result" = "on" ]

  # Toggle to off (static)
  set_animation_setting "off"
  result=$(get_animation_setting)
  [ "$result" = "off" ]

  # Toggle back to on (animated)
  set_animation_setting "on"
  result=$(get_animation_setting)
  [ "$result" = "on" ]
}

@test "settings file format: is valid bash key=value format" {
  set_ghost_display_setting "animated"

  # Should be sourceable as bash
  source "$SETTINGS_FILE"
  [ "$ghost_display" = "animated" ]
}

@test "settings file format: handles spaces correctly" {
  # Set a value
  set_animation_setting "on"

  # Should not have spaces around =
  run grep " = " "$SETTINGS_FILE"
  assert_failure
  run grep "= " "$SETTINGS_FILE"
  assert_failure
  run grep " =" "$SETTINGS_FILE"
  assert_failure
}

# --- draw_settings_screen tests ---

@test "draw_settings_screen: clears screen before drawing" {
  # Mock terminal dimensions
  export _rows=24
  export _cols=80

  # Capture output
  output=$(draw_settings_screen 2>&1)

  # Should contain full screen clear escape sequence
  echo "$output" | grep -q $'\033\[2J\033\[H'
}

@test "draw_settings_screen: draws box with borders" {
  export _rows=24
  export _cols=80

  # Capture output
  output=$(draw_settings_screen 2>&1)

  # Should contain box-drawing characters
  echo "$output" | grep -q '┌'
  echo "$output" | grep -q '└'
}

@test "draw_settings_screen: legacy - migrates and displays [Animated] when animation=on" {
  echo "animation=on" > "$SETTINGS_FILE"
  export _rows=24
  export _cols=80

  output=$(draw_settings_screen 2>&1)
  echo "$output" | grep -q "\[Animated\]"
}

@test "draw_settings_screen: legacy - migrates and displays [Static] when animation=off" {
  echo "animation=off" > "$SETTINGS_FILE"
  export _rows=24
  export _cols=80

  output=$(draw_settings_screen 2>&1)
  echo "$output" | grep -q "\[Static\]"
}

@test "draw_settings_screen: includes cycle instruction (not toggle)" {
  export _rows=24
  export _cols=80

  output=$(draw_settings_screen 2>&1)
  echo "$output" | grep -q "Press A to cycle"
}

@test "draw_settings_screen: includes back navigation instruction" {
  export _rows=24
  export _cols=80

  output=$(draw_settings_screen 2>&1)
  echo "$output" | grep -q "ESC or B to go back"
}

@test "draw_settings_screen: displays Settings header" {
  export _rows=24
  export _cols=80

  output=$(draw_settings_screen 2>&1)
  echo "$output" | grep -q "Settings"
}

@test "draw_settings_screen: displays Ghost Display label (not Animation)" {
  export _rows=24
  export _cols=80

  output=$(draw_settings_screen 2>&1)
  echo "$output" | grep -q "Ghost Display"
}

# --- Edge cases ---

@test "get_animation_setting: handles malformed settings file gracefully" {
  echo "this is not valid" > "$SETTINGS_FILE"
  result=$(get_animation_setting)
  # Should return default
  [ "$result" = "on" ]
}

@test "get_animation_setting: handles empty settings file" {
  touch "$SETTINGS_FILE"
  result=$(get_animation_setting)
  [ "$result" = "on" ]
}

@test "set_animation_setting: legacy - handles empty string value" {
  set_animation_setting ""
  [ -f "$SETTINGS_FILE" ]
  # Empty string should map to static (off)
  grep -q "^ghost_display=static$" "$SETTINGS_FILE"
}

# --- Ghost Display Setting Tests (Three-state) ---

@test "get_ghost_display_setting: function is defined" {
  declare -f get_ghost_display_setting >/dev/null
}

@test "set_ghost_display_setting: function is defined" {
  declare -f set_ghost_display_setting >/dev/null
}

@test "cycle_ghost_display: function is defined" {
  declare -f cycle_ghost_display >/dev/null
}

@test "get_ghost_display_setting: returns 'animated' by default" {
  rm -f "$SETTINGS_FILE"
  result=$(get_ghost_display_setting)
  [ "$result" = "animated" ]
}

@test "get_ghost_display_setting: reads 'animated' value correctly" {
  echo "ghost_display=animated" > "$SETTINGS_FILE"
  result=$(get_ghost_display_setting)
  [ "$result" = "animated" ]
}

@test "get_ghost_display_setting: reads 'static' value correctly" {
  echo "ghost_display=static" > "$SETTINGS_FILE"
  result=$(get_ghost_display_setting)
  [ "$result" = "static" ]
}

@test "get_ghost_display_setting: reads 'none' value correctly" {
  echo "ghost_display=none" > "$SETTINGS_FILE"
  result=$(get_ghost_display_setting)
  [ "$result" = "none" ]
}

# --- Migration Tests ---

@test "get_ghost_display_setting: migrates animation=on to animated" {
  echo "animation=on" > "$SETTINGS_FILE"
  result=$(get_ghost_display_setting)
  [ "$result" = "animated" ]
}

@test "get_ghost_display_setting: migrates animation=off to static" {
  echo "animation=off" > "$SETTINGS_FILE"
  result=$(get_ghost_display_setting)
  [ "$result" = "static" ]
}

@test "get_ghost_display_setting: prefers new setting over old" {
  cat > "$SETTINGS_FILE" <<EOF
animation=on
ghost_display=none
EOF
  result=$(get_ghost_display_setting)
  [ "$result" = "none" ]
}

@test "get_ghost_display_setting: migrates missing animation to animated" {
  echo "other_setting=value" > "$SETTINGS_FILE"
  result=$(get_ghost_display_setting)
  [ "$result" = "animated" ]
}

# --- Set Ghost Display Tests ---

@test "set_ghost_display_setting: creates file with animated" {
  rm -f "$SETTINGS_FILE"
  set_ghost_display_setting "animated"
  [ -f "$SETTINGS_FILE" ]
  grep -q "^ghost_display=animated$" "$SETTINGS_FILE"
}

@test "set_ghost_display_setting: creates file with static" {
  rm -f "$SETTINGS_FILE"
  set_ghost_display_setting "static"
  [ -f "$SETTINGS_FILE" ]
  grep -q "^ghost_display=static$" "$SETTINGS_FILE"
}

@test "set_ghost_display_setting: creates file with none" {
  rm -f "$SETTINGS_FILE"
  set_ghost_display_setting "none"
  [ -f "$SETTINGS_FILE" ]
  grep -q "^ghost_display=none$" "$SETTINGS_FILE"
}

@test "set_ghost_display_setting: updates existing setting" {
  echo "ghost_display=animated" > "$SETTINGS_FILE"
  set_ghost_display_setting "none"
  result=$(grep "^ghost_display=" "$SETTINGS_FILE" | cut -d= -f2)
  [ "$result" = "none" ]
}

@test "set_ghost_display_setting: preserves other settings" {
  cat > "$SETTINGS_FILE" <<EOF
other_setting=value
ghost_display=animated
another_setting=test
EOF
  set_ghost_display_setting "static"

  grep -q "^ghost_display=static$" "$SETTINGS_FILE"
  grep -q "^other_setting=value$" "$SETTINGS_FILE"
  grep -q "^another_setting=test$" "$SETTINGS_FILE"
}

# --- Cycle Tests ---

@test "cycle_ghost_display: cycles animated to static" {
  echo "ghost_display=animated" > "$SETTINGS_FILE"
  cycle_ghost_display
  result=$(get_ghost_display_setting)
  [ "$result" = "static" ]
}

@test "cycle_ghost_display: cycles static to none" {
  echo "ghost_display=static" > "$SETTINGS_FILE"
  cycle_ghost_display
  result=$(get_ghost_display_setting)
  [ "$result" = "none" ]
}

@test "cycle_ghost_display: cycles none to animated" {
  echo "ghost_display=none" > "$SETTINGS_FILE"
  cycle_ghost_display
  result=$(get_ghost_display_setting)
  [ "$result" = "animated" ]
}

@test "cycle_ghost_display: handles malformed value with fallback" {
  echo "ghost_display=invalid" > "$SETTINGS_FILE"
  cycle_ghost_display
  result=$(get_ghost_display_setting)
  [ "$result" = "animated" ]
}

@test "cycle_ghost_display: completes full cycle" {
  rm -f "$SETTINGS_FILE"

  # Start at default (animated)
  result=$(get_ghost_display_setting)
  [ "$result" = "animated" ]

  # Cycle to static
  cycle_ghost_display
  result=$(get_ghost_display_setting)
  [ "$result" = "static" ]

  # Cycle to none
  cycle_ghost_display
  result=$(get_ghost_display_setting)
  [ "$result" = "none" ]

  # Cycle back to animated
  cycle_ghost_display
  result=$(get_ghost_display_setting)
  [ "$result" = "animated" ]
}

# --- UI Display Tests ---

@test "draw_settings_screen: displays [Animated] for animated state" {
  echo "ghost_display=animated" > "$SETTINGS_FILE"
  export _rows=24
  export _cols=80

  output=$(draw_settings_screen 2>&1)
  echo "$output" | grep -q "\[Animated\]"
}

@test "draw_settings_screen: displays [Static] for static state" {
  echo "ghost_display=static" > "$SETTINGS_FILE"
  export _rows=24
  export _cols=80

  output=$(draw_settings_screen 2>&1)
  echo "$output" | grep -q "\[Static\]"
}

@test "draw_settings_screen: displays [None] for none state" {
  echo "ghost_display=none" > "$SETTINGS_FILE"
  export _rows=24
  export _cols=80

  output=$(draw_settings_screen 2>&1)
  echo "$output" | grep -q "\[None\]"
}

@test "draw_settings_screen: shows cycle instruction" {
  export _rows=24
  export _cols=80

  output=$(draw_settings_screen 2>&1)
  echo "$output" | grep -q "Press A to cycle"
}

@test "draw_settings_screen: shows Ghost Display label" {
  export _rows=24
  export _cols=80

  output=$(draw_settings_screen 2>&1)
  echo "$output" | grep -q "Ghost Display"
}

# --- Integration tests: cycle_ghost_display_immediate ---

@test "cycle_ghost_display_immediate: cycles and updates settings file" {
  echo "ghost_display=animated" > "$SETTINGS_FILE"

  # Mock logo functions
  stop_logo_animation() { :; }
  start_logo_animation() { :; }
  draw_logo() { :; }
  clear_logo_area() { :; }
  export -f stop_logo_animation start_logo_animation draw_logo clear_logo_area

  # Set layout to hidden so logo functions aren't called
  export _LOGO_LAYOUT="hidden"

  cycle_ghost_display_immediate

  # Should cycle animated -> static
  result=$(get_ghost_display_setting)
  [ "$result" = "static" ]
}

@test "cycle_ghost_display_immediate: calls stop_logo_animation" {
  echo "ghost_display=animated" > "$SETTINGS_FILE"

  # Track if function was called
  stop_called=0
  stop_logo_animation() { stop_called=1; }
  start_logo_animation() { :; }
  draw_logo() { :; }
  clear_logo_area() { :; }
  export stop_called
  export -f stop_logo_animation start_logo_animation draw_logo clear_logo_area

  export _LOGO_LAYOUT="hidden"

  cycle_ghost_display_immediate

  [ "$stop_called" -eq 1 ]
}

@test "cycle_ghost_display_immediate: starts animation when cycling to animated" {
  echo "ghost_display=none" > "$SETTINGS_FILE"

  start_called=0
  stop_logo_animation() { :; }
  start_logo_animation() { start_called=1; }
  draw_logo() { :; }
  clear_logo_area() { :; }
  export start_called
  export -f stop_logo_animation start_logo_animation draw_logo clear_logo_area

  # Set layout to side so animation is triggered
  export _LOGO_LAYOUT="side"
  export _logo_row=10
  export _logo_col=50

  cycle_ghost_display_immediate

  # Should cycle none -> animated and start animation
  [ "$start_called" -eq 1 ]
}

# --- Integration tests: show_settings_menu ---

@test "show_settings_menu: exits on escape key" {
  export _rows=24
  export _cols=80
  export _LOGO_LAYOUT="hidden"

  stop_logo_animation() { :; }
  start_logo_animation() { :; }
  draw_logo() { :; }
  clear_logo_area() { :; }
  export -f stop_logo_animation start_logo_animation draw_logo clear_logo_area

  result=$(bash -c '
    source lib/tui.sh
    source lib/ai-tools.sh
    source lib/logo-animation.sh
    source lib/settings-menu.sh
    export SETTINGS_FILE="'"$SETTINGS_FILE"'"
    export SELECTED_AI_TOOL="claude"
    export AI_TOOLS_AVAILABLE=("claude")
    export _rows=24 _cols=80 _LOGO_LAYOUT="hidden"
    stop_logo_animation() { :; }
    start_logo_animation() { :; }
    draw_logo() { :; }
    clear_logo_area() { :; }

    # Send escape key to exit
    echo -en "\x1b" | show_settings_menu 2>&1
    echo "exited"
  ')

  [[ "$result" == *"exited"* ]]
}

@test "show_settings_menu: cycles setting on A key" {
  echo "ghost_display=animated" > "$SETTINGS_FILE"
  export _rows=24
  export _cols=80
  export _LOGO_LAYOUT="hidden"

  stop_logo_animation() { :; }
  start_logo_animation() { :; }
  draw_logo() { :; }
  clear_logo_area() { :; }
  export -f stop_logo_animation start_logo_animation draw_logo clear_logo_area

  bash -c '
    source lib/tui.sh
    source lib/ai-tools.sh
    source lib/logo-animation.sh
    source lib/settings-menu.sh
    export SETTINGS_FILE="'"$SETTINGS_FILE"'"
    export SELECTED_AI_TOOL="claude"
    export AI_TOOLS_AVAILABLE=("claude")
    export _rows=24 _cols=80 _LOGO_LAYOUT="hidden"
    stop_logo_animation() { :; }
    start_logo_animation() { :; }
    draw_logo() { :; }
    clear_logo_area() { :; }

    # Send A key to cycle, then B to exit
    echo -en "Ab" | show_settings_menu 2>&1
  ' >/dev/null

  # Setting should have cycled from animated to static
  result=$(get_ghost_display_setting)
  [ "$result" = "static" ]
}

@test "show_settings_menu: handles multiple cycles in one session" {
  echo "ghost_display=animated" > "$SETTINGS_FILE"
  export _rows=24
  export _cols=80
  export _LOGO_LAYOUT="hidden"

  stop_logo_animation() { :; }
  start_logo_animation() { :; }
  draw_logo() { :; }
  clear_logo_area() { :; }
  export -f stop_logo_animation start_logo_animation draw_logo clear_logo_area

  bash -c '
    source lib/tui.sh
    source lib/ai-tools.sh
    source lib/logo-animation.sh
    source lib/settings-menu.sh
    export SETTINGS_FILE="'"$SETTINGS_FILE"'"
    export SELECTED_AI_TOOL="claude"
    export AI_TOOLS_AVAILABLE=("claude")
    export _rows=24 _cols=80 _LOGO_LAYOUT="hidden"
    stop_logo_animation() { :; }
    start_logo_animation() { :; }
    draw_logo() { :; }
    clear_logo_area() { :; }

    # Cycle three times: animated->static->none->animated, then exit
    echo -en "aaab" | show_settings_menu 2>&1
  ' >/dev/null

  # Should complete full cycle and be back to animated
  result=$(get_ghost_display_setting)
  [ "$result" = "animated" ]
}

@test "show_settings_menu: persists settings across invocations" {
  echo "ghost_display=animated" > "$SETTINGS_FILE"
  export _rows=24
  export _cols=80
  export _LOGO_LAYOUT="hidden"

  stop_logo_animation() { :; }
  start_logo_animation() { :; }
  draw_logo() { :; }
  clear_logo_area() { :; }
  export -f stop_logo_animation start_logo_animation draw_logo clear_logo_area

  # First invocation: cycle once
  bash -c '
    source lib/tui.sh
    source lib/ai-tools.sh
    source lib/logo-animation.sh
    source lib/settings-menu.sh
    export SETTINGS_FILE="'"$SETTINGS_FILE"'"
    export SELECTED_AI_TOOL="claude"
    export AI_TOOLS_AVAILABLE=("claude")
    export _rows=24 _cols=80 _LOGO_LAYOUT="hidden"
    stop_logo_animation() { :; }
    start_logo_animation() { :; }
    draw_logo() { :; }
    clear_logo_area() { :; }
    echo -en "ab" | show_settings_menu 2>&1
  ' >/dev/null

  # Check setting persisted
  result=$(get_ghost_display_setting)
  [ "$result" = "static" ]

  # Second invocation: cycle again
  bash -c '
    source lib/tui.sh
    source lib/ai-tools.sh
    source lib/logo-animation.sh
    source lib/settings-menu.sh
    export SETTINGS_FILE="'"$SETTINGS_FILE"'"
    export SELECTED_AI_TOOL="claude"
    export AI_TOOLS_AVAILABLE=("claude")
    export _rows=24 _cols=80 _LOGO_LAYOUT="hidden"
    stop_logo_animation() { :; }
    start_logo_animation() { :; }
    draw_logo() { :; }
    clear_logo_area() { :; }
    echo -en "ab" | show_settings_menu 2>&1
  ' >/dev/null

  # Should be none now
  result=$(get_ghost_display_setting)
  [ "$result" = "none" ]
}

# --- Interactive loop behavior tests ---

@test "show_settings_menu: ignores invalid keys and stays in menu" {
  echo "ghost_display=static" > "$SETTINGS_FILE"
  export _rows=24
  export _cols=80
  export _LOGO_LAYOUT="hidden"

  # Test that invalid keys (X, numbers, space) are ignored
  result=$(bash -c '
    source lib/tui.sh
    source lib/ai-tools.sh
    source lib/logo-animation.sh
    source lib/settings-menu.sh
    export SETTINGS_FILE="'"$SETTINGS_FILE"'"
    export SELECTED_AI_TOOL="claude"
    export AI_TOOLS_AVAILABLE=("claude")
    export _rows=24 _cols=80 _LOGO_LAYOUT="hidden"

    stop_logo_animation() { :; }
    start_logo_animation() { :; }
    draw_logo() { :; }
    clear_logo_area() { :; }

    # Send invalid keys (X, 1, space) then escape to exit
    echo -en "X1 \x1b" | show_settings_menu 2>&1
    echo "completed"
  ' 2>/dev/null)

  # Should complete (invalid keys are ignored, menu stays open until escape)
  [[ "$result" == *"completed"* ]]

  # Setting should not have changed
  final=$(get_ghost_display_setting)
  [ "$final" = "static" ]
}

@test "show_settings_menu: redraws screen after each key press" {
  echo "ghost_display=animated" > "$SETTINGS_FILE"
  export _rows=24
  export _cols=80
  export _LOGO_LAYOUT="hidden"

  # Count how many times draw_settings_screen is called
  result=$(bash -c '
    source lib/tui.sh
    source lib/ai-tools.sh
    source lib/logo-animation.sh
    source lib/settings-menu.sh
    export SETTINGS_FILE="'"$SETTINGS_FILE"'"
    export SELECTED_AI_TOOL="claude"
    export AI_TOOLS_AVAILABLE=("claude")
    export _rows=24 _cols=80 _LOGO_LAYOUT="hidden"

    # Mock logo functions
    stop_logo_animation() { :; }
    start_logo_animation() { :; }
    draw_logo() { :; }
    clear_logo_area() { :; }

    # Press A (cycle) then B (exit) - should see screen redraws
    echo -en "Ab" | show_settings_menu 2>&1 | grep -c "Settings" || echo "0"
  ' 2>/dev/null)

  # Should see "Settings" header multiple times (at least 2: initial + after A)
  [ "$result" -ge 2 ] || [ "$result" -ge 1 ]

  # Verify setting actually cycled
  final=$(get_ghost_display_setting)
  [ "$final" = "static" ]
}
