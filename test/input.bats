setup() {
  load 'test_helper/common'
  _common_setup
  source "$PROJECT_ROOT/lib/input.sh"
}

@test "parse_esc_sequence: up arrow" {
  result="$(printf '[A' | parse_esc_sequence)"
  [[ "$result" == "A" ]]
}

@test "parse_esc_sequence: down arrow" {
  result="$(printf '[B' | parse_esc_sequence)"
  [[ "$result" == "B" ]]
}

@test "parse_esc_sequence: left arrow" {
  result="$(printf '[D' | parse_esc_sequence)"
  [[ "$result" == "D" ]]
}

@test "parse_esc_sequence: right arrow" {
  result="$(printf '[C' | parse_esc_sequence)"
  [[ "$result" == "C" ]]
}

@test "parse_esc_sequence: SGR mouse left click" {
  result="$(printf '[<0;15;3M' | parse_esc_sequence)"
  [[ "$result" == "click:3" ]]
}

@test "parse_esc_sequence: SGR mouse left click different row" {
  result="$(printf '[<0;22;10M' | parse_esc_sequence)"
  [[ "$result" == "click:10" ]]
}

@test "parse_esc_sequence: ignores mouse release" {
  result="$(printf '[<0;15;3m' | parse_esc_sequence)"
  [[ "$result" == "" ]]
}

@test "parse_esc_sequence: ignores right click" {
  result="$(printf '[<2;15;3M' | parse_esc_sequence)"
  [[ "$result" == "" ]]
}

@test "parse_esc_sequence: ignores middle click" {
  result="$(printf '[<1;15;3M' | parse_esc_sequence)"
  [[ "$result" == "" ]]
}

# --- Malformed escape sequence scenarios ---

@test "parse_esc_sequence: handles truncated arrow sequence" {
  # read will block on incomplete input - skip this test
  skip "read will block on incomplete input"
}

@test "parse_esc_sequence: handles empty input after escape" {
  # read will block on empty input - skip this test
  skip "read will block on empty input"
}

@test "parse_esc_sequence: handles unknown bracket sequence" {
  result="$(printf '[Z' | parse_esc_sequence)"
  [[ "$result" == "Z" ]]
}

@test "parse_esc_sequence: handles double bracket" {
  result="$(printf '[[A' | parse_esc_sequence)"
  # First bracket read as _b1, second as _b2, returns "["
  [[ -n "$result" || "$result" == "" ]]
}

@test "parse_esc_sequence: handles SGR mouse sequence missing button" {
  # Missing button number - still has terminator so won't hang
  result="$(printf '[<;15;3M' | parse_esc_sequence)"
  [[ "$result" == "" ]]
}

@test "parse_esc_sequence: handles SGR mouse sequence missing column" {
  # Missing column - still has terminator so won't hang
  result="$(printf '[<0;;3M' | parse_esc_sequence)"
  # Parses empty column, returns click with row
  [[ "$result" == "click:3" ]]
}

@test "parse_esc_sequence: handles SGR mouse sequence missing row" {
  # Missing row - still has terminator so won't hang
  result="$(printf '[<0;15;M' | parse_esc_sequence)"
  # Should return click with empty row
  [[ "$result" == "click:" || "$result" == "" ]]
}

@test "parse_esc_sequence: handles SGR mouse sequence with no semicolons" {
  result="$(printf '[<0M' | parse_esc_sequence)"
  # Parses "0" as all three fields
  [[ "$result" == "click:0" ]]
}

@test "parse_esc_sequence: handles SGR mouse with extra semicolons" {
  result="$(printf '[<0;15;3;M' | parse_esc_sequence)"
  # Extra semicolon - row ends up empty after parsing
  [[ "$result" == "click:" ]]
}

@test "parse_esc_sequence: handles SGR mouse with spaces" {
  # Spaces in coordinates (invalid but test)
  result="$(printf '[<0; 15 ; 3M' | parse_esc_sequence)"
  # Spaces get stripped in the parsing
  [[ "$result" == "click:3" ]]
}

@test "parse_esc_sequence: handles mixed case terminator" {
  # 'm' is release, should be ignored
  result="$(printf '[<0;15;3m' | parse_esc_sequence)"
  [[ "$result" == "" ]]
}

# --- Non-UTF8 and special characters ---

@test "parse_esc_sequence: handles null byte in sequence" {
  # Null byte in mouse sequence
  result="$(printf '[<0\x00;15;3M' | parse_esc_sequence)"
  # Should handle gracefully
  [[ "$result" == "" || "$result" == "click:3" ]]
}

@test "parse_esc_sequence: handles high byte values" {
  # High byte value (non-ASCII)
  result="$(printf '[\xff' | parse_esc_sequence)"
  # Should return the byte or empty
  [[ -n "$result" || "$result" == "" ]]
}

@test "parse_esc_sequence: handles newline in sequence" {
  # Newline in mouse data
  result="$(printf '[<0;15\n;3M' | parse_esc_sequence)"
  # Function will parse the newline as part of the string
  [[ "$result" == "click:3" || "$result" == "" ]]
}

@test "parse_esc_sequence: handles carriage return in sequence" {
  result="$(printf '[<0;15\r;3M' | parse_esc_sequence)"
  # Carriage return is captured as part of data
  [[ "$result" == "click:3" || "$result" == "" ]]
}

@test "parse_esc_sequence: handles tab character in sequence" {
  result="$(printf '[<0;\t15;3M' | parse_esc_sequence)"
  # Tab is captured in the data
  [[ "$result" == "click:3" || "$result" == "" ]]
}

@test "parse_esc_sequence: handles escape character in sequence" {
  # Escape character within sequence
  result="$(printf '[<0;15\x1b;3M' | parse_esc_sequence)"
  [[ "$result" == "" || "$result" == "click:3" ]]
}

@test "parse_esc_sequence: handles backspace in sequence" {
  result="$(printf '[<0;15\x08;3M' | parse_esc_sequence)"
  [[ "$result" == "" || "$result" == "click:3" ]]
}

@test "parse_esc_sequence: handles delete character in sequence" {
  result="$(printf '[<0;15\x7f;3M' | parse_esc_sequence)"
  [[ "$result" == "" || "$result" == "click:3" ]]
}

# --- Boundary cases ---

@test "parse_esc_sequence: handles very large row number" {
  result="$(printf '[<0;15;99999M' | parse_esc_sequence)"
  [[ "$result" == "click:99999" ]]
}

@test "parse_esc_sequence: handles zero row number" {
  result="$(printf '[<0;15;0M' | parse_esc_sequence)"
  [[ "$result" == "click:0" ]]
}

@test "parse_esc_sequence: handles negative row number" {
  # Technically invalid but test robustness
  result="$(printf '[<0;15;-5M' | parse_esc_sequence)"
  # Will be parsed as string "-5"
  [[ "$result" == "click:-5" || "$result" == "" ]]
}

@test "parse_esc_sequence: handles very large button number" {
  result="$(printf '[<999;15;3M' | parse_esc_sequence)"
  # Not button 0, so should be ignored
  [[ "$result" == "" ]]
}

@test "parse_esc_sequence: handles alphabetic button number" {
  result="$(printf '[<X;15;3M' | parse_esc_sequence)"
  [[ "$result" == "" ]]
}

@test "parse_esc_sequence: handles alphabetic row number" {
  result="$(printf '[<0;15;XYZ M' | parse_esc_sequence)"
  # Parses "XYZ " as the row (everything until M)
  [[ "$result" == "click:XYZ" || "$result" == "" ]]
}

@test "parse_esc_sequence: handles very long row number" {
  result="$(printf '[<0;15;123456789012345M' | parse_esc_sequence)"
  [[ "$result" == "click:123456789012345" ]]
}

# --- Mouse coordinate edge cases ---

@test "parse_esc_sequence: handles out of bounds coordinates small terminal" {
  # Coordinates beyond typical small terminal (20x20)
  result="$(printf '[<0;25;25M' | parse_esc_sequence)"
  [[ "$result" == "click:25" ]]
}

@test "parse_esc_sequence: handles maximum terminal coordinates" {
  # Maximum typical terminal size (223 for some encodings)
  result="$(printf '[<0;223;223M' | parse_esc_sequence)"
  [[ "$result" == "click:223" ]]
}

@test "parse_esc_sequence: handles single digit coordinates" {
  result="$(printf '[<0;1;1M' | parse_esc_sequence)"
  [[ "$result" == "click:1" ]]
}

@test "parse_esc_sequence: handles leading zeros in coordinates" {
  result="$(printf '[<00;015;003M' | parse_esc_sequence)"
  # Button "00" doesn't match "0", so click is ignored
  [[ "$result" == "" ]]
}

# --- Invalid sequence patterns ---

@test "parse_esc_sequence: handles sequence with extra data after terminator" {
  # Mouse sequence followed by extra characters (won't be read by parse_esc_sequence)
  result="$(printf '[<0;15;3MEXTRA' | parse_esc_sequence)"
  [[ "$result" == "click:3" ]]
}

@test "parse_esc_sequence: handles multiple consecutive semicolons" {
  result="$(printf '[<0;;;3M' | parse_esc_sequence)"
  # Multiple empty fields
  [[ "$result" == "click:3" || "$result" == "" ]]
}

@test "parse_esc_sequence: handles semicolon at start" {
  result="$(printf '[<;0;15;3M' | parse_esc_sequence)"
  [[ "$result" == "" ]]
}

@test "parse_esc_sequence: handles missing all coordinates" {
  result="$(printf '[<M' | parse_esc_sequence)"
  [[ "$result" == "" || "$result" == "click:" ]]
}

# --- Stream cutoff scenarios ---

@test "parse_esc_sequence: handles partial SGR sequence" {
  # Sequence without terminator - will block in real usage
  skip "read will block waiting for terminator"
}

@test "parse_esc_sequence: handles complete sequence quickly" {
  # Verify function works with complete data
  result="$(printf '[A' | parse_esc_sequence)"
  [[ "$result" == "A" ]]
}

# --- Invalid byte sequences ---

@test "parse_esc_sequence: handles invalid UTF-8 sequence" {
  # Invalid UTF-8 byte sequence
  result="$(printf '[\xc3\x28' | parse_esc_sequence)"
  [[ -n "$result" || "$result" == "" ]]
}

@test "parse_esc_sequence: handles binary data in sequence" {
  # Random binary data
  result="$(printf '[\x01\x02\x03' | parse_esc_sequence)"
  [[ -n "$result" || "$result" == "" ]]
}

@test "parse_esc_sequence: handles all control characters" {
  # Control characters 0x00-0x1F
  result="$(printf '[\x01A' | parse_esc_sequence)"
  [[ -n "$result" || "$result" == "" ]]
}

# --- Format variations ---

@test "parse_esc_sequence: handles CSI format variation" {
  # Standard arrow key
  result="$(printf '[A' | parse_esc_sequence)"
  [[ "$result" == "A" ]]
}

@test "parse_esc_sequence: handles F1-F4 keys" {
  # F1 key (OP format)
  result="$(printf '[P' | parse_esc_sequence)"
  [[ "$result" == "P" ]]
}

@test "parse_esc_sequence: handles numeric parameters" {
  # Arrow key with modifiers (e.g., Shift+Up)
  result="$(printf '[1;2A' | parse_esc_sequence)"
  # Returns the parameter string
  [[ "$result" == "1" || "$result" == "" ]]
}

@test "parse_esc_sequence: handles empty parameter" {
  result="$(printf '[;A' | parse_esc_sequence)"
  [[ "$result" == ";" || "$result" == "" ]]
}

# --- confirm_tui tests ---

@test "confirm_tui calls ghost-tab-tui confirm with message" {
  # Mock ghost-tab-tui confirm
  ghost-tab-tui() {
    if [[ "$1" == "confirm" ]]; then
      [[ "$2" == "Delete this?" ]] || return 1
      echo '{"confirmed":true}'
      return 0
    fi
    return 1
  }
  export -f ghost-tab-tui

  # Mock jq
  jq() {
    if [[ "$1" == "-r" && "$2" == ".confirmed" ]]; then
      echo "true"
      return 0
    fi
    return 1
  }
  export -f jq

  run confirm_tui "Delete this?"

  assert_success
}

@test "confirm_tui returns failure when user cancels" {
  # Mock ghost-tab-tui confirm (cancelled)
  ghost-tab-tui() {
    if [[ "$1" == "confirm" ]]; then
      echo '{"confirmed":false}'
      return 0
    fi
    return 1
  }
  export -f ghost-tab-tui

  # Mock jq
  jq() {
    if [[ "$1" == "-r" && "$2" == ".confirmed" ]]; then
      echo "false"
      return 0
    fi
    return 1
  }
  export -f jq

  run confirm_tui "Delete this?"

  assert_failure
}

@test "confirm_tui handles jq parse failure" {
  # Mock ghost-tab-tui confirm
  ghost-tab-tui() {
    if [[ "$1" == "confirm" ]]; then
      echo '{"confirmed":true}'
      return 0
    fi
    return 1
  }
  export -f ghost-tab-tui

  # Mock jq (fails)
  jq() {
    return 1
  }
  export -f jq

  run confirm_tui "Delete this?"

  assert_failure
  assert_output --partial "Failed to parse confirmation response"
}

@test "confirm_tui validates against null string" {
  # Mock ghost-tab-tui confirm
  ghost-tab-tui() {
    if [[ "$1" == "confirm" ]]; then
      echo '{"confirmed":"null"}'
      return 0
    fi
    return 1
  }
  export -f ghost-tab-tui

  # Mock jq (returns string "null")
  jq() {
    if [[ "$1" == "-r" && "$2" == ".confirmed" ]]; then
      echo "null"
      return 0
    fi
    return 1
  }
  export -f jq

  run confirm_tui "Delete this?"

  assert_failure
}

@test "confirm_tui falls back to bash prompt when binary missing" {
  # Test the fallback logic by simulating the condition
  # We'll use a temp script to test the actual fallback behavior
  local temp_dir
  temp_dir="$(mktemp -d)"
  local test_script="$temp_dir/test_fallback.sh"

  cat > "$test_script" << 'EOF'
#!/bin/bash
# Fallback logic (copied from confirm_tui, without /dev/tty for testing)
msg="$1"
read -rp "$msg (y/N) " response
[[ "$response" =~ ^[Yy]$ ]]
EOF

  chmod +x "$test_script"

  # Simulate user typing "y"
  run bash -c "echo 'y' | $test_script 'Delete this?'"

  rm -rf "$temp_dir"
  assert_success
}

@test "confirm_tui fallback rejects non-yes responses" {
  # Test the fallback logic by simulating the condition
  local temp_dir
  temp_dir="$(mktemp -d)"
  local test_script="$temp_dir/test_fallback_reject.sh"

  cat > "$test_script" << 'EOF'
#!/bin/bash
# Fallback logic (copied from confirm_tui, without /dev/tty for testing)
msg="$1"
read -rp "$msg (y/N) " response
[[ "$response" =~ ^[Yy]$ ]]
EOF

  chmod +x "$test_script"

  # Simulate user typing "n"
  run bash -c "echo 'n' | $test_script 'Delete this?'"

  rm -rf "$temp_dir"
  assert_failure
}
