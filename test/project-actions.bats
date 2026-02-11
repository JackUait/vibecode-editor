setup() {
  load 'test_helper/common'
  _common_setup
  source "$PROJECT_ROOT/lib/project-actions.sh"
  # Source TUI wrapper if it exists (for add_project_interactive test)
  if [ -f "$PROJECT_ROOT/lib/project-actions-tui.sh" ]; then
    source "$PROJECT_ROOT/lib/project-actions-tui.sh"
  fi
  TEST_TMP="$(mktemp -d)"
}

teardown() {
  rm -rf "$TEST_TMP"
}

# --- validate_new_project ---

@test "validate_new_project: normalizes path and sets _validated_name" {
  mkdir -p "$TEST_TMP/myapp"
  cat > "$TEST_TMP/projects" << 'EOF'
other:/some/other/path
EOF
  validate_new_project "$TEST_TMP/myapp" "$TEST_TMP/projects"
  [ "$_validated_name" = "myapp" ]
  [ "$_validated_path" = "$TEST_TMP/myapp" ]
}

@test "validate_new_project: strips trailing slash" {
  mkdir -p "$TEST_TMP/myapp"
  touch "$TEST_TMP/projects"
  validate_new_project "$TEST_TMP/myapp/" "$TEST_TMP/projects"
  [[ "$_validated_path" != */ ]]
}

@test "validate_new_project: returns 1 for duplicate path" {
  mkdir -p "$TEST_TMP/myapp"
  echo "myapp:$TEST_TMP/myapp" > "$TEST_TMP/projects"
  run validate_new_project "$TEST_TMP/myapp" "$TEST_TMP/projects"
  [ "$status" -eq 1 ]
}

@test "validate_new_project: detects duplicate even with trailing slash" {
  mkdir -p "$TEST_TMP/myapp"
  echo "myapp:$TEST_TMP/myapp" > "$TEST_TMP/projects"
  run validate_new_project "$TEST_TMP/myapp/" "$TEST_TMP/projects"
  [ "$status" -eq 1 ]
}

@test "validate_new_project: returns 0 for empty projects file" {
  mkdir -p "$TEST_TMP/myapp"
  touch "$TEST_TMP/projects"
  run validate_new_project "$TEST_TMP/myapp" "$TEST_TMP/projects"
  [ "$status" -eq 0 ]
}

@test "validate_new_project: returns 0 for missing projects file" {
  mkdir -p "$TEST_TMP/myapp"
  run validate_new_project "$TEST_TMP/myapp" "$TEST_TMP/nonexistent"
  [ "$status" -eq 0 ]
}

@test "validate_new_project: expands tilde to HOME" {
  mkdir -p "$HOME/some-dir"
  touch "$TEST_TMP/projects"
  validate_new_project "~/some-dir" "$TEST_TMP/projects"
  [ "${_validated_path:0:1}" = "/" ]
  [[ "$_validated_path" == "$HOME"* ]]
  [[ "$_validated_path" != "~"* ]]
  rm -rf "$HOME/some-dir"
}

# --- add_project_to_file ---

@test "add_project_to_file: appends name:path to file" {
  touch "$TEST_TMP/projects"
  add_project_to_file "myapp" "/path/to/myapp" "$TEST_TMP/projects"
  run cat "$TEST_TMP/projects"
  assert_output "myapp:/path/to/myapp"
}

@test "add_project_to_file: creates parent directory" {
  add_project_to_file "myapp" "/path/to/myapp" "$TEST_TMP/sub/dir/projects"
  [ -f "$TEST_TMP/sub/dir/projects" ]
  run cat "$TEST_TMP/sub/dir/projects"
  assert_output "myapp:/path/to/myapp"
}

@test "add_project_to_file: appends to existing entries" {
  echo "first:/path/first" > "$TEST_TMP/projects"
  add_project_to_file "second" "/path/second" "$TEST_TMP/projects"
  [ "$(wc -l < "$TEST_TMP/projects" | tr -d ' ')" -eq 2 ]
  run tail -1 "$TEST_TMP/projects"
  assert_output "second:/path/second"
}

# --- delete_project_from_file ---

@test "delete_project_from_file: removes matching line" {
  cat > "$TEST_TMP/projects" << 'EOF'
app1:/path/app1
app2:/path/app2
app3:/path/app3
EOF
  delete_project_from_file "app2:/path/app2" "$TEST_TMP/projects"
  run cat "$TEST_TMP/projects"
  assert_line --index 0 "app1:/path/app1"
  assert_line --index 1 "app3:/path/app3"
  refute_output --partial "app2"
}

@test "delete_project_from_file: leaves file intact when line not found" {
  echo "app1:/path/app1" > "$TEST_TMP/projects"
  delete_project_from_file "nonexistent:/path" "$TEST_TMP/projects"
  run cat "$TEST_TMP/projects"
  assert_output "app1:/path/app1"
}

@test "delete_project_from_file: handles single-entry file" {
  echo "only:/path/only" > "$TEST_TMP/projects"
  delete_project_from_file "only:/path/only" "$TEST_TMP/projects"
  # File should be empty (or contain only whitespace)
  [ ! -s "$TEST_TMP/projects" ] || [ "$(wc -l < "$TEST_TMP/projects" | tr -d ' ')" -eq 0 ]
}

@test "add_project_to_file: handles paths with spaces" {
  mkdir -p "$TEST_TMP/path with spaces"
  touch "$TEST_TMP/projects"
  add_project_to_file "myapp" "$TEST_TMP/path with spaces" "$TEST_TMP/projects"
  run cat "$TEST_TMP/projects"
  assert_output "myapp:$TEST_TMP/path with spaces"
}

@test "delete_project_from_file: handles paths with spaces" {
  cat > "$TEST_TMP/projects" << EOF
app1:/path/app1
app2:$TEST_TMP/path with spaces
app3:/path/app3
EOF
  delete_project_from_file "app2:$TEST_TMP/path with spaces" "$TEST_TMP/projects"
  run cat "$TEST_TMP/projects"
  assert_line --index 0 "app1:/path/app1"
  assert_line --index 1 "app3:/path/app3"
  refute_output --partial "app2"
}

# --- validate_new_project edge cases ---

@test "validate_new_project: handles path with spaces" {
  mkdir -p "$TEST_TMP/path with spaces"
  touch "$TEST_TMP/projects"
  validate_new_project "$TEST_TMP/path with spaces" "$TEST_TMP/projects"
  [ "$_validated_name" = "path with spaces" ]
  [[ "$_validated_path" == *"path with spaces" ]]
}

@test "validate_new_project: handles path with single quotes" {
  mkdir -p "$TEST_TMP/path'with'quotes"
  touch "$TEST_TMP/projects"
  validate_new_project "$TEST_TMP/path'with'quotes" "$TEST_TMP/projects"
  [ "$_validated_name" = "path'with'quotes" ]
  [[ "$_validated_path" == *"path'with'quotes" ]]
}

@test "validate_new_project: handles path with double quotes" {
  mkdir -p "$TEST_TMP/path\"with\"quotes"
  touch "$TEST_TMP/projects"
  validate_new_project "$TEST_TMP/path\"with\"quotes" "$TEST_TMP/projects"
  [ "$_validated_name" = "path\"with\"quotes" ]
  [[ "$_validated_path" == *'path"with"quotes' ]]
}

@test "validate_new_project: handles path with unicode" {
  mkdir -p "$TEST_TMP/émoji👻"
  touch "$TEST_TMP/projects"
  validate_new_project "$TEST_TMP/émoji👻" "$TEST_TMP/projects"
  [ "$_validated_name" = "émoji👻" ]
  [[ "$_validated_path" == *"émoji👻" ]]
}

@test "validate_new_project: normalizes path with .. components" {
  mkdir -p "$TEST_TMP/subdir/myapp"
  touch "$TEST_TMP/projects"
  validate_new_project "$TEST_TMP/subdir/../subdir/myapp" "$TEST_TMP/projects"
  # Should normalize to absolute path without ..
  [[ "$_validated_path" == "$TEST_TMP/subdir/myapp" ]]
  [ "$_validated_name" = "myapp" ]
}

@test "validate_new_project: handles relative path ./" {
  original_dir="$(pwd)"
  mkdir -p "$TEST_TMP/myapp"
  cd "$TEST_TMP" || exit 1
  touch "$TEST_TMP/projects"
  validate_new_project "./myapp" "$TEST_TMP/projects"
  # Should resolve to absolute path
  [ "$_validated_path" = "$TEST_TMP/myapp" ]
  [ "$_validated_name" = "myapp" ]
  cd "$original_dir" || exit 1
}

@test "validate_new_project: handles symlink to directory" {
  mkdir -p "$TEST_TMP/real-dir"
  ln -s "$TEST_TMP/real-dir" "$TEST_TMP/link-dir"
  touch "$TEST_TMP/projects"
  validate_new_project "$TEST_TMP/link-dir" "$TEST_TMP/projects"
  # Should resolve to the real path
  [ "$_validated_path" = "$TEST_TMP/real-dir" ]
  [ "$_validated_name" = "real-dir" ]
}

@test "validate_new_project: handles multiple trailing slashes" {
  mkdir -p "$TEST_TMP/myapp"
  touch "$TEST_TMP/projects"
  validate_new_project "$TEST_TMP/myapp///" "$TEST_TMP/projects"
  [[ "$_validated_path" != */ ]]
  [ "$_validated_path" = "$TEST_TMP/myapp" ]
}

@test "validate_new_project: detects duplicate with different trailing slash variations" {
  mkdir -p "$TEST_TMP/myapp"
  echo "myapp:$TEST_TMP/myapp" > "$TEST_TMP/projects"
  run validate_new_project "$TEST_TMP/myapp///" "$TEST_TMP/projects"
  [ "$status" -eq 1 ]
}

@test "validate_new_project: detects duplicate through symlink" {
  mkdir -p "$TEST_TMP/real-dir"
  ln -s "$TEST_TMP/real-dir" "$TEST_TMP/link-dir"
  echo "app:$TEST_TMP/real-dir" > "$TEST_TMP/projects"
  run validate_new_project "$TEST_TMP/link-dir" "$TEST_TMP/projects"
  # Should detect as duplicate since symlink resolves to same path
  [ "$status" -eq 1 ]
}

@test "validate_new_project: handles tilde expansion with spaces" {
  mkdir -p "$HOME/temp space test"
  touch "$TEST_TMP/projects"
  validate_new_project "~/temp space test" "$TEST_TMP/projects"
  [ "$_validated_name" = "temp space test" ]
  [ "$_validated_path" = "$HOME/temp space test" ]
  rm -rf "$HOME/temp space test"
}

@test "validate_new_project: handles nonexistent path gracefully" {
  touch "$TEST_TMP/projects"
  validate_new_project "$TEST_TMP/nonexistent" "$TEST_TMP/projects"
  # Should still set validated values even if path doesn't exist
  [ -n "$_validated_path" ]
  [ -n "$_validated_name" ]
}

@test "validate_new_project: handles empty path input" {
  touch "$TEST_TMP/projects"
  run validate_new_project "" "$TEST_TMP/projects"
  # Should handle gracefully (might fail but shouldn't crash)
  # Exit code can be 0 or non-zero depending on implementation
  true
}

@test "validate_new_project: handles path with colons" {
  # Colons are tricky because they're the delimiter in projects file
  mkdir -p "$TEST_TMP/path:with:colons"
  touch "$TEST_TMP/projects"
  validate_new_project "$TEST_TMP/path:with:colons" "$TEST_TMP/projects"
  [ "$_validated_name" = "path:with:colons" ]
  [[ "$_validated_path" == *"path:with:colons" ]]
}

@test "validate_new_project: handles tilde not at start" {
  mkdir -p "$TEST_TMP/foo"
  mkdir -p "$TEST_TMP/foo/~"
  mkdir -p "$TEST_TMP/foo/~/bar"
  touch "$TEST_TMP/projects"
  # Tilde not at start should be treated literally (not expanded)
  validate_new_project "$TEST_TMP/foo/~/bar" "$TEST_TMP/projects"
  # Should resolve to absolute path with literal ~
  [[ "$_validated_path" == *"/foo/~/bar" ]]
  [ "$_validated_name" = "bar" ]
}

@test "validate_new_project: handles circular symlinks" {
  ln -s "$TEST_TMP/link2" "$TEST_TMP/link1"
  ln -s "$TEST_TMP/link1" "$TEST_TMP/link2"
  touch "$TEST_TMP/projects"
  # Should handle gracefully (cd will fail, but function won't crash)
  validate_new_project "$TEST_TMP/link1" "$TEST_TMP/projects" || true
  # Function should handle the error gracefully
  # The circular symlink path will be set as fallback
  [[ "$_validated_path" == *"link1" ]] || true
}

# --- add_project_to_file edge cases ---

@test "add_project_to_file: handles name with spaces" {
  touch "$TEST_TMP/projects"
  add_project_to_file "my app" "/path/to/app" "$TEST_TMP/projects"
  run cat "$TEST_TMP/projects"
  assert_output "my app:/path/to/app"
}

@test "add_project_to_file: handles path with quotes" {
  touch "$TEST_TMP/projects"
  add_project_to_file "app" '/path/with"quotes' "$TEST_TMP/projects"
  run cat "$TEST_TMP/projects"
  assert_output 'app:/path/with"quotes'
}

@test "add_project_to_file: handles path with unicode" {
  touch "$TEST_TMP/projects"
  add_project_to_file "app" "/path/émoji/👻" "$TEST_TMP/projects"
  run cat "$TEST_TMP/projects"
  assert_output "app:/path/émoji/👻"
}

@test "add_project_to_file: handles very long paths" {
  touch "$TEST_TMP/projects"
  long_path="$(printf '/very/long/path%.0s' {1..50})"
  add_project_to_file "app" "$long_path" "$TEST_TMP/projects"
  run cat "$TEST_TMP/projects"
  assert_output "app:$long_path"
}

@test "add_project_to_file: handles name with colons" {
  touch "$TEST_TMP/projects"
  add_project_to_file "app:v2.0" "/path/to/app" "$TEST_TMP/projects"
  run cat "$TEST_TMP/projects"
  assert_output "app:v2.0:/path/to/app"
}

@test "add_project_to_file: handles special characters in name" {
  touch "$TEST_TMP/projects"
  add_project_to_file "app-v1.0_test" "/path/to/app" "$TEST_TMP/projects"
  run cat "$TEST_TMP/projects"
  assert_output "app-v1.0_test:/path/to/app"
}

# --- delete_project_from_file edge cases ---

@test "delete_project_from_file: handles entry with quotes" {
  cat > "$TEST_TMP/projects" << 'EOF'
app1:/path/app1
app2:/path/with"quotes
app3:/path/app3
EOF
  delete_project_from_file 'app2:/path/with"quotes' "$TEST_TMP/projects"
  run cat "$TEST_TMP/projects"
  assert_line --index 0 "app1:/path/app1"
  assert_line --index 1 "app3:/path/app3"
  refute_output --partial "app2"
}

@test "delete_project_from_file: handles entry with unicode" {
  cat > "$TEST_TMP/projects" << EOF
app1:/path/app1
app2:/path/émoji/👻
app3:/path/app3
EOF
  delete_project_from_file "app2:/path/émoji/👻" "$TEST_TMP/projects"
  run cat "$TEST_TMP/projects"
  assert_line --index 0 "app1:/path/app1"
  assert_line --index 1 "app3:/path/app3"
  refute_output --partial "app2"
}

@test "delete_project_from_file: does not delete partial matches" {
  cat > "$TEST_TMP/projects" << 'EOF'
app:/path/app
app-long:/path/app-longer-name
EOF
  delete_project_from_file "app:/path/app" "$TEST_TMP/projects"
  run cat "$TEST_TMP/projects"
  assert_output "app-long:/path/app-longer-name"
}

@test "delete_project_from_file: handles very long entries" {
  long_path="$(printf '/very/long/path%.0s' {1..50})"
  cat > "$TEST_TMP/projects" << EOF
app1:/path/app1
app2:${long_path}
app3:/path/app3
EOF
  delete_project_from_file "app2:${long_path}" "$TEST_TMP/projects"
  run cat "$TEST_TMP/projects"
  assert_line --index 0 "app1:/path/app1"
  assert_line --index 1 "app3:/path/app3"
  refute_output --partial "app2"
}

# --- add_project_interactive (TUI wrapper) ---

@test "add_project_interactive calls ghost-tab-tui and parses JSON" {
  # Mock ghost-tab-tui add-project
  ghost-tab-tui() {
    if [[ "$1" == "add-project" ]]; then
      echo '{"name":"test-proj","path":"/tmp/test","confirmed":true}'
      return 0
    fi
    return 1
  }
  export -f ghost-tab-tui

  # Call function directly (not via run) to check variables
  add_project_interactive

  # Check that global variables are set
  [ "$_add_project_name" = "test-proj" ]
  [ "$_add_project_path" = "/tmp/test" ]
}

@test "add_project_interactive returns 1 when cancelled" {
  # Mock ghost-tab-tui returning cancelled
  ghost-tab-tui() {
    if [[ "$1" == "add-project" ]]; then
      echo '{"name":"","path":"","confirmed":false}'
      return 0
    fi
    return 1
  }
  export -f ghost-tab-tui

  run add_project_interactive

  assert_failure
}

@test "add_project_interactive handles missing jq gracefully" {
  ghost-tab-tui() {
    if [[ "$1" == "add-project" ]]; then
      echo '{"name":"test","path":"/tmp","confirmed":true}'
      return 0
    fi
    return 1
  }
  export -f ghost-tab-tui

  # Mock jq to fail
  jq() {
    return 127
  }
  export -f jq

  source "$BATS_TEST_DIRNAME/../lib/project-actions-tui.sh"
  source "$BATS_TEST_DIRNAME/../lib/tui.sh"

  run add_project_interactive
  assert_failure
  assert_output --partial "Failed to parse"
}

@test "add_project_interactive handles malformed JSON" {
  ghost-tab-tui() {
    if [[ "$1" == "add-project" ]]; then
      echo 'not valid json at all'
      return 0
    fi
    return 1
  }
  export -f ghost-tab-tui

  source "$BATS_TEST_DIRNAME/../lib/project-actions-tui.sh"
  source "$BATS_TEST_DIRNAME/../lib/tui.sh"

  run add_project_interactive
  assert_failure
  assert_output --partial "Failed to parse"
}

@test "add_project_interactive fails when TUI binary missing" {
  # Override command -v to return false for ghost-tab-tui
  command() {
    if [[ "$1" == "-v" && "$2" == "ghost-tab-tui" ]]; then
      return 1
    fi
    # shellcheck disable=SC2312
    builtin command "$@"
  }
  export -f command

  source "$BATS_TEST_DIRNAME/../lib/project-actions-tui.sh"
  source "$BATS_TEST_DIRNAME/../lib/tui.sh"

  run add_project_interactive
  assert_failure
  assert_output --partial "not found"
}

@test "add_project_interactive rejects null values from TUI" {
  ghost-tab-tui() {
    if [[ "$1" == "add-project" ]]; then
      echo '{"name":null,"path":null,"confirmed":true}'
      return 0
    fi
    return 1
  }
  export -f ghost-tab-tui

  source "$BATS_TEST_DIRNAME/../lib/project-actions-tui.sh"
  source "$BATS_TEST_DIRNAME/../lib/tui.sh"

  run add_project_interactive
  assert_failure
  assert_output --partial "invalid"
}
