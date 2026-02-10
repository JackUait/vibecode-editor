#!/bin/bash
# AI tool helper functions — pure, no side effects on source.

ai_tool_display_name() {
  case "$1" in
    claude)   echo "Claude Code" ;;
    codex)    echo "Codex CLI" ;;
    copilot)  echo "Copilot CLI" ;;
    opencode) echo "OpenCode" ;;
    *)        echo "$1" ;;
  esac
}

ai_tool_color() {
  case "$1" in
    claude)   printf '\033[38;5;209m' ;;
    codex)    printf '\033[38;5;114m' ;;
    copilot)  printf '\033[38;5;141m' ;;
    opencode) printf '\033[38;5;250m' ;;
    *)        printf '\033[0;36m' ;;
  esac
}

ai_tool_dim_color() {
  case "$1" in
    claude)   printf '\033[2;38;5;209m' ;;
    codex)    printf '\033[2;38;5;114m' ;;
    copilot)  printf '\033[2;38;5;141m' ;;
    opencode) printf '\033[2;38;5;244m' ;;
    *)        printf '\033[2;36m' ;;
  esac
}

ai_tool_bright_color() {
  case "$1" in
    claude)   printf '\033[38;5;209m' ;;
    codex)    printf '\033[38;5;114m' ;;
    copilot)  printf '\033[38;5;141m' ;;
    opencode) printf '\033[1;38;5;255m' ;;
    *)        printf '\033[1;37m' ;;
  esac
}

# Cycles SELECTED_AI_TOOL through AI_TOOLS_AVAILABLE array.
# Expects both globals to be set. Does NOT write to disk.
cycle_ai_tool() {
  local direction="$1" i
  [ ${#AI_TOOLS_AVAILABLE[@]} -le 1 ] && return
  for i in "${!AI_TOOLS_AVAILABLE[@]}"; do
    if [ "${AI_TOOLS_AVAILABLE[$i]}" == "$SELECTED_AI_TOOL" ]; then
      if [ "$direction" == "next" ]; then
        SELECTED_AI_TOOL="${AI_TOOLS_AVAILABLE[$(( (i + 1) % ${#AI_TOOLS_AVAILABLE[@]} ))]}"
      else
        SELECTED_AI_TOOL="${AI_TOOLS_AVAILABLE[$(( (i - 1 + ${#AI_TOOLS_AVAILABLE[@]}) % ${#AI_TOOLS_AVAILABLE[@]} ))]}"
      fi
      break
    fi
  done
}

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
