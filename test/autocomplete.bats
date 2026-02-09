setup() {
  load 'test_helper/common'
  _common_setup
  source "$PROJECT_ROOT/lib/tui.sh"
  source "$PROJECT_ROOT/lib/autocomplete.sh"

  # Create temp directory structure for testing
  TEST_TMP="$(mktemp -d)"
  mkdir -p "$TEST_TMP/alpha" "$TEST_TMP/beta" "$TEST_TMP/Beta-upper" "$TEST_TMP/gamma"
  touch "$TEST_TMP/file.txt"
}

teardown() {
  rm -rf "$TEST_TMP"
}

# --- get_suggestions ---

@test "get_suggestions: lists directory contents for trailing slash" {
  get_suggestions "$TEST_TMP/"
  [ "${#_suggestions[@]}" -gt 0 ]
}

@test "get_suggestions: filters by prefix" {
  get_suggestions "$TEST_TMP/al"
  [ "${#_suggestions[@]}" -eq 1 ]
  [[ "${_suggestions[0]}" == "alpha/" ]]
}

@test "get_suggestions: appends slash to directories" {
  get_suggestions "$TEST_TMP/gam"
  [[ "${_suggestions[0]}" == "gamma/" ]]
}

@test "get_suggestions: does not append slash to files" {
  get_suggestions "$TEST_TMP/fi"
  [[ "${_suggestions[0]}" == "file.txt" ]]
}

@test "get_suggestions: case-insensitive matching" {
  get_suggestions "$TEST_TMP/beta"
  # Should match both "beta" and "Beta-upper"
  [ "${#_suggestions[@]}" -ge 2 ]
}

@test "get_suggestions: limits to 8 results" {
  # Create many entries
  for i in $(seq 1 12); do
    mkdir -p "$TEST_TMP/many/item$i"
  done
  get_suggestions "$TEST_TMP/many/"
  [ "${#_suggestions[@]}" -le 8 ]
}

@test "get_suggestions: empty input defaults to ~/" {
  get_suggestions ""
  # Should get suggestions from home directory
  [ "${#_suggestions[@]}" -gt 0 ]
}

@test "get_suggestions: nonexistent directory returns empty" {
  get_suggestions "/nonexistent_path_xyz/"
  [ "${#_suggestions[@]}" -eq 0 ]
}
