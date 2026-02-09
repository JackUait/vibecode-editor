#!/bin/bash
# Visual test for ghost mascots — run to preview all four ghosts.
source "$(dirname "$0")/../lib/logo-animation.sh"

print_ghost() {
  local name="$1"
  printf "\n  === %s (h=%d w=%d) ===\n\n" "$name" "$_LOGO_HEIGHT" "$_LOGO_WIDTH"
  for line in "${_LOGO_LINES[@]}"; do
    printf "  %b\n" "$line"
  done

  # Verify all lines are the expected width
  local i=0 ok=1
  for line in "${_LOGO_LINES[@]}"; do
    local stripped
    stripped=$(printf "%b" "$line" | sed 's/\x1b\[[0-9;]*m//g')
    local w=${#stripped}
    if [ "$w" -ne "$_LOGO_WIDTH" ]; then
      printf "  !! Line %d: width=%d (expected %d)\n" "$i" "$w" "$_LOGO_WIDTH"
      ok=0
    fi
    ((i++))
  done
  [ "$ok" -eq 1 ] && printf "  [OK] All %d lines verified at width %d\n" "$_LOGO_HEIGHT" "$_LOGO_WIDTH"
}

logo_art_claude;   print_ghost "Claude Ghost (orange/starburst)"
logo_art_codex;    print_ghost "Codex Ghost (green/hexagons)"
logo_art_copilot;  print_ghost "Copilot Ghost (purple/goggles)"
logo_art_opencode; print_ghost "OpenCode Ghost (blue/brackets)"

echo ""
