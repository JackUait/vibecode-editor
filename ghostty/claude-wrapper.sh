#!/bin/bash
export PATH="$HOME/.local/bin:/opt/homebrew/bin:/usr/local/bin:$PATH"

# Load shared library functions
_WRAPPER_DIR="$(cd "$(dirname "$0")" && pwd)"
source "$_WRAPPER_DIR/lib/ai-tools.sh"
source "$_WRAPPER_DIR/lib/projects.sh"
source "$_WRAPPER_DIR/lib/process.sh"
source "$_WRAPPER_DIR/lib/input.sh"
source "$_WRAPPER_DIR/lib/tui.sh"
source "$_WRAPPER_DIR/lib/update.sh"
source "$_WRAPPER_DIR/lib/menu.sh"
source "$_WRAPPER_DIR/lib/autocomplete.sh"
source "$_WRAPPER_DIR/lib/project-actions.sh"
source "$_WRAPPER_DIR/lib/tmux-session.sh"

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
UPDATE_CACHE="${XDG_CONFIG_HOME:-$HOME/.config}/ghost-tab/.update-check"
_update_version=""

check_for_update

# Select working directory
if [ -n "$1" ] && [ -d "$1" ]; then
  cd "$1"
  shift
elif [ -z "$1" ]; then
  tui_init_interactive

  # Restore cursor and disable mouse on exit
  trap 'printf "${_SHOW_CURSOR}${_MOUSE_OFF}"; printf "\\033[?7h"' EXIT

  # Wait for terminal to fully initialize and report correct size
  sleep 0.1

  # Set terminal title for project selection screen
  printf '\033]0;👻 Ghost Tab\007'

  while true; do
    # Reload projects each iteration
    projects=()
    if [ -f "$PROJECTS_FILE" ]; then
      while IFS= read -r line; do
        [[ -z "$line" || "$line" == \#* ]] && continue
        projects+=("$line")
      done < "$PROJECTS_FILE"
    fi

    # Build menu items
    menu_labels=()
    menu_subs=()
    menu_types=()
    menu_hi=()
    for i in "${!projects[@]}"; do
      name="${projects[$i]%%:*}"
      dir="${projects[$i]#*:}"
      display_dir="${dir/#$HOME/~}"
      menu_labels+=("$name")
      menu_subs+=("$display_dir")
      menu_types+=("project")
      menu_hi+=("${_INVERSE}")
    done
    menu_labels+=("Add new project" "Delete a project" "Open once" "Plain terminal")
    menu_subs+=("" "" "" "")
    menu_types+=("add" "delete" "open_once" "plain")
    menu_hi+=("${_BG_BLUE}${_WHITE}" "${_BG_RED}${_WHITE}" "${_INVERSE}" "${_DIM}")

    _action_hints=("A" "D" "O" "P")
    total=${#menu_labels[@]}
    selected=0
    box_w=44

    _add_mode=0
    _add_input=""
    _add_msg=""
    _del_mode=0
    _open_mode=0

    printf "${_HIDE_CURSOR}${_MOUSE_ON}"
    printf '\033[2J\033[H'
    draw_menu

    # Input loop
    while true; do
      if [ "$_add_mode" -eq 1 ]; then
        # Add mode: read path with autocomplete
        _sub_row=$(( _top_row + 3 + _update_line + _add_idx * 2 + _sep_count + 1 ))
        read_path_autocomplete "$_sub_row" "$_left_col"
        _add_input="$_path_result"

        if [ -z "$_add_input" ]; then
          # Empty = cancel
          _add_mode=0
          menu_labels[$_add_idx]="Add new project"
          menu_subs[$_add_idx]=""
          printf "${_HIDE_CURSOR}"
          draw_menu
          continue
        fi

        if ! validate_new_project "$_add_input" "$PROJECTS_FILE"; then
          _add_mode=0
          menu_labels[$_add_idx]="Add new project"
          menu_subs[$_add_idx]=""
          draw_menu
          moveto "$_sub_row" "$_left_col"
          printf "    ${_YELLOW}!${_NC} Project ${_BOLD}${_validated_name}${_NC} already exists\033[K"
          sleep 1
          printf "${_HIDE_CURSOR}"
          draw_menu
          continue
        fi

        add_project_to_file "$_validated_name" "$_validated_path" "$PROJECTS_FILE"
        _add_mode=0
        menu_labels[$_add_idx]="Add new project"
        menu_subs[$_add_idx]=""
        draw_menu
        moveto "$_sub_row" "$_left_col"
        printf "    ${_GREEN}✓${_NC} Added ${_BOLD}${_validated_name}${_NC}\033[K"
        sleep 0.8
        printf "${_HIDE_CURSOR}"
        # Redraw with new project list
        break
      fi

      if [ "$_open_mode" -eq 1 ]; then
        # Open once mode: read path with autocomplete
        _sub_row=$(( _top_row + 3 + _update_line + _open_idx * 2 + _sep_count + 1 ))
        read_path_autocomplete "$_sub_row" "$_left_col"
        _open_input="$_path_result"

        if [ -z "$_open_input" ]; then
          # Empty = cancel
          _open_mode=0
          menu_labels[$_open_idx]="Open once"
          menu_subs[$_open_idx]=""
          printf "${_HIDE_CURSOR}${_MOUSE_ON}"
          draw_menu
          continue
        fi

        expanded="${_open_input/#\~/$HOME}"
        if [ -d "$expanded" ]; then
          printf "${_SHOW_CURSOR}${_MOUSE_OFF}"
          printf '\033[2J\033[H'
          cd "$expanded"
          break 2
        else
          _open_mode=0
          menu_labels[$_open_idx]="Open once"
          menu_subs[$_open_idx]=""
          draw_menu
          moveto "$_sub_row" "$_left_col"
          printf "    ${_YELLOW}!${_NC} Directory not found\033[K"
          sleep 0.8
          printf "${_HIDE_CURSOR}${_MOUSE_ON}"
          draw_menu
          continue
        fi
      fi

      if [ "$_del_mode" -eq 1 ]; then
        # In delete mode: arrow-navigate or number-select project to delete
        _del_sub_row=$(( _top_row + 3 + _update_line + _del_idx * 2 + _sep_count + 1 ))
        # Render current selection on subtitle line
        _dn="${projects[$_del_sel]%%:*}"
        moveto "$_del_sub_row" "$_left_col"
        printf "    ${_BG_RED}${_WHITE}${_BOLD} %d) %s ${_NC}  ${_DIM}↑↓ navigate  1-9 jump  ⏎ delete  q cancel${_NC}\033[K" "$((_del_sel+1))" "$_dn"

        read -rsn1 key
        if [[ "$key" == $'\x1b' ]]; then
          _esc_seq="$(parse_esc_sequence)"
          case "$_esc_seq" in
            "A") _del_sel=$(( (_del_sel - 1 + ${#projects[@]}) % ${#projects[@]} )) ;;
            "B") _del_sel=$(( (_del_sel + 1) % ${#projects[@]} )) ;;
          esac
          continue
        elif [[ "$key" == "q" ]]; then
          _del_mode=0
          menu_labels[$_del_idx]="Delete a project"
          menu_subs[$_del_idx]=""
          printf "${_HIDE_CURSOR}"
          draw_menu
          continue
        elif [[ "$key" =~ ^[1-9]$ ]] && [ "$key" -le "${#projects[@]}" ]; then
          _del_sel=$((key - 1))
          # fall through to delete
        elif [[ "$key" == "" ]]; then
          # Enter: confirm current selection
          :
        else
          continue
        fi
        # Perform deletion
        del_name="${projects[$_del_sel]%%:*}"
        del_line="${projects[$_del_sel]}"
        delete_project_from_file "$del_line" "$PROJECTS_FILE"
        _del_mode=0
        menu_labels[$_del_idx]="Delete a project"
        menu_subs[$_del_idx]=""
        printf "${_HIDE_CURSOR}"
        draw_menu
        moveto "$_del_sub_row" "$_left_col"
        printf "    ${_GREEN}✓${_NC} Deleted ${_BOLD}${del_name}${_NC}\033[K"
        sleep 0.5
        break
      fi

      _do_select=0
      read -rsn1 key
      if [[ "$key" == $'\x1b' ]]; then
        _esc_seq="$(parse_esc_sequence)"
        if [[ "$_esc_seq" == click:* ]]; then
          _click_row="${_esc_seq#click:}"
          for _ci in $(seq 0 $((total - 1))); do
            if [ "$_click_row" -eq "${_item_rows[$_ci]}" ] || [ "$_click_row" -eq "$(( ${_item_rows[$_ci]} + 1 ))" ]; then
              selected=$_ci
              _do_select=1
              break
            fi
          done
          if [ "$_do_select" -eq 0 ]; then continue; fi
        else
          case "$_esc_seq" in
            "A") selected=$(( (selected - 1 + total) % total )); draw_menu ;;
            "B") selected=$(( (selected + 1) % total )); draw_menu ;;
            "C") cycle_ai_tool "next"; echo "$SELECTED_AI_TOOL" > "$AI_TOOL_PREF_FILE"; draw_menu ;;
            "D") cycle_ai_tool "prev"; echo "$SELECTED_AI_TOOL" > "$AI_TOOL_PREF_FILE"; draw_menu ;;
          esac
        fi
      fi
      if [[ "$key" =~ ^[1-9]$ ]] && [ "$key" -le "${#projects[@]}" ]; then
        selected=$((key - 1))
        _do_select=1
      fi
      _n=${#projects[@]}
      case "$key" in
        a|A) selected=$((_n)); _do_select=1 ;;
        d|D) selected=$((_n + 1)); _do_select=1 ;;
        o|O) selected=$((_n + 2)); _do_select=1 ;;
        p|P) selected=$((_n + 3)); _do_select=1 ;;
      esac
      if [[ "$key" == "" ]] || [ "$_do_select" -eq 1 ]; then
        case "${menu_types[$selected]}" in
          project)
            printf "${_SHOW_CURSOR}${_MOUSE_OFF}"
            printf '\033[2J\033[H'
            PROJECT_NAME="${projects[$selected]%%:*}"
            dir="${projects[$selected]#*:}"
            cd "$dir"
            break 2
            ;;
          add)
            printf "${_MOUSE_OFF}"
            _add_mode=1
            _add_idx=$selected
            _add_input=""
            menu_labels[$selected]="Enter project path:  ${_DIM}(empty to cancel)${_NC}"
            menu_subs[$selected]=""
            draw_menu
            ;;
          delete)
            if [ ${#projects[@]} -eq 0 ]; then
              draw_menu
              moveto "$(( _top_row + 3 + _update_line + selected * 2 + _sep_count + 1 ))" "$_left_col"
              printf "    ${_DIM}No projects to delete.${_NC}\033[K"
              sleep 0.8
              draw_menu
            else
              _del_mode=1
              _del_idx=$selected
              _del_sel=0
              menu_labels[$selected]="Select project to delete:"
              menu_subs[$selected]=""
              draw_menu
            fi
            ;;
          open_once)
            printf "${_MOUSE_OFF}"
            _open_mode=1
            _open_idx=$selected
            menu_labels[$selected]="Enter path to open:  ${_DIM}(empty to cancel)${_NC}"
            menu_subs[$selected]=""
            draw_menu
            ;;
          plain)
            printf "${_SHOW_CURSOR}${_MOUSE_OFF}"
            printf '\033[2J\033[H'
            exec bash
            ;;
        esac
      fi
    done
    # If we broke out of inner loop (not break 2), continue outer loop to redraw
    continue
  done
fi

export PROJECT_DIR="$(pwd)"
export PROJECT_NAME="${PROJECT_NAME:-$(basename "$PROJECT_DIR")}"
SESSION_NAME="dev-${PROJECT_NAME}-$$"

# Set terminal/tab title
printf '\033]0;%s\007' "$PROJECT_NAME"

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
