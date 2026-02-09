setup() {
  load 'test_helper/common'
  _common_setup
  source "$PROJECT_ROOT/lib/project-actions.sh"
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
