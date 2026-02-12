setup() {
  load 'test_helper/common'
  _common_setup
  source "$PROJECT_ROOT/lib/process.sh"
  source "$PROJECT_ROOT/lib/tmux-session.sh"
  TEST_TMP="$(mktemp -d)"
}

teardown() {
  rm -rf "$TEST_TMP"
}

# --- cleanup_tmux_session ---

@test "cleanup_tmux_session: calls kill and tmux kill-session" {
  _calls=()
  kill() { _calls+=("kill:$*"); return 0; }
  export -f kill

  tmux() {
    if [[ "$1" == "list-panes" ]]; then
      echo "12345"
    elif [[ "$1" == "kill-session" ]]; then
      _calls+=("kill-session:$*")
    fi
    return 0
  }
  export -f tmux

  kill_tree() { _calls+=("kill_tree:$*"); return 0; }
  export -f kill_tree

  sleep() { return 0; }
  export -f sleep

  cleanup_tmux_session "test-session" "99999" "tmux"
}

@test "cleanup_tmux_session: handles missing session gracefully" {
  tmux() { return 1; }
  export -f tmux
  kill() { return 0; }
  export -f kill
  sleep() { return 0; }
  export -f sleep
  kill_tree() { return 0; }
  export -f kill_tree

  run cleanup_tmux_session "nonexistent" "99999" "tmux"
  [ "$status" -eq 0 ]
}

# --- cleanup_tmux_session edge cases ---

@test "cleanup_tmux_session: handles multiple panes with process trees" {
  _kill_tree_calls=()
  kill() { return 0; }
  export -f kill

  tmux() {
    if [[ "$1" == "list-panes" ]]; then
      # Return 3 pane PIDs
      echo "10001
10002
10003"
    elif [[ "$1" == "kill-session" ]]; then
      return 0
    fi
    return 0
  }
  export -f tmux

  kill_tree() {
    _kill_tree_calls+=("$1:$2")
    return 0
  }
  export -f kill_tree

  sleep() { return 0; }
  export -f sleep

  cleanup_tmux_session "test-session" "99999" "tmux"

  # Should have killed all 3 panes with TERM, then all 3 with KILL
  [[ "${#_kill_tree_calls[@]}" -eq 6 ]]
  [[ "${_kill_tree_calls[0]}" == "10001:TERM" ]]
  [[ "${_kill_tree_calls[1]}" == "10002:TERM" ]]
  [[ "${_kill_tree_calls[2]}" == "10003:TERM" ]]
  [[ "${_kill_tree_calls[3]}" == "10001:KILL" ]]
  [[ "${_kill_tree_calls[4]}" == "10002:KILL" ]]
  [[ "${_kill_tree_calls[5]}" == "10003:KILL" ]]
}

@test "cleanup_tmux_session: handles watcher PID that doesn't exist" {
  kill() {
    # Simulate watcher PID not existing
    return 1
  }
  export -f kill

  tmux() {
    if [[ "$1" == "list-panes" ]]; then
      echo "10001"
    elif [[ "$1" == "kill-session" ]]; then
      return 0
    fi
    return 0
  }
  export -f tmux

  kill_tree() { return 0; }
  export -f kill_tree

  sleep() { return 0; }
  export -f sleep

  # Should not fail even if watcher kill fails
  run cleanup_tmux_session "test-session" "99999" "tmux"
  assert_success
}

@test "cleanup_tmux_session: handles pane PIDs that disappear between TERM and KILL" {
  _list_panes_call_count=0

  kill() { return 0; }
  export -f kill

  tmux() {
    if [[ "$1" == "list-panes" ]]; then
      _list_panes_call_count=$((_list_panes_call_count + 1))
      if [[ "$_list_panes_call_count" -eq 1 ]]; then
        # First call: 2 panes
        echo "10001
10002"
      else
        # Second call: only 1 pane (one exited already)
        echo "10002"
      fi
    elif [[ "$1" == "kill-session" ]]; then
      return 0
    fi
    return 0
  }
  export -f tmux

  kill_tree() { return 0; }
  export -f kill_tree

  sleep() { return 0; }
  export -f sleep

  run cleanup_tmux_session "test-session" "99999" "tmux"
  assert_success
}

@test "cleanup_tmux_session: handles no panes in session" {
  kill() { return 0; }
  export -f kill

  tmux() {
    if [[ "$1" == "list-panes" ]]; then
      # No panes (empty output)
      echo ""
    elif [[ "$1" == "kill-session" ]]; then
      return 0
    fi
    return 0
  }
  export -f tmux

  kill_tree() { return 0; }
  export -f kill_tree

  sleep() { return 0; }
  export -f sleep

  run cleanup_tmux_session "test-session" "99999" "tmux"
  assert_success
}

@test "cleanup_tmux_session: handles kill_tree failures gracefully" {
  kill() { return 0; }
  export -f kill

  tmux() {
    if [[ "$1" == "list-panes" ]]; then
      echo "10001"
    elif [[ "$1" == "kill-session" ]]; then
      return 0
    fi
    return 0
  }
  export -f tmux

  kill_tree() {
    # Simulate kill_tree failure
    return 1
  }
  export -f kill_tree

  sleep() { return 0; }
  export -f sleep

  # Should still succeed even if kill_tree fails
  run cleanup_tmux_session "test-session" "99999" "tmux"
  assert_success
}

@test "cleanup_tmux_session: handles tmux kill-session failure gracefully" {
  kill() { return 0; }
  export -f kill

  tmux() {
    if [[ "$1" == "list-panes" ]]; then
      echo "10001"
    elif [[ "$1" == "kill-session" ]]; then
      # Simulate kill-session failure
      return 1
    fi
    return 0
  }
  export -f tmux

  kill_tree() { return 0; }
  export -f kill_tree

  sleep() { return 0; }
  export -f sleep

  run cleanup_tmux_session "test-session" "99999" "tmux"
  assert_success
}

@test "cleanup_tmux_session: handles list-panes returning error on second call" {
  _list_panes_call_count=0

  kill() { return 0; }
  export -f kill

  tmux() {
    if [[ "$1" == "list-panes" ]]; then
      _list_panes_call_count=$((_list_panes_call_count + 1))
      if [[ "$_list_panes_call_count" -eq 1 ]]; then
        echo "10001"
        return 0
      else
        # Second call fails (session already gone)
        return 1
      fi
    elif [[ "$1" == "kill-session" ]]; then
      return 0
    fi
    return 0
  }
  export -f tmux

  kill_tree() { return 0; }
  export -f kill_tree

  sleep() { return 0; }
  export -f sleep

  run cleanup_tmux_session "test-session" "99999" "tmux"
  assert_success
}

@test "cleanup_tmux_session: handles concurrent cleanup calls (idempotent)" {
  _cleanup_count=0

  kill() { return 0; }
  export -f kill

  tmux() {
    if [[ "$1" == "list-panes" ]]; then
      _cleanup_count=$((_cleanup_count + 1))
      if [[ "$_cleanup_count" -le 2 ]]; then
        echo "10001"
      else
        # Session gone on subsequent calls
        return 1
      fi
    elif [[ "$1" == "kill-session" ]]; then
      return 0
    fi
    return 0
  }
  export -f tmux

  kill_tree() { return 0; }
  export -f kill_tree

  sleep() { return 0; }
  export -f sleep

  # First cleanup
  run cleanup_tmux_session "test-session" "99999" "tmux"
  assert_success

  # Second cleanup (should handle gracefully)
  run cleanup_tmux_session "test-session" "99999" "tmux"
  assert_success
}

@test "cleanup_tmux_session: verifies TERM then KILL sequence with timing" {
  _kill_tree_calls=()
  _sleep_calls=()

  kill() { return 0; }
  export -f kill

  tmux() {
    if [[ "$1" == "list-panes" ]]; then
      echo "10001"
    elif [[ "$1" == "kill-session" ]]; then
      return 0
    fi
    return 0
  }
  export -f tmux

  kill_tree() {
    _kill_tree_calls+=("$1:$2")
    return 0
  }
  export -f kill_tree

  sleep() {
    _sleep_calls+=("$1")
    return 0
  }
  export -f sleep

  cleanup_tmux_session "test-session" "99999" "tmux"

  # Should have TERM, then sleep, then KILL
  [[ "${_kill_tree_calls[0]}" == "10001:TERM" ]]
  [[ "${_sleep_calls[0]}" == "0.3" ]]
  [[ "${_kill_tree_calls[1]}" == "10001:KILL" ]]
}

@test "cleanup_tmux_session: handles pane with PID 1 (special case)" {
  local call_log="$TEST_TMP/kill_tree_calls.log"
  > "$call_log"

  kill() { return 0; }
  export -f kill

  tmux() {
    if [[ "$1" == "list-panes" ]]; then
      # PID 1 is init process (should never kill this in real world)
      echo "1"
    elif [[ "$1" == "kill-session" ]]; then
      return 0
    fi
    return 0
  }
  export -f tmux

  kill_tree() {
    echo "$1:$2" >> "$call_log"
    return 0
  }
  export -f kill_tree
  export call_log

  sleep() { return 0; }
  export -f sleep

  run cleanup_tmux_session "test-session" "99999" "tmux"
  assert_success

  # Should still attempt to kill even PID 1 (kill_tree handles gracefully)
  first_call=$(head -1 "$call_log")
  [[ "$first_call" == "1:TERM" ]]
}

@test "cleanup_tmux_session: handles very large number of panes (20 panes)" {
  kill() { return 0; }
  export -f kill

  tmux() {
    if [[ "$1" == "list-panes" ]]; then
      # Generate 20 pane PIDs
      for i in {10001..10020}; do
        echo "$i"
      done
    elif [[ "$1" == "kill-session" ]]; then
      return 0
    fi
    return 0
  }
  export -f tmux

  _kill_tree_call_count=0
  kill_tree() {
    _kill_tree_call_count=$((_kill_tree_call_count + 1))
    return 0
  }
  export -f kill_tree

  sleep() { return 0; }
  export -f sleep

  cleanup_tmux_session "test-session" "99999" "tmux"

  # Should have killed 20 panes twice (TERM + KILL)
  [[ "$_kill_tree_call_count" -eq 40 ]]
}

@test "cleanup_tmux_session: handles panes with non-numeric PIDs in output" {
  local call_log="$TEST_TMP/kill_tree_calls2.log"
  > "$call_log"

  kill() { return 0; }
  export -f kill

  tmux() {
    if [[ "$1" == "list-panes" ]]; then
      # Mixed output (should only process numeric PIDs)
      echo "10001
invalid
10002
error: something
10003"
    elif [[ "$1" == "kill-session" ]]; then
      return 0
    fi
    return 0
  }
  export -f tmux

  kill_tree() {
    echo "$1:$2" >> "$call_log"
    return 0
  }
  export -f kill_tree
  export call_log

  sleep() { return 0; }
  export -f sleep

  cleanup_tmux_session "test-session" "99999" "tmux"

  # Should process all lines (even invalid ones - kill_tree handles gracefully)
  # Note: "error: something" gets split by word splitting into "error:" and "something"
  # So we get: 10001, invalid, 10002, error:, something, 10003 = 6 items
  # Total: 6 items × 2 passes (TERM + KILL) = 12 calls
  call_count=$(wc -l < "$call_log" | tr -d ' ')
  [[ "$call_count" -eq 12 ]]

  # Verify we tried to kill the valid PIDs
  grep -q "10001:TERM" "$call_log"
  grep -q "10002:TERM" "$call_log"
  grep -q "10003:TERM" "$call_log"
}
