#!/bin/bash
export PATH="$HOME/.local/bin:/opt/homebrew/bin:/usr/local/bin:$PATH"

# Self-healing: Check if ghost-tab-tui exists, rebuild if missing
if ! command -v ghost-tab-tui &>/dev/null; then
  # Simple inline rebuild without TUI functions (not loaded yet)
  if command -v go &>/dev/null; then
    printf 'Rebuilding ghost-tab-tui...\n' >&2
    mkdir -p "$HOME/.local/bin"
    # Determine SHARE_DIR location
    # 1. Try wrapper script's actual location (for dev/symlink setups)
    _wrapper_real_path="$(cd "$(dirname "$(readlink "$0" || echo "$0")")" && pwd)"
    _candidate_share_dir="$(cd "$_wrapper_real_path/.." && pwd)"
    if [ -f "$_candidate_share_dir/cmd/ghost-tab-tui/main.go" ]; then
      SHARE_DIR="$_candidate_share_dir"
    # 2. Try standard install locations
    elif [ -f "/opt/homebrew/share/ghost-tab/cmd/ghost-tab-tui/main.go" ]; then
      SHARE_DIR="/opt/homebrew/share/ghost-tab"
    elif [ -f "/usr/local/share/ghost-tab/cmd/ghost-tab-tui/main.go" ]; then
      SHARE_DIR="/usr/local/share/ghost-tab"
    elif [ -f "$HOME/.local/share/ghost-tab/cmd/ghost-tab-tui/main.go" ]; then
      SHARE_DIR="$HOME/.local/share/ghost-tab"
    else
      printf '\033[31mError:\033[0m Cannot find ghost-tab source code\n' >&2
      printf 'Run \033[1mghost-tab\033[0m to reinstall.\n' >&2
      printf 'Press any key to exit...\n' >&2
      read -rsn1
      exit 1
    fi
    # Build from module root with relative path to cmd
    if (cd "$SHARE_DIR" && go build -o "$HOME/.local/bin/ghost-tab-tui" ./cmd/ghost-tab-tui) 2>/dev/null; then
      printf 'ghost-tab-tui rebuilt successfully\n' >&2
      export PATH="$HOME/.local/bin:$PATH"
    else
      printf '\033[31mError:\033[0m Failed to rebuild ghost-tab-tui\n' >&2
      printf 'Run \033[1mghost-tab\033[0m to reinstall.\n' >&2
      printf 'Press any key to exit...\n' >&2
      read -rsn1
      exit 1
    fi
  else
    printf '\033[31mError:\033[0m ghost-tab-tui binary not found and Go not installed\n' >&2
    printf 'Run \033[1mghost-tab\033[0m to reinstall.\n' >&2
    printf 'Press any key to exit...\n' >&2
    read -rsn1
    exit 1
  fi
fi

# Load shared library functions
_WRAPPER_DIR="$(cd "$(dirname "$0")" && pwd)"

if [ ! -d "$_WRAPPER_DIR/lib" ]; then
  printf '\033[31mError:\033[0m Ghost Tab libraries not found at %s/lib\n' "$_WRAPPER_DIR" >&2
  printf 'Run \033[1mghost-tab\033[0m to reinstall.\n' >&2
  printf 'Press any key to exit...\n' >&2
  read -rsn1
  exit 1
fi

_gt_libs=(ai-tools projects process input tui update menu-tui project-actions project-actions-tui tmux-session settings-menu-tui)
for _gt_lib in "${_gt_libs[@]}"; do
  if [ ! -f "$_WRAPPER_DIR/lib/${_gt_lib}.sh" ]; then
    printf '\033[31mError:\033[0m Missing library %s/lib/%s.sh\n' "$_WRAPPER_DIR" "$_gt_lib" >&2
    printf 'Run \033[1mghost-tab\033[0m to reinstall.\n' >&2
    printf 'Press any key to exit...\n' >&2
    read -rsn1
    exit 1
  fi
  # shellcheck disable=SC1090  # Dynamic module loading
  source "$_WRAPPER_DIR/lib/${_gt_lib}.sh"
done
unset _gt_libs _gt_lib

TMUX_CMD="$(command -v tmux)"
LAZYGIT_CMD="$(command -v lazygit)"
BROOT_CMD="$(command -v broot)"
CLAUDE_CMD="$(command -v claude)"
CODEX_CMD="$(command -v codex)"
COPILOT_CMD="$(command -v copilot)"
OPENCODE_CMD="$(command -v opencode)"

# AI tool preference
AI_TOOL_PREF_FILE="${XDG_CONFIG_HOME:-$HOME/.config}/ghost-tab/ai-tool"
AI_TOOLS_AVAILABLE=()
[ -n "$CLAUDE_CMD" ] && AI_TOOLS_AVAILABLE+=("claude")
[ -n "$CODEX_CMD" ] && AI_TOOLS_AVAILABLE+=("codex")
[ -n "$COPILOT_CMD" ] && AI_TOOLS_AVAILABLE+=("copilot")
[ -n "$OPENCODE_CMD" ] && AI_TOOLS_AVAILABLE+=("opencode")

# Read saved preference, default to first available
SELECTED_AI_TOOL=""
if [ -f "$AI_TOOL_PREF_FILE" ]; then
  SELECTED_AI_TOOL="$(cat "$AI_TOOL_PREF_FILE" 2>/dev/null | tr -d '[:space:]')"
fi
# Validate saved preference is still installed
validate_ai_tool

# Load user projects from config file if it exists
PROJECTS_FILE="${XDG_CONFIG_HOME:-$HOME/.config}/ghost-tab/projects"

# Version update check (Homebrew only)
# shellcheck disable=SC2034  # Used in sourced update.sh module
UPDATE_CACHE="${XDG_CONFIG_HOME:-$HOME/.config}/ghost-tab/.update-check"
_update_version=""

check_for_update

# Select working directory
if [ -n "$1" ] && [ -d "$1" ]; then
  cd "$1" || exit 1
  shift
elif [ -z "$1" ]; then
  # Use TUI for project selection
  printf '\033]0;👻 Ghost Tab\007'

  while true; do
    if select_project_interactive "$PROJECTS_FILE"; then
      # Update AI tool if user cycled it in the menu (for all actions)
      if [[ -n "${_selected_ai_tool:-}" ]]; then
        SELECTED_AI_TOOL="$_selected_ai_tool"
      fi
      # shellcheck disable=SC2154
      case "$_selected_project_action" in
        select-project)
          PROJECT_NAME="$_selected_project_name"
          # shellcheck disable=SC2154
          cd "$_selected_project_path" || exit 1
          break
          ;;
        add-project)
          if add_project_interactive; then
            # shellcheck disable=SC2154
            if validate_new_project "$_add_project_path" "$PROJECTS_FILE"; then
              add_project_to_file "$_add_project_name" "$_validated_path" "$PROJECTS_FILE"
            fi
          fi
          continue
          ;;
        delete-project)
          # Show project list for deletion — reuse select-project TUI
          # For now, loop back to main menu (delete handled in TUI in future)
          continue
          ;;
        open-once)
          # Prompt for path via /dev/tty
          printf 'Project path: ' >/dev/tty
          read -r open_path </dev/tty
          if [[ -n "$open_path" ]]; then
            open_path="${open_path/#\~/$HOME}"
            if [[ -d "$open_path" ]]; then
              cd "$open_path" || exit 1
              PROJECT_NAME="$(basename "$open_path")"
              break
            fi
          fi
          continue
          ;;
        plain-terminal)
          exec "$SHELL"
          ;;
        *)
          # settings or unknown — loop back to menu
          continue
          ;;
      esac
    else
      # User quit (ESC/Ctrl-C)
      exit 0
    fi
  done
fi

PROJECT_DIR="$(pwd)"
export PROJECT_DIR
export PROJECT_NAME="${PROJECT_NAME:-$(basename "$PROJECT_DIR")}"
SESSION_NAME="dev-${PROJECT_NAME}-$$"

# Set terminal/tab title based on tab_title setting
_tab_title_setting="full"
_settings_file="${XDG_CONFIG_HOME:-$HOME/.config}/ghost-tab/settings"
if [ -f "$_settings_file" ]; then
  _saved_tab_title=$(grep '^tab_title=' "$_settings_file" 2>/dev/null | cut -d= -f2)
  if [ -n "$_saved_tab_title" ]; then
    _tab_title_setting="$_saved_tab_title"
  fi
fi
if [ "$_tab_title_setting" = "full" ]; then
  set_tab_title "$PROJECT_NAME" "$SELECTED_AI_TOOL"
else
  set_tab_title "$PROJECT_NAME"
fi

# Background watcher: switch to Claude pane once it's ready
(
  while true; do
    sleep 0.5
    content=$("$TMUX_CMD" capture-pane -t "$SESSION_NAME:0.1" -p 2>/dev/null)
    # All three tools show a prompt character when ready
    if echo "$content" | grep -qE '[>$❯]'; then
      "$TMUX_CMD" select-pane -t "$SESSION_NAME:0.1"
      break
    fi
  done
) &
WATCHER_PID=$!

cleanup() {
  cleanup_tmux_session "$SESSION_NAME" "$WATCHER_PID" "$TMUX_CMD"
}
trap cleanup EXIT HUP TERM INT

# Build the AI tool launch command
case "$SELECTED_AI_TOOL" in
  codex|opencode)
    AI_LAUNCH_CMD="$(build_ai_launch_cmd "$SELECTED_AI_TOOL" "$CLAUDE_CMD" "$CODEX_CMD" "$COPILOT_CMD" "$OPENCODE_CMD" "$PROJECT_DIR")"
    ;;
  *)
    AI_LAUNCH_CMD="$(build_ai_launch_cmd "$SELECTED_AI_TOOL" "$CLAUDE_CMD" "$CODEX_CMD" "$COPILOT_CMD" "$OPENCODE_CMD" "$*")"
    ;;
esac

"$TMUX_CMD" new-session -s "$SESSION_NAME" -e "PATH=$PATH" -c "$PROJECT_DIR" \
  "$LAZYGIT_CMD; exec bash" \; \
  set-option status-left " ⬡ ${PROJECT_NAME} " \; \
  set-option status-left-style "fg=white,bg=colour236,bold" \; \
  set-option status-style "bg=colour235" \; \
  set-option status-right "" \; \
  set-option exit-unattached on \; \
  split-window -h -p 50 -c "$PROJECT_DIR" \
  "$AI_LAUNCH_CMD; exec bash" \; \
  select-pane -t 0 \; \
  split-window -v -p 50 -c "$PROJECT_DIR" \
  "trap exit TERM; while true; do $BROOT_CMD $PROJECT_DIR; done" \; \
  split-window -v -p 30 -c "$PROJECT_DIR" \; \
  select-pane -t 3
