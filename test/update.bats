setup() {
  load 'test_helper/common'
  _common_setup

  # Create temp dir for cache file
  TEST_TMP="$(mktemp -d)"
  UPDATE_CACHE="$TEST_TMP/.update-check"
  _update_version=""

  source "$PROJECT_ROOT/lib/update.sh"
}

teardown() {
  rm -rf "$TEST_TMP"
}

# --- check_for_update: no brew ---

@test "check_for_update: returns early when brew not found" {
  # Hide brew by clearing PATH so command -v brew fails
  PATH="/nonexistent" check_for_update
  [ -z "$_update_version" ]
}

# --- check_for_update: fresh cache ---

@test "check_for_update: reads fresh cache with newer version" {
  # Write a fresh cache (timestamp = now)
  printf '2.0.0\n%s\n' "$(date +%s)" > "$UPDATE_CACHE"

  # Mock brew to return installed version 1.0.0
  brew() {
    if [ "$1" = "list" ] && [ "$2" = "--versions" ]; then
      echo "ghost-tab 1.0.0"
      return 0
    fi
    return 0
  }
  export -f brew

  check_for_update
  [ "$_update_version" = "2.0.0" ]
}

@test "check_for_update: ignores cache when versions match" {
  printf '1.0.0\n%s\n' "$(date +%s)" > "$UPDATE_CACHE"

  brew() {
    if [ "$1" = "list" ] && [ "$2" = "--versions" ]; then
      echo "ghost-tab 1.0.0"
      return 0
    fi
    return 0
  }
  export -f brew

  check_for_update
  [ -z "$_update_version" ]
}

# --- check_for_update: empty cache version ---

@test "check_for_update: handles empty version in cache" {
  printf '\n%s\n' "$(date +%s)" > "$UPDATE_CACHE"

  brew() { return 0; }
  export -f brew

  check_for_update
  [ -z "$_update_version" ]
}

# --- Network failure and timeout scenarios ---

@test "check_for_update: handles brew outdated network timeout" {
  brew() {
    if [[ "$*" == *"outdated"* ]]; then
      echo "Error: Operation timed out" >&2
      return 1
    fi
    if [[ "$*" == *"list --versions"* ]]; then
      echo "ghost-tab 1.0.0"
      return 0
    fi
    return 0
  }
  export -f brew

  check_for_update
  # Function spawns background process, verify it doesn't crash
  [ "$?" -eq 0 ]
}

@test "check_for_update: handles brew command hanging" {
  brew() {
    if [[ "$*" == *"outdated"* ]]; then
      # Simulate hanging (would timeout in real scenario)
      sleep 5 &
      return 1
    fi
    return 0
  }
  export -f brew

  check_for_update
  # Should return immediately, not wait for brew
  [ "$?" -eq 0 ]
}

@test "check_for_update: handles connection refused" {
  brew() {
    if [[ "$*" == *"outdated"* ]]; then
      echo "Error: Failed to connect to github.com: Connection refused" >&2
      return 7
    fi
    return 0
  }
  export -f brew

  check_for_update
  [ "$?" -eq 0 ]
}

@test "check_for_update: handles DNS resolution failure" {
  brew() {
    if [[ "$*" == *"outdated"* ]]; then
      echo "Error: Could not resolve host: github.com" >&2
      return 6
    fi
    return 0
  }
  export -f brew

  check_for_update
  [ "$?" -eq 0 ]
}

@test "check_for_update: handles HTTP 404 from brew" {
  brew() {
    if [[ "$*" == *"outdated"* ]]; then
      echo "Error: 404: Not Found" >&2
      return 1
    fi
    return 0
  }
  export -f brew

  check_for_update
  [ "$?" -eq 0 ]
}

@test "check_for_update: handles HTTP 503 service unavailable" {
  brew() {
    if [[ "$*" == *"outdated"* ]]; then
      echo "Error: 503 Service Unavailable" >&2
      return 1
    fi
    return 0
  }
  export -f brew

  check_for_update
  [ "$?" -eq 0 ]
}

# --- Invalid version string scenarios ---

@test "check_for_update: handles brew outdated returning malformed output" {
  brew() {
    if [[ "$*" == *"outdated"* ]]; then
      echo "CORRUPT@#$%DATA"
      return 0
    fi
    if [[ "$*" == *"list --versions"* ]]; then
      echo "ghost-tab 1.0.0"
      return 0
    fi
    return 0
  }
  export -f brew

  check_for_update
  [ "$?" -eq 0 ]
}

@test "check_for_update: handles brew list returning unparseable version" {
  printf '2.0.0\n%s\n' "$(date +%s)" > "$UPDATE_CACHE"

  brew() {
    if [[ "$*" == *"list --versions"* ]]; then
      echo "ghost-tab INVALID_VERSION_STRING"
      return 0
    fi
    return 0
  }
  export -f brew

  check_for_update
  [ "$?" -eq 0 ]
}

@test "check_for_update: handles brew list returning empty string" {
  printf '2.0.0\n%s\n' "$(date +%s)" > "$UPDATE_CACHE"

  brew() {
    if [[ "$*" == *"list --versions"* ]]; then
      echo ""
      return 0
    fi
    return 0
  }
  export -f brew

  check_for_update
  [ "$?" -eq 0 ]
}

@test "check_for_update: handles brew outdated output missing version number" {
  brew() {
    if [[ "$*" == *"outdated"* ]]; then
      echo "ghost-tab (1.0.0) <"
      return 0
    fi
    return 0
  }
  export -f brew

  check_for_update
  [ "$?" -eq 0 ]
}

@test "check_for_update: handles brew outdated returning multiple packages" {
  brew() {
    if [[ "$*" == *"outdated"* ]]; then
      echo "other-pkg (1.0.0) < 2.0.0"
      echo "ghost-tab (1.0.0) < 1.5.0"
      return 0
    fi
    return 0
  }
  export -f brew

  check_for_update
  [ "$?" -eq 0 ]
}

@test "check_for_update: handles brew returning version with special characters" {
  brew() {
    if [[ "$*" == *"outdated"* ]]; then
      echo "ghost-tab (1.0.0) < 2.0.0-beta+build.123"
      return 0
    fi
    return 0
  }
  export -f brew

  check_for_update
  [ "$?" -eq 0 ]
}

# --- Malformed cache file scenarios ---

@test "check_for_update: handles corrupted cache file" {
  printf '2.0.0\nNOT_A_NUMBER\n' > "$UPDATE_CACHE"

  brew() {
    if [[ "$*" == *"list --versions"* ]]; then
      echo "ghost-tab 1.0.0"
      return 0
    fi
    return 0
  }
  export -f brew

  check_for_update
  [ "$?" -eq 0 ]
}

@test "check_for_update: handles cache file with only one line" {
  echo "2.0.0" > "$UPDATE_CACHE"

  brew() {
    if [[ "$*" == *"list --versions"* ]]; then
      echo "ghost-tab 1.0.0"
      return 0
    fi
    return 0
  }
  export -f brew

  check_for_update
  [ "$?" -eq 0 ]
}

@test "check_for_update: handles cache file with future timestamp" {
  printf '2.0.0\n9999999999\n' > "$UPDATE_CACHE"

  brew() { return 0; }
  export -f brew

  check_for_update
  [ "$?" -eq 0 ]
}

@test "check_for_update: handles cache file with negative timestamp" {
  printf '2.0.0\n-1000\n' > "$UPDATE_CACHE"

  brew() { return 0; }
  export -f brew

  check_for_update
  [ "$?" -eq 0 ]
}

@test "check_for_update: handles cache directory creation failure" {
  TEST_CACHE_DIR="$TEST_TMP/readonly"
  mkdir -p "$TEST_CACHE_DIR"
  chmod 444 "$TEST_CACHE_DIR"
  UPDATE_CACHE="$TEST_CACHE_DIR/.update-check"

  brew() {
    if [[ "$*" == *"outdated"* ]]; then
      echo "ghost-tab (1.0.0) < 2.0.0"
      return 0
    fi
    return 0
  }
  export -f brew

  check_for_update
  # Should not crash even if cache write fails
  [ "$?" -eq 0 ]

  chmod 755 "$TEST_CACHE_DIR"
}

@test "check_for_update: handles cache file with non-ASCII characters" {
  printf '2.0.0\342\234\223\n%s\n' "$(date +%s)" > "$UPDATE_CACHE"

  brew() {
    if [[ "$*" == *"list --versions"* ]]; then
      echo "ghost-tab 1.0.0"
      return 0
    fi
    return 0
  }
  export -f brew

  check_for_update
  [ "$?" -eq 0 ]
}

# --- Command not found scenarios ---

@test "check_for_update: returns early when brew not in PATH" {
  OLD_PATH="$PATH"
  PATH="/nonexistent"

  check_for_update
  result="$?"
  [ -z "$_update_version" ]
  [ "$result" -eq 0 ]

  PATH="$OLD_PATH"
}

@test "check_for_update: handles brew command returning 127" {
  brew() {
    return 127
  }
  export -f brew

  check_for_update
  [ "$?" -eq 0 ]
}

@test "check_for_update: handles corrupted cache that sed cannot parse" {
  # Create cache with binary data that sed might fail on
  printf '\xff\xfe2.0.0\n%s\n' "$(date +%s)" > "$UPDATE_CACHE"

  brew() {
    if [[ "$*" == *"list --versions"* ]]; then
      echo "ghost-tab 1.0.0"
      return 0
    fi
    return 0
  }
  export -f brew

  check_for_update
  # Should handle gracefully even with corrupt cache
  [ "$?" -eq 0 ]
}
