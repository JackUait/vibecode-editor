setup() {
  load 'test_helper/common'
  _common_setup
  TEST_TMP="$(mktemp -d)"
}

teardown() {
  rm -rf "$TEST_TMP"
}

@test "select_ai_tool_interactive calls ghost-tab-tui multi-select-ai-tool" {
  source "$PROJECT_ROOT/lib/ai-select-tui.sh"

  # Mock ghost-tab-tui
  ghost-tab-tui() {
    if [[ "$1" == "multi-select-ai-tool" ]]; then
      echo '{"tools":["claude","codex"],"confirmed":true}'
      return 0
    fi
    return 1
  }
  export -f ghost-tab-tui

  # Mock jq
  echo "0" > "$TEST_TMP/jq_calls"
  jq() {
    local count
    count=$(<"$TEST_TMP/jq_calls")
    count=$((count + 1))
    echo "$count" > "$TEST_TMP/jq_calls"

    if [[ "$2" == ".confirmed" ]]; then
      echo "true"
    elif [[ "$2" == ".tools[]" ]]; then
      printf "claude\ncodex\n"
    fi
    return 0
  }
  export -f jq
  export TEST_TMP

  # Mock grep and head for priority selection
  # Don't use run - need to check variable in current shell
  select_ai_tool_interactive
  local result=$?

  [[ $result -eq 0 ]]
  [[ "$_selected_ai_tool" == "claude" ]]
}

@test "select_ai_tool_interactive sets _selected_ai_tools" {
  source "$PROJECT_ROOT/lib/ai-select-tui.sh"

  # Mock ghost-tab-tui
  ghost-tab-tui() {
    if [[ "$1" == "multi-select-ai-tool" ]]; then
      echo '{"tools":["codex","copilot"],"confirmed":true}'
      return 0
    fi
    return 1
  }
  export -f ghost-tab-tui

  # Mock jq
  echo "0" > "$TEST_TMP/jq_calls"
  jq() {
    local count
    count=$(<"$TEST_TMP/jq_calls")
    count=$((count + 1))
    echo "$count" > "$TEST_TMP/jq_calls"

    if [[ "$2" == ".confirmed" ]]; then
      echo "true"
    elif [[ "$2" == ".tools[]" ]]; then
      printf "codex\ncopilot\n"
    fi
    return 0
  }
  export -f jq
  export TEST_TMP

  select_ai_tool_interactive
  local result=$?

  [[ $result -eq 0 ]]
  # _selected_ai_tools contains both tools
  echo "$_selected_ai_tools" | grep -q "codex"
  echo "$_selected_ai_tools" | grep -q "copilot"
}

@test "select_ai_tool_interactive picks claude by priority" {
  source "$PROJECT_ROOT/lib/ai-select-tui.sh"

  # Mock ghost-tab-tui
  ghost-tab-tui() {
    if [[ "$1" == "multi-select-ai-tool" ]]; then
      echo '{"tools":["copilot","claude"],"confirmed":true}'
      return 0
    fi
    return 1
  }
  export -f ghost-tab-tui

  # Mock jq
  echo "0" > "$TEST_TMP/jq_calls"
  jq() {
    local count
    count=$(<"$TEST_TMP/jq_calls")
    count=$((count + 1))
    echo "$count" > "$TEST_TMP/jq_calls"

    if [[ "$2" == ".confirmed" ]]; then
      echo "true"
    elif [[ "$2" == ".tools[]" ]]; then
      printf "copilot\nclaude\n"
    fi
    return 0
  }
  export -f jq
  export TEST_TMP

  select_ai_tool_interactive
  local result=$?

  [[ $result -eq 0 ]]
  # Should pick claude despite copilot being first in list
  [[ "$_selected_ai_tool" == "claude" ]]
}

@test "select_ai_tool_interactive picks codex when claude not selected" {
  source "$PROJECT_ROOT/lib/ai-select-tui.sh"

  # Mock ghost-tab-tui
  ghost-tab-tui() {
    if [[ "$1" == "multi-select-ai-tool" ]]; then
      echo '{"tools":["copilot","codex"],"confirmed":true}'
      return 0
    fi
    return 1
  }
  export -f ghost-tab-tui

  # Mock jq
  echo "0" > "$TEST_TMP/jq_calls"
  jq() {
    local count
    count=$(<"$TEST_TMP/jq_calls")
    count=$((count + 1))
    echo "$count" > "$TEST_TMP/jq_calls"

    if [[ "$2" == ".confirmed" ]]; then
      echo "true"
    elif [[ "$2" == ".tools[]" ]]; then
      printf "copilot\ncodex\n"
    fi
    return 0
  }
  export -f jq
  export TEST_TMP

  select_ai_tool_interactive
  local result=$?

  [[ $result -eq 0 ]]
  [[ "$_selected_ai_tool" == "codex" ]]
}

@test "select_ai_tool_interactive returns failure when cancelled" {
  source "$PROJECT_ROOT/lib/ai-select-tui.sh"

  # Mock ghost-tab-tui (cancelled)
  ghost-tab-tui() {
    if [[ "$1" == "multi-select-ai-tool" ]]; then
      echo '{"confirmed":false}'
      return 0
    fi
    return 1
  }
  export -f ghost-tab-tui

  # Mock jq
  jq() {
    if [[ "$2" == ".confirmed" ]]; then
      echo "false"
    fi
    return 0
  }
  export -f jq

  run select_ai_tool_interactive

  assert_failure
}

@test "select_ai_tool_interactive handles binary missing" {
  source "$PROJECT_ROOT/lib/ai-select-tui.sh"

  # Mock error function
  error() {
    echo "ERROR: $*" >&2
  }
  export -f error

  # Override command to simulate missing binary
  command() {
    if [[ "$1" == "-v" && "$2" == "ghost-tab-tui" ]]; then
      return 1
    fi
    builtin command "$@"
  }
  export -f command

  run select_ai_tool_interactive

  assert_failure
  assert_output --partial "ghost-tab-tui binary not found"
}

@test "select_ai_tool_interactive handles jq parse failure for confirmed" {
  source "$PROJECT_ROOT/lib/ai-select-tui.sh"

  # Mock error function
  error() {
    echo "ERROR: $*" >&2
  }
  export -f error

  # Mock ghost-tab-tui
  ghost-tab-tui() {
    echo '{"tools":["claude"],"confirmed":true}'
    return 0
  }
  export -f ghost-tab-tui

  # Mock jq (fails on first call)
  jq() {
    return 1
  }
  export -f jq

  run select_ai_tool_interactive

  assert_failure
  assert_output --partial "Failed to parse AI tool selection response"
}

@test "select_ai_tool_interactive handles jq parse failure for tools" {
  source "$PROJECT_ROOT/lib/ai-select-tui.sh"

  # Mock error function
  error() {
    echo "ERROR: $*" >&2
  }
  export -f error

  # Mock ghost-tab-tui
  ghost-tab-tui() {
    echo '{"tools":["claude"],"confirmed":true}'
    return 0
  }
  export -f ghost-tab-tui

  # Mock jq (first call succeeds, second fails) - use file to track calls
  echo "0" > "$TEST_TMP/jq_calls"
  jq() {
    local count
    count=$(<"$TEST_TMP/jq_calls")
    count=$((count + 1))
    echo "$count" > "$TEST_TMP/jq_calls"

    if [[ $count -eq 1 ]]; then
      echo "true"
      return 0
    fi
    return 1
  }
  export -f jq
  export TEST_TMP

  run select_ai_tool_interactive

  assert_failure
  assert_output --partial "Failed to parse selected tools"
}

@test "select_ai_tool_interactive validates against null confirmed" {
  source "$PROJECT_ROOT/lib/ai-select-tui.sh"

  # Mock error function
  error() {
    echo "ERROR: $*" >&2
  }
  export -f error

  # Mock ghost-tab-tui
  ghost-tab-tui() {
    echo '{"confirmed":"null"}'
    return 0
  }
  export -f ghost-tab-tui

  # Mock jq
  jq() {
    if [[ "$2" == ".confirmed" ]]; then
      echo "null"
    fi
    return 0
  }
  export -f jq

  run select_ai_tool_interactive

  assert_failure
  assert_output --partial "TUI returned invalid confirmation status"
}

@test "select_ai_tool_interactive validates against empty tools" {
  source "$PROJECT_ROOT/lib/ai-select-tui.sh"

  # Mock error function
  error() {
    echo "ERROR: $*" >&2
  }
  export -f error

  # Mock ghost-tab-tui
  ghost-tab-tui() {
    echo '{"tools":[],"confirmed":true}'
    return 0
  }
  export -f ghost-tab-tui

  # Mock jq - use file to track calls
  echo "0" > "$TEST_TMP/jq_calls"
  jq() {
    local count
    count=$(<"$TEST_TMP/jq_calls")
    count=$((count + 1))
    echo "$count" > "$TEST_TMP/jq_calls"

    if [[ $count -eq 1 ]]; then
      echo "true"
    else
      echo ""
    fi
    return 0
  }
  export -f jq
  export TEST_TMP

  run select_ai_tool_interactive

  assert_failure
  assert_output --partial "TUI returned empty tool selection"
}
