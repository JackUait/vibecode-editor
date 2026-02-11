setup() {
  load 'test_helper/common'
  _common_setup
  TEST_TMP="$(mktemp -d)"
}

teardown() {
  rm -rf "$TEST_TMP"
}

@test "settings_menu_interactive calls ghost-tab-tui and parses JSON" {
  source "$PROJECT_ROOT/lib/settings-menu-tui.sh"

  # Mock ghost-tab-tui
  ghost-tab-tui() {
    if [[ "$1" == "settings-menu" ]]; then
      echo '{"action":"toggle-ghost"}'
      return 0
    fi
    return 1
  }
  export -f ghost-tab-tui

  # Mock jq
  jq() {
    if [[ "$2" == ".action" ]]; then
      echo "toggle-ghost"
    fi
    return 0
  }
  export -f jq

  run settings_menu_interactive

  assert_success
  assert_output "toggle-ghost"
}

@test "settings_menu_interactive handles quit action" {
  source "$PROJECT_ROOT/lib/settings-menu-tui.sh"

  # Mock ghost-tab-tui
  ghost-tab-tui() {
    if [[ "$1" == "settings-menu" ]]; then
      echo '{"action":"quit"}'
      return 0
    fi
    return 1
  }
  export -f ghost-tab-tui

  # Mock jq
  jq() {
    if [[ "$2" == ".action" ]]; then
      echo "quit"
    fi
    return 0
  }
  export -f jq

  run settings_menu_interactive

  assert_success
  assert_output "quit"
}

@test "settings_menu_interactive handles binary missing" {
  source "$PROJECT_ROOT/lib/settings-menu-tui.sh"

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

  run settings_menu_interactive

  assert_failure
  assert_output --partial "ghost-tab-tui binary not found"
}

@test "settings_menu_interactive handles jq parse failure" {
  source "$PROJECT_ROOT/lib/settings-menu-tui.sh"

  # Mock error function
  error() {
    echo "ERROR: $*" >&2
  }
  export -f error

  # Mock ghost-tab-tui
  ghost-tab-tui() {
    echo '{"action":"toggle-ghost"}'
    return 0
  }
  export -f ghost-tab-tui

  # Mock jq (fails)
  jq() {
    return 1
  }
  export -f jq

  run settings_menu_interactive

  assert_failure
  assert_output --partial "Failed to parse"
}

@test "settings_menu_interactive validates against null action" {
  source "$PROJECT_ROOT/lib/settings-menu-tui.sh"

  # Mock error function
  error() {
    echo "ERROR: $*" >&2
  }
  export -f error

  # Mock ghost-tab-tui
  ghost-tab-tui() {
    echo '{"action":null}'
    return 0
  }
  export -f ghost-tab-tui

  # Mock jq
  jq() {
    if [[ "$2" == ".action" ]]; then
      echo "null"
    fi
    return 0
  }
  export -f jq

  run settings_menu_interactive

  assert_failure
  assert_output --partial "invalid action"
}

@test "settings_menu_interactive allows empty action for quit" {
  source "$PROJECT_ROOT/lib/settings-menu-tui.sh"

  # Mock ghost-tab-tui
  ghost-tab-tui() {
    echo '{"action":""}'
    return 0
  }
  export -f ghost-tab-tui

  # Mock jq
  jq() {
    if [[ "$2" == ".action" ]]; then
      echo ""
    fi
    return 0
  }
  export -f jq

  run settings_menu_interactive

  assert_success
  assert_output ""
}
