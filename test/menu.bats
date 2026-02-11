setup() {
  load 'test_helper/common'
  _common_setup
  TEST_TMP="$(mktemp -d)"
}

teardown() {
  rm -rf "$TEST_TMP"
}

@test "select_project_interactive calls ghost-tab-tui and parses JSON" {
  source "$PROJECT_ROOT/lib/menu-tui.sh"

  PROJECTS_FILE="$TEST_TMP/projects"
  echo "proj1:/tmp/p1" > "$PROJECTS_FILE"

  # Mock ghost-tab-tui
  ghost-tab-tui() {
    if [[ "$1" == "select-project" ]]; then
      echo '{"name":"proj1","path":"/tmp/p1","selected":true}'
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

    case $count in
      1) echo "true" ;;      # .selected
      2) echo "proj1" ;;     # .name
      3) echo "/tmp/p1" ;;   # .path
    esac
    return 0
  }
  export -f jq
  export TEST_TMP

  # Call function without run to preserve variable assignments
  select_project_interactive "$PROJECTS_FILE"
  local result=$?

  # Check return code and variables
  [[ $result -eq 0 ]]
  [[ "$_selected_project_name" == "proj1" ]]
  [[ "$_selected_project_path" == "/tmp/p1" ]]
}

@test "select_project_interactive handles cancellation" {
  source "$PROJECT_ROOT/lib/menu-tui.sh"

  PROJECTS_FILE="$TEST_TMP/projects"
  echo "proj1:/tmp/p1" > "$PROJECTS_FILE"

  # Mock ghost-tab-tui to return cancelled
  ghost-tab-tui() {
    if [[ "$1" == "select-project" ]]; then
      echo '{"selected":false}'
      return 0
    fi
    return 1
  }
  export -f ghost-tab-tui

  # Mock jq
  jq() {
    if [[ "$2" == ".selected" ]]; then
      echo "false"
    fi
    return 0
  }
  export -f jq

  run select_project_interactive "$PROJECTS_FILE"

  assert_failure
}

@test "select_project_interactive validates null name" {
  source "$PROJECT_ROOT/lib/menu-tui.sh"

  PROJECTS_FILE="$TEST_TMP/projects"

  # Mock error function
  error() {
    echo "ERROR: $*" >&2
  }
  export -f error

  # Mock ghost-tab-tui to return null name
  ghost-tab-tui() {
    if [[ "$1" == "select-project" ]]; then
      echo '{"name":null,"path":"/tmp/p1","selected":true}'
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

    case $count in
      1) echo "true" ;;   # .selected
      2) echo "null" ;;   # .name
    esac
    return 0
  }
  export -f jq
  export TEST_TMP

  run select_project_interactive "$PROJECTS_FILE"

  assert_failure
  assert_output --partial "invalid project name"
}

@test "select_project_interactive validates null path" {
  source "$PROJECT_ROOT/lib/menu-tui.sh"

  PROJECTS_FILE="$TEST_TMP/projects"

  # Mock error function
  error() {
    echo "ERROR: $*" >&2
  }
  export -f error

  # Mock ghost-tab-tui
  ghost-tab-tui() {
    if [[ "$1" == "select-project" ]]; then
      echo '{"name":"proj1","path":null,"selected":true}'
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

    case $count in
      1) echo "true" ;;    # .selected
      2) echo "proj1" ;;   # .name
      3) echo "null" ;;    # .path
    esac
    return 0
  }
  export -f jq
  export TEST_TMP

  run select_project_interactive "$PROJECTS_FILE"

  assert_failure
  assert_output --partial "invalid project path"
}

@test "select_project_interactive handles jq parse failure" {
  source "$PROJECT_ROOT/lib/menu-tui.sh"

  PROJECTS_FILE="$TEST_TMP/projects"

  # Mock error function
  error() {
    echo "ERROR: $*" >&2
  }
  export -f error

  # Mock ghost-tab-tui
  ghost-tab-tui() {
    echo '{"name":"proj1","path":"/tmp/p1","selected":true}'
    return 0
  }
  export -f ghost-tab-tui

  # Mock jq (fails)
  jq() {
    return 1
  }
  export -f jq

  run select_project_interactive "$PROJECTS_FILE"

  assert_failure
  assert_output --partial "Failed to parse"
}

@test "select_project_interactive handles binary missing" {
  source "$PROJECT_ROOT/lib/menu-tui.sh"

  PROJECTS_FILE="$TEST_TMP/projects"

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

  run select_project_interactive "$PROJECTS_FILE"

  assert_failure
  assert_output --partial "ghost-tab-tui binary not found"
}
