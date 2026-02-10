setup() {
  load 'test_helper/common'
  _common_setup
  source "$PROJECT_ROOT/lib/process.sh"
  TEST_PIDS=()
}

teardown() {
  # CRITICAL: Clean up ALL test processes
  for pid in "${TEST_PIDS[@]}"; do
    kill -9 "$pid" 2>/dev/null || true
  done
  # Also clean up any bash children of test process
  local test_children
  test_children=$(pgrep -P $$ 2>/dev/null || true)
  for pid in $test_children; do
    kill -9 "$pid" 2>/dev/null || true
  done
  sleep 0.1
}

@test "kill_tree: kills a parent and its children" {
  # Spawn a parent that spawns a child
  bash -c 'sleep 300 & sleep 300 & wait' &
  parent_pid=$!
  TEST_PIDS+=("$parent_pid")
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
  TEST_PIDS+=("$pid")
  kill_tree "$pid" 2>/dev/null || true
  sleep 0.2
  run kill -0 "$pid"
  assert_failure
}

# --- EDGE CASE TESTS ---

@test "kill_tree: kills deep process tree (4 levels)" {
  # Create 4-level deep process tree
  # Level 1 spawns Level 2, Level 2 spawns Level 3, Level 3 spawns Level 4
  bash -c 'bash -c "bash -c \"sleep 100\" & sleep 100" & sleep 100' &
  parent_pid=$!
  TEST_PIDS+=("$parent_pid")
  sleep 0.5  # Let all levels start

  # Verify parent is running
  kill -0 "$parent_pid" 2>/dev/null

  # Get all descendants before kill
  local descendants
  descendants=$(pgrep -P "$parent_pid" 2>/dev/null || true)

  # Kill the tree
  kill_tree "$parent_pid" TERM 2>/dev/null || true
  sleep 0.5

  # Parent should be dead
  run kill -0 "$parent_pid"
  assert_failure

  # All descendants should be dead
  for pid in $descendants; do
    run kill -0 "$pid"
    assert_failure
  done
}

@test "kill_tree: handles process that forks during kill with timeout" {
  # Create a process that spawns a new child when receiving TERM
  # Use timeout to prevent test hanging
  bash -c 'trap "bash -c \"sleep 5\" &" TERM; sleep 100' &
  parent_pid=$!
  TEST_PIDS+=("$parent_pid")
  sleep 0.3

  # Kill the tree (with timeout safety)
  timeout 3 bash -c "
    source '$PROJECT_ROOT/lib/process.sh'
    kill_tree $parent_pid TERM 2>/dev/null || true
    sleep 0.5
    # Force kill any survivors
    kill_tree $parent_pid KILL 2>/dev/null || true
  " || true

  sleep 0.3

  # Parent should be dead
  run kill -0 "$parent_pid"
  assert_failure

  # Verify no children left running
  local children
  children=$(pgrep -P "$parent_pid" 2>/dev/null || true)
  [[ -z "$children" ]]
}

@test "kill_tree: handles zombie processes in tree" {
  # Create parent with zombie child (child exits immediately, parent sleeps)
  bash -c '(exit 0) & sleep 5' &
  parent_pid=$!
  TEST_PIDS+=("$parent_pid")
  sleep 0.3

  # Verify parent is running
  kill -0 "$parent_pid" 2>/dev/null

  # Kill the tree
  kill_tree "$parent_pid" TERM 2>/dev/null || true
  sleep 0.3

  # Parent should be dead (zombie cleans up automatically)
  run kill -0 "$parent_pid"
  assert_failure
}

@test "kill_tree: multiple calls are idempotent" {
  sleep 100 &
  pid=$!
  TEST_PIDS+=("$pid")
  sleep 0.2

  # First kill
  run kill_tree "$pid" TERM
  assert_success
  sleep 0.2

  # Second kill on same (now dead) PID should not error
  run kill_tree "$pid" TERM
  assert_success

  # Third kill should also succeed
  run kill_tree "$pid" TERM
  assert_success
}

@test "kill_tree: works with KILL signal (immediate termination)" {
  # Create process that ignores TERM via trap
  bash -c 'trap "" TERM; sleep 100' &
  pid=$!
  TEST_PIDS+=("$pid")
  sleep 0.3

  # TERM won't work (process ignores it)
  kill_tree "$pid" TERM 2>/dev/null || true
  sleep 0.3

  # Process still alive
  kill -0 "$pid" 2>/dev/null

  # KILL should work immediately
  kill_tree "$pid" KILL 2>/dev/null || true
  sleep 0.2

  # Process should be dead
  run kill -0 "$pid"
  assert_failure
}

@test "kill_tree: works with HUP signal" {
  sleep 100 &
  pid=$!
  TEST_PIDS+=("$pid")
  sleep 0.2

  kill_tree "$pid" HUP 2>/dev/null || true
  sleep 0.2

  run kill -0 "$pid"
  assert_failure
}

@test "kill_tree: handles moderate-sized process tree (15 processes)" {
  # Create a process that spawns 14 children
  bash -c '
    for i in {1..14}; do
      sleep 100 &
    done
    wait
  ' &
  parent_pid=$!
  TEST_PIDS+=("$parent_pid")
  sleep 0.5  # Let all children start

  # Count children
  local child_count
  child_count=$(pgrep -P "$parent_pid" 2>/dev/null | wc -l)
  [[ "$child_count" -eq 14 ]]

  # Kill the tree
  kill_tree "$parent_pid" TERM 2>/dev/null || true
  sleep 0.5

  # Parent should be dead
  run kill -0 "$parent_pid"
  assert_failure

  # No children should remain
  local remaining
  remaining=$(pgrep -P "$parent_pid" 2>/dev/null || true)
  [[ -z "$remaining" ]]
}

@test "kill_tree: handles process that exits before KILL" {
  # Start a short-lived process
  bash -c 'sleep 0.2' &
  pid=$!
  TEST_PIDS+=("$pid")

  # Send TERM, then process exits naturally
  kill_tree "$pid" TERM 2>/dev/null || true
  sleep 0.5  # Process already gone

  # Second kill should handle gracefully
  run kill_tree "$pid" KILL
  assert_success
}

@test "kill_tree: concurrent kills don't cause errors" {
  sleep 100 &
  pid=$!
  TEST_PIDS+=("$pid")
  sleep 0.2

  # Launch two concurrent kills in background
  (kill_tree "$pid" TERM 2>/dev/null || true) &
  (kill_tree "$pid" TERM 2>/dev/null || true) &

  # Wait for both to complete
  wait
  sleep 0.3

  # Process should be dead
  run kill -0 "$pid"
  assert_failure
}

@test "kill_tree: handles empty process tree (no children)" {
  # Single process with no children
  sleep 100 &
  pid=$!
  TEST_PIDS+=("$pid")
  sleep 0.2

  # Should work fine with no children
  run kill_tree "$pid" TERM
  assert_success
  sleep 0.2

  run kill -0 "$pid"
  assert_failure
}
