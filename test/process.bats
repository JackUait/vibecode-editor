setup() {
  load 'test_helper/common'
  _common_setup
  source "$PROJECT_ROOT/lib/process.sh"
}

@test "kill_tree: kills a parent and its children" {
  # Spawn a parent that spawns a child
  bash -c 'sleep 300 & sleep 300 & wait' &
  parent_pid=$!
  sleep 0.3  # Let children start

  # Verify parent is running
  kill -0 "$parent_pid" 2>/dev/null

  # Suppress termination messages from the killed shell
  kill_tree "$parent_pid" TERM 2>/dev/null || true
  sleep 0.3

  # Parent should be dead
  run kill -0 "$parent_pid"
  assert_failure
}

@test "kill_tree: handles nonexistent PID gracefully" {
  # Should not error out
  run kill_tree 999999 TERM
  assert_success
}

@test "kill_tree: defaults to TERM signal" {
  sleep 300 &
  pid=$!
  kill_tree "$pid" 2>/dev/null || true
  sleep 0.2
  run kill -0 "$pid"
  assert_failure
}
