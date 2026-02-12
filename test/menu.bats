setup() {
  load 'test_helper/common'
  _common_setup
  TEST_TMP="$(mktemp -d)"

  # Source required libs for every test
  source "$PROJECT_ROOT/lib/tui.sh" 2>/dev/null || true
  source "$PROJECT_ROOT/lib/menu-tui.sh"

  # Default mocks/vars used by most tests
  PROJECTS_FILE="$TEST_TMP/projects"
  echo "proj1:/tmp/p1" > "$PROJECTS_FILE"
  AI_TOOLS_AVAILABLE=("claude")
  SELECTED_AI_TOOL="claude"
  _update_version=""

  # Mock error function
  error() { echo "ERROR: $*" >&2; }
  export -f error
}

teardown() {
  rm -rf "$TEST_TMP"
}

@test "select_project_interactive calls main-menu and parses select-project JSON" {
  ghost-tab-tui() {
    if [[ "$1" == "main-menu" ]]; then
      echo '{"action":"select-project","name":"proj1","path":"/tmp/p1","ai_tool":"claude"}'
      return 0
    fi
    return 1
  }
  export -f ghost-tab-tui

  select_project_interactive "$PROJECTS_FILE"
  local result=$?

  [[ $result -eq 0 ]]
  [[ "$_selected_project_name" == "proj1" ]]
  [[ "$_selected_project_path" == "/tmp/p1" ]]
  [[ "$_selected_project_action" == "select-project" ]]
}

@test "select_project_interactive passes correct flags to main-menu" {
  AI_TOOLS_AVAILABLE=("claude" "codex")
  SELECTED_AI_TOOL="codex"
  _update_version="2.0.0"

  # Override XDG path
  XDG_CONFIG_HOME="$TEST_TMP"
  mkdir -p "$TEST_TMP/ghost-tab"
  echo "ghost_display=static" > "$TEST_TMP/ghost-tab/settings"

  ghost-tab-tui() {
    echo "$*" > "$TEST_TMP/captured_args"
    echo '{"action":"quit"}'
    return 0
  }
  export -f ghost-tab-tui
  export TEST_TMP

  # Run it (will return failure for quit, but we want to check args)
  select_project_interactive "$PROJECTS_FILE" || true

  # Verify main-menu was called with correct flags
  local captured_args
  captured_args=$(<"$TEST_TMP/captured_args")
  [[ "$captured_args" == *"main-menu"* ]]
  [[ "$captured_args" == *"--projects-file"* ]]
  [[ "$captured_args" == *"--ai-tool"* ]]
  [[ "$captured_args" == *"codex"* ]]
  [[ "$captured_args" == *"--ai-tools"* ]]
  [[ "$captured_args" == *"claude,codex"* ]]
  [[ "$captured_args" == *"--ghost-display"* ]]
  [[ "$captured_args" == *"static"* ]]
  [[ "$captured_args" == *"--update-version"* ]]
  [[ "$captured_args" == *"2.0.0"* ]]
}

@test "select_project_interactive handles AI tool change" {
  AI_TOOLS_AVAILABLE=("claude" "codex")
  SELECTED_AI_TOOL="claude"

  ghost-tab-tui() {
    echo '{"action":"select-project","name":"proj1","path":"/tmp/p1","ai_tool":"codex"}'
    return 0
  }
  export -f ghost-tab-tui

  select_project_interactive "$PROJECTS_FILE"

  [[ "$_selected_ai_tool" == "codex" ]]
}

@test "select_project_interactive handles quit action" {
  ghost-tab-tui() {
    echo '{"action":"quit"}'
    return 0
  }
  export -f ghost-tab-tui

  run select_project_interactive "$PROJECTS_FILE"

  assert_failure
}

@test "select_project_interactive handles open-once action with name and path" {
  ghost-tab-tui() {
    echo '{"action":"open-once","name":"temp","path":"/tmp/temp","ai_tool":"claude"}'
    return 0
  }
  export -f ghost-tab-tui

  select_project_interactive "$PROJECTS_FILE"
  local result=$?

  [[ $result -eq 0 ]]
  [[ "$_selected_project_action" == "open-once" ]]
  [[ "$_selected_project_name" == "temp" ]]
  [[ "$_selected_project_path" == "/tmp/temp" ]]
}

@test "select_project_interactive handles plain-terminal action" {
  ghost-tab-tui() {
    echo '{"action":"plain-terminal","ai_tool":"claude"}'
    return 0
  }
  export -f ghost-tab-tui

  select_project_interactive "$PROJECTS_FILE"
  local result=$?

  [[ $result -eq 0 ]]
  [[ "$_selected_project_action" == "plain-terminal" ]]
}

@test "select_project_interactive handles settings action" {
  ghost-tab-tui() {
    echo '{"action":"settings","ai_tool":"claude"}'
    return 0
  }
  export -f ghost-tab-tui

  select_project_interactive "$PROJECTS_FILE"
  local result=$?

  [[ $result -eq 0 ]]
  [[ "$_selected_project_action" == "settings" ]]
}

@test "select_project_interactive persists ghost_display change" {
  XDG_CONFIG_HOME="$TEST_TMP"
  mkdir -p "$TEST_TMP/ghost-tab"

  ghost-tab-tui() {
    echo '{"action":"select-project","name":"proj1","path":"/tmp/p1","ai_tool":"claude","ghost_display":"static"}'
    return 0
  }
  export -f ghost-tab-tui

  select_project_interactive "$PROJECTS_FILE"

  # Check settings file was written
  [[ -f "$TEST_TMP/ghost-tab/settings" ]]
  grep -q 'ghost_display=static' "$TEST_TMP/ghost-tab/settings"
}

@test "select_project_interactive updates existing ghost_display in settings" {
  XDG_CONFIG_HOME="$TEST_TMP"
  mkdir -p "$TEST_TMP/ghost-tab"
  echo "ghost_display=animated" > "$TEST_TMP/ghost-tab/settings"

  ghost-tab-tui() {
    echo '{"action":"select-project","name":"proj1","path":"/tmp/p1","ai_tool":"claude","ghost_display":"none"}'
    return 0
  }
  export -f ghost-tab-tui

  select_project_interactive "$PROJECTS_FILE"

  grep -q 'ghost_display=none' "$TEST_TMP/ghost-tab/settings"
  # Make sure old value is gone
  ! grep -q 'ghost_display=animated' "$TEST_TMP/ghost-tab/settings"
}

@test "select_project_interactive reads ghost_display from settings file" {
  XDG_CONFIG_HOME="$TEST_TMP"
  mkdir -p "$TEST_TMP/ghost-tab"
  echo "ghost_display=none" > "$TEST_TMP/ghost-tab/settings"

  ghost-tab-tui() {
    echo "$*" > "$TEST_TMP/captured_args"
    echo '{"action":"quit"}'
    return 0
  }
  export -f ghost-tab-tui
  export TEST_TMP

  select_project_interactive "$PROJECTS_FILE" || true

  local captured_args
  captured_args=$(<"$TEST_TMP/captured_args")
  [[ "$captured_args" == *"--ghost-display"* ]]
  [[ "$captured_args" == *"none"* ]]
}

@test "select_project_interactive defaults ghost_display to animated" {
  XDG_CONFIG_HOME="$TEST_TMP"
  # No settings file

  ghost-tab-tui() {
    echo "$*" > "$TEST_TMP/captured_args"
    echo '{"action":"quit"}'
    return 0
  }
  export -f ghost-tab-tui
  export TEST_TMP

  select_project_interactive "$PROJECTS_FILE" || true

  local captured_args
  captured_args=$(<"$TEST_TMP/captured_args")
  [[ "$captured_args" == *"--ghost-display"* ]]
  [[ "$captured_args" == *"animated"* ]]
}

@test "select_project_interactive validates null name on select-project" {
  ghost-tab-tui() {
    echo '{"action":"select-project","name":null,"path":"/tmp/p1","ai_tool":"claude"}'
    return 0
  }
  export -f ghost-tab-tui

  run select_project_interactive "$PROJECTS_FILE"

  assert_failure
  assert_output --partial "invalid project name"
}

@test "select_project_interactive validates null path on select-project" {
  ghost-tab-tui() {
    echo '{"action":"select-project","name":"proj1","path":null,"ai_tool":"claude"}'
    return 0
  }
  export -f ghost-tab-tui

  run select_project_interactive "$PROJECTS_FILE"

  assert_failure
  assert_output --partial "invalid project path"
}

@test "select_project_interactive handles jq parse failure" {
  ghost-tab-tui() {
    echo 'not json at all'
    return 0
  }
  export -f ghost-tab-tui

  # Mock jq to fail
  jq() {
    return 1
  }
  export -f jq

  run select_project_interactive "$PROJECTS_FILE"

  assert_failure
  assert_output --partial "Failed to parse"
}

@test "select_project_interactive handles binary missing" {
  # Override command to simulate missing binary
  command() {
    if [[ "$1" == "-v" && "$2" == "ghost-tab-tui" ]]; then
      return 1
    fi
    builtin command "$@"
  }
  export -f command

  run select_project_interactive "$PROJECTS_FILE"

  assert_failure
  assert_output --partial "ghost-tab-tui binary not found"
}

@test "select_project_interactive handles ghost-tab-tui failure" {
  ghost-tab-tui() {
    return 1
  }
  export -f ghost-tab-tui

  run select_project_interactive "$PROJECTS_FILE"

  assert_failure
}

@test "select_project_interactive omits update-version flag when empty" {
  _update_version=""

  ghost-tab-tui() {
    echo "$*" > "$TEST_TMP/captured_args"
    echo '{"action":"quit"}'
    return 0
  }
  export -f ghost-tab-tui
  export TEST_TMP

  select_project_interactive "$PROJECTS_FILE" || true

  local captured_args
  captured_args=$(<"$TEST_TMP/captured_args")
  [[ "$captured_args" != *"--update-version"* ]]
}

@test "select_project_interactive reads tab_title from settings file" {
  XDG_CONFIG_HOME="$TEST_TMP"
  mkdir -p "$TEST_TMP/ghost-tab"
  echo "tab_title=project" > "$TEST_TMP/ghost-tab/settings"

  ghost-tab-tui() {
    echo "$*" > "$TEST_TMP/captured_args"
    echo '{"action":"quit"}'
    return 0
  }
  export -f ghost-tab-tui
  export TEST_TMP

  select_project_interactive "$PROJECTS_FILE" || true

  local captured_args
  captured_args=$(<"$TEST_TMP/captured_args")
  [[ "$captured_args" == *"--tab-title project"* ]]
}

@test "select_project_interactive defaults tab_title to full" {
  XDG_CONFIG_HOME="$TEST_TMP"
  # No settings file

  ghost-tab-tui() {
    echo "$*" > "$TEST_TMP/captured_args"
    echo '{"action":"quit"}'
    return 0
  }
  export -f ghost-tab-tui
  export TEST_TMP

  select_project_interactive "$PROJECTS_FILE" || true

  local captured_args
  captured_args=$(<"$TEST_TMP/captured_args")
  [[ "$captured_args" == *"--tab-title"* ]]
  [[ "$captured_args" == *"full"* ]]
}

@test "select_project_interactive persists tab_title change" {
  XDG_CONFIG_HOME="$TEST_TMP"
  mkdir -p "$TEST_TMP/ghost-tab"

  ghost-tab-tui() {
    echo '{"action":"select-project","name":"proj1","path":"/tmp/p1","ai_tool":"claude","tab_title":"project"}'
    return 0
  }
  export -f ghost-tab-tui

  select_project_interactive "$PROJECTS_FILE"

  [[ -f "$TEST_TMP/ghost-tab/settings" ]]
  grep -q 'tab_title=project' "$TEST_TMP/ghost-tab/settings"
}

@test "select_project_interactive persists ai_tool change to ai-tool file" {
  XDG_CONFIG_HOME="$TEST_TMP"
  mkdir -p "$TEST_TMP/ghost-tab"

  AI_TOOLS_AVAILABLE=("claude" "codex")
  SELECTED_AI_TOOL="claude"

  ghost-tab-tui() {
    echo '{"action":"select-project","name":"proj1","path":"/tmp/p1","ai_tool":"codex"}'
    return 0
  }
  export -f ghost-tab-tui

  select_project_interactive "$PROJECTS_FILE"

  # AI tool preference file should be updated
  [[ -f "$TEST_TMP/ghost-tab/ai-tool" ]]
  [[ "$(cat "$TEST_TMP/ghost-tab/ai-tool")" == "codex" ]]
}

@test "select_project_interactive does not write ai-tool file when tool unchanged" {
  XDG_CONFIG_HOME="$TEST_TMP"
  mkdir -p "$TEST_TMP/ghost-tab"

  AI_TOOLS_AVAILABLE=("claude")
  SELECTED_AI_TOOL="claude"

  ghost-tab-tui() {
    echo '{"action":"select-project","name":"proj1","path":"/tmp/p1","ai_tool":"claude"}'
    return 0
  }
  export -f ghost-tab-tui

  select_project_interactive "$PROJECTS_FILE"

  # Should not write file when tool didn't change
  [[ ! -f "$TEST_TMP/ghost-tab/ai-tool" ]]
}

@test "select_project_interactive sets _selected_ai_tool for settings action" {
  AI_TOOLS_AVAILABLE=("claude" "codex")
  SELECTED_AI_TOOL="claude"

  ghost-tab-tui() {
    echo '{"action":"settings","ai_tool":"codex"}'
    return 0
  }
  export -f ghost-tab-tui

  select_project_interactive "$PROJECTS_FILE"

  [[ "$_selected_ai_tool" == "codex" ]]
}

@test "select_project_interactive updates existing tab_title in settings" {
  XDG_CONFIG_HOME="$TEST_TMP"
  mkdir -p "$TEST_TMP/ghost-tab"
  echo "tab_title=full" > "$TEST_TMP/ghost-tab/settings"

  ghost-tab-tui() {
    echo '{"action":"select-project","name":"proj1","path":"/tmp/p1","ai_tool":"claude","tab_title":"project"}'
    return 0
  }
  export -f ghost-tab-tui

  select_project_interactive "$PROJECTS_FILE"

  grep -q 'tab_title=project' "$TEST_TMP/ghost-tab/settings"
  ! grep -q 'tab_title=full' "$TEST_TMP/ghost-tab/settings"
}
