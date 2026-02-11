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
  if ! confirmed=$(echo "$result" | jq -r '.confirmed' 2>/dev/null); then
    error "Failed to parse TUI response"
    return 1
  fi

  if [[ "$confirmed" != "true" ]]; then
    return 1
  fi

  local name path
  if ! name=$(echo "$result" | jq -r '.name' 2>/dev/null); then
    error "Failed to parse project name"
    return 1
  fi

  if [[ -z "$name" || "$name" == "null" ]]; then
    error "TUI returned invalid project name"
    return 1
  fi

  if ! path=$(echo "$result" | jq -r '.path' 2>/dev/null); then
    error "Failed to parse project path"
    return 1
  fi

  if [[ -z "$path" || "$path" == "null" ]]; then
    error "TUI returned invalid project path"
    return 1
  fi

  _add_project_name="$name"
  _add_project_path="$path"

  return 0
}
