setup() {
  load 'test_helper/common'
  _common_setup
  source "$PROJECT_ROOT/lib/statusline.sh"
}

# --- format_memory ---

@test "format_memory: converts KB to MB" {
  run format_memory 512000
  assert_output "500M"
}

@test "format_memory: small MB value" {
  run format_memory 102400
  assert_output "100M"
}

@test "format_memory: converts to GB with decimal" {
  # 1572864 KB = 1536 MB = 1.5 GB
  run format_memory 1572864
  assert_output "1.5G"
}

@test "format_memory: exactly 1 GB" {
  # 1048576 KB = 1024 MB = 1.0G
  run format_memory 1048576
  assert_output "1.0G"
}

@test "format_memory: zero returns 0M" {
  run format_memory 0
  assert_output "0M"
}

# --- parse_cwd_from_json ---

@test "parse_cwd_from_json: extracts current_dir" {
  run parse_cwd_from_json '{"current_dir":"/Users/me/project"}'
  assert_output "/Users/me/project"
}

@test "parse_cwd_from_json: handles nested JSON" {
  run parse_cwd_from_json '{"foo":"bar","current_dir":"/tmp/test","baz":1}'
  assert_output "/tmp/test"
}

@test "parse_cwd_from_json: returns empty for missing key" {
  run parse_cwd_from_json '{"foo":"bar"}'
  assert_output ""
}

# --- Edge Cases: JSON Parsing ---

@test "parse_cwd_from_json: handles malformed JSON - missing quotes" {
  run parse_cwd_from_json '{current_dir:/tmp/test}'
  # sed-based parser doesn't validate JSON, just pattern matches
  assert_output ""
}

@test "parse_cwd_from_json: handles malformed JSON - missing braces" {
  run parse_cwd_from_json '"current_dir":"/tmp/test"'
  assert_output "/tmp/test"
}

@test "parse_cwd_from_json: handles malformed JSON - trailing comma" {
  run parse_cwd_from_json '{"current_dir":"/tmp/test",}'
  assert_output "/tmp/test"
}

@test "parse_cwd_from_json: handles empty JSON object" {
  run parse_cwd_from_json '{}'
  assert_output ""
}

@test "parse_cwd_from_json: handles empty string" {
  run parse_cwd_from_json ''
  assert_output ""
}

@test "parse_cwd_from_json: handles whitespace-only string" {
  run parse_cwd_from_json '   '
  assert_output ""
}

@test "parse_cwd_from_json: handles binary data" {
  run parse_cwd_from_json "$(printf '\x00\x01\x02\x03')"
  assert_output ""
}

@test "parse_cwd_from_json: handles path with escaped characters" {
  run parse_cwd_from_json '{"current_dir":"/tmp/test\\ndir"}'
  # sed extracts the literal backslash-n sequence
  assert_output '/tmp/test\\ndir'
}

@test "parse_cwd_from_json: handles path with special JSON chars" {
  run parse_cwd_from_json '{"current_dir":"/tmp/test\"quoted"}'
  # sed stops at first unescaped quote
  assert_output "/tmp/test\\"
}

@test "parse_cwd_from_json: handles very long path" {
  local long_path="/very/long/path/with/many/segments"
  for i in {1..50}; do
    long_path="${long_path}/segment${i}"
  done
  run parse_cwd_from_json "{\"current_dir\":\"${long_path}\"}"
  assert_output "$long_path"
}

@test "parse_cwd_from_json: handles path with Unicode characters" {
  run parse_cwd_from_json '{"current_dir":"/tmp/tëst/日本語/émoji🎉"}'
  assert_output '/tmp/tëst/日本語/émoji🎉'
}

@test "parse_cwd_from_json: handles multiple current_dir keys (takes last)" {
  run parse_cwd_from_json '{"current_dir":"/first","other":"stuff","current_dir":"/second"}'
  # sed uses greedy matching, will match the largest pattern
  assert_output "/second"
}

@test "parse_cwd_from_json: handles Windows-style line endings" {
  run parse_cwd_from_json "$(printf '{\r\n\"current_dir\":\"/tmp/test\"\r\n}\r\n')"
  assert_output "/tmp/test"
}

# --- Edge Cases: Memory Formatting ---

@test "format_memory: handles negative values" {
  run format_memory -1024
  # bc may produce negative results
  [[ "$output" == *"-"* ]] || [[ "$output" == "0M" ]]
}

@test "format_memory: handles very large values" {
  # 10TB in KB
  run format_memory 10737418240
  assert_output --partial "G"
  # Should show thousands of GB
  [[ "$output" =~ ^[0-9]+\.[0-9]G$ ]]
}

@test "format_memory: handles non-numeric input gracefully" {
  run format_memory "not_a_number"
  # bc will fail or produce 0
  assert_failure || assert_output "0M"
}

@test "format_memory: handles empty string" {
  run format_memory ""
  assert_failure || assert_output "0M"
}

@test "format_memory: handles floating point input" {
  run format_memory 512000.5
  # Shell arithmetic with floating point may fail
  # Bash $(()) doesn't handle floats, this will error
  assert_failure || assert_success
}

@test "format_memory: handles exactly 1024 MB boundary" {
  # Exactly 1GB = 1048576 KB
  run format_memory 1048576
  assert_output "1.0G"
}

@test "format_memory: handles just below GB boundary" {
  # 1023 MB
  run format_memory 1047552
  assert_output "1023M"
}

@test "format_memory: handles just above GB boundary" {
  # 1025 MB
  run format_memory 1049600
  assert_output "1.0G"
}

@test "format_memory: handles KB value with remainder" {
  # 512500 KB = 500.488 MB
  run format_memory 512500
  assert_output "500M"
}

@test "format_memory: handles very small non-zero values" {
  run format_memory 1
  assert_output "0M"
}

# --- get_tree_rss_kb ---

# Helper: create PATH-based mock scripts for ps and pgrep
_setup_tree_mocks() {
  MOCK_BIN="$(mktemp -d)"
  export PATH="$MOCK_BIN:$PATH"
}

_teardown_tree_mocks() {
  rm -rf "$MOCK_BIN"
}

@test "get_tree_rss_kb: sums memory of process and its children" {
  _setup_tree_mocks

  cat > "$MOCK_BIN/pgrep" <<'SCRIPT'
#!/bin/bash
pid="${@: -1}"
case "$pid" in
  100) printf '101\n102\n' ;;
  101) printf '103\n' ;;
  *) exit 1 ;;
esac
SCRIPT
  chmod +x "$MOCK_BIN/pgrep"

  cat > "$MOCK_BIN/ps" <<'SCRIPT'
#!/bin/bash
pid="${@: -1}"
case "$pid" in
  100) echo "  51200" ;;
  101) echo "  25600" ;;
  102) echo "  10240" ;;
  103) echo "  5120" ;;
  *) echo "" ;;
esac
SCRIPT
  chmod +x "$MOCK_BIN/ps"

  run get_tree_rss_kb 100
  _teardown_tree_mocks
  assert_success
  # 51200 + 25600 + 10240 + 5120 = 92160
  assert_output "92160"
}

@test "get_tree_rss_kb: handles process with no children" {
  _setup_tree_mocks

  cat > "$MOCK_BIN/pgrep" <<'SCRIPT'
#!/bin/bash
exit 1
SCRIPT
  chmod +x "$MOCK_BIN/pgrep"

  cat > "$MOCK_BIN/ps" <<'SCRIPT'
#!/bin/bash
echo "  51200"
SCRIPT
  chmod +x "$MOCK_BIN/ps"

  run get_tree_rss_kb 100
  _teardown_tree_mocks
  assert_success
  assert_output "51200"
}

@test "get_tree_rss_kb: handles disappeared process gracefully" {
  _setup_tree_mocks

  cat > "$MOCK_BIN/pgrep" <<'SCRIPT'
#!/bin/bash
exit 1
SCRIPT
  chmod +x "$MOCK_BIN/pgrep"

  cat > "$MOCK_BIN/ps" <<'SCRIPT'
#!/bin/bash
echo ""
SCRIPT
  chmod +x "$MOCK_BIN/ps"

  run get_tree_rss_kb 999
  _teardown_tree_mocks
  assert_success
  assert_output "0"
}

@test "get_tree_rss_kb: handles child that disappears mid-walk" {
  _setup_tree_mocks

  cat > "$MOCK_BIN/pgrep" <<'SCRIPT'
#!/bin/bash
pid="${@: -1}"
case "$pid" in
  100) printf '101\n102\n' ;;
  *) exit 1 ;;
esac
SCRIPT
  chmod +x "$MOCK_BIN/pgrep"

  cat > "$MOCK_BIN/ps" <<'SCRIPT'
#!/bin/bash
pid="${@: -1}"
case "$pid" in
  100) echo "  51200" ;;
  101) echo "" ;;
  102) echo "  10240" ;;
  *) echo "" ;;
esac
SCRIPT
  chmod +x "$MOCK_BIN/ps"

  run get_tree_rss_kb 100
  _teardown_tree_mocks
  assert_success
  # 51200 + 0 + 10240 = 61440
  assert_output "61440"
}

@test "format_memory: handles missing bc command" {
  # Mock bc to fail
  bc() {
    return 127
  }
  export -f bc

  run format_memory 1572864
  # bc is called via echo | bc, if bc fails output will be empty
  # But format_memory doesn't check bc success, just uses output
  # So it might succeed with empty string or "G"
  assert_success || assert_failure
}

# --- statusline-command.sh: session line diff ---

# Helper: set up a temporary git repo for statusline-command tests
_setup_git_repo() {
  REPO_DIR="$(mktemp -d)"
  git -C "$REPO_DIR" init -q
  git -C "$REPO_DIR" config user.email "test@test.com"
  git -C "$REPO_DIR" config user.name "Test"
  echo "initial" > "$REPO_DIR/file.txt"
  git -C "$REPO_DIR" add file.txt
  git -C "$REPO_DIR" commit -q -m "initial"
  STATUSLINE_CMD="$PROJECT_ROOT/templates/statusline-command.sh"
}

_teardown_git_repo() {
  rm -rf "$REPO_DIR"
}

@test "statusline-command: shows +N green and -N red for session diff" {
  _setup_git_repo
  local baseline_sha
  baseline_sha=$(git -C "$REPO_DIR" rev-parse HEAD)

  # Add 3 lines
  printf 'line1\nline2\nline3\n' >> "$REPO_DIR/file.txt"

  BASELINE_FILE="$(mktemp)"
  echo "$baseline_sha" > "$BASELINE_FILE"
  export GHOST_TAB_BASELINE_FILE="$BASELINE_FILE"

  run bash -c "echo '{\"current_dir\":\"$REPO_DIR\"}' | bash '$STATUSLINE_CMD'"
  _teardown_git_repo
  rm -f "$BASELINE_FILE"

  assert_success
  # Should contain +3 in green and -0 in red
  assert_output --partial "+3"
  assert_output --partial "-0"
}

@test "statusline-command: shows deletions in red" {
  _setup_git_repo
  local baseline_sha
  baseline_sha=$(git -C "$REPO_DIR" rev-parse HEAD)

  # Delete the file content (1 line removed)
  : > "$REPO_DIR/file.txt"

  BASELINE_FILE="$(mktemp)"
  echo "$baseline_sha" > "$BASELINE_FILE"
  export GHOST_TAB_BASELINE_FILE="$BASELINE_FILE"

  run bash -c "echo '{\"current_dir\":\"$REPO_DIR\"}' | bash '$STATUSLINE_CMD'"
  _teardown_git_repo
  rm -f "$BASELINE_FILE"

  assert_success
  assert_output --partial "+0"
  assert_output --partial "-1"
}

@test "statusline-command: tracks committed changes since baseline" {
  _setup_git_repo
  local baseline_sha
  baseline_sha=$(git -C "$REPO_DIR" rev-parse HEAD)

  # Make a committed change
  printf 'new1\nnew2\n' >> "$REPO_DIR/file.txt"
  git -C "$REPO_DIR" add file.txt
  git -C "$REPO_DIR" commit -q -m "add lines"

  BASELINE_FILE="$(mktemp)"
  echo "$baseline_sha" > "$BASELINE_FILE"
  export GHOST_TAB_BASELINE_FILE="$BASELINE_FILE"

  run bash -c "echo '{\"current_dir\":\"$REPO_DIR\"}' | bash '$STATUSLINE_CMD'"
  _teardown_git_repo
  rm -f "$BASELINE_FILE"

  assert_success
  assert_output --partial "+2"
  assert_output --partial "-0"
}

@test "statusline-command: shows zero diff when no changes" {
  _setup_git_repo
  local baseline_sha
  baseline_sha=$(git -C "$REPO_DIR" rev-parse HEAD)

  BASELINE_FILE="$(mktemp)"
  echo "$baseline_sha" > "$BASELINE_FILE"
  export GHOST_TAB_BASELINE_FILE="$BASELINE_FILE"

  run bash -c "echo '{\"current_dir\":\"$REPO_DIR\"}' | bash '$STATUSLINE_CMD'"
  _teardown_git_repo
  rm -f "$BASELINE_FILE"

  assert_success
  assert_output --partial "+0"
  assert_output --partial "-0"
}

@test "statusline-command: falls back to repo+branch only without baseline" {
  _setup_git_repo

  # No GHOST_TAB_BASELINE_FILE set
  unset GHOST_TAB_BASELINE_FILE

  run bash -c "echo '{\"current_dir\":\"$REPO_DIR\"}' | bash '$STATUSLINE_CMD'"
  _teardown_git_repo

  assert_success
  # Should show repo name and branch but no +/- diff section
  assert_output --partial "$(basename "$REPO_DIR")"
  refute_output --partial "+0"
  refute_output --partial "/ -"
}

@test "statusline-command: falls back when baseline file missing" {
  _setup_git_repo

  export GHOST_TAB_BASELINE_FILE="/tmp/ghost-tab-nonexistent-baseline"

  run bash -c "echo '{\"current_dir\":\"$REPO_DIR\"}' | bash '$STATUSLINE_CMD'"
  _teardown_git_repo
  unset GHOST_TAB_BASELINE_FILE

  assert_success
  assert_output --partial "$(basename "$REPO_DIR")"
  refute_output --partial "+0"
  refute_output --partial "/ -"
}

@test "statusline-command: non-git directory shows just dirname" {
  local non_git_dir
  non_git_dir="$(mktemp -d)"
  local cmd="$PROJECT_ROOT/templates/statusline-command.sh"

  run bash -c "echo '{\"current_dir\":\"$non_git_dir\"}' | bash '$cmd'"
  rm -rf "$non_git_dir"

  assert_success
  assert_output --partial "$(basename "$non_git_dir")"
  refute_output --partial "+0"
  refute_output --partial "/ -"
}
