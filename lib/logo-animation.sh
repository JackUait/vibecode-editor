#!/bin/bash
# Pixel-art ghost mascots for each AI tool.
# Each function sets _LOGO_LINES (array), _LOGO_HEIGHT, _LOGO_WIDTH.
# Art uses Unicode block chars with embedded ANSI 256-color escapes.
#
# Every line is exactly 28 visible characters wide.
# Ghost silhouette (15 lines):
#   Line  0:  7sp + 14▄ + 7sp
#   Line  1:  5sp + ▄ + 16█ + ▄ + 5sp
#   Line  2:  4sp + ▄ + 18█ + ▄ + 4sp
#   Line  3:  3sp + 22█ + 3sp
#   Lines 4-12: 2sp + 24█ + 2sp
#   Line 13:  2sp + 24 (feet gaps) + 2sp
#   Line 14:  2sp + 24 (feet tips) + 2sp

# ── Helpers ─────────────────────────────────────────────────────
_c() { printf '\033[38;5;%dm' "$1"; }
_r() { printf '\033[0m'; }

# ─────────────────────────────────────────────────────────────────
# Claude Ghost — orange/warm, starburst/spark motif
# Colors: 209 orange, 208 deeper, 166 dark, 223 peach, 220 gold
# ─────────────────────────────────────────────────────────────────
logo_art_claude() {
  local O D B L W K Y R
  O=$(_c 209) D=$(_c 208) B=$(_c 166) L=$(_c 223)
  W=$(_c 255) K=$(_c 232) Y=$(_c 220) R=$(_r)

  _LOGO_LINES=(
    "${R}       ${L}▄▄▄▄▄▄▄▄▄▄▄▄▄▄${R}       "
    "${R}     ${L}▄${O}████████████████${L}▄${R}     "
    "${R}    ${L}▄${O}██████████████████${L}▄${R}    "
    "${R}   ${O}██████████████████████${R}   "
    "${R}  ${O}████████████████████████${R}  "
    "${R}  ${O}████${W}███${K}██${O}██████${W}███${K}██${O}████${R}  "
    "${R}  ${O}████${W}███${K}██${O}██████${W}███${K}██${O}████${R}  "
    "${R}  ${D}████████████████████████${R}  "
    "${R}  ${D}█████████${Y}██${D}█████████████${R}  "
    "${R}  ${D}████████${Y}█▀▀█${D}████████████${R}  "
    "${R}  ${D}████████${Y}█▄▄█${D}████████████${R}  "
    "${R}  ${D}█████████${Y}██${D}█████████████${R}  "
    "${R}  ${B}████████████████████████${R}  "
    "${R}  ${B}██ █████ ██████ █████ ██${R}  "
    "${R}  ${B}█${R}  ${B}▀████▀${R} ${B}████${R} ${B}▀████▀${R}  ${B}█${R}  "
  )

  _LOGO_HEIGHT=${#_LOGO_LINES[@]}
  _LOGO_WIDTH=28
}

# ─────────────────────────────────────────────────────────────────
# Codex Ghost — green, hexagonal/honeycomb pattern
# Colors: 114 green, 113 yellow-green, 71 dark, 157 light, 78 teal
# ─────────────────────────────────────────────────────────────────
logo_art_codex() {
  local G Y D L W K H R
  G=$(_c 114) Y=$(_c 113) D=$(_c 71) L=$(_c 157)
  W=$(_c 255) K=$(_c 232) H=$(_c 78) R=$(_r)

  _LOGO_LINES=(
    "${R}       ${L}▄▄▄▄▄▄▄▄▄▄▄▄▄▄${R}       "
    "${R}     ${L}▄${G}████████████████${L}▄${R}     "
    "${R}    ${L}▄${G}██████████████████${L}▄${R}    "
    "${R}   ${G}██████████████████████${R}   "
    "${R}  ${G}████████████████████████${R}  "
    "${R}  ${G}████${W}███${K}██${G}██████${W}███${K}██${G}████${R}  "
    "${R}  ${G}████${W}███${K}██${G}██████${W}███${K}██${G}████${R}  "
    "${R}  ${Y}████████████████████████${R}  "
    "${R}  ${Y}████${H}▄▀▀▀▄${Y}██████${H}▄▀▀▀▄${Y}████${R}  "
    "${R}  ${Y}████${H}█${Y}████${H}█${Y}████${H}█${Y}████${H}█${Y}████${R}  "
    "${R}  ${Y}████${H}▀▄▄▄▀${Y}██████${H}▀▄▄▄▀${Y}████${R}  "
    "${R}  ${Y}████████████████████████${R}  "
    "${R}  ${D}████████████████████████${R}  "
    "${R}  ${D}██ █████ ██████ █████ ██${R}  "
    "${R}  ${D}█${R}  ${D}▀████▀${R} ${D}████${R} ${D}▀████▀${R}  ${D}█${R}  "
  )

  _LOGO_HEIGHT=${#_LOGO_LINES[@]}
  _LOGO_WIDTH=28
}

# ─────────────────────────────────────────────────────────────────
# Copilot Ghost — purple, aviator goggles
# Colors: 141 purple, 140 light, 98 dark, 183 lavender, 134 magenta
# ─────────────────────────────────────────────────────────────────
logo_art_copilot() {
  local P LP DP LV W K M R
  P=$(_c 141) LP=$(_c 140) DP=$(_c 98) LV=$(_c 183)
  W=$(_c 255) K=$(_c 232) M=$(_c 134) R=$(_r)

  _LOGO_LINES=(
    "${R}       ${LV}▄▄▄▄▄▄▄▄▄▄▄▄▄▄${R}       "
    "${R}     ${LV}▄${P}████████████████${LV}▄${R}     "
    "${R}    ${LV}▄${P}██████████████████${LV}▄${R}    "
    "${R}   ${P}██████████████████████${R}   "
    "${R}  ${P}████████████████████████${R}  "
    "${R}  ${P}██${M}▄▄▄▄▄▄${P}████████${M}▄▄▄▄▄▄${P}██${R}  "
    "${R}  ${P}██${M}▌${W}██${K}█${W}██${M}▐${P}██████${M}▌${W}██${K}█${W}██${M}▐${P}██${R}  "
    "${R}  ${P}██${M}▀▀▀▀▀▀${P}████████${M}▀▀▀▀▀▀${P}██${R}  "
    "${R}  ${LP}████████████████████████${R}  "
    "${R}  ${LP}████████████████████████${R}  "
    "${R}  ${LP}████████████████████████${R}  "
    "${R}  ${LP}████████████████████████${R}  "
    "${R}  ${DP}████████████████████████${R}  "
    "${R}  ${DP}██ █████ ██████ █████ ██${R}  "
    "${R}  ${DP}█${R}  ${DP}▀████▀${R} ${DP}████${R} ${DP}▀████▀${R}  ${DP}█${R}  "
  )

  _LOGO_HEIGHT=${#_LOGO_LINES[@]}
  _LOGO_WIDTH=28
}

# ─────────────────────────────────────────────────────────────────
# OpenCode Ghost — monochrome friendly, clean and professional
# Colors: 255 white, 252 very light, 250 light, 246 med-light,
#         244 medium, 242 lower body, 240 feet, 238 eyes/smile
# ─────────────────────────────────────────────────────────────────
logo_art_opencode() {
  local W VL L ML M MD D K SM R
  W=$(_c 255) VL=$(_c 252) L=$(_c 250) ML=$(_c 246)
  M=$(_c 244) MD=$(_c 242) D=$(_c 240) K=$(_c 238)
  SM=$(_c 240) R=$(_r)

  _LOGO_LINES=(
    "${R}       ${VL}▄▄▄▄▄▄▄▄▄▄▄▄▄▄${R}       "
    "${R}     ${VL}▄${W}████████████████${VL}▄${R}     "
    "${R}    ${VL}▄${W}██████████████████${VL}▄${R}    "
    "${R}   ${L}██████████████████████${R}   "
    "${R}  ${L}████████████████████████${R}  "
    "${R}  ${ML}████${W}███${K}██${ML}██████${W}███${K}██${ML}████${R}  "
    "${R}  ${ML}████${W}███${K}██${ML}██████${W}███${K}██${ML}████${R}  "
    "${R}  ${M}████████████████████████${R}  "
    "${R}  ${M}████████████████████████${R}  "
    "${R}  ${M}████████${SM}█▀▀█${M}████████████${R}  "
    "${R}  ${M}████████████████████████${R}  "
    "${R}  ${MD}████████████████████████${R}  "
    "${R}  ${MD}████████████████████████${R}  "
    "${R}  ${D}██ █████ ██████ █████ ██${R}  "
    "${R}  ${D}█${R}  ${D}▀████▀${R} ${D}████${R} ${D}▀████▀${R}  ${D}█${R}  "
  )

  _LOGO_HEIGHT=${#_LOGO_LINES[@]}
  _LOGO_WIDTH=28
}

# ─────────────────────────────────────────────────────────────────
# Rendering & animation helpers
# ─────────────────────────────────────────────────────────────────

# draw_logo row col tool_name
#   Populate _LOGO_LINES via logo_art_<tool_name>, then render each
#   line at the given terminal position using moveto (from tui.sh).
draw_logo() {
  local row=$1 col=$2 tool=$3 line
  "logo_art_${tool}"
  for line in "${_LOGO_LINES[@]}"; do
    moveto "$row" "$col"
    printf '%b' "$line"
    row=$((row + 1))
  done
}

# clear_logo_area row col height width
#   Overwrite the rectangular region with spaces to erase the logo.
clear_logo_area() {
  local row=$1 col=$2 height=$3 width=$4
  local end=$((row + height - 1)) blank
  blank=$(printf "%*s" "$width" "")
  while [ "$row" -le "$end" ]; do
    moveto "$row" "$col"
    printf '%s' "$blank"
    row=$((row + 1))
  done
}

# Bob offsets following a sine-wave pattern (range 0–1 rows).
# 14 entries: slow at extremes (ease), smooth gentle bob.
# Max 1-row difference between consecutive entries.
_BOB_OFFSETS=(0 0 0 0 1 1 1 1 1 1 0 0 0 0)
_BOB_MAX=1

# start_logo_animation row col tool_name
#   Launch a background bobbing animation: the ghost floats smoothly
#   following a sine-wave offset pattern with 0.18 s per step.
#   A flag file gates the loop so stop_logo_animation can halt it.
start_logo_animation() {
  local row=$1 col=$2 tool=$3
  local flagfile="/tmp/ghost-tab-anim-$$"

  touch "$flagfile"
  # Pre-populate art arrays so the subshell inherits them
  "logo_art_${tool}"

  (
    local prev_off=-1 off idx=0 n=${#_BOB_OFFSETS[@]}
    while [ -f "$flagfile" ]; do
      off=${_BOB_OFFSETS[$idx]}
      if [ "$off" -ne "$prev_off" ]; then
        # Draw new position first to minimize flashing
        draw_logo $((row + off)) "$col" "$tool"

        # Clear only the exposed row
        if [ "$prev_off" -ge 0 ]; then
          if [ "$off" -gt "$prev_off" ]; then
            # Moving down: clear top exposed row
            clear_logo_area $((row + prev_off)) "$col" 1 "$_LOGO_WIDTH"
          else
            # Moving up: clear bottom exposed row (last line of old position)
            clear_logo_area $((row + prev_off + _LOGO_HEIGHT - 1)) "$col" 1 "$_LOGO_WIDTH"
          fi
        fi

        prev_off=$off
      fi
      sleep 0.18
      idx=$(( (idx + 1) % n ))
    done
  ) &

  _LOGO_ANIM_PID=$!
  _LOGO_CUR_ROW=$row
  _LOGO_CUR_COL=$col
  _LOGO_CUR_TOOL=$tool
}

# stop_logo_animation
#   Tear down the background animation and erase the ghost at all
#   possible bob positions so no artefacts remain on screen.
stop_logo_animation() {
  rm -f "/tmp/ghost-tab-anim-$$"

  if [ -n "$_LOGO_ANIM_PID" ]; then
    kill "$_LOGO_ANIM_PID" 2>/dev/null
    wait "$_LOGO_ANIM_PID" 2>/dev/null
  fi

  if [ -n "$_LOGO_CUR_ROW" ]; then
    local i
    for i in $(seq 0 "$_BOB_MAX"); do
      clear_logo_area $((_LOGO_CUR_ROW + i)) "$_LOGO_CUR_COL" "$_LOGO_HEIGHT" "$_LOGO_WIDTH"
    done
  fi

  unset _LOGO_ANIM_PID
}
