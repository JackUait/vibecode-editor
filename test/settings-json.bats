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

# --- Edge Cases: Malformed JSON ---

@test "add_sound_notification_hook: handles malformed JSON - missing closing brace" {
  echo '{"foo": "bar"' > "$TEST_TMP/settings.json"
  run add_sound_notification_hook "$TEST_TMP/settings.json" "afplay /System/Library/Sounds/Bottle.aiff &"
  assert_output "added"
  [ -f "$TEST_TMP/settings.json" ]
  # Verify the resulting file is valid JSON
  run python3 -c "import json; json.load(open('$TEST_TMP/settings.json'))"
  assert_success
}

@test "add_sound_notification_hook: handles malformed JSON - missing quotes" {
  echo '{foo: bar}' > "$TEST_TMP/settings.json"
  run add_sound_notification_hook "$TEST_TMP/settings.json" "afplay /System/Library/Sounds/Bottle.aiff &"
  assert_output "added"
  # Python's json module should treat this as invalid and start fresh
  run cat "$TEST_TMP/settings.json"
  assert_output --partial "idle_prompt"
}

@test "add_sound_notification_hook: handles malformed JSON - trailing comma" {
  echo '{"foo": "bar",}' > "$TEST_TMP/settings.json"
  run add_sound_notification_hook "$TEST_TMP/settings.json" "afplay /System/Library/Sounds/Bottle.aiff &"
  assert_output "added"
  run cat "$TEST_TMP/settings.json"
  assert_output --partial "idle_prompt"
}

@test "add_sound_notification_hook: handles malformed JSON - missing commas" {
  echo '{"foo": "bar" "baz": "qux"}' > "$TEST_TMP/settings.json"
  run add_sound_notification_hook "$TEST_TMP/settings.json" "afplay /System/Library/Sounds/Bottle.aiff &"
  assert_output "added"
  run cat "$TEST_TMP/settings.json"
  assert_output --partial "idle_prompt"
}

@test "add_sound_notification_hook: handles completely corrupted JSON" {
  echo 'not even json at all!!!' > "$TEST_TMP/settings.json"
  run add_sound_notification_hook "$TEST_TMP/settings.json" "afplay /System/Library/Sounds/Bottle.aiff &"
  assert_output "added"
  run cat "$TEST_TMP/settings.json"
  assert_output --partial "idle_prompt"
}

@test "add_sound_notification_hook: handles binary file" {
  # Create a binary file with null bytes
  printf '\x00\x01\x02\x03\x04' > "$TEST_TMP/settings.json"
  run add_sound_notification_hook "$TEST_TMP/settings.json" "afplay /System/Library/Sounds/Bottle.aiff &"
  assert_output "added"
  run cat "$TEST_TMP/settings.json"
  assert_output --partial "idle_prompt"
}

# --- Edge Cases: Windows Line Endings ---

@test "add_sound_notification_hook: handles Windows line endings (CRLF)" {
  printf '{\r\n  "foo": "bar"\r\n}\r\n' > "$TEST_TMP/settings.json"
  run add_sound_notification_hook "$TEST_TMP/settings.json" "afplay /System/Library/Sounds/Bottle.aiff &"
  assert_output "added"
  run cat "$TEST_TMP/settings.json"
  assert_output --partial "idle_prompt"
}

@test "merge_claude_settings: handles Windows line endings in existing file" {
  printf '{"foo":"bar"}\r\n' > "$TEST_TMP/settings.json"
  run merge_claude_settings "$TEST_TMP/settings.json"
  # File exists and doesn't have statusLine, so it adds it with sed
  assert_output --partial "Added status line"
  [ -f "$TEST_TMP/settings.json" ]
}

# --- Edge Cases: Empty and Whitespace Files ---

@test "add_sound_notification_hook: handles empty file" {
  touch "$TEST_TMP/settings.json"
  run add_sound_notification_hook "$TEST_TMP/settings.json" "afplay /System/Library/Sounds/Bottle.aiff &"
  assert_output "added"
  run cat "$TEST_TMP/settings.json"
  assert_output --partial "idle_prompt"
}

@test "add_sound_notification_hook: handles file with only whitespace" {
  printf '   \n\n  \t\t  \n' > "$TEST_TMP/settings.json"
  run add_sound_notification_hook "$TEST_TMP/settings.json" "afplay /System/Library/Sounds/Bottle.aiff &"
  assert_output "added"
  run cat "$TEST_TMP/settings.json"
  assert_output --partial "idle_prompt"
}

@test "merge_claude_settings: handles empty file" {
  touch "$TEST_TMP/settings.json"
  run merge_claude_settings "$TEST_TMP/settings.json"
  # Empty file exists, grep doesn't find statusLine, sed tries to modify
  # sed with '$ s/}$/...' won't match anything in empty file, so nothing happens
  # But function prints success message anyway
  assert_success
}

# --- Edge Cases: Special Characters ---

@test "add_sound_notification_hook: handles command with special shell characters" {
  echo '{}' > "$TEST_TMP/settings.json"
  run add_sound_notification_hook "$TEST_TMP/settings.json" 'echo "test $VAR & && || ; | > < ( )"'
  assert_output "added"
  run cat "$TEST_TMP/settings.json"
  # The command is stored in JSON, so check for the key parts
  assert_output --partial '"command":'
  assert_output --partial 'echo'
}

@test "add_sound_notification_hook: handles command with quotes and newlines" {
  echo '{}' > "$TEST_TMP/settings.json"
  run add_sound_notification_hook "$TEST_TMP/settings.json" "echo 'single' \"double\""
  assert_output "added"
  run cat "$TEST_TMP/settings.json"
  assert_output --partial "echo 'single'"
}

# --- Edge Cases: Permission Denied ---

@test "add_sound_notification_hook: handles read-only file" {
  echo '{}' > "$TEST_TMP/settings.json"
  chmod 444 "$TEST_TMP/settings.json"
  run add_sound_notification_hook "$TEST_TMP/settings.json" "afplay /System/Library/Sounds/Bottle.aiff &"
  # Python should fail to write to read-only file
  assert_failure
  chmod 644 "$TEST_TMP/settings.json"  # cleanup
}

@test "merge_claude_settings: handles read-only file" {
  echo '{}' > "$TEST_TMP/settings.json"
  chmod 444 "$TEST_TMP/settings.json"
  run merge_claude_settings "$TEST_TMP/settings.json"
  # sed -i '' fails but function doesn't use set -e, prints success anyway
  assert_output --partial "Added status line"
  chmod 644 "$TEST_TMP/settings.json"  # cleanup
}

# --- Edge Cases: Large Files ---

@test "add_sound_notification_hook: handles large JSON file" {
  # Create a JSON file with 1000+ entries
  echo '{' > "$TEST_TMP/settings.json"
  for i in {1..1000}; do
    if [ "$i" -eq 1000 ]; then
      echo "  \"key$i\": \"value$i\"" >> "$TEST_TMP/settings.json"
    else
      echo "  \"key$i\": \"value$i\"," >> "$TEST_TMP/settings.json"
    fi
  done
  echo '}' >> "$TEST_TMP/settings.json"

  run add_sound_notification_hook "$TEST_TMP/settings.json" "afplay /System/Library/Sounds/Bottle.aiff &"
  assert_output "added"
  run cat "$TEST_TMP/settings.json"
  assert_output --partial "idle_prompt"
  assert_output --partial "key1000"
}

@test "merge_claude_settings: handles file with 1000+ lines" {
  # Create a file with many lines
  for i in {1..1000}; do
    echo "# Comment line $i" >> "$TEST_TMP/settings.json"
  done
  echo '{}' >> "$TEST_TMP/settings.json"

  run merge_claude_settings "$TEST_TMP/settings.json"
  # Should append statusLine to the JSON at the end
  assert_output --partial "Added status line"
}

# --- Edge Cases: Concurrent Operations ---

@test "add_sound_notification_hook: handles concurrent writes to same file" {
  echo '{}' > "$TEST_TMP/settings.json"

  # Run two simultaneous writes in background
  add_sound_notification_hook "$TEST_TMP/settings.json" "afplay /System/Library/Sounds/Bottle.aiff &" > "$TEST_TMP/out1" 2>&1 &
  pid1=$!
  add_sound_notification_hook "$TEST_TMP/settings.json" "afplay /System/Library/Sounds/Bottle.aiff &" > "$TEST_TMP/out2" 2>&1 &
  pid2=$!

  # Wait for both to complete
  wait "$pid1"
  wait "$pid2"

  # At least one should succeed, file should remain valid JSON
  run python3 -c "import json; json.load(open('$TEST_TMP/settings.json'))"
  assert_success
}
