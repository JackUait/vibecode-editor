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

  # Clear screen to remove main menu, ghost will be redrawn after
  printf '\033[2J\033[H'

  # Box dimensions (matching main menu style)
  local box_w=48
  local box_h=6
  local left_col=$(( (_cols - box_w) / 2 + 1 ))
  [ "$left_col" -lt 1 ] && left_col=1
  local top_row=$(( (_rows - box_h) / 2 ))
  [ "$top_row" -lt 1 ] && top_row=1
  local content_col=$((left_col + 1))
  local r="$top_row"

  # Border colors matching AI tool (same as main menu)
  local _bdr_clr _acc_clr _inner_w _right_col _hline
  _bdr_clr="$(ai_tool_dim_color "$SELECTED_AI_TOOL")"
  _acc_clr="$(ai_tool_color "$SELECTED_AI_TOOL")"
  _inner_w=$((box_w - 2))
  _right_col=$((left_col + box_w - 1))
  printf -v _hline '%*s' "$_inner_w" ""
  _hline="${_hline// /─}"

  # Helper: print right border and clear rest of line
  _rbdr() { moveto "$1" "$_right_col"; printf "${_bdr_clr}│${_NC}\\033[K"; }

  # ── Top border ──
  moveto "$r" "$left_col"
  printf "${_bdr_clr}┌%s┐${_NC}\\033[K" "$_hline"
  r=$((r+1))

  # ── Title row ──
  moveto "$r" "$left_col"
  printf "${_bdr_clr}│${_NC}\\033[K"
  printf " ${_BOLD}${_acc_clr}⬡  Settings${_NC}"
  _rbdr "$r"
  r=$((r+1))

  # ── Separator ──
  moveto "$r" "$left_col"
  printf "${_bdr_clr}├%s┤${_NC}\\033[K" "$_hline"
  r=$((r+1))

  # ── Ghost Display setting ──
  moveto "$r" "$left_col"
  printf "${_bdr_clr}│${_NC}\\033[K"
  printf " Ghost Display    %b  ${_DIM}Press A to cycle${_NC}" "$state_display"
  _rbdr "$r"
  r=$((r+1))

  # ── Separator ──
  moveto "$r" "$left_col"
  printf "${_bdr_clr}├%s┤${_NC}\\033[K" "$_hline"
  r=$((r+1))

  # ── Footer ──
  moveto "$r" "$left_col"
  printf "${_bdr_clr}│${_NC}\\033[K"
  printf "  ${_DIM}ESC or B to go back${_NC}"
  _rbdr "$r"
  r=$((r+1))

  # ── Bottom border ──
  moveto "$r" "$left_col"
  printf "${_bdr_clr}└%s┘${_NC}\\033[K" "$_hline"
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

    # Redraw ghost after settings screen if it should be visible
    if [ "$_LOGO_LAYOUT" != "hidden" ]; then
      local ghost_display=$(get_ghost_display_setting)
      # Stop any existing animation first to prevent multiple processes
      stop_logo_animation 2>/dev/null
      case "$ghost_display" in
        animated)
          start_logo_animation "$_logo_row" "$_logo_col" "$SELECTED_AI_TOOL"
          ;;
        static)
          draw_logo "$_logo_row" "$_logo_col" "$SELECTED_AI_TOOL"
          ;;
        none)
          # Don't draw ghost
          ;;
      esac
    fi

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
