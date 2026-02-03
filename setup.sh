#!/bin/bash
set -e

# Wrap everything in a function so `curl | bash` reads the entire
# script into memory before executing.  Without this, bash reads
# line-by-line from the pipe and `read` consumes script lines as input.
main() {

# Colors (with fallback)
if [ -t 1 ] && [ "$(tput colors 2>/dev/null)" -ge 8 ] 2>/dev/null; then
  GREEN='\033[0;32m'
  YELLOW='\033[0;33m'
  RED='\033[0;31m'
  BLUE='\033[0;34m'
  BOLD='\033[1m'
  NC='\033[0m'
else
  GREEN='' YELLOW='' RED='' BLUE='' BOLD='' NC=''
fi

success() { echo -e "${GREEN}✓${NC} $1"; }
warn()    { echo -e "${YELLOW}!${NC} $1"; }
error()   { echo -e "${RED}✗${NC} $1"; }
info()    { echo -e "${BLUE}→${NC} $1"; }
header()  { echo -e "\n${BOLD}$1${NC}"; }

# ---------- Embedded files ----------

WRAPPER_SCRIPT='#!/bin/bash
export PATH="/opt/homebrew/bin:/usr/local/bin:$PATH"

TMUX_CMD="$(command -v tmux)"
LAZYGIT_CMD="$(command -v lazygit)"
BROOT_CMD="$(command -v broot)"
CLAUDE_CMD="$(command -v claude)"

# Load user projects from config file if it exists
PROJECTS_FILE="${XDG_CONFIG_HOME:-$HOME/.config}/vibecode-editor/projects"

# Select working directory
if [ -n "$1" ] && [ -d "$1" ]; then
  cd "$1"
  shift
elif [ -z "$1" ]; then
  # Colors for interactive menu
  _CYAN=$'\''\033[0;36m'\''
  _GREEN=$'\''\033[0;32m'\''
  _YELLOW=$'\''\033[0;33m'\''
  _BLUE=$'\''\033[0;34m'\''
  _BOLD=$'\''\033[1m'\''
  _DIM=$'\''\033[2m'\''
  _NC=$'\''\033[0m'\''
  _INVERSE=$'\''\033[7m'\''
  _BG_BLUE=$'\''\033[48;5;27m'\''
  _BG_RED=$'\''\033[48;5;160m'\''
  _WHITE=$'\''\033[1;37m'\''
  _HIDE_CURSOR=$'\''\033[?25l'\''
  _SHOW_CURSOR=$'\''\033[?25h'\''
  _MOUSE_ON=$'\''\033[?1000h\033[?1006h'\''
  _MOUSE_OFF=$'\''\033[?1000l\033[?1006l'\''

  # Restore cursor and disable mouse on exit
  trap '\''printf "${_SHOW_CURSOR}${_MOUSE_OFF}"; printf "\\033[?7h"'\'' EXIT

  # Wait for terminal to fully initialize and report correct size
  sleep 0.1

  # Set terminal title for project selection screen
  printf '\''\033]0;👻 Ghost Tab\007'\''

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
    menu_labels+=("Add new project" "Delete a project" "Open once")
    menu_subs+=("" "" "")
    menu_types+=("add" "delete" "open_once")
    menu_hi+=("${_BG_BLUE}${_WHITE}" "${_BG_RED}${_WHITE}" "${_INVERSE}")

    total=${#menu_labels[@]}
    selected=0
    box_w=44

    # Move cursor to row;col
    moveto() { printf '\''\033[%d;%dH'\'' "$1" "$2"; }

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
      printf '\''\033[s\033[9999;9999H'\''
      IFS='\''[;'\'' read -rs -d R -p $'\''\033[6n'\'' _ _rows _cols </dev/tty
      printf '\''\033[u'\''
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
            printf "${menu_hi[$i]}${_BOLD}  ❯ %s  ${_NC}\033[K" "${menu_labels[$i]}"
          fi
        else
          if [ "$i" -lt "${#projects[@]}" ]; then
            printf "  ${_DIM}%d${_NC} %s\033[K" "$((i+1))" "${menu_labels[$i]}"
          else
            printf "  ${_DIM}·${_NC} %s\033[K" "${menu_labels[$i]}"
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
    printf '\''\033[2J\033[H'\''
    draw_menu

    # Input loop
    while true; do
      if [ "$_add_mode" -eq 1 ]; then
        # Add mode: read full path with readline support
        _sub_row=$(( _top_row + 3 + _add_idx * 2 + _sep_count + 1 ))
        printf "${_SHOW_CURSOR}"
        moveto "$_sub_row" "$_left_col"
        printf "    ${_CYAN}\033[K"
        read -r _add_input
        printf "${_NC}"

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
        new_name="$(basename "$expanded")"
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
        # Open once mode: read path and open without saving
        _sub_row=$(( _top_row + 3 + _open_idx * 2 + _sep_count + 1 ))
        printf "${_SHOW_CURSOR}"
        moveto "$_sub_row" "$_left_col"
        printf "    ${_CYAN}\033[K"
        read -r _open_input
        printf "${_NC}"

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
          printf '\''\033[2J\033[H'\''
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
        if [[ "$key" == $'\''\x1b'\'' ]]; then
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
      if [[ "$key" == $'\''\x1b'\'' ]]; then
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
      if [[ "$key" == "" ]] || [ "$_do_select" -eq 1 ]; then
        case "${menu_types[$selected]}" in
          project)
            printf "${_SHOW_CURSOR}${_MOUSE_OFF}"
            printf '\''\033[2J\033[H'\''
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
printf '\''\033]0;%s\007'\'' "$PROJECT_NAME"

# Background watcher: switch to Claude pane once it'\''s ready
(
  while true; do
    sleep 0.5
    content=$("$TMUX_CMD" capture-pane -t "$SESSION_NAME:0.1" -p 2>/dev/null)
    if echo "$content" | grep -q '\''>'\''; then
      "$TMUX_CMD" select-pane -t "$SESSION_NAME:0.1"
      break
    fi
  done
) &
WATCHER_PID=$!

# Kill all processes in the tmux session when the terminal closes
cleanup() {
  # Get PIDs of all pane shell processes in this session, then kill their entire process trees
  for pane_pid in $("$TMUX_CMD" list-panes -s -t "$SESSION_NAME" -F '\''#{pane_pid}'\'' 2>/dev/null); do
    pkill -TERM -P "$pane_pid" 2>/dev/null
  done
  kill $WATCHER_PID 2>/dev/null
  "$TMUX_CMD" kill-session -t "$SESSION_NAME" 2>/dev/null
}
trap cleanup EXIT HUP TERM INT

"$TMUX_CMD" new-session -s "$SESSION_NAME" -e "PATH=$PATH" -c "$PROJECT_DIR" \
  "$LAZYGIT_CMD; exec bash" \; \
  set-option status-left " ⬡ ${PROJECT_NAME} " \; \
  set-option status-left-style "fg=white,bg=colour236,bold" \; \
  set-option status-style "bg=colour235" \; \
  set-option status-right "" \; \
  split-window -h -p 50 -c "$PROJECT_DIR" \
  "$CLAUDE_CMD $*; exec bash" \; \
  select-pane -t 0 \; \
  split-window -v -p 50 -c "$PROJECT_DIR" \
  "while true; do $BROOT_CMD $PROJECT_DIR; done" \; \
  split-window -v -p 30 -c "$PROJECT_DIR" \; \
  select-pane -t 3'

GHOSTTY_DEFAULT_CONFIG='keybind = cmd+shift+left=previous_tab
keybind = cmd+shift+right=next_tab
keybind = cmd+t=new_tab
macos-option-as-alt = left
command = ~/.config/ghostty/claude-wrapper.sh'

# ---------- OS check ----------
header "Checking platform..."
if [ "$(uname)" != "Darwin" ]; then
  error "This setup script only supports macOS."
  exit 1
fi
success "macOS detected"

# ---------- Homebrew ----------
header "Checking Homebrew..."
if ! command -v brew &>/dev/null; then
  info "Installing Homebrew..."
  /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
  eval "$(/opt/homebrew/bin/brew shellenv 2>/dev/null || /usr/local/bin/brew shellenv 2>/dev/null)"
  success "Homebrew installed"
else
  success "Homebrew found"
fi

# ---------- Dependencies ----------
header "Installing dependencies..."
for pkg in tmux lazygit broot claude; do
  brew_pkg="$pkg"
  [ "$pkg" = "claude" ] && brew_pkg="claude-code"

  if brew list "$brew_pkg" &>/dev/null; then
    success "$pkg already installed"
  else
    info "Installing $pkg..."
    if brew install "$brew_pkg"; then
      success "$pkg installed"
      [ "$pkg" = "claude" ] && info "Run 'claude' to authenticate before opening Ghostty."
    else
      if [ "$pkg" = "claude" ]; then
        error "Claude Code installation failed."
        info "Install manually: brew install claude-code"
        info "Then run 'claude' to authenticate before re-running this script."
        exit 1
      else
        warn "Failed to install $pkg — install it manually with: brew install $brew_pkg"
      fi
    fi
  fi
done

# ---------- Ghostty ----------
header "Checking Ghostty..."
if [ -d "/Applications/Ghostty.app" ]; then
  success "Ghostty found"
else
  info "Installing Ghostty..."
  if brew install --cask ghostty; then
    success "Ghostty installed"
  else
    error "Ghostty installation failed."
    info "Install manually from https://ghostty.org or run: brew install --cask ghostty"
    exit 1
  fi
fi

# ---------- Wrapper script ----------
header "Setting up wrapper script..."
mkdir -p ~/.config/ghostty
echo "$WRAPPER_SCRIPT" > ~/.config/ghostty/claude-wrapper.sh
chmod +x ~/.config/ghostty/claude-wrapper.sh
success "Created claude-wrapper.sh in ~/.config/ghostty/"

# ---------- Ghostty config ----------
header "Setting up Ghostty config..."
GHOSTTY_CONFIG="$HOME/.config/ghostty/config"
WRAPPER_LINE="command = ~/.config/ghostty/claude-wrapper.sh"

if [ -f "$GHOSTTY_CONFIG" ]; then
  warn "Existing Ghostty config found at $GHOSTTY_CONFIG"
  echo ""
  echo -e "  ${BOLD}1)${NC} Merge — add the wrapper command to your existing config"
  echo -e "  ${BOLD}2)${NC} Backup & replace — save current config and use ours"
  echo ""
  read -rn1 -p "$(echo -e "${BLUE}Choose (1/2):${NC} ")" config_choice </dev/tty
  echo ""

  case "$config_choice" in
    1)
      if grep -q '^command\s*=' "$GHOSTTY_CONFIG"; then
        sed -i '' 's|^command\s*=.*|'"$WRAPPER_LINE"'|' "$GHOSTTY_CONFIG"
        success "Replaced existing command line in config"
      else
        echo "$WRAPPER_LINE" >> "$GHOSTTY_CONFIG"
        success "Appended wrapper command to config"
      fi
      ;;
    2)
      BACKUP="$GHOSTTY_CONFIG.backup.$(date +%s)"
      cp "$GHOSTTY_CONFIG" "$BACKUP"
      success "Backed up existing config to $BACKUP"
      echo "$GHOSTTY_DEFAULT_CONFIG" > "$GHOSTTY_CONFIG"
      success "Replaced config with vibecode-editor defaults"
      ;;
    *)
      warn "Invalid choice, skipping config setup"
      ;;
  esac
else
  echo "$GHOSTTY_DEFAULT_CONFIG" > "$GHOSTTY_CONFIG"
  success "Created Ghostty config at $GHOSTTY_CONFIG"
fi

# ---------- Projects ----------
header "Setting up projects..."
PROJECTS_DIR="${XDG_CONFIG_HOME:-$HOME/.config}/vibecode-editor"
PROJECTS_FILE="$PROJECTS_DIR/projects"
mkdir -p "$PROJECTS_DIR"

echo ""
read -rn1 -p "$(echo -e "${BLUE}Add a project? (y/n):${NC} ")" add_project </dev/tty
echo ""

while [[ "$add_project" =~ ^[yY]$ ]]; do
  read -rp "$(echo -e "${BLUE}Project name:${NC} ")" proj_name </dev/tty
  read -rp "$(echo -e "${BLUE}Project path:${NC} ")" proj_path </dev/tty

  # Expand ~ to $HOME for validation
  expanded_path="${proj_path/#\~/$HOME}"

  if [ -d "$expanded_path" ]; then
    echo "$proj_name:$expanded_path" >> "$PROJECTS_FILE"
    success "Added $proj_name"
  else
    warn "Path $proj_path does not exist yet — adding anyway"
    echo "$proj_name:$expanded_path" >> "$PROJECTS_FILE"
  fi

  echo ""
  read -rn1 -p "$(echo -e "${BLUE}Add another? (y/n):${NC} ")" add_project </dev/tty
  echo ""
done

if [ -f "$PROJECTS_FILE" ] && [ -s "$PROJECTS_FILE" ]; then
  success "Projects saved to $PROJECTS_FILE"
else
  info "No projects added. Add them later to $PROJECTS_FILE"
fi

# ---------- Summary ----------
header "Setup complete!"
echo ""
success "Wrapper script: ~/.config/ghostty/claude-wrapper.sh"
success "Ghostty config:  ~/.config/ghostty/config"
success "Projects file:   $PROJECTS_FILE"
echo ""
info "Open a new Ghostty window to start coding."

} # end main

main "$@"
