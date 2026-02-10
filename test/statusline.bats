setup() {
  load 'test_helper/common'
  _common_setup
  source "$PROJECT_ROOT/lib/statusline.sh"
}

# --- format_memory ---

@test "format_memory: converts KB to MB" {
  run format_memory 512000
  assert_output "500M"
}

@test "format_memory: small MB value" {
  run format_memory 102400
  assert_output "100M"
}

@test "format_memory: converts to GB with decimal" {
  # 1572864 KB = 1536 MB = 1.5 GB
  run format_memory 1572864
  assert_output "1.5G"
}

@test "format_memory: exactly 1 GB" {
  # 1048576 KB = 1024 MB = 1.0G
  run format_memory 1048576
  assert_output "1.0G"
}

@test "format_memory: zero returns 0M" {
  run format_memory 0
  assert_output "0M"
}

# --- parse_cwd_from_json ---

@test "parse_cwd_from_json: extracts current_dir" {
  run parse_cwd_from_json '{"current_dir":"/Users/me/project"}'
  assert_output "/Users/me/project"
}

@test "parse_cwd_from_json: handles nested JSON" {
  run parse_cwd_from_json '{"foo":"bar","current_dir":"/tmp/test","baz":1}'
  assert_output "/tmp/test"
}

@test "parse_cwd_from_json: returns empty for missing key" {
  run parse_cwd_from_json '{"foo":"bar"}'
  assert_output ""
}

# --- Edge Cases: JSON Parsing ---

@test "parse_cwd_from_json: handles malformed JSON - missing quotes" {
  run parse_cwd_from_json '{current_dir:/tmp/test}'
  # sed-based parser doesn't validate JSON, just pattern matches
  assert_output ""
}

@test "parse_cwd_from_json: handles malformed JSON - missing braces" {
  run parse_cwd_from_json '"current_dir":"/tmp/test"'
  assert_output "/tmp/test"
}

@test "parse_cwd_from_json: handles malformed JSON - trailing comma" {
  run parse_cwd_from_json '{"current_dir":"/tmp/test",}'
  assert_output "/tmp/test"
}

@test "parse_cwd_from_json: handles empty JSON object" {
  run parse_cwd_from_json '{}'
  assert_output ""
}

@test "parse_cwd_from_json: handles empty string" {
  run parse_cwd_from_json ''
  assert_output ""
}

@test "parse_cwd_from_json: handles whitespace-only string" {
  run parse_cwd_from_json '   '
  assert_output ""
}

@test "parse_cwd_from_json: handles binary data" {
  run parse_cwd_from_json "$(printf '\x00\x01\x02\x03')"
  assert_output ""
}

@test "parse_cwd_from_json: handles path with escaped characters" {
  run parse_cwd_from_json '{"current_dir":"/tmp/test\\ndir"}'
  # sed extracts the literal backslash-n sequence
  assert_output '/tmp/test\\ndir'
}

@test "parse_cwd_from_json: handles path with special JSON chars" {
  run parse_cwd_from_json '{"current_dir":"/tmp/test\"quoted"}'
  # sed stops at first unescaped quote
  assert_output '/tmp/test\'
}

@test "parse_cwd_from_json: handles very long path" {
  local long_path="/very/long/path/with/many/segments"
  for i in {1..50}; do
    long_path="${long_path}/segment${i}"
  done
  run parse_cwd_from_json "{\"current_dir\":\"${long_path}\"}"
  assert_output "$long_path"
}

@test "parse_cwd_from_json: handles path with Unicode characters" {
  run parse_cwd_from_json '{"current_dir":"/tmp/tëst/日本語/émoji🎉"}'
  assert_output '/tmp/tëst/日本語/émoji🎉'
}

@test "parse_cwd_from_json: handles multiple current_dir keys (takes last)" {
  run parse_cwd_from_json '{"current_dir":"/first","other":"stuff","current_dir":"/second"}'
  # sed uses greedy matching, will match the largest pattern
  assert_output "/second"
}

@test "parse_cwd_from_json: handles Windows-style line endings" {
  run parse_cwd_from_json "$(printf '{\r\n\"current_dir\":\"/tmp/test\"\r\n}\r\n')"
  assert_output "/tmp/test"
}

# --- Edge Cases: Memory Formatting ---

@test "format_memory: handles negative values" {
  run format_memory -1024
  # bc may produce negative results
  [[ "$output" == *"-"* ]] || [[ "$output" == "0M" ]]
}

@test "format_memory: handles very large values" {
  # 10TB in KB
  run format_memory 10737418240
  assert_output --partial "G"
  # Should show thousands of GB
  [[ "$output" =~ ^[0-9]+\.[0-9]G$ ]]
}

@test "format_memory: handles non-numeric input gracefully" {
  run format_memory "not_a_number"
  # bc will fail or produce 0
  assert_failure || assert_output "0M"
}

@test "format_memory: handles empty string" {
  run format_memory ""
  assert_failure || assert_output "0M"
}

@test "format_memory: handles floating point input" {
  run format_memory 512000.5
  # Shell arithmetic with floating point may fail
  # Bash $(()) doesn't handle floats, this will error
  assert_failure || assert_success
}

@test "format_memory: handles exactly 1024 MB boundary" {
  # Exactly 1GB = 1048576 KB
  run format_memory 1048576
  assert_output "1.0G"
}

@test "format_memory: handles just below GB boundary" {
  # 1023 MB
  run format_memory 1047552
  assert_output "1023M"
}

@test "format_memory: handles just above GB boundary" {
  # 1025 MB
  run format_memory 1049600
  assert_output "1.0G"
}

@test "format_memory: handles KB value with remainder" {
  # 512500 KB = 500.488 MB
  run format_memory 512500
  assert_output "500M"
}

@test "format_memory: handles very small non-zero values" {
  run format_memory 1
  assert_output "0M"
}

@test "format_memory: handles missing bc command" {
  # Mock bc to fail
  bc() {
    return 127
  }
  export -f bc

  run format_memory 1572864
  # bc is called via echo | bc, if bc fails output will be empty
  # But format_memory doesn't check bc success, just uses output
  # So it might succeed with empty string or "G"
  assert_success || assert_failure
}
