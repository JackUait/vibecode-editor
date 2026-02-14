#!/bin/bash
# Loading screen with random ASCII art, color palettes, and animation.

LOADING_ART_COUNT=10
LOADING_PALETTE_COUNT=5

# Get ASCII art by index (0-9). Prints to stdout.
get_loading_art() {
  case "$1" in
    0) cat << 'ART'
  ____ _               _     _____     _
 / ___| |__   ___  ___| |_  |_   _|_ _| |__
| |  _| '_ \ / _ \/ __| __|   | |/ _` | '_ \
| |_| | | | | (_) \__ \ |_    | | (_| | |_) |
 \____|_| |_|\___/|___/\__|   |_|\__,_|_.__/
ART
    ;;
    1) cat << 'ART'
   ________               __     ______      __
  / ____/ /_  ____  _____/ /_   /_  __/___ _/ /_
 / / __/ __ \/ __ \/ ___/ __/    / / / __ `/ __ \
/ /_/ / / / / /_/ (__  ) /_     / / / /_/ / /_/ /
\____/_/ /_/\____/____/\__/    /_/  \__,_/_.___/
ART
    ;;
    2) cat << 'ART'
  ___ _           _     _____     _
 / __| |_  ___ __| |_  |_   _|_ _| |__
| (_ | ' \/ _ (_-<  _|   | |/ _` | '_ \
 \___|_||_\___/__/\__|   |_|\__,_|_.__/
ART
    ;;
    3) cat << 'ART'
______________              _____     ________      ______
__  ____/__  /________________  /_    ___  __/_____ ___  /_
_  / __ __  __ \  __ \_  ___/  __/    __  /  _  __ `/_  __ \
/ /_/ / _  / / / /_/ /(__  )/ /_      _  /   / /_/ /_  /_/ /
\____/  /_/ /_/\____//____/ \__/      /_/    \__,_/ /_.___/
ART
    ;;
    4) cat << 'ART'
   ___ _               _     _____      _
  / _ \ |__   ___  ___| |_  /__   \__ _| |__
 / /_\/ '_ \ / _ \/ __| __|   / /\/ _` | '_ \
/ /_\\| | | | (_) \__ \ |_   / / | (_| | |_) |
\____/|_| |_|\___/|___/\__|  \/   \__,_|_.__/
ART
    ;;
    5) cat << 'ART'
 .d8888b. 888                     888      88888888888     888
d88P  Y88b888                     888          888         888
888    888888                     888          888         888
888       88888b.  .d88b. .d8888b 888888       888  8888b. 88888b.
888  88888888 "88bd88""88b88K     888          888     "88b888 "88b
888    888888  888888  888"Y8888b.888          888 .d888888888  888
Y88b  d88P888  888Y88..88P     X88Y88b.        888 888  888888 d88P
 "Y8888P88888  888 "Y88P"  88888P' "Y888       888 "Y88888888888P"
ART
    ;;
    6) cat << 'ART'
  ___| |                |   __ __|     |
 |     __ \   _ \   __| __|    |  _` | __ \
 |   | | | | (   |\__ \ |      | (   | |   |
\____|_| |_|\___/ ____/\__|   _|\__,_|_.__/
ART
    ;;
    7) cat << 'ART'
 _____ _           _      _____     _
|   __| |_ ___ ___| |_   |_   _|___| |_
|  |  |   | . |_ -|  _|    | | | .'| . |
|_____|_|_|___|___|_|      |_| |__,|___|
ART
    ;;
    8) cat << 'ART'
 _______ __                 __     _______         __
|     __|  |--.-----.-----.|  |_  |_     _|.---.-.|  |--.
|    |  |     |  _  |__ --||   _|   |   |  |  _  ||  _  |
|_______|__|__|_____|_____||____|   |___|  |___._||_____|
ART
    ;;
    9) cat << 'ART'
 ____    __                       __        ______          __
/\  _`\ /\ \                     /\ \__    /\__  _\        /\ \
\ \ \L\_\ \ \___     ___     ____\ \ ,_\   \/_/\ \/    __  \ \ \____
 \ \ \L_L\ \  _ `\  / __`\  /',__\\ \ \/      \ \ \  /'__`\ \ \ '__`\
  \ \ \/, \ \ \ \ \/\ \L\ \/\__, `\\ \ \_      \ \ \/\ \L\.\_\ \ \L\ \
   \ \____/\ \_\ \_\ \____/\/\____/ \ \__\      \ \_\ \__/.\_\\ \_,__/
    \/___/  \/_/\/_/\/___/  \/___/   \/__/       \/_/\/__/\/_/ \/___/
ART
    ;;
  esac
}

# Get color palette by index (0-4). Prints space-separated 256-color codes.
get_loading_palette() {
  case "$1" in
    0) echo "55 91 127 163 169 175 176 177" ;;   # purple aurora
    1) echo "125 161 162 197 198 205 206 213" ;;  # hot pink
    2) echo "17 18 24 25 31 33 39 45" ;;          # deep ocean
    3) echo "130 166 172 208 209 214 215 220" ;;  # fire / sunset
    4) echo "22 28 29 34 35 41 42 47" ;;          # emerald
  esac
}

# Render a single frame of the loading screen.
# Args: art_index palette_index frame_number term_cols term_rows
render_loading_frame() {
  local art_idx="$1" pal_idx="$2" frame="$3"
  local cols="${4:-80}" rows="${5:-24}"

  # Get art lines into array
  local art
  art="$(get_loading_art "$art_idx")"
  local -a lines=()
  while IFS= read -r line; do
    lines+=("$line")
  done <<< "$art"

  # Get palette
  local -a palette
  read -ra palette <<< "$(get_loading_palette "$pal_idx")"
  local pal_len=${#palette[@]}

  # Calculate art dimensions
  local art_height=${#lines[@]}
  local art_width=0
  for line in "${lines[@]}"; do
    local len=${#line}
    if (( len > art_width )); then
      art_width=$len
    fi
  done

  # Center position
  local start_row=$(( (rows - art_height) / 2 ))
  local start_col=$(( (cols - art_width) / 2 ))
  if (( start_row < 1 )); then start_row=1; fi
  if (( start_col < 1 )); then start_col=1; fi

  # Draw each line with gradient color shifted by frame
  local i
  for i in "${!lines[@]}"; do
    local color_idx=$(( (i + frame) % pal_len ))
    local color="${palette[$color_idx]}"
    printf '\033[%d;%dH\033[38;5;%dm%s' \
      "$((start_row + i))" "$start_col" "$color" "${lines[$i]}"
  done

  printf '\033[0m'
}

# PID of the background animation process.
_LOADING_SCREEN_PID=""

# Detect terminal dimensions reliably. Prints "rows cols" to stdout.
_detect_term_size() {
  local _r _c

  # Method 1: stty size via /dev/tty (most reliable in pty context)
  local _size
  if _size=$( (stty size </dev/tty) 2>/dev/null ) && read -r _r _c <<< "$_size"; then
    if (( _r > 0 && _c > 0 )); then
      echo "$_r $_c"
      return
    fi
  fi

  # Method 2: stty size from stdin
  if read -r _r _c < <(stty size 2>/dev/null); then
    if (( _r > 0 && _c > 0 )); then
      echo "$_r $_c"
      return
    fi
  fi

  # Method 3: tput (uses terminfo + ioctl)
  _c=$(tput cols 2>/dev/null || echo 0)
  _r=$(tput lines 2>/dev/null || echo 0)
  if (( _r > 0 && _c > 0 )); then
    echo "$_r $_c"
    return
  fi

  # Fallback
  echo "24 80"
}

# Show animated loading screen with random art and colors.
# Sets _LOADING_SCREEN_PID for the caller to stop later.
show_loading_screen() {
  local art_idx=$(( RANDOM % LOADING_ART_COUNT ))
  local pal_idx=$(( RANDOM % LOADING_PALETTE_COUNT ))
  local rows cols
  read -r rows cols <<< "$(_detect_term_size)"

  # Clear screen, hide cursor
  printf '\033[2J\033[H\033[?25l'

  # Draw initial frame
  render_loading_frame "$art_idx" "$pal_idx" 0 "$cols" "$rows"

  # Symbols for floating particles
  local symbols=('·' '•' '°' '∘' '⋅' '∙')

  # Start animation in background
  (
    trap 'printf "\033[?25h\033[0m"; exit 0' TERM INT HUP
    local frame=1
    local -a prev_sym_positions=()

    while true; do
      # Redraw art with shifted colors
      render_loading_frame "$art_idx" "$pal_idx" "$frame" "$cols" "$rows"

      # Clear previous floating symbols
      for pos in "${prev_sym_positions[@]}"; do
        local sr sc
        IFS=';' read -r sr sc <<< "$pos"
        printf '\033[%d;%dH ' "$sr" "$sc"
      done
      prev_sym_positions=()

      # Draw new floating symbols
      local -a palette
      read -ra palette <<< "$(get_loading_palette "$pal_idx")"
      local pal_len=${#palette[@]}
      local _s
      for _s in 0 1 2; do
        local sym_row=$(( RANDOM % rows + 1 ))
        local sym_col=$(( RANDOM % cols + 1 ))
        local sym_color="${palette[$(( RANDOM % pal_len ))]}"
        local sym="${symbols[$(( RANDOM % ${#symbols[@]} ))]}"
        printf '\033[%d;%dH\033[2m\033[38;5;%dm%s\033[0m' \
          "$sym_row" "$sym_col" "$sym_color" "$sym"
        prev_sym_positions+=("${sym_row};${sym_col}")
      done

      frame=$(( (frame + 1) % pal_len ))
      sleep 0.15
    done
  ) &
  _LOADING_SCREEN_PID=$!
}

# Stop loading screen animation and restore terminal.
stop_loading_screen() {
  if [ -n "${_LOADING_SCREEN_PID:-}" ]; then
    kill "$_LOADING_SCREEN_PID" 2>/dev/null
    wait "$_LOADING_SCREEN_PID" 2>/dev/null
    _LOADING_SCREEN_PID=""
  fi
  printf '\033[?25h\033[0m'
}
