#!/bin/bash
# Input parsing helpers — no side effects on source.

# Interactive confirmation using ghost-tab-tui
# Usage: confirm_tui "Delete project 'foo'?"
# Returns: 0 if confirmed, 1 if cancelled
confirm_tui() {
  local msg="$1"

  if ! command -v ghost-tab-tui &>/dev/null; then
    # Fallback to simple bash prompt
    read -rp "$msg (y/N) " response </dev/tty
    [[ "$response" =~ ^[Yy]$ ]]
    return $?
  fi

  local result
  if ! result=$(ghost-tab-tui confirm "$msg" 2>/dev/null); then
    return 1
  fi

  local confirmed
  if ! confirmed=$(echo "$result" | jq -r '.confirmed' 2>/dev/null); then
    # Source tui.sh for error function if not already loaded
    if ! declare -F error &>/dev/null; then
      echo "ERROR: Failed to parse confirmation response" >&2
    else
      error "Failed to parse confirmation response"
    fi
    return 1
  fi

  # Validate against "null" string (learned from Task 3)
  if [[ "$confirmed" == "null" || -z "$confirmed" ]]; then
    return 1
  fi

  [[ "$confirmed" == "true" ]]
}

# Parses an escape sequence from stdin (bytes AFTER the initial \x1b).
# Outputs to stdout:
#   "A"/"B"/"C"/"D" for arrow keys
#   "click:ROW" for SGR mouse left-click press
#   "" (empty) for ignored events (release, non-left-click)
parse_esc_sequence() {
  local _b1 _b2 _mc _mouse_data _mouse_btn _mouse_rest _mouse_col _mouse_row

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
        _mouse_btn="${_mouse_data%%;*}"
        _mouse_rest="${_mouse_data#*;}"
        _mouse_col="${_mouse_rest%%;*}"
        _mouse_row="${_mouse_rest##*;}"
        if [[ "$_mouse_btn" == "0" ]]; then
          echo "click:${_mouse_row}"
        fi
      fi
    else
      echo "$_b2"
    fi
  fi
}
