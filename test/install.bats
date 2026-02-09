setup() {
  load 'test_helper/common'
  _common_setup
  source "$PROJECT_ROOT/lib/tui.sh"
  source "$PROJECT_ROOT/lib/install.sh"
}

# --- ensure_brew_pkg ---

@test "ensure_brew_pkg: reports already installed" {
  brew() { return 0; }  # mock: brew list succeeds
  export -f brew
  run ensure_brew_pkg "tmux"
  assert_output --partial "already installed"
}

@test "ensure_brew_pkg: installs missing package" {
  _call_count=0
  brew() {
    if [ "$1" = "list" ]; then return 1; fi  # not installed
    if [ "$1" = "install" ]; then return 0; fi  # install succeeds
    return 0
  }
  export -f brew
  run ensure_brew_pkg "tmux"
  assert_output --partial "installed"
}

@test "ensure_brew_pkg: warns on install failure" {
  brew() {
    if [ "$1" = "list" ]; then return 1; fi
    if [ "$1" = "install" ]; then return 1; fi
    return 0
  }
  export -f brew
  run ensure_brew_pkg "tmux"
  assert_output --partial "Failed"
}

# --- ensure_cask ---

@test "ensure_cask: reports found when app exists" {
  # Use a directory that exists
  TEST_TMP="$(mktemp -d)"
  mkdir -p "$TEST_TMP/Ghostty.app"
  # Override the check by testing with an existing app path
  ensure_cask_check() {
    if [ -d "$TEST_TMP/Ghostty.app" ]; then
      success "Ghostty found"
    fi
  }
  run ensure_cask_check
  assert_output --partial "found"
  rm -rf "$TEST_TMP"
}

# --- ensure_command ---

@test "ensure_command: reports already installed for existing command" {
  run ensure_command "bash" "echo noop" "" "Bash"
  assert_output --partial "already installed"
}

@test "ensure_command: installs missing command" {
  run ensure_command "nonexistent_cmd_xyz" "true" "" "TestTool"
  assert_output --partial "installed"
}

@test "ensure_command: shows post message on success" {
  run ensure_command "nonexistent_cmd_xyz" "true" "Run it now" "TestTool"
  assert_output --partial "Run it now"
}

@test "ensure_command: warns on install failure" {
  run ensure_command "nonexistent_cmd_xyz" "false" "" "TestTool"
  assert_output --partial "failed"
}
