setup() {
  load 'test_helper/common'
  _common_setup
  source "$PROJECT_ROOT/lib/statusline.sh"
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
