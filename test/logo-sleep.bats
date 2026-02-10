#!/usr/bin/env bats

load test_helper/bats-support/load
load test_helper/bats-assert/load

setup() {
  source "$BATS_TEST_DIRNAME/../lib/logo-animation.sh"
}

@test "logo_art_claude_sleeping function exists" {
  run type logo_art_claude_sleeping
  assert_success
}

@test "logo_art_claude_sleeping sets required variables" {
  logo_art_claude_sleeping

  assert [ "${#_LOGO_LINES[@]}" -eq 15 ]
  assert [ "$_LOGO_HEIGHT" -eq 15 ]
  assert [ "$_LOGO_WIDTH" -eq 28 ]
}

@test "logo_art_claude_sleeping has closed eyes" {
  logo_art_claude_sleeping

  # Line 5 should have closed eyes (line 6 is solid body in sleeping variant)
  # Check that line 5 contains the closed eye pattern
  [[ "${_LOGO_LINES[5]}" =~ "▬▬▬▬" ]]
}

@test "all sleeping ghost lines are exactly 28 visible characters" {
  logo_art_claude_sleeping

  for i in $(seq 0 14); do
    # Strip ANSI codes and count visible chars
    local line="${_LOGO_LINES[$i]}"
    # Remove all ANSI escape sequences
    line=$(echo -e "$line" | sed 's/\x1b\[[0-9;]*m//g')
    local len=${#line}
    assert [ "$len" -eq 28 ]
  done
}

@test "logo_art_codex_sleeping function exists and sets variables" {
  logo_art_codex_sleeping
  assert [ "${#_LOGO_LINES[@]}" -eq 15 ]
  assert [ "$_LOGO_HEIGHT" -eq 15 ]
  assert [ "$_LOGO_WIDTH" -eq 28 ]
}

@test "logo_art_copilot_sleeping function exists and sets variables" {
  logo_art_copilot_sleeping
  assert [ "${#_LOGO_LINES[@]}" -eq 15 ]
  assert [ "$_LOGO_HEIGHT" -eq 15 ]
  assert [ "$_LOGO_WIDTH" -eq 28 ]
}

@test "logo_art_opencode_sleeping function exists and sets variables" {
  logo_art_opencode_sleeping
  assert [ "${#_LOGO_LINES[@]}" -eq 15 ]
  assert [ "$_LOGO_HEIGHT" -eq 15 ]
  assert [ "$_LOGO_WIDTH" -eq 28 ]
}

@test "draw_zzz function exists" {
  run type draw_zzz
  assert_success
}

@test "clear_zzz function exists" {
  run type clear_zzz
  assert_success
}

@test "draw_logo_sleeping function exists" {
  run type draw_logo_sleeping
  assert_success
}
