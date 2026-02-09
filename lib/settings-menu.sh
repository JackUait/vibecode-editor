#!/bin/bash
# Settings menu for ghost-tab configuration

SETTINGS_FILE="${XDG_CONFIG_HOME:-$HOME/.config}/ghost-tab/settings"

# ── Setting Getters/Setters ─────────────────────────────────────────

get_animation_setting() {
  local value=$(grep "^animation=" "$SETTINGS_FILE" 2>/dev/null | cut -d= -f2)
  echo "${value:-on}"  # Default to "on"
}

set_animation_setting() {
  mkdir -p "$(dirname "$SETTINGS_FILE")"

  if grep -q "^animation=" "$SETTINGS_FILE" 2>/dev/null; then
    sed -i '' "s/^animation=.*/animation=$1/" "$SETTINGS_FILE"
  else
    echo "animation=$1" >> "$SETTINGS_FILE"
  fi
}

# ── Settings Menu UI ────────────────────────────────────────────────

draw_settings_screen() {
  local current=$(get_animation_setting)
  local state_display

  if [ "$current" = "on" ]; then
    state_display="$(_c 114)[ON]$(_r)"  # Green
  else
    state_display="$(_c 240)[OFF]$(_r)"  # Dim gray
  fi

  # Clear screen completely
  printf '\033[2J\033[H'

  # Center calculation
  local settings_w=42
  local left_col=$(( (_cols - settings_w) / 2 ))
  local top_row=$(( (_rows - 6) / 2 ))

  # Header
  moveto "$top_row" "$left_col"
  printf '%b' "$(_c 75)⬡  Settings$(_r)"

  # Top border
  top_row=$((top_row + 1))
  moveto "$top_row" "$left_col"
  printf '%b' "$(_c 240)──────────────────────────────────────────$(_r)"

  # Animation setting
  top_row=$((top_row + 1))
  moveto "$top_row" "$left_col"
  printf ' Ghost Animation    %b  Press A to toggle' "$state_display"

  # Bottom border
  top_row=$((top_row + 1))
  moveto "$top_row" "$left_col"
  printf '%b' "$(_c 240)──────────────────────────────────────────$(_r)"

  # Footer
  top_row=$((top_row + 1))
  moveto "$top_row" "$left_col"
  printf '%b' "$(_c 240) ESC or B to go back$(_r)"
}

# ── Toggle Animation Immediately ────────────────────────────────────

toggle_animation_immediate() {
  local current=$(get_animation_setting)

  if [ "$current" = "on" ]; then
    set_animation_setting "off"
    stop_logo_animation
  else
    set_animation_setting "on"
    # Restart animation if logo is visible
    if [ "$_LOGO_LAYOUT" != "hidden" ] && [ -n "$_logo_row" ] && [ -n "$_logo_col" ]; then
      start_logo_animation "$_logo_row" "$_logo_col" "$SELECTED_AI_TOOL"
    fi
  fi
}

# ── Settings Menu Main Loop ─────────────────────────────────────────

show_settings_menu() {
  while true; do
    draw_settings_screen

    read -rsn1 key
    case "$key" in
      A|a)
        toggle_animation_immediate
        # Screen will redraw on next loop iteration
        ;;
      $'\e'|B|b)
        # Return to main menu
        return
        ;;
    esac
  done
}
