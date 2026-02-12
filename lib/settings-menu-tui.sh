#!/bin/bash
# TUI wrapper for settings menu
# Uses ghost-tab-tui settings-menu subcommand

# Interactive settings menu using ghost-tab-tui
# Returns action string (or empty if quit)
settings_menu_interactive() {
  if ! command -v ghost-tab-tui &>/dev/null; then
    error "ghost-tab-tui binary not found. Please reinstall."
    return 1
  fi

  local result
  if ! result=$(ghost-tab-tui settings-menu 2>/dev/null); then
    return 1
  fi

  local action
  if ! action=$(echo "$result" | jq -r '.action' 2>/dev/null); then
    error "Failed to parse settings menu response"
    return 1
  fi

  # Validate null/empty (empty is OK for quit, but null is not)
  if [[ "$action" == "null" ]]; then
    error "TUI returned invalid action"
    return 1
  fi

  echo "$action"
  return 0
}
