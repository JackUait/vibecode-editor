setup() {
  load 'test_helper/common'
  _common_setup
  source "$PROJECT_ROOT/lib/tui.sh"
  source "$PROJECT_ROOT/lib/ghostty-config.sh"

  TEST_TMP="$(mktemp -d)"
}

teardown() {
  rm -rf "$TEST_TMP"
}

# --- merge_ghostty_config ---

@test "merge_ghostty_config: replaces existing command line" {
  echo 'command = /old/path' > "$TEST_TMP/config"
  run merge_ghostty_config "$TEST_TMP/config" "command = /new/path"
  assert_output --partial "Replaced"
  run cat "$TEST_TMP/config"
  assert_output "command = /new/path"
}

@test "merge_ghostty_config: appends when no command line exists" {
  echo 'font-size = 14' > "$TEST_TMP/config"
  run merge_ghostty_config "$TEST_TMP/config" "command = /new/path"
  assert_output --partial "Appended"
  run cat "$TEST_TMP/config"
  assert_line "font-size = 14"
  assert_line "command = /new/path"
}

# --- backup_replace_ghostty_config ---

@test "backup_replace_ghostty_config: creates backup and replaces" {
  echo 'old content' > "$TEST_TMP/config"
  echo 'new content' > "$TEST_TMP/source"
  run backup_replace_ghostty_config "$TEST_TMP/config" "$TEST_TMP/source"
  assert_output --partial "Backed up"
  assert_output --partial "Replaced"

  # Verify the config was replaced
  run cat "$TEST_TMP/config"
  assert_output "new content"

  # Verify a backup file exists
  local backup_count
  backup_count=$(ls "$TEST_TMP"/config.backup.* 2>/dev/null | wc -l | tr -d ' ')
  [ "$backup_count" -eq 1 ]
}
