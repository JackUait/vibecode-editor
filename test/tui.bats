setup() {
  load 'test_helper/common'
  _common_setup
  source "$PROJECT_ROOT/lib/tui.sh"
}

# --- success ---

@test "success: outputs checkmark and message" {
  run success "all good"
  assert_output --partial "✓"
  assert_output --partial "all good"
}

# --- warn ---

@test "warn: outputs exclamation and message" {
  run warn "be careful"
  assert_output --partial "!"
  assert_output --partial "be careful"
}

# --- error ---

@test "error: outputs cross and message" {
  run error "something broke"
  assert_output --partial "✗"
  assert_output --partial "something broke"
}

# --- info ---

@test "info: outputs arrow and message" {
  run info "starting now"
  assert_output --partial "→"
  assert_output --partial "starting now"
}

# --- header ---

@test "header: outputs the message" {
  run header "My Section"
  assert_output --partial "My Section"
}

# --- pad ---

@test "pad 5: outputs exactly 5 characters" {
  run pad 5
  [ "${#output}" -eq 5 ]
}

@test "pad 0: outputs empty string" {
  run pad 0
  [ "${#output}" -eq 0 ]
}

# --- moveto ---

@test "moveto 3 7: outputs correct escape sequence" {
  result="$(moveto 3 7)"
  [[ "$result" == $'\033[3;7H' ]]
}

# --- tui_init_interactive ---

@test "tui_init_interactive: sets _CYAN non-empty" {
  tui_init_interactive
  [ -n "$_CYAN" ]
}

@test "tui_init_interactive: sets _HIDE_CURSOR non-empty" {
  tui_init_interactive
  [ -n "$_HIDE_CURSOR" ]
}
