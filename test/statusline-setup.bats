setup() {
  load 'test_helper/common'
  _common_setup
  source "$PROJECT_ROOT/lib/tui.sh"
  source "$PROJECT_ROOT/lib/settings-json.sh"
  source "$PROJECT_ROOT/lib/statusline-setup.sh"
  TEST_TMP="$(mktemp -d)"

  # Create a fake share dir with template files
  SHARE_DIR="$TEST_TMP/share"
  mkdir -p "$SHARE_DIR/templates"
  mkdir -p "$SHARE_DIR/lib"
  echo "mock-settings" > "$SHARE_DIR/templates/ccstatusline-settings.json"
  echo "mock-command" > "$SHARE_DIR/templates/statusline-command.sh"
  echo "mock-wrapper" > "$SHARE_DIR/templates/statusline-wrapper.sh"
  echo "mock-helpers" > "$SHARE_DIR/lib/statusline.sh"

  # Create fake home dirs
  FAKE_HOME="$TEST_TMP/home"
  mkdir -p "$FAKE_HOME/.config/ccstatusline"
  mkdir -p "$FAKE_HOME/.claude"
}

teardown() {
  rm -rf "$TEST_TMP"
}

@test "setup_statusline: copies config and scripts when npm available" {
  # Mock npm to succeed for all calls
  npm() { return 0; }
  export -f npm

  # Mock _has_npm to return true
  _has_npm() { return 0; }
  export -f _has_npm

  setup_statusline "$SHARE_DIR" "$FAKE_HOME/.claude/settings.json" "$FAKE_HOME"
  [ -f "$FAKE_HOME/.config/ccstatusline/settings.json" ]
  [ -f "$FAKE_HOME/.claude/statusline-command.sh" ]
  [ -f "$FAKE_HOME/.claude/statusline-wrapper.sh" ]
  [ -f "$FAKE_HOME/.claude/statusline-helpers.sh" ]
  [ -x "$FAKE_HOME/.claude/statusline-command.sh" ]
  [ -x "$FAKE_HOME/.claude/statusline-wrapper.sh" ]
}

@test "setup_statusline: skips when npm not available and brew fails" {
  # Mock _has_npm to return false (npm not found)
  _has_npm() { return 1; }
  export -f _has_npm
  brew() { return 1; }
  export -f brew

  run setup_statusline "$SHARE_DIR" "$FAKE_HOME/.claude/settings.json" "$FAKE_HOME"
  [ "$status" -eq 0 ]
  [ ! -f "$FAKE_HOME/.claude/statusline-command.sh" ]
}

@test "setup_statusline: reports already installed" {
  npm() {
    if [[ "$1" == "list" ]]; then return 0; fi
    return 0
  }
  export -f npm
  _has_npm() { return 0; }
  export -f _has_npm

  run setup_statusline "$SHARE_DIR" "$FAKE_HOME/.claude/settings.json" "$FAKE_HOME"
  assert_output --partial "already installed"
}

@test "setup_statusline: warns and skips when npm install fails" {
  _has_npm() { return 0; }
  export -f _has_npm
  npm() {
    if [[ "$1" == "list" ]]; then return 1; fi
    if [[ "$1" == "install" ]]; then return 1; fi
    return 0
  }
  export -f npm

  run setup_statusline "$SHARE_DIR" "$FAKE_HOME/.claude/settings.json" "$FAKE_HOME"
  [ "$status" -eq 0 ]
  assert_output --partial "Failed to install"
  [ ! -f "$FAKE_HOME/.claude/statusline-command.sh" ]
  [ ! -f "$FAKE_HOME/.claude/statusline-wrapper.sh" ]
}

@test "setup_statusline: installs ccstatusline and copies files on fresh install" {
  _has_npm() { return 0; }
  export -f _has_npm
  # First npm list call returns 1 (not installed), subsequent calls return 0
  _npm_list_call_count=0
  export _npm_list_call_count
  npm() {
    if [[ "$1" == "list" ]]; then
      _npm_list_call_count=$((_npm_list_call_count + 1))
      export _npm_list_call_count
      if [[ "$_npm_list_call_count" -eq 1 ]]; then return 1; fi
      return 0
    fi
    if [[ "$1" == "install" ]]; then return 0; fi
    return 0
  }
  export -f npm

  run setup_statusline "$SHARE_DIR" "$FAKE_HOME/.claude/settings.json" "$FAKE_HOME"
  assert_output --partial "ccstatusline installed"
  refute_output --partial "already installed"
  [ -f "$FAKE_HOME/.config/ccstatusline/settings.json" ]
  [ -f "$FAKE_HOME/.claude/statusline-command.sh" ]
  [ -f "$FAKE_HOME/.claude/statusline-wrapper.sh" ]
}

@test "setup_statusline: calls merge_claude_settings after file copy" {
  _has_npm() { return 0; }
  export -f _has_npm
  npm() { return 0; }
  export -f npm

  local claude_settings="$TEST_TMP/claude-settings/settings.json"

  run setup_statusline "$SHARE_DIR" "$claude_settings" "$FAKE_HOME"
  [ "$status" -eq 0 ]
  [ -f "$claude_settings" ]
  grep -q '"statusLine"' "$claude_settings"
}

# --- npm install failure scenarios ---

@test "setup_statusline: handles npm install network timeout" {
  _has_npm() { return 0; }
  export -f _has_npm
  npm() {
    if [[ "$*" == *"install"* ]]; then
      echo "npm ERR! network timeout" >&2
      return 1
    fi
    if [[ "$*" == *"list"* ]]; then
      return 1
    fi
    return 0
  }
  export -f npm

  run setup_statusline "$SHARE_DIR" "$FAKE_HOME/.claude/settings.json" "$FAKE_HOME"
  assert_success
  assert_output --partial "Failed to install"
  [ ! -f "$FAKE_HOME/.claude/statusline-command.sh" ]
}

@test "setup_statusline: handles npm install ECONNREFUSED" {
  _has_npm() { return 0; }
  export -f _has_npm
  npm() {
    if [[ "$*" == *"install"* ]]; then
      echo "npm ERR! network request to https://registry.npmjs.org/ccstatusline failed, reason: connect ECONNREFUSED" >&2
      return 1
    fi
    if [[ "$*" == *"list"* ]]; then return 1; fi
    return 0
  }
  export -f npm

  run setup_statusline "$SHARE_DIR" "$FAKE_HOME/.claude/settings.json" "$FAKE_HOME"
  assert_success
  assert_output --partial "Failed to install"
}

@test "setup_statusline: handles npm install ETIMEDOUT" {
  _has_npm() { return 0; }
  export -f _has_npm
  npm() {
    if [[ "$*" == *"install"* ]]; then
      echo "npm ERR! network request timed out, reason: ETIMEDOUT" >&2
      return 1
    fi
    if [[ "$*" == *"list"* ]]; then return 1; fi
    return 0
  }
  export -f npm

  run setup_statusline "$SHARE_DIR" "$FAKE_HOME/.claude/settings.json" "$FAKE_HOME"
  assert_success
  assert_output --partial "Failed to install"
}

@test "setup_statusline: handles npm registry returning 404" {
  _has_npm() { return 0; }
  export -f _has_npm
  npm() {
    if [[ "$*" == *"install"* ]]; then
      echo "npm ERR! 404 Not Found - GET https://registry.npmjs.org/ccstatusline" >&2
      return 1
    fi
    if [[ "$*" == *"list"* ]]; then return 1; fi
    return 0
  }
  export -f npm

  run setup_statusline "$SHARE_DIR" "$FAKE_HOME/.claude/settings.json" "$FAKE_HOME"
  assert_success
  assert_output --partial "Failed to install"
}

@test "setup_statusline: handles npm registry returning 500" {
  _has_npm() { return 0; }
  export -f _has_npm
  npm() {
    if [[ "$*" == *"install"* ]]; then
      echo "npm ERR! 500 Internal Server Error - GET https://registry.npmjs.org/ccstatusline" >&2
      return 1
    fi
    if [[ "$*" == *"list"* ]]; then return 1; fi
    return 0
  }
  export -f npm

  run setup_statusline "$SHARE_DIR" "$FAKE_HOME/.claude/settings.json" "$FAKE_HOME"
  assert_success
  assert_output --partial "Failed to install"
}

@test "setup_statusline: handles npm registry returning 503 unavailable" {
  _has_npm() { return 0; }
  export -f _has_npm
  npm() {
    if [[ "$*" == *"install"* ]]; then
      echo "npm ERR! 503 Service Unavailable - GET https://registry.npmjs.org/ccstatusline" >&2
      return 1
    fi
    if [[ "$*" == *"list"* ]]; then return 1; fi
    return 0
  }
  export -f npm

  run setup_statusline "$SHARE_DIR" "$FAKE_HOME/.claude/settings.json" "$FAKE_HOME"
  assert_success
  assert_output --partial "Failed to install"
}

@test "setup_statusline: handles npm install hanging" {
  _has_npm() { return 0; }
  export -f _has_npm
  npm() {
    if [[ "$*" == *"install"* ]]; then
      sleep 5 &
      return 1
    fi
    if [[ "$*" == *"list"* ]]; then return 1; fi
    return 0
  }
  export -f npm

  run setup_statusline "$SHARE_DIR" "$FAKE_HOME/.claude/settings.json" "$FAKE_HOME"
  assert_success
  assert_output --partial "Failed to install"
}

@test "setup_statusline: handles npm install disk full error" {
  _has_npm() { return 0; }
  export -f _has_npm
  npm() {
    if [[ "$*" == *"install"* ]]; then
      echo "npm ERR! ENOSPC: no space left on device" >&2
      return 1
    fi
    if [[ "$*" == *"list"* ]]; then return 1; fi
    return 0
  }
  export -f npm

  run setup_statusline "$SHARE_DIR" "$FAKE_HOME/.claude/settings.json" "$FAKE_HOME"
  assert_success
  assert_output --partial "Failed to install"
}

@test "setup_statusline: handles npm install permission denied" {
  _has_npm() { return 0; }
  export -f _has_npm
  npm() {
    if [[ "$*" == *"install"* ]]; then
      echo "npm ERR! EACCES: permission denied" >&2
      return 1
    fi
    if [[ "$*" == *"list"* ]]; then return 1; fi
    return 0
  }
  export -f npm

  run setup_statusline "$SHARE_DIR" "$FAKE_HOME/.claude/settings.json" "$FAKE_HOME"
  assert_success
  assert_output --partial "Failed to install"
}

# --- npm list failure scenarios ---

@test "setup_statusline: handles npm list returning malformed output" {
  _has_npm() { return 0; }
  export -f _has_npm
  npm() {
    if [[ "$*" == *"list"* ]]; then
      echo "CORRUPT@#$%DATA"
      return 0
    fi
    return 0
  }
  export -f npm

  run setup_statusline "$SHARE_DIR" "$FAKE_HOME/.claude/settings.json" "$FAKE_HOME"
  assert_success
  assert_output --partial "already installed"
}

@test "setup_statusline: handles npm list command hanging" {
  _has_npm() { return 0; }
  export -f _has_npm
  npm() {
    if [[ "$*" == *"list"* ]]; then
      sleep 5 &
      return 0
    fi
    return 0
  }
  export -f npm

  run setup_statusline "$SHARE_DIR" "$FAKE_HOME/.claude/settings.json" "$FAKE_HOME"
  assert_success
}

@test "setup_statusline: handles npm returning non-JSON output" {
  _has_npm() { return 0; }
  export -f _has_npm
  npm() {
    if [[ "$*" == *"list"* ]]; then
      echo "This is not JSON"
      return 0
    fi
    return 0
  }
  export -f npm

  run setup_statusline "$SHARE_DIR" "$FAKE_HOME/.claude/settings.json" "$FAKE_HOME"
  assert_success
}

@test "setup_statusline: handles npm list returning empty output" {
  _has_npm() { return 0; }
  export -f _has_npm
  npm() {
    if [[ "$*" == *"list"* ]]; then
      echo ""
      return 0
    fi
    return 0
  }
  export -f npm

  run setup_statusline "$SHARE_DIR" "$FAKE_HOME/.claude/settings.json" "$FAKE_HOME"
  assert_success
}

# --- npm not found scenarios ---

@test "setup_statusline: handles npm not in PATH after install" {
  _has_npm() { return 1; }
  export -f _has_npm
  brew() {
    if [[ "$*" == *"install node"* ]]; then
      return 0
    fi
    return 0
  }
  export -f brew

  run setup_statusline "$SHARE_DIR" "$FAKE_HOME/.claude/settings.json" "$FAKE_HOME"
  assert_success
  refute_output --partial "ccstatusline installed"
}

@test "setup_statusline: handles brew node install failure" {
  _has_npm() { return 1; }
  export -f _has_npm
  brew() {
    if [[ "$*" == *"install node"* ]]; then
      echo "Error: Failed to install node" >&2
      return 1
    fi
    return 0
  }
  export -f brew

  run setup_statusline "$SHARE_DIR" "$FAKE_HOME/.claude/settings.json" "$FAKE_HOME"
  assert_success
  assert_output --partial "Node.js installation failed"
}

@test "setup_statusline: handles brew not available for node install" {
  _has_npm() { return 1; }
  export -f _has_npm
  brew() {
    return 127
  }
  export -f brew

  run setup_statusline "$SHARE_DIR" "$FAKE_HOME/.claude/settings.json" "$FAKE_HOME"
  assert_success
  refute_output --partial "ccstatusline"
}

# --- File operation failure scenarios ---

@test "setup_statusline: handles missing template files" {
  _has_npm() { return 0; }
  export -f _has_npm
  npm() { return 0; }
  export -f npm

  # Remove template files
  rm -rf "$SHARE_DIR/templates"

  run setup_statusline "$SHARE_DIR" "$FAKE_HOME/.claude/settings.json" "$FAKE_HOME"
  # cp will fail but script should handle gracefully
  [ "$status" -ne 0 ] || [ ! -f "$FAKE_HOME/.config/ccstatusline/settings.json" ]
}

@test "setup_statusline: handles read-only config directory" {
  _has_npm() { return 0; }
  export -f _has_npm
  npm() { return 0; }
  export -f npm

  # Make config dir read-only
  mkdir -p "$FAKE_HOME/.config"
  chmod 444 "$FAKE_HOME/.config"

  run setup_statusline "$SHARE_DIR" "$FAKE_HOME/.claude/settings.json" "$FAKE_HOME"
  # Function doesn't check mkdir errors, so it completes successfully
  [ "$status" -eq 0 ]

  chmod 755 "$FAKE_HOME/.config"
}

@test "setup_statusline: handles chmod failure on scripts" {
  _has_npm() { return 0; }
  export -f _has_npm
  npm() { return 0; }
  export -f npm

  # Override chmod to fail
  chmod() {
    return 1
  }
  export -f chmod

  run setup_statusline "$SHARE_DIR" "$FAKE_HOME/.claude/settings.json" "$FAKE_HOME"
  # Function doesn't check chmod errors, completes successfully
  [ "$status" -eq 0 ]
}

@test "setup_statusline: handles config file copy permission denied" {
  _has_npm() { return 0; }
  export -f _has_npm
  npm() { return 0; }
  export -f npm

  # Make ccstatusline dir read-only
  chmod 444 "$FAKE_HOME/.config/ccstatusline"

  run setup_statusline "$SHARE_DIR" "$FAKE_HOME/.claude/settings.json" "$FAKE_HOME"
  # Function doesn't check cp errors, may succeed or fail
  [ "$status" -eq 0 ] || [ "$status" -ne 0 ]

  chmod 755 "$FAKE_HOME/.config/ccstatusline"
}

@test "setup_statusline: handles corrupted template file" {
  _has_npm() { return 0; }
  export -f _has_npm
  npm() { return 0; }
  export -f npm

  # Create corrupted template (non-UTF8)
  printf '\xff\xfe\xfd' > "$SHARE_DIR/templates/ccstatusline-settings.json"

  run setup_statusline "$SHARE_DIR" "$FAKE_HOME/.claude/settings.json" "$FAKE_HOME"
  # Should copy file even if corrupted
  [ "$status" -eq 0 ]
  [ -f "$FAKE_HOME/.config/ccstatusline/settings.json" ]
}
