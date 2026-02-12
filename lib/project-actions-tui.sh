#!/bin/bash
# Interactive TUI wrappers for project actions using ghost-tab-tui

# Interactive project addition using ghost-tab-tui
# Returns 0 if confirmed, 1 if cancelled
# Sets: _add_project_name, _add_project_path
add_project_interactive() {
  if ! command -v ghost-tab-tui &>/dev/null; then
    error "ghost-tab-tui binary not found. Please reinstall."
    return 1
  fi

  local result
  if ! result=$(ghost-tab-tui add-project 2>/dev/null); then
    return 1
  fi

  local confirmed
  confirmed=$(echo "$result" | jq -r '.confirmed')

  if [[ "$confirmed" != "true" ]]; then
    return 1
  fi

  _add_project_name=$(echo "$result" | jq -r '.name')
  _add_project_path=$(echo "$result" | jq -r '.path')

  return 0
}
