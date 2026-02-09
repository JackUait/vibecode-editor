#!/bin/bash
# Shared TUI helpers — colors, logging, cursor utilities.

# Basic colors (with terminal capability fallback)
if [ -t 1 ] && [ "$(tput colors 2>/dev/null)" -ge 8 ] 2>/dev/null; then
  _GREEN='\033[0;32m'
  _YELLOW='\033[0;33m'
  _RED='\033[0;31m'
  _BLUE='\033[0;34m'
  _BOLD='\033[1m'
  _NC='\033[0m'
else
  _GREEN='' _YELLOW='' _RED='' _BLUE='' _BOLD='' _NC=''
fi

# Logging helpers
success() { echo -e "${_GREEN}✓${_NC} $1"; }
warn()    { echo -e "${_YELLOW}!${_NC} $1"; }
error()   { echo -e "${_RED}✗${_NC} $1"; }
info()    { echo -e "${_BLUE}→${_NC} $1"; }
header()  { echo -e "\n${_BOLD}$1${_NC}"; }

# Extended TUI variables for interactive full-screen UIs.
# Call this before using any of the extended variables.
tui_init_interactive() {
  _CYAN=$'\033[0;36m'
  _DIM=$'\033[2m'
  _INVERSE=$'\033[7m'
  _BG_BLUE=$'\033[48;5;27m'
  _BG_RED=$'\033[48;5;160m'
  _WHITE=$'\033[1;37m'
  _HIDE_CURSOR=$'\033[?25l'
  _SHOW_CURSOR=$'\033[?25h'
  _MOUSE_ON=$'\033[?1000h\033[?1006h'
  _MOUSE_OFF=$'\033[?1000l\033[?1006l'
}

# Move cursor to row;col
moveto() { printf '\033[%d;%dH' "$1" "$2"; }

# Print N spaces
pad() { printf "%*s" "$1" ""; }
