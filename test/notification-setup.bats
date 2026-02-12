setup() {
  load 'test_helper/common'
  _common_setup
  source "$PROJECT_ROOT/lib/tui.sh"
  source "$PROJECT_ROOT/lib/settings-json.sh"
  source "$PROJECT_ROOT/lib/notification-setup.sh"
  TEST_TMP="$(mktemp -d)"

  FAKE_HOME="$TEST_TMP/home"
  mkdir -p "$FAKE_HOME/.claude"
}

teardown() {
  rm -rf "$TEST_TMP"
}

# --- setup_sound_notification ---

@test "setup_sound_notification: adds Stop hook to empty settings" {
  echo '{}' > "$TEST_TMP/settings.json"
  run setup_sound_notification "$TEST_TMP/settings.json" "afplay /System/Library/Sounds/Bottle.aiff &"
  assert_output --partial "configured"
  run cat "$TEST_TMP/settings.json"
  assert_output --partial '"Stop"'
  refute_output --partial "idle_prompt"
}

@test "setup_sound_notification: reports already exists" {
  cat > "$TEST_TMP/settings.json" << 'EOF'
{
  "hooks": {
    "Stop": [
      {
        "hooks": [{"type": "command", "command": "afplay sound &"}]
      }
    ]
  }
}
EOF
  run setup_sound_notification "$TEST_TMP/settings.json" "afplay sound &"
  assert_output --partial "already configured"
}

@test "setup_sound_notification: creates file when missing" {
  run setup_sound_notification "$TEST_TMP/new-settings.json" "afplay sound &"
  assert_output --partial "configured"
  [ -f "$TEST_TMP/new-settings.json" ]
}

# --- is_sound_enabled ---

@test "is_sound_enabled: returns false when features file missing" {
  run is_sound_enabled "claude" "$TEST_TMP/nonexistent"
  assert_success
  assert_output "false"
}

@test "is_sound_enabled: returns false when sound key missing" {
  echo '{}' > "$TEST_TMP/claude-features.json"
  run is_sound_enabled "claude" "$TEST_TMP"
  assert_success
  assert_output "false"
}

@test "is_sound_enabled: returns true when sound is true" {
  echo '{"sound": true}' > "$TEST_TMP/claude-features.json"
  run is_sound_enabled "claude" "$TEST_TMP"
  assert_success
  assert_output "true"
}

@test "is_sound_enabled: returns false when sound is false" {
  echo '{"sound": false}' > "$TEST_TMP/claude-features.json"
  run is_sound_enabled "claude" "$TEST_TMP"
  assert_success
  assert_output "false"
}

# --- remove_sound_notification ---

@test "remove_sound_notification: removes Stop hook from settings" {
  cat > "$TEST_TMP/settings.json" << 'EOF'
{
  "hooks": {
    "Stop": [
      {
        "hooks": [{"type": "command", "command": "afplay /System/Library/Sounds/Bottle.aiff &"}]
      }
    ]
  }
}
EOF
  run remove_sound_notification "$TEST_TMP/settings.json" "afplay /System/Library/Sounds/Bottle.aiff &"
  assert_success
  assert_output --partial "removed"
  run cat "$TEST_TMP/settings.json"
  refute_output --partial "afplay"
}

@test "remove_sound_notification: noop when hook not present" {
  echo '{}' > "$TEST_TMP/settings.json"
  run remove_sound_notification "$TEST_TMP/settings.json" "afplay /System/Library/Sounds/Bottle.aiff &"
  assert_success
  assert_output --partial "not_found"
}

@test "remove_sound_notification: removes matching hook, keeps others" {
  cat > "$TEST_TMP/settings.json" << 'EOF'
{
  "hooks": {
    "Stop": [
      {
        "hooks": [{"type": "command", "command": "afplay /System/Library/Sounds/Bottle.aiff &"}]
      },
      {
        "hooks": [{"type": "command", "command": "other-command"}]
      }
    ]
  }
}
EOF
  run remove_sound_notification "$TEST_TMP/settings.json" "afplay /System/Library/Sounds/Bottle.aiff &"
  assert_success
  assert_output --partial "removed"
  run cat "$TEST_TMP/settings.json"
  refute_output --partial "afplay"
  assert_output --partial "other-command"
}

# --- set_sound_feature_flag ---

@test "set_sound_feature_flag: creates file with sound true" {
  run set_sound_feature_flag "claude" "$TEST_TMP" true
  assert_success
  [ -f "$TEST_TMP/claude-features.json" ]
  run python3 -c "import json; print(json.load(open('$TEST_TMP/claude-features.json'))['sound'])"
  assert_output "True"
}

@test "set_sound_feature_flag: sets sound false in existing file" {
  echo '{"sound": true}' > "$TEST_TMP/claude-features.json"
  run set_sound_feature_flag "claude" "$TEST_TMP" false
  assert_success
  run python3 -c "import json; print(json.load(open('$TEST_TMP/claude-features.json'))['sound'])"
  assert_output "False"
}

@test "set_sound_feature_flag: preserves other keys" {
  echo '{"sound": false, "other": 42}' > "$TEST_TMP/claude-features.json"
  run set_sound_feature_flag "claude" "$TEST_TMP" true
  assert_success
  run python3 -c "import json; d=json.load(open('$TEST_TMP/claude-features.json')); print(d['sound'], d['other'])"
  assert_output "True 42"
}

# --- toggle_sound_notification ---

@test "toggle_sound_notification: enables for claude" {
  source "$PROJECT_ROOT/lib/settings-json.sh"
  local config_dir="$TEST_TMP/config"
  mkdir -p "$config_dir"
  echo '{}' > "$TEST_TMP/settings.json"

  run toggle_sound_notification "claude" "$config_dir" "$TEST_TMP/settings.json"
  assert_success
  assert_output --partial "enabled"

  # Verify feature flag was set
  run python3 -c "import json; print(json.load(open('$config_dir/claude-features.json'))['sound'])"
  assert_output "True"

  # Verify hook was added
  run cat "$TEST_TMP/settings.json"
  assert_output --partial "Stop"
}

@test "toggle_sound_notification: disables for claude" {
  source "$PROJECT_ROOT/lib/settings-json.sh"
  local config_dir="$TEST_TMP/config"
  mkdir -p "$config_dir"
  echo '{"sound": true}' > "$config_dir/claude-features.json"
  cat > "$TEST_TMP/settings.json" << 'EOF'
{
  "hooks": {
    "Stop": [
      {
        "hooks": [{"type": "command", "command": "afplay /System/Library/Sounds/Bottle.aiff &"}]
      }
    ]
  }
}
EOF

  run toggle_sound_notification "claude" "$config_dir" "$TEST_TMP/settings.json"
  assert_success
  assert_output --partial "disabled"

  # Verify feature flag was set
  run python3 -c "import json; print(json.load(open('$config_dir/claude-features.json'))['sound'])"
  assert_output "False"

  # Verify hook was removed
  run cat "$TEST_TMP/settings.json"
  refute_output --partial "afplay"
}
