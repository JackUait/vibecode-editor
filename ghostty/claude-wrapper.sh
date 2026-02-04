#!/bin/bash
export PATH="/opt/homebrew/bin:/usr/local/bin:$PATH"

TMUX_CMD="$(command -v tmux)"
LAZYGIT_CMD="$(command -v lazygit)"
BROOT_CMD="$(command -v broot)"
CLAUDE_CMD="$(command -v claude)"

# Load user projects from config file if it exists
PROJECTS_FILE="${XDG_CONFIG_HOME:-$HOME/.config}/ghost-tab/projects"

# Version update check (Homebrew only)
UPDATE_CACHE="${XDG_CONFIG_HOME:-$HOME/.config}/ghost-tab/.update-check"
_update_version=""

check_for_update() {
  local cache_ts now age latest
  # Only check if brew is available (Homebrew install)
  command -v brew &>/dev/null || return

  # Read cache if it exists
  if [ -f "$UPDATE_CACHE" ]; then
    latest="$(sed -n '1p' "$UPDATE_CACHE")"
    cache_ts="$(sed -n '2p' "$UPDATE_CACHE")"
    now="$(date +%s)"
    age=$(( now - ${cache_ts:-0} ))
    # Use cached result if less than 24 hours old
    if [ "$age" -lt 86400 ]; then
      _update_version="$latest"
      return
    fi
  fi

  # Spawn background check (non-blocking)
  (
    result="$(brew outdated --verbose --formula ghost-tab 2>/dev/null)"
    mkdir -p "$(dirname "$UPDATE_CACHE")"
    if [ -n "$result" ]; then
      # Extract new version: "ghost-tab (1.0.0) < 1.1.0" -> "1.1.0"
      new_ver="$(echo "$result" | sed -n 's/.*< *//p')"
      printf '%s\n%s\n' "$new_ver" "$(date +%s)" > "$UPDATE_CACHE.tmp"
      mv "$UPDATE_CACHE.tmp" "$UPDATE_CACHE"
    else
      printf '\n%s\n' "$(date +%s)" > "$UPDATE_CACHE.tmp"
      mv "$UPDATE_CACHE.tmp" "$UPDATE_CACHE"
    fi
  ) &
  disown
}

check_for_update

# Select working directory
if [ -n "$1" ] && [ -d "$1" ]; then
  cd "$1"
  shift
elif [ -z "$1" ]; then
  # Colors for interactive menu
  _CYAN=$'\033[0;36m'
  _GREEN=$'\033[0;32m'
  _YELLOW=$'\033[0;33m'
  _BLUE=$'\033[0;34m'
  _BOLD=$'\033[1m'
  _DIM=$'\033[2m'
  _NC=$'\033[0m'
  _INVERSE=$'\033[7m'
  _BG_BLUE=$'\033[48;5;27m'
  _BG_RED=$'\033[48;5;160m'
  _WHITE=$'\033[1;37m'
  _HIDE_CURSOR=$'\033[?25l'
  _SHOW_CURSOR=$'\033[?25h'
  _MOUSE_ON=$'\033[?1000h\033[?1006h'
  _MOUSE_OFF=$'\033[?1000l\033[?1006l'

  # Restore cursor and disable mouse on exit
  trap 'printf "${_SHOW_CURSOR}${_MOUSE_OFF}"; printf "\\033[?7h"' EXIT

  # Wait for terminal to fully initialize and report correct size
  sleep 0.1

  # Set terminal title for project selection screen
  printf '\033]0;👻 Ghost Tab\007'

  # Padding helper: prints N spaces
  pad() { printf "%*s" "$1" ""; }

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

    # Move cursor to row;col
    moveto() { printf '\033[%d;%dH' "$1" "$2"; }

    # Get directory suggestions for autocomplete
    get_suggestions() {
      local input="$1" expanded dir base
      _suggestions=()

      [ -z "$input" ] && input="~/"
      expanded="${input/#\~/$HOME}"

      if [[ "$expanded" == */ ]] && [ -d "$expanded" ]; then
        dir="$expanded"
        base=""
      else
        dir="$(dirname "$expanded")/"
        base="$(basename "$expanded")"
      fi

      [ ! -d "$dir" ] && return

      local item name
      while IFS= read -r item; do
        [ -z "$item" ] && continue
        name="$(basename "$item")"
        [ -d "$item" ] && name="${name}/"
        _suggestions+=("$name")
      done < <(find "$dir" -maxdepth 1 -iname "${base}*" 2>/dev/null | sort -f | head -8)
    }

    # Draw autocomplete suggestions box
    draw_suggestions() {
      local input_row="$1" input_col="$2" i
      local box_row=$((input_row + 1))
      local box_col=$((input_col + 2))

      # Clear suggestion area (below input)
      for i in $(seq 0 11); do
        moveto "$((box_row + i))" "$box_col"
        printf "\033[K"
      done

      [ ${#_suggestions[@]} -eq 0 ] && return

      moveto "$box_row" "$box_col"
      printf "${_DIM}┌────────────────────────────────────┐${_NC}"

      for i in "${!_suggestions[@]}"; do
        moveto "$((box_row + 1 + i))" "$box_col"
        if [ "$i" -eq "$_sug_sel" ]; then
          printf "${_DIM}│${_NC}${_INVERSE} %-34.34s ${_NC}${_DIM}│${_NC}" "${_suggestions[$i]}"
        else
          printf "${_DIM}│${_NC} %-34.34s ${_DIM}│${_NC}" "${_suggestions[$i]}"
        fi
      done

      moveto "$((box_row + 1 + ${#_suggestions[@]}))" "$box_col"
      printf "${_DIM}└────────────────────────────────────┘${_NC}"
      moveto "$((box_row + 2 + ${#_suggestions[@]}))" "$box_col"
      printf "${_DIM}↑↓${_NC} navigate  ${_DIM}⏎${_NC} complete  ${_DIM}Esc${_NC} cancel"
    }

    # Read path with autocomplete - sets _path_result
    read_path_autocomplete() {
      local prompt_row="$1" prompt_col="$2"
      _path_result=""
      _path_input=""
      _sug_sel=0
      local _just_completed=0

      printf "${_SHOW_CURSOR}"

      while true; do
        # Get and draw suggestions (skip if just completed)
        if [ "$_just_completed" -eq 0 ]; then
          get_suggestions "$_path_input"
          [ "$_sug_sel" -ge "${#_suggestions[@]}" ] && _sug_sel=0
          draw_suggestions "$prompt_row" "$prompt_col"
        fi

        # Draw input
        moveto "$prompt_row" "$prompt_col"
        printf "    ${_CYAN}%-30.30s${_NC}" "$_path_input"
        moveto "$prompt_row" "$(( prompt_col + 4 + ${#_path_input} ))"

        read -rsn1 key

        if [[ "$key" == $'\x1b' ]]; then
          read -rsn1 seq1
          if [[ "$seq1" == "[" ]]; then
            read -rsn1 seq2
            case "$seq2" in
              "A") # Up arrow
                if [ ${#_suggestions[@]} -gt 0 ]; then
                  _sug_sel=$(( (_sug_sel - 1 + ${#_suggestions[@]}) % ${#_suggestions[@]} ))
                fi
                ;;
              "B") # Down arrow
                if [ ${#_suggestions[@]} -gt 0 ]; then
                  _sug_sel=$(( (_sug_sel + 1) % ${#_suggestions[@]} ))
                fi
                ;;
            esac
          else
            # Escape pressed - cancel
            _path_result=""
            break
          fi
        elif [[ "$key" == $'\x7f' ]] || [[ "$key" == $'\x08' ]]; then
          # Backspace
          [ -n "$_path_input" ] && _path_input="${_path_input%?}"
          _sug_sel=0
          _just_completed=0
        elif [[ "$key" == $'\t' ]] || [[ "$key" == "" ]]; then
          # Tab or Enter
          if [[ "$key" == "" ]] && [ "$_just_completed" -eq 1 ]; then
            # Second Enter after completion - confirm
            _path_result="$_path_input"
            break
          elif [ ${#_suggestions[@]} -gt 0 ]; then
            local expanded="${_path_input/#\~/$HOME}"
            local dir
            if [[ "$expanded" == */ ]] && [ -d "$expanded" ]; then
              dir="$expanded"
            else
              dir="$(dirname "$expanded")/"
            fi
            local selected_sug="${_suggestions[$_sug_sel]}"
            local completed="${dir}${selected_sug}"
            local new_input="${completed/#$HOME/\~}"
            _path_input="$new_input"
            _sug_sel=0
            _just_completed=1
            # Clear suggestions and redraw menu below
            for i in $(seq 0 12); do
              moveto "$((prompt_row + 1 + i))" "$(( prompt_col + 2 ))"
              printf "\033[K"
            done
            draw_menu
            # Redraw input
            moveto "$prompt_row" "$prompt_col"
            printf "    ${_CYAN}%-30.30s${_NC}" "$_path_input"
            moveto "$prompt_row" "$(( prompt_col + 4 + ${#_path_input} ))"
          elif [[ "$key" == "" ]]; then
            # Enter with no suggestions - confirm current input
            _path_result="$_path_input"
            break
          fi
        elif [[ "$key" =~ [[:print:]] ]]; then
          _path_input="${_path_input}${key}"
          _sug_sel=0
          _just_completed=0
        fi
      done

      # Clear suggestion box and redraw menu
      for i in $(seq 0 12); do
        moveto "$((prompt_row + 1 + i))" "$(( prompt_col + 2 ))"
        printf "\033[K"
      done
      draw_menu
    }

    # Read escape sequence after \x1b
    # Sets _esc_seq to: A/B (arrows), "click:ROW" (mouse click), or empty
    read_esc() {
      _esc_seq=""
      read -rsn1 _b1
      if [[ "$_b1" == "[" ]]; then
        read -rsn1 _b2
        if [[ "$_b2" == "<" ]]; then
          # SGR mouse: read until M (press) or m (release)
          _mouse_data=""
          while true; do
            read -rsn1 _mc
            if [[ "$_mc" == "M" || "$_mc" == "m" ]]; then
              break
            fi
            _mouse_data="${_mouse_data}${_mc}"
          done
          # Only handle press (M), ignore release (m)
          if [[ "$_mc" == "M" ]]; then
            # Parse button;col;row — we want the row
            _mouse_btn="${_mouse_data%%;*}"
            _mouse_rest="${_mouse_data#*;}"
            _mouse_col="${_mouse_rest%%;*}"
            _mouse_row="${_mouse_rest##*;}"
            # Only left click (button 0)
            if [[ "$_mouse_btn" == "0" ]]; then
              _esc_seq="click:${_mouse_row}"
            fi
          fi
        else
          _esc_seq="$_b2"
        fi
      fi
    }

    draw_menu() {
      local i r c

      # Read terminal dimensions using cursor position trick
      # Move cursor to far corner, query position to get actual size
      printf '\033[s\033[9999;9999H'
      IFS='[;' read -rs -d R -p $'\033[6n' _ _rows _cols </dev/tty
      printf '\033[u'
      : "${_rows:=24}" "${_cols:=80}"

      _sep_count=0
      [ "${#projects[@]}" -gt 0 ] && _sep_count=1
      _menu_h=$(( 3 + total * 2 + _sep_count + 2 ))

      _top_row=$(( (_rows - _menu_h) / 2 ))
      [ "$_top_row" -lt 1 ] && _top_row=1
      _left_col=$(( (_cols - box_w) / 2 + 1 ))
      [ "$_left_col" -lt 1 ] && _left_col=1

      c="$_left_col"
      r="$_top_row"

      # Title
      moveto "$r" "$c"; printf "${_BOLD}${_CYAN}⬡  Ghost Tab${_NC}\033[K"; r=$((r+1))
      moveto "$r" "$c"; printf "${_DIM}──────────────────────────────────────${_NC}\033[K"; r=$((r+1))
      moveto "$r" "$c"; printf "\033[K"; r=$((r+1))

      _item_rows=()
      for i in $(seq 0 $((total - 1))); do
        # Separator before action items
        if [ "$i" -eq "${#projects[@]}" ] && [ "${#projects[@]}" -gt 0 ]; then
          moveto "$r" "$c"; printf "${_DIM}──────────────────────────────────────${_NC}\033[K"; r=$((r+1))
        fi

        _item_rows+=("$r")
        moveto "$r" "$c"
        if [ "$i" -eq "$selected" ]; then
          if [ "$i" -lt "${#projects[@]}" ]; then
            printf "${menu_hi[$i]}${_BOLD} %d❯ %s  ${_NC}\033[K" "$((i+1))" "${menu_labels[$i]}"
          else
            printf "${menu_hi[$i]}${_BOLD} %s❯ %s  ${_NC}\033[K" "${_action_hints[$((i - ${#projects[@]}))]}" "${menu_labels[$i]}"
          fi
        else
          if [ "$i" -lt "${#projects[@]}" ]; then
            printf "  ${_DIM}%d${_NC} %s\033[K" "$((i+1))" "${menu_labels[$i]}"
          else
            printf "  ${_DIM}%s${_NC} %s\033[K" "${_action_hints[$((i - ${#projects[@]}))]}" "${menu_labels[$i]}"
          fi
        fi
        r=$((r+1))

        # Subtitle line
        moveto "$r" "$c"
        if [ -n "${menu_subs[$i]}" ]; then
          if [ "$i" -eq "$selected" ]; then
            printf "    ${_CYAN}%s${_NC}\033[K" "${menu_subs[$i]}"
          else
            printf "    ${_DIM}%s${_NC}\033[K" "${menu_subs[$i]}"
          fi
        else
          printf "\033[K"
        fi
        r=$((r+1))
      done

      moveto "$r" "$c"; printf "${_DIM}──────────────────────────────────────${_NC}\033[K"; r=$((r+1))
      moveto "$r" "$c"; printf "${_DIM}  ↑↓${_NC} navigate  ${_DIM}⏎${_NC} select\033[K"
    }

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
        _sub_row=$(( _top_row + 3 + _add_idx * 2 + _sep_count + 1 ))
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

        expanded="${_add_input/#\~/$HOME}"
        # Normalize: resolve to absolute path, remove trailing slash
        expanded="$(cd "$expanded" 2>/dev/null && pwd)" || expanded="${_add_input/#\~/$HOME}"
        expanded="${expanded%/}"
        new_name="$(basename "$expanded")"

        # Check if project already exists (compare normalized paths)
        _already_exists=0
        for _proj in "${projects[@]}"; do
          _proj_path="${_proj#*:}"
          _proj_path="${_proj_path%/}"
          if [[ "$_proj_path" == "$expanded" ]]; then
            _already_exists=1
            break
          fi
        done

        if [ "$_already_exists" -eq 1 ]; then
          _add_mode=0
          menu_labels[$_add_idx]="Add new project"
          menu_subs[$_add_idx]=""
          draw_menu
          moveto "$_sub_row" "$_left_col"
          printf "    ${_YELLOW}!${_NC} Project ${_BOLD}${new_name}${_NC} already exists\033[K"
          sleep 1
          printf "${_HIDE_CURSOR}"
          draw_menu
          continue
        fi

        mkdir -p "$(dirname "$PROJECTS_FILE")"
        echo "${new_name}:${expanded}" >> "$PROJECTS_FILE"
        _add_mode=0
        menu_labels[$_add_idx]="Add new project"
        menu_subs[$_add_idx]=""
        draw_menu
        moveto "$_sub_row" "$_left_col"
        printf "    ${_GREEN}✓${_NC} Added ${_BOLD}${new_name}${_NC}\033[K"
        sleep 0.8
        printf "${_HIDE_CURSOR}"
        # Redraw with new project list
        break
      fi

      if [ "$_open_mode" -eq 1 ]; then
        # Open once mode: read path with autocomplete
        _sub_row=$(( _top_row + 3 + _open_idx * 2 + _sep_count + 1 ))
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
        _del_sub_row=$(( _top_row + 3 + _del_idx * 2 + _sep_count + 1 ))
        # Render current selection on subtitle line
        _dn="${projects[$_del_sel]%%:*}"
        moveto "$_del_sub_row" "$_left_col"
        printf "    ${_BG_RED}${_WHITE}${_BOLD} %d) %s ${_NC}  ${_DIM}↑↓ navigate  1-9 jump  ⏎ delete  q cancel${_NC}\033[K" "$((_del_sel+1))" "$_dn"

        read -rsn1 key
        if [[ "$key" == $'\x1b' ]]; then
          read_esc
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
        grep -vxF "$del_line" "$PROJECTS_FILE" > "$PROJECTS_FILE.tmp" && mv "$PROJECTS_FILE.tmp" "$PROJECTS_FILE"
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
        read_esc
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
              moveto "$(( _top_row + 3 + selected * 2 + _sep_count + 1 ))" "$_left_col"
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
    if echo "$content" | grep -q '>'; then
      "$TMUX_CMD" select-pane -t "$SESSION_NAME:0.1"
      break
    fi
  done
) &
WATCHER_PID=$!

# Kill all processes in the tmux session when the terminal closes
kill_tree() {
  local pid=$1
  local sig=${2:-TERM}
  # Kill children first (depth-first), then the process itself
  for child in $(pgrep -P "$pid" 2>/dev/null); do
    kill_tree "$child" "$sig"
  done
  kill -"$sig" "$pid" 2>/dev/null
}

cleanup() {
  kill $WATCHER_PID 2>/dev/null

  # SIGTERM the full process tree of every pane
  for pane_pid in $("$TMUX_CMD" list-panes -s -t "$SESSION_NAME" -F '#{pane_pid}' 2>/dev/null); do
    kill_tree "$pane_pid" TERM
  done

  # Brief grace period, then SIGKILL any survivors
  sleep 0.3
  for pane_pid in $("$TMUX_CMD" list-panes -s -t "$SESSION_NAME" -F '#{pane_pid}' 2>/dev/null); do
    kill_tree "$pane_pid" KILL
  done

  "$TMUX_CMD" kill-session -t "$SESSION_NAME" 2>/dev/null
}
trap cleanup EXIT HUP TERM INT

"$TMUX_CMD" new-session -s "$SESSION_NAME" -e "PATH=$PATH" -c "$PROJECT_DIR" \
  "$LAZYGIT_CMD; exec bash" \; \
  set-option status-left " ⬡ ${PROJECT_NAME} " \; \
  set-option status-left-style "fg=white,bg=colour236,bold" \; \
  set-option status-style "bg=colour235" \; \
  set-option status-right "" \; \
  set-option exit-unattached on \; \
  split-window -h -p 50 -c "$PROJECT_DIR" \
  "$CLAUDE_CMD $*; exec bash" \; \
  select-pane -t 0 \; \
  split-window -v -p 50 -c "$PROJECT_DIR" \
  "trap exit TERM; while true; do $BROOT_CMD $PROJECT_DIR; done" \; \
  split-window -v -p 30 -c "$PROJECT_DIR" \; \
  select-pane -t 3
