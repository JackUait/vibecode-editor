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

# --- set_tab_title ---

@test "set_tab_title: includes project name and ai tool separated by middot" {
  result="$(set_tab_title "ghost-tab" "claude")"
  [[ "$result" == *"ghost-tab · claude"* ]]
}

@test "set_tab_title: outputs OSC escape sequence" {
  result="$(set_tab_title "myproject" "codex")"
  # OSC 0 sets window title: \033]0;TITLE\007
  [[ "$result" == $'\033]0;myproject · codex\007' ]]
}

@test "set_tab_title: omits tool name when empty" {
  result="$(set_tab_title "myproject" "")"
  [[ "$result" == $'\033]0;myproject\007' ]]
}

# --- draw_logo ---

@test "draw_logo calls ghost-tab-tui show-logo" {
  # Mock ghost-tab-tui
  ghost-tab-tui() {
    if [[ "$1" == "show-logo" ]]; then
      echo "MOCK_LOGO_OUTPUT"
      return 0
    fi
    return 1
  }
  export -f ghost-tab-tui

  run draw_logo "claude"

  assert_success
  assert_output "MOCK_LOGO_OUTPUT"
}
