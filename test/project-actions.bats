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

  source "$BATS_TEST_DIRNAME/../lib/project-actions-tui.sh"

  # Call function directly (not via run) to check variables
  add_project_interactive

  # Check that global variables are set
  [ "$_add_project_name" = "test-proj" ]
  [ "$_add_project_path" = "/tmp/test" ]
}

@test "add_project_interactive returns 1 when user cancels" {
  # Mock ghost-tab-tui returning cancelled
  ghost-tab-tui() {
    if [[ "$1" == "add-project" ]]; then
      echo '{"name":"","path":"","confirmed":false}'
      return 0
    fi
    return 1
  }
  export -f ghost-tab-tui

  source "$BATS_TEST_DIRNAME/../lib/project-actions-tui.sh"

  run add_project_interactive

  assert_failure
}
