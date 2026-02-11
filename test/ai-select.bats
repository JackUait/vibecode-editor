setup() {
  load 'test_helper/common'
  _common_setup
  source "$PROJECT_ROOT/lib/tui.sh"
  source "$PROJECT_ROOT/lib/ai-select.sh"
  TEST_TMP="$(mktemp -d)"
}

teardown() {
  rm -rf "$TEST_TMP"
}

# --- save_ai_tool_preference ---

@test "save_ai_tool_preference: claude first when selected" {
  save_ai_tool_preference 1 1 0 0 "$TEST_TMP"
  run cat "$TEST_TMP/ai-tool"
  assert_output "claude"
}

@test "save_ai_tool_preference: codex when claude not selected" {
  save_ai_tool_preference 0 1 0 0 "$TEST_TMP"
  run cat "$TEST_TMP/ai-tool"
  assert_output "codex"
}

@test "save_ai_tool_preference: copilot when claude and codex not selected" {
  save_ai_tool_preference 0 0 1 0 "$TEST_TMP"
  run cat "$TEST_TMP/ai-tool"
  assert_output "copilot"
}

@test "save_ai_tool_preference: opencode as last fallback" {
  save_ai_tool_preference 0 0 0 1 "$TEST_TMP"
  run cat "$TEST_TMP/ai-tool"
  assert_output "opencode"
}

@test "save_ai_tool_preference: creates parent directory" {
  save_ai_tool_preference 1 0 0 0 "$TEST_TMP/nested/dir"
  [ -f "$TEST_TMP/nested/dir/ai-tool" ]
  run cat "$TEST_TMP/nested/dir/ai-tool"
  assert_output "claude"
}

@test "save_ai_tool_preference: does nothing when none selected" {
  save_ai_tool_preference 0 0 0 0 "$TEST_TMP"
  [ ! -f "$TEST_TMP/ai-tool" ]
}

# --- run_ai_tool_select ---

@test "run_ai_tool_select: function is defined" {
  declare -f run_ai_tool_select >/dev/null
}

# Helper: run run_ai_tool_select in a child bash process where </dev/tty is
# replaced with a file descriptor fed from a prepared byte sequence.
# Usage: _run_ai_select_with_keys <key_bytes_file> <cc> <codex> <copilot> <oc>
# Prints _sel_claude _sel_codex _sel_copilot _sel_opencode on stdout.
_run_ai_select_with_keys() {
  local keyfile="$1" cc="$2" codex="$3" copilot="$4" oc="$5"
  bash -c '
    source "'"$PROJECT_ROOT"'/lib/tui.sh"
    source "'"$PROJECT_ROOT"'/lib/ai-select.sh"
    # Re-declare run_ai_tool_select with </dev/tty replaced by <&3
    new_body="$(declare -f run_ai_tool_select | sed "s|</dev/tty|<\&3|g")"
    eval "$new_body"
    exec 3< "'"$keyfile"'"
    run_ai_tool_select '"$cc"' '"$codex"' '"$copilot"' '"$oc"' >/dev/null 2>&1
    echo "$_sel_claude $_sel_codex $_sel_copilot $_sel_opencode"
  '
}

@test "run_ai_tool_select: defaults claude on, others match installed state" {
  # Input: just press Enter (empty byte = newline) to confirm defaults
  local keyfile="$TEST_TMP/keys"
  printf '\n' > "$keyfile"

  run _run_ai_select_with_keys "$keyfile" 0 1 0 1
  assert_success
  # _sel_claude=1 (always pre-checked), _sel_codex=1 (installed),
  # _sel_copilot=0 (not installed), _sel_opencode=1 (installed)
  assert_output "1 1 0 1"
}

@test "run_ai_tool_select: rejects empty selection then accepts after toggle" {
  # Sequence: Space (uncheck claude) -> Enter (rejected) -> Space (recheck) -> Enter (accepted)
  # Space = 0x20, Enter = 0x0a (newline)
  local keyfile="$TEST_TMP/keys"
  printf ' \n \n' > "$keyfile"

  run _run_ai_select_with_keys "$keyfile" 0 0 0 0
  assert_success
  # After re-checking claude, _sel_claude=1; all others stay 0
  assert_output "1 0 0 0"
}

# --- select_ai_tool_interactive ---

@test "select_ai_tool_interactive: calls ghost-tab-tui and parses JSON" {
  # Source the TUI wrapper module
  source "$PROJECT_ROOT/lib/ai-select-tui.sh"

  # Mock ghost-tab-tui
  ghost-tab-tui() {
    if [[ "$1" == "select-ai-tool" ]]; then
      echo '{"tool":"claude","command":"claude","selected":true}'
      return 0
    fi
    return 1
  }
  export -f ghost-tab-tui

  # Call function directly (not via run) to check variables
  select_ai_tool_interactive

  # Check that global variable is set
  [ "$_selected_ai_tool" = "claude" ]
}

@test "select_ai_tool_interactive: returns error when cancelled" {
  source "$PROJECT_ROOT/lib/ai-select-tui.sh"

  # Mock ghost-tab-tui returning cancelled
  ghost-tab-tui() {
    if [[ "$1" == "select-ai-tool" ]]; then
      echo '{"tool":"","command":"","selected":false}'
      return 0
    fi
    return 1
  }
  export -f ghost-tab-tui

  run select_ai_tool_interactive

  assert_failure
}

@test "select_ai_tool_interactive: handles jq parse error" {
  source "$PROJECT_ROOT/lib/ai-select-tui.sh"

  # Mock ghost-tab-tui returning invalid JSON
  ghost-tab-tui() {
    echo "not valid json"
    return 0
  }
  export -f ghost-tab-tui

  run select_ai_tool_interactive

  assert_failure
  assert_output --partial "Failed to parse"
}

@test "select_ai_tool_interactive: validates null values" {
  source "$PROJECT_ROOT/lib/ai-select-tui.sh"

  # Mock ghost-tab-tui returning null tool
  ghost-tab-tui() {
    echo '{"tool":null,"command":null,"selected":true}'
    return 0
  }
  export -f ghost-tab-tui

  run select_ai_tool_interactive

  assert_failure
  assert_output --partial "invalid tool name"
}

@test "select_ai_tool_interactive: selects codex" {
  source "$PROJECT_ROOT/lib/ai-select-tui.sh"

  ghost-tab-tui() {
    echo '{"tool":"codex","command":"codex","selected":true}'
    return 0
  }
  export -f ghost-tab-tui

  # Call function directly (not via run) to check variables
  select_ai_tool_interactive

  # Check that global variable is set
  [ "$_selected_ai_tool" = "codex" ]
}
