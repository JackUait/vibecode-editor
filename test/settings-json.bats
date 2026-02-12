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

# --- Migration: Notification.idle_prompt to Stop ---

@test "add_sound_notification_hook: migrates old Notification.idle_prompt to Stop" {
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
  assert_output "added"
  run cat "$TEST_TMP/settings.json"
  assert_output --partial '"Stop"'
  refute_output --partial "idle_prompt"
}

@test "add_sound_notification_hook: migration preserves other Notification hooks" {
  cat > "$TEST_TMP/settings.json" << 'EOF'
{
  "hooks": {
    "Notification": [
      {
        "matcher": "idle_prompt",
        "hooks": [{"type": "command", "command": "afplay /System/Library/Sounds/Bottle.aiff &"}]
      },
      {
        "matcher": "permission_prompt",
        "hooks": [{"type": "command", "command": "echo permission"}]
      }
    ]
  }
}
EOF
  run add_sound_notification_hook "$TEST_TMP/settings.json" "afplay /System/Library/Sounds/Bottle.aiff &"
  assert_output "added"
  run cat "$TEST_TMP/settings.json"
  assert_output --partial '"Stop"'
  assert_output --partial "permission_prompt"
  refute_output --partial "idle_prompt"
}
