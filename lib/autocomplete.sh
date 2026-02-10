#!/bin/bash
# Path autocomplete — directory suggestions and interactive input.
# Depends on: tui.sh, menu.sh (draw_menu)

# Populate _suggestions array for the given input path.
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
  done < <(find "$dir" -mindepth 1 -maxdepth 1 -iname "${base}*" 2>/dev/null | sort -f | head -8)
}

# Draw autocomplete suggestions box below the input field.
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

# Read path with autocomplete — sets _path_result.
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
