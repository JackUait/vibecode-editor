#!/bin/bash
# AI tool helper functions — pure, no side effects on source.

# Validates SELECTED_AI_TOOL against AI_TOOLS_AVAILABLE.
# Falls back to first available if current selection is invalid.
validate_ai_tool() {
  local _valid=0 _t
  for _t in "${AI_TOOLS_AVAILABLE[@]}"; do
    [ "$_t" == "$SELECTED_AI_TOOL" ] && _valid=1
  done
  if [ "$_valid" -eq 0 ] && [ ${#AI_TOOLS_AVAILABLE[@]} -gt 0 ]; then
    SELECTED_AI_TOOL="${AI_TOOLS_AVAILABLE[0]}"
  fi
}
