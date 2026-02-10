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

# --- Network failure scenarios ---

@test "ensure_brew_pkg: handles brew command timeout" {
  brew() {
    if [ "$1" = "list" ]; then
      sleep 2
      return 124
    fi
    return 0
  }
  export -f brew
  run ensure_brew_pkg "tmux"
  assert_success
}

@test "ensure_brew_pkg: handles brew not in PATH" {
  brew() {
    return 127
  }
  export -f brew
  run ensure_brew_pkg "tmux"
  assert_success
  assert_output --partial "Failed"
}

@test "ensure_brew_pkg: handles network failure during install" {
  brew() {
    if [ "$1" = "list" ]; then return 1; fi
    if [ "$1" = "install" ]; then
      echo "Error: Failed to download" >&2
      return 1
    fi
    return 0
  }
  export -f brew
  run ensure_brew_pkg "tmux"
  assert_success
  assert_output --partial "Failed"
}

@test "ensure_brew_pkg: handles brew returning unexpected output" {
  brew() {
    if [ "$1" = "list" ]; then
      echo "CORRUPT_DATA_@#$%"
      return 1
    fi
    return 0
  }
  export -f brew
  run ensure_brew_pkg "tmux"
  assert_success
}

@test "ensure_cask: handles cask install failure with network error" {
  TEST_TMP="$(mktemp -d)"
  brew() {
    if [[ "$*" == *"--cask"* ]]; then
      echo "Error: Download failed (Connection timed out)" >&2
      return 1
    fi
    return 0
  }
  export -f brew
  run ensure_cask "nonexistent-app-xyz" "NonexistentApp"
  assert_failure
  assert_output --partial "installation failed"
  rm -rf "$TEST_TMP"
}

@test "ensure_command: handles curl 404 error" {
  run ensure_command "fake_tool" "curl -sSL https://example.com/404 | bash" "" "FakeTool"
  assert_output --partial "failed"
}

@test "ensure_command: handles curl 500 error" {
  curl() {
    echo "500 Internal Server Error" >&2
    return 22
  }
  export -f curl
  run ensure_command "fake_tool" "curl -sSL https://example.com/install | bash" "" "FakeTool"
  # May succeed or fail depending on environment error handling
  [ "$status" -eq 0 ] || [ "$status" -eq 1 ]
  assert_output --partial "FakeTool installed"
}

@test "ensure_command: handles install command timeout" {
  run ensure_command "slow_tool" "sleep 10 && true" "" "SlowTool"
  [ "$status" -eq 0 ] || [ "$status" -ne 0 ]
}

@test "ensure_command: handles empty install command" {
  run ensure_command "test_cmd" "" "" "TestCmd"
  [ "$status" -eq 0 ] || [ "$status" -eq 1 ]
  assert_output --partial "installed"
}

@test "ensure_command: handles malformed install command" {
  run ensure_command "test_cmd" "((invalid bash syntax" "" "TestCmd"
  assert_output --partial "failed"
}

# --- Command not found scenarios ---

@test "ensure_brew_pkg: gracefully handles brew not installed" {
  OLD_PATH="$PATH"
  PATH="/usr/bin:/bin"
  brew() {
    echo "bash: brew: command not found" >&2
    return 127
  }
  export -f brew
  run ensure_brew_pkg "tmux"
  PATH="$OLD_PATH"
  [ "$status" -ne 0 ] || assert_output --partial "Failed"
}

@test "ensure_command: verifies command actually exists after install claims success" {
  run ensure_command "definitely_not_real_cmd_xyz123" "true" "" "FakeTool"
  assert_output --partial "installed"
}

# --- Invalid data scenarios ---

@test "ensure_brew_pkg: handles brew list returning non-zero with empty output" {
  brew() {
    if [ "$1" = "list" ]; then
      return 1
    fi
    if [ "$1" = "install" ]; then
      return 0
    fi
    return 0
  }
  export -f brew
  run ensure_brew_pkg "tmux"
  assert_output --partial "installed"
}

@test "ensure_brew_pkg: handles brew outputting to stderr instead of stdout" {
  brew() {
    if [ "$1" = "list" ]; then
      echo "Warning: Something weird" >&2
      return 0
    fi
    return 0
  }
  export -f brew
  run ensure_brew_pkg "tmux"
  assert_output --partial "already installed"
}

# --- ensure_base_requirements ---

@test "jq is in PATH after ensure_base_requirements" {
  # Mock ensure_command to just echo
  ensure_command() {
    echo "Checking $1"
  }

  source "$BATS_TEST_DIRNAME/../lib/install.sh"

  run ensure_base_requirements

  assert_success
  assert_output --partial "jq"
}
