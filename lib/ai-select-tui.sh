#!/bin/bash
# AI tool selection TUI wrapper using ghost-tab-tui

# Interactive AI tool selection
# Returns 0 if selected, 1 if cancelled
# Sets: _selected_ai_tool
select_ai_tool_interactive() {
  if ! command -v ghost-tab-tui &>/dev/null; then
    error "ghost-tab-tui binary not found. Please reinstall."
    return 1
  fi

  local result
  if ! result=$(ghost-tab-tui select-ai-tool 2>/dev/null); then
    return 1
  fi

  local selected
  if ! selected=$(echo "$result" | jq -r '.selected' 2>/dev/null); then
    error "Failed to parse AI tool selection response"
    return 1
  fi

  # Validate against null/empty
  if [[ -z "$selected" || "$selected" == "null" ]]; then
    error "TUI returned invalid selection status"
    return 1
  fi

  if [[ "$selected" != "true" ]]; then
    return 1
  fi

  local tool
  if ! tool=$(echo "$result" | jq -r '.tool' 2>/dev/null); then
    error "Failed to parse selected tool"
    return 1
  fi

  # Validate against null/empty
  if [[ -z "$tool" || "$tool" == "null" ]]; then
    error "TUI returned invalid tool name"
    return 1
  fi

  _selected_ai_tool="$tool"

  return 0
}
