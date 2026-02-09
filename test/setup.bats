setup() {
  load 'test_helper/common'
  _common_setup
  source "$PROJECT_ROOT/lib/setup.sh"
  TEST_DIR="$(mktemp -d)"
}

teardown() {
  rm -rf "$TEST_DIR"
}

@test "resolve_share_dir: returns brew share when in brew prefix" {
  run resolve_share_dir "/opt/homebrew/bin" "/opt/homebrew"
  assert_output "/opt/homebrew/share/ghost-tab"
}

@test "resolve_share_dir: returns parent dir when not in brew prefix" {
  mkdir -p "$TEST_DIR/ghost-tab/bin"
  run resolve_share_dir "$TEST_DIR/ghost-tab/bin" ""
  assert_output "$TEST_DIR/ghost-tab"
}

@test "resolve_share_dir: returns parent dir when brew prefix is empty" {
  mkdir -p "$TEST_DIR/ghost-tab/bin"
  run resolve_share_dir "$TEST_DIR/ghost-tab/bin" ""
  assert_output "$TEST_DIR/ghost-tab"
}
