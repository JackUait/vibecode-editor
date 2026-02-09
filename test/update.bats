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
