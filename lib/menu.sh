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
  _menu_h=$(( 3 + _update_line + total * 2 + _sep_count + 2 ))

  _top_row=$(( (_rows - _menu_h) / 2 ))
  [ "$_top_row" -lt 1 ] && _top_row=1
  _left_col=$(( (_cols - box_w) / 2 + 1 ))
  [ "$_left_col" -lt 1 ] && _left_col=1

  c="$_left_col"
  r="$_top_row"

  # Title with AI tool toggle (right-aligned to separator width)
  local _sep_w=38 _title_w=13
  moveto "$r" "$c"
  if [ ${#AI_TOOLS_AVAILABLE[@]} -gt 1 ]; then
    local _ai_name
    _ai_name="$(ai_tool_display_name "$SELECTED_AI_TOOL")"
    local _pad=$(( _sep_w - _title_w - ${#_ai_name} - 4 ))
    [ "$_pad" -lt 2 ] && _pad=2
    local _ai_clr
    _ai_clr="$(ai_tool_color "$SELECTED_AI_TOOL")"
    printf "${_BOLD}${_CYAN}⬡  Ghost Tab${_NC}%*s${_DIM}◀${_NC} ${_ai_clr}%s${_NC} ${_DIM}▶${_NC}\033[K" \
      "$_pad" "" "$_ai_name"
  elif [ ${#AI_TOOLS_AVAILABLE[@]} -eq 1 ]; then
    local _ai_name
    _ai_name="$(ai_tool_display_name "$SELECTED_AI_TOOL")"
    local _pad=$(( _sep_w - _title_w - ${#_ai_name} ))
    [ "$_pad" -lt 2 ] && _pad=2
    local _ai_clr
    _ai_clr="$(ai_tool_color "$SELECTED_AI_TOOL")"
    printf "${_BOLD}${_CYAN}⬡  Ghost Tab${_NC}%*s${_ai_clr}%s${_NC}\033[K" \
      "$_pad" "" "$_ai_name"
  else
    printf "${_BOLD}${_CYAN}⬡  Ghost Tab${_NC}\033[K"
  fi
  r=$((r+1))
  if [ -n "$_update_version" ]; then
    moveto "$r" "$c"; printf "  ${_YELLOW}Update available: v${_update_version}${_NC} ${_DIM}(brew upgrade ghost-tab)${_NC}\033[K"; r=$((r+1))
  fi
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
      local _sub="${menu_subs[$i]}"
      local _max_sub=$(( box_w - 6 ))
      if [ "${#_sub}" -gt "$_max_sub" ]; then
        local _half=$(( (_max_sub - 3) / 2 ))
        _sub="${_sub:0:$_half}...${_sub: -$_half}"
      fi
      if [ "$i" -eq "$selected" ]; then
        printf "    ${_CYAN}%s${_NC}\033[K" "$_sub"
      else
        printf "    ${_DIM}%s${_NC}\033[K" "$_sub"
      fi
    else
      printf "\033[K"
    fi
    r=$((r+1))
  done

  moveto "$r" "$c"; printf "${_DIM}──────────────────────────────────────${_NC}\033[K"; r=$((r+1))
  if [ ${#AI_TOOLS_AVAILABLE[@]} -gt 1 ]; then
    moveto "$r" "$c"; printf "${_DIM}  ↑↓${_NC} navigate  ${_DIM}←→${_NC} AI tool  ${_DIM}⏎${_NC} select\033[K"
  else
    moveto "$r" "$c"; printf "${_DIM}  ↑↓${_NC} navigate  ${_DIM}⏎${_NC} select\033[K"
  fi
}
