setup() {
  load 'test_helper/common'
  _common_setup
  source "$PROJECT_ROOT/lib/tui.sh"
  source "$PROJECT_ROOT/lib/settings-json.sh"

  TEST_TMP="$(mktemp -d)"
}

teardown() {
  rm -rf "$TEST_TMP"
}

# --- merge_claude_settings ---

@test "merge_claude_settings: creates new file with statusLine" {
  run merge_claude_settings "$TEST_TMP/settings.json"
  assert_output --partial "Created Claude settings"
  [ -f "$TEST_TMP/settings.json" ]
  run cat "$TEST_TMP/settings.json"
  assert_output --partial "statusLine"
}

@test "merge_claude_settings: skips when statusLine already exists" {
  echo '{"statusLine": {"type": "command"}}' > "$TEST_TMP/settings.json"
  run merge_claude_settings "$TEST_TMP/settings.json"
  assert_output --partial "already configured"
}

# --- add_sound_notification_hook ---

@test "add_sound_notification_hook: adds hook to empty settings" {
  echo '{}' > "$TEST_TMP/settings.json"
  run add_sound_notification_hook "$TEST_TMP/settings.json" "afplay /System/Library/Sounds/Bottle.aiff &"
  assert_output "added"
  run cat "$TEST_TMP/settings.json"
  assert_output --partial "idle_prompt"
  assert_output --partial "Bottle.aiff"
}

@test "add_sound_notification_hook: skips when already exists" {
  cat > "$TEST_TMP/settings.json" << 'EOF'
{
  "hooks": {
    "Notification": [
      {
        "matcher": "idle_prompt",
        "hooks": [{"type": "command", "command": "afplay /System/Library/Sounds/Bottle.aiff &"}]
      }
    ]
  }
}
EOF
  run add_sound_notification_hook "$TEST_TMP/settings.json" "afplay /System/Library/Sounds/Bottle.aiff &"
  assert_output "exists"
}

@test "add_sound_notification_hook: creates file when missing" {
  run add_sound_notification_hook "$TEST_TMP/new-settings.json" "afplay /System/Library/Sounds/Bottle.aiff &"
  assert_output "added"
  [ -f "$TEST_TMP/new-settings.json" ]
}

# --- add_spinner_hooks ---

@test "add_spinner_hooks: adds both hooks to empty settings" {
  echo '{}' > "$TEST_TMP/settings.json"
  add_spinner_hooks "$TEST_TMP/settings.json" "bash ~/.claude/tab-spinner-start.sh &" "bash ~/.claude/tab-spinner-stop.sh"
  run cat "$TEST_TMP/settings.json"
  assert_output --partial "tab-spinner-start"
  assert_output --partial "tab-spinner-stop"
  assert_output --partial "Notification"
  assert_output --partial "UserPromptSubmit"
}

@test "add_spinner_hooks: idempotent — running twice doesn't duplicate" {
  echo '{}' > "$TEST_TMP/settings.json"
  add_spinner_hooks "$TEST_TMP/settings.json" "bash start.sh &" "bash stop.sh"
  add_spinner_hooks "$TEST_TMP/settings.json" "bash start.sh &" "bash stop.sh"

  # Count occurrences of the start command — should be exactly 1
  local count
  count=$(grep -c "start.sh" "$TEST_TMP/settings.json" || true)
  [ "$count" -eq 1 ]
}
