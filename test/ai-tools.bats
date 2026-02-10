setup() {
  load 'test_helper/common'
  _common_setup
  source "$PROJECT_ROOT/lib/ai-tools.sh"
}

# --- ai_tool_display_name ---

@test "ai_tool_display_name: claude -> Claude Code" {
  run ai_tool_display_name "claude"
  assert_output "Claude Code"
}

@test "ai_tool_display_name: codex -> Codex CLI" {
  run ai_tool_display_name "codex"
  assert_output "Codex CLI"
}

@test "ai_tool_display_name: copilot -> Copilot CLI" {
  run ai_tool_display_name "copilot"
  assert_output "Copilot CLI"
}

@test "ai_tool_display_name: opencode -> OpenCode" {
  run ai_tool_display_name "opencode"
  assert_output "OpenCode"
}

@test "ai_tool_display_name: unknown passes through" {
  run ai_tool_display_name "vim"
  assert_output "vim"
}

# --- ai_tool_color ---

@test "ai_tool_color: claude returns orange ANSI" {
  result="$(ai_tool_color "claude")"
  [[ "$result" == $'\033[38;5;209m' ]]
}

@test "ai_tool_color: codex returns green ANSI" {
  result="$(ai_tool_color "codex")"
  [[ "$result" == $'\033[38;5;114m' ]]
}

@test "ai_tool_color: copilot returns purple ANSI" {
  result="$(ai_tool_color "copilot")"
  [[ "$result" == $'\033[38;5;141m' ]]
}

@test "ai_tool_color: opencode returns light gray ANSI" {
  result="$(ai_tool_color "opencode")"
  [[ "$result" == $'\033[38;5;250m' ]]
}

@test "ai_tool_color: unknown returns default cyan" {
  result="$(ai_tool_color "vim")"
  [[ "$result" == $'\033[0;36m' ]]
}

# --- ai_tool_bright_color ---

@test "ai_tool_bright_color: claude returns orange ANSI" {
  result="$(ai_tool_bright_color "claude")"
  [[ "$result" == $'\033[38;5;209m' ]]
}

@test "ai_tool_bright_color: codex returns green ANSI" {
  result="$(ai_tool_bright_color "codex")"
  [[ "$result" == $'\033[38;5;114m' ]]
}

@test "ai_tool_bright_color: copilot returns purple ANSI" {
  result="$(ai_tool_bright_color "copilot")"
  [[ "$result" == $'\033[38;5;141m' ]]
}

@test "ai_tool_bright_color: opencode returns bold white ANSI" {
  result="$(ai_tool_bright_color "opencode")"
  [[ "$result" == $'\033[1;38;5;255m' ]]
}

@test "ai_tool_bright_color: unknown returns bold white" {
  result="$(ai_tool_bright_color "vim")"
  [[ "$result" == $'\033[1;37m' ]]
}

# --- cycle_ai_tool ---

@test "cycle_ai_tool: next wraps from last to first" {
  AI_TOOLS_AVAILABLE=("claude" "codex" "opencode")
  SELECTED_AI_TOOL="opencode"
  cycle_ai_tool "next"
  [[ "$SELECTED_AI_TOOL" == "claude" ]]
}

@test "cycle_ai_tool: next advances by one" {
  AI_TOOLS_AVAILABLE=("claude" "codex" "opencode")
  SELECTED_AI_TOOL="claude"
  cycle_ai_tool "next"
  [[ "$SELECTED_AI_TOOL" == "codex" ]]
}

@test "cycle_ai_tool: prev wraps from first to last" {
  AI_TOOLS_AVAILABLE=("claude" "codex" "opencode")
  SELECTED_AI_TOOL="claude"
  cycle_ai_tool "prev"
  [[ "$SELECTED_AI_TOOL" == "opencode" ]]
}

@test "cycle_ai_tool: no-op with single tool" {
  AI_TOOLS_AVAILABLE=("claude")
  SELECTED_AI_TOOL="claude"
  cycle_ai_tool "next"
  [[ "$SELECTED_AI_TOOL" == "claude" ]]
}

# --- validate_ai_tool ---

@test "validate_ai_tool: keeps valid preference" {
  AI_TOOLS_AVAILABLE=("claude" "codex")
  SELECTED_AI_TOOL="codex"
  validate_ai_tool
  [[ "$SELECTED_AI_TOOL" == "codex" ]]
}

@test "validate_ai_tool: falls back to first when pref is invalid" {
  AI_TOOLS_AVAILABLE=("claude" "codex")
  SELECTED_AI_TOOL="vim"
  validate_ai_tool
  [[ "$SELECTED_AI_TOOL" == "claude" ]]
}

@test "validate_ai_tool: falls back to first when pref is empty" {
  AI_TOOLS_AVAILABLE=("codex" "opencode")
  SELECTED_AI_TOOL=""
  validate_ai_tool
  [[ "$SELECTED_AI_TOOL" == "codex" ]]
}
