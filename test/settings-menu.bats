setup() {
  load 'test_helper/common'
  _common_setup
  source "$PROJECT_ROOT/lib/tui.sh"
  source "$PROJECT_ROOT/lib/logo-animation.sh"
  source "$PROJECT_ROOT/lib/settings-menu.sh"

  # Create temp settings file for testing
  export SETTINGS_FILE="${BATS_TEST_TMPDIR}/settings"
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

@test "set_animation_setting: creates file with animation=on" {
  rm -f "$SETTINGS_FILE"
  set_animation_setting "on"
  [ -f "$SETTINGS_FILE" ]
  grep -q "^animation=on$" "$SETTINGS_FILE"
}

@test "set_animation_setting: creates file with animation=off" {
  rm -f "$SETTINGS_FILE"
  set_animation_setting "off"
  [ -f "$SETTINGS_FILE" ]
  grep -q "^animation=off$" "$SETTINGS_FILE"
}

@test "set_animation_setting: updates existing animation setting" {
  echo "animation=on" > "$SETTINGS_FILE"
  set_animation_setting "off"
  result=$(grep "^animation=" "$SETTINGS_FILE" | cut -d= -f2)
  [ "$result" = "off" ]
}

@test "set_animation_setting: preserves other settings when updating" {
  cat > "$SETTINGS_FILE" <<EOF
other_setting=value
animation=on
another_setting=test
EOF
  set_animation_setting "off"

  # Check animation was updated
  grep -q "^animation=off$" "$SETTINGS_FILE"

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

@test "set_animation_setting: only updates first animation= line if multiple exist" {
  cat > "$SETTINGS_FILE" <<EOF
animation=on
other_setting=value
animation=on
EOF
  set_animation_setting "off"

  # Should update first occurrence
  first_line=$(grep -n "^animation=" "$SETTINGS_FILE" | head -1 | cut -d: -f2)
  [ "$first_line" = "animation=off" ]

  # Second occurrence should remain
  line_count=$(grep -c "^animation=" "$SETTINGS_FILE")
  [ "$line_count" -eq 2 ]
}

# --- Integration tests ---

@test "settings workflow: toggle from on to off and back" {
  rm -f "$SETTINGS_FILE"

  # Default is on
  result=$(get_animation_setting)
  [ "$result" = "on" ]

  # Toggle to off
  set_animation_setting "off"
  result=$(get_animation_setting)
  [ "$result" = "off" ]

  # Toggle back to on
  set_animation_setting "on"
  result=$(get_animation_setting)
  [ "$result" = "on" ]
}

@test "settings file format: is valid bash key=value format" {
  set_animation_setting "on"

  # Should be sourceable as bash
  source "$SETTINGS_FILE"
  [ "$animation" = "on" ]
}

@test "settings file format: handles spaces correctly" {
  # Set a value
  set_animation_setting "on"

  # Should not have spaces around =
  ! grep -q " = " "$SETTINGS_FILE"
  ! grep -q "= " "$SETTINGS_FILE"
  ! grep -q " =" "$SETTINGS_FILE"
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

@test "set_animation_setting: handles empty string value" {
  set_animation_setting ""
  [ -f "$SETTINGS_FILE" ]
  grep -q "^animation=$" "$SETTINGS_FILE"
}
