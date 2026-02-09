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
