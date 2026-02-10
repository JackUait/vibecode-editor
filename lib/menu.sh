#!/bin/bash
# Menu drawing — project selection screen.
# Depends on: tui.sh, ai-tools.sh, and caller globals (projects, menu_*, selected, etc.)

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
  _update_line=0
  [ -n "$_update_version" ] && _update_line=1
  _menu_h=$(( 7 + _update_line + total * 2 + _sep_count ))

  _top_row=$(( (_rows - _menu_h) / 2 ))
  [ "$_top_row" -lt 1 ] && _top_row=1
  _left_col=$(( (_cols - box_w) / 2 + 1 ))
  [ "$_left_col" -lt 1 ] && _left_col=1
  _content_col=$(( _left_col + 1 ))

  # ── Logo layout ──
  # Need _LOGO_HEIGHT and _LOGO_WIDTH from the art function
  "logo_art_${SELECTED_AI_TOOL}"
  local _logo_need_side=$(( box_w + 3 + _LOGO_WIDTH + 3 ))

  if [ "$_logo_need_side" -le "$_cols" ]; then
    _LOGO_LAYOUT="side"
    _logo_col=$(( _left_col + box_w + 3 ))
    _logo_row=$(( _top_row + (_menu_h - _LOGO_HEIGHT) / 2 ))
    [ "$_logo_row" -lt 1 ] && _logo_row=1
    # Ensure room for full bob range
    if [ $((_logo_row + _LOGO_HEIGHT + _BOB_MAX)) -gt "$_rows" ]; then
      _logo_row=$((_rows - _LOGO_HEIGHT - _BOB_MAX))
      [ "$_logo_row" -lt 1 ] && _logo_row=1
    fi
  elif [ "$_rows" -ge "$(( _menu_h + _LOGO_HEIGHT + _BOB_MAX + 1 ))" ]; then
    _LOGO_LAYOUT="above"
    _top_row=$(( (_rows - _menu_h - _LOGO_HEIGHT - 1) / 2 + _LOGO_HEIGHT + 1 ))
    [ "$_top_row" -lt $(( _LOGO_HEIGHT + 2 )) ] && _top_row=$(( _LOGO_HEIGHT + 2 ))
    _logo_row=$(( _top_row - _LOGO_HEIGHT - 1 ))
    [ "$_logo_row" -lt 1 ] && _logo_row=1
    _logo_col=$(( (_cols - _LOGO_WIDTH) / 2 + 1 ))
    # Recalculate left_col and content_col since _top_row changed
    _left_col=$(( (_cols - box_w) / 2 + 1 ))
    [ "$_left_col" -lt 1 ] && _left_col=1
    _content_col=$(( _left_col + 1 ))
  else
    _LOGO_LAYOUT="hidden"
  fi

  c="$_left_col"
  r="$_top_row"

  # Precompute border colors and horizontal line
  local _bdr_clr _acc_clr _bright_clr _inner_w _right_col _hline
  _bdr_clr="$(ai_tool_dim_color "$SELECTED_AI_TOOL")"
  _acc_clr="$(ai_tool_color "$SELECTED_AI_TOOL")"
  _bright_clr="$(ai_tool_bright_color "$SELECTED_AI_TOOL")"
  _inner_w=$(( box_w - 2 ))
  _right_col=$(( c + box_w - 1 ))
  printf -v _hline '%*s' "$_inner_w" ""
  _hline="${_hline// /─}"

  # Helper: print right border at fixed column and clear rest of line
  _rbdr() { moveto "$1" "$_right_col"; printf "${_bdr_clr}│${_NC}\033[K"; }

  # ── Top border ──
  moveto "$r" "$c"
  printf "${_bdr_clr}┌%s┐${_NC}\033[K" "$_hline"
  r=$((r+1))

  # ── Title row ──
  local _title_w=13 _layout_w=$(( _inner_w - 2 ))
  moveto "$r" "$c"
  printf "${_bdr_clr}│${_NC}\033[K"
  if [ ${#AI_TOOLS_AVAILABLE[@]} -gt 1 ]; then
    local _ai_name
    _ai_name="$(ai_tool_display_name "$SELECTED_AI_TOOL")"
    local _pad=$(( _layout_w - _title_w - ${#_ai_name} - 4 ))
    [ "$_pad" -lt 2 ] && _pad=2
    local _ai_clr
    _ai_clr="$(ai_tool_color "$SELECTED_AI_TOOL")"
    printf " ${_BOLD}${_acc_clr}⬡  Ghost Tab${_NC}%*s${_DIM}◂${_NC} ${_ai_clr}%s${_NC} ${_DIM}▸${_NC} " \
      "$_pad" "" "$_ai_name"
  elif [ ${#AI_TOOLS_AVAILABLE[@]} -eq 1 ]; then
    local _ai_name
    _ai_name="$(ai_tool_display_name "$SELECTED_AI_TOOL")"
    local _pad=$(( _layout_w - _title_w - ${#_ai_name} ))
    [ "$_pad" -lt 2 ] && _pad=2
    local _ai_clr
    _ai_clr="$(ai_tool_color "$SELECTED_AI_TOOL")"
    printf " ${_BOLD}${_acc_clr}⬡  Ghost Tab${_NC}%*s${_ai_clr}%s${_NC} " \
      "$_pad" "" "$_ai_name"
  else
    printf " ${_BOLD}${_acc_clr}⬡  Ghost Tab${_NC}"
  fi
  _rbdr "$r"
  r=$((r+1))

  # ── Update notification ──
  if [ -n "$_update_version" ]; then
    moveto "$r" "$c"
    printf "${_bdr_clr}│${_NC}\033[K  ${_YELLOW}Update available: v${_update_version}${_NC} ${_DIM}(brew upgrade ghost-tab)${_NC}"
    _rbdr "$r"
    r=$((r+1))
  fi

  # ── Separator after title ──
  moveto "$r" "$c"
  printf "${_bdr_clr}├%s┤${_NC}\033[K" "$_hline"
  r=$((r+1))

  # ── Blank row ──
  moveto "$r" "$c"
  printf "${_bdr_clr}│${_NC}\033[K"
  _rbdr "$r"
  r=$((r+1))

  # ── Menu items ──
  local _max_label=$(( _inner_w - 8 ))
  _item_rows=()
  for i in $(seq 0 $((total - 1))); do
    # Separator before action items
    if [ "$i" -eq "${#projects[@]}" ] && [ "${#projects[@]}" -gt 0 ]; then
      moveto "$r" "$c"
      printf "${_bdr_clr}├%s┤${_NC}\033[K" "$_hline"
      r=$((r+1))
    fi

    # Truncate label if needed
    local _label="${menu_labels[$i]}"
    if [ "${#_label}" -gt "$_max_label" ]; then
      _label="${_label:0:$((_max_label-1))}…"
    fi

    _item_rows+=("$r")
    moveto "$r" "$c"
    printf "${_bdr_clr}│${_NC}\033[K"
    if [ "$i" -eq "$selected" ]; then
      if [ "$i" -lt "${#projects[@]}" ]; then
        printf "  ${_acc_clr}▎${_NC} ${_DIM}%d${_NC}  ${_bright_clr}${_BOLD}%s${_NC}" "$((i+1))" "$_label"
      else
        local _ai=$(( i - ${#projects[@]} ))
        printf " ${_action_bar[$_ai]}▎${_NC}${menu_hi[$i]}${_BOLD} %s  %s ${_NC}" "${_action_hints[$_ai]}" "$_label"
      fi
    else
      if [ "$i" -lt "${#projects[@]}" ]; then
        printf "    ${_DIM}%d${_NC}  %s" "$((i+1))" "$_label"
      else
        printf "    ${_DIM}%s${_NC}  %s" "${_action_hints[$((i - ${#projects[@]}))]}" "$_label"
      fi
    fi
    _rbdr "$r"
    r=$((r+1))

    # Subtitle line
    moveto "$r" "$c"
    printf "${_bdr_clr}│${_NC}\033[K"
    if [ -n "${menu_subs[$i]}" ]; then
      local _sub="${menu_subs[$i]}"
      local _max_sub=$(( _inner_w - 7 ))
      if [ "${#_sub}" -gt "$_max_sub" ]; then
        local _half=$(( (_max_sub - 3) / 2 ))
        _sub="${_sub:0:$_half}...${_sub: -$_half}"
      fi
      if [ "$i" -eq "$selected" ]; then
        printf "      ${_acc_clr}%s${_NC}" "$_sub"
      else
        printf "      ${_DIM}%s${_NC}" "$_sub"
      fi
    fi
    _rbdr "$r"
    r=$((r+1))
  done

  # ── Separator before help ──
  moveto "$r" "$c"
  printf "${_bdr_clr}├%s┤${_NC}\033[K" "$_hline"
  r=$((r+1))

  # ── Help row ──
  moveto "$r" "$c"
  printf "${_bdr_clr}│${_NC}\033[K"
  if [ ${#AI_TOOLS_AVAILABLE[@]} -gt 1 ]; then
    printf " ${_DIM}↑↓${_NC} navigate ${_DIM}←→${_NC} AI tool ${_DIM}S${_NC} settings ${_DIM}⏎${_NC} select "
  else
    printf " ${_DIM}↑↓${_NC} navigate ${_DIM}S${_NC} settings ${_DIM}⏎${_NC} select "
  fi
  _rbdr "$r"
  r=$((r+1))

  # ── Bottom border ──
  moveto "$r" "$c"
  printf "${_bdr_clr}└%s┘${_NC}\033[K" "$_hline"

  # ── Logo ──
  if [ "$_LOGO_LAYOUT" != "hidden" ]; then
    local ghost_display=$(get_ghost_display_setting)
    # Only draw static ghost if not set to "none"
    [ "$ghost_display" != "none" ] && draw_logo "$_logo_row" "$_logo_col" "$SELECTED_AI_TOOL"
  fi
}
