#!/bin/bash
# Settings menu for ghost-tab configuration

SETTINGS_FILE="${XDG_CONFIG_HOME:-$HOME/.config}/ghost-tab/settings"

# ── Setting Getters/Setters ─────────────────────────────────────────

get_ghost_display_setting() {
  # Check for new setting first
  local value=$(grep "^ghost_display=" "$SETTINGS_FILE" 2>/dev/null | cut -d= -f2)

  if [ -n "$value" ]; then
    echo "$value"
  else
    # Migrate from old animation setting
    local old_animation=$(grep "^animation=" "$SETTINGS_FILE" 2>/dev/null | cut -d= -f2)
    if [ "$old_animation" = "off" ]; then
      echo "static"
    else
      echo "animated"  # Default and old "on" → animated
    fi
  fi
}

set_ghost_display_setting() {
  mkdir -p "$(dirname "$SETTINGS_FILE")"

  if grep -q "^ghost_display=" "$SETTINGS_FILE" 2>/dev/null; then
    sed -i '' "s/^ghost_display=.*/ghost_display=$1/" "$SETTINGS_FILE"
  else
    echo "ghost_display=$1" >> "$SETTINGS_FILE"
  fi
}

cycle_ghost_display() {
  local current=$(get_ghost_display_setting)
  local next

  case "$current" in
    animated) next="static" ;;
    static) next="none" ;;
    none) next="animated" ;;
    *) next="animated" ;;  # Fallback
  esac

  set_ghost_display_setting "$next"
}

# Legacy functions for backward compatibility
get_animation_setting() {
  local display=$(get_ghost_display_setting)
  [ "$display" = "animated" ] && echo "on" || echo "off"
}

set_animation_setting() {
  [ "$1" = "on" ] && set_ghost_display_setting "animated" || set_ghost_display_setting "static"
}

# ── Settings Menu UI ────────────────────────────────────────────────

draw_settings_screen() {
  local current=$(get_ghost_display_setting)
  local state_display

  case "$current" in
    animated)
      state_display="$(_c 114)[Animated]$(_r)"  # Green
      ;;
    static)
      state_display="$(_c 220)[Static]$(_r)"  # Yellow
      ;;
    none)
      state_display="$(_c 240)[None]$(_r)"  # Dim gray
      ;;
    *)
      state_display="$(_c 114)[Animated]$(_r)"  # Default
      ;;
  esac

  # Clear screen completely
  printf '\033[2J\033[H'

  # Center calculation
  local settings_w=48
  local left_col=$(( (_cols - settings_w) / 2 ))
  local top_row=$(( (_rows - 6) / 2 ))

  # Header
  moveto "$top_row" "$left_col"
  printf '%b' "$(_c 75)⬡  Settings$(_r)"

  # Top border
  top_row=$((top_row + 1))
  moveto "$top_row" "$left_col"
  printf '%b' "$(_c 240)────────────────────────────────────────────────$(_r)"

  # Ghost display setting
  top_row=$((top_row + 1))
  moveto "$top_row" "$left_col"
  printf ' Ghost Display    %b  Press A to cycle' "$state_display"

  # Bottom border
  top_row=$((top_row + 1))
  moveto "$top_row" "$left_col"
  printf '%b' "$(_c 240)────────────────────────────────────────────────$(_r)"

  # Footer
  top_row=$((top_row + 1))
  moveto "$top_row" "$left_col"
  printf '%b' "$(_c 240) ESC or B to go back$(_r)"
}

# ── Cycle Ghost Display Immediately ────────────────────────────────

cycle_ghost_display_immediate() {
  cycle_ghost_display
  local new_state=$(get_ghost_display_setting)

  # Stop animation in all cases first
  stop_logo_animation 2>/dev/null

  # Apply new state if layout allows and logo position is known
  if [ "$_LOGO_LAYOUT" != "hidden" ] && [ -n "$_logo_row" ] && [ -n "$_logo_col" ]; then
    case "$new_state" in
      animated)
        # Start animation
        start_logo_animation "$_logo_row" "$_logo_col" "$SELECTED_AI_TOOL"
        ;;
      static)
        # Draw static ghost
        draw_logo "$_logo_row" "$_logo_col" "$SELECTED_AI_TOOL"
        ;;
      none)
        # Clear ghost area
        clear_logo_area "$_logo_row" "$_logo_col" "$_LOGO_HEIGHT" "$_LOGO_WIDTH"
        ;;
    esac
  fi
}

# Legacy function for backward compatibility
toggle_animation_immediate() {
  cycle_ghost_display_immediate
}

# ── Settings Menu Main Loop ─────────────────────────────────────────

show_settings_menu() {
  while true; do
    draw_settings_screen

    read -rsn1 key
    case "$key" in
      A|a)
        cycle_ghost_display_immediate
        # Screen will redraw on next loop iteration
        ;;
      $'\e'|B|b)
        # Return to main menu
        return
        ;;
    esac
  done
}
