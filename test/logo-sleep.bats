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

  # Lines 5 and 6 should have closed eyes (no white 255 color in eyes area)
  # Check that line 5 contains the closed eye pattern
  [[ "${_LOGO_LINES[5]}" =~ "▬▬▬▬" ]]
}
