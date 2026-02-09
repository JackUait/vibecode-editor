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

@test "setup_sound_notification: adds hook to empty settings" {
  echo '{}' > "$TEST_TMP/settings.json"
  run setup_sound_notification "$TEST_TMP/settings.json" "afplay /System/Library/Sounds/Bottle.aiff &"
  assert_output --partial "configured"
  run cat "$TEST_TMP/settings.json"
  assert_output --partial "idle_prompt"
}

@test "setup_sound_notification: reports already exists" {
  cat > "$TEST_TMP/settings.json" << 'EOF'
{
  "hooks": {
    "Notification": [
      {
        "matcher": "idle_prompt",
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


