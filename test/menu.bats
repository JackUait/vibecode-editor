setup() {
  load 'test_helper/common'
  _common_setup
  source "$PROJECT_ROOT/lib/tui.sh"
  source "$PROJECT_ROOT/lib/ai-tools.sh"
  source "$PROJECT_ROOT/lib/menu.sh"
}

# --- draw_menu: function exists ---

@test "draw_menu: function is defined" {
  declare -f draw_menu >/dev/null
}

@test "draw_menu: function is callable (type -t)" {
  [ "$(type -t draw_menu)" = "function" ]
}
