#!/bin/bash
# TUI wrapper for project selection menu
# Uses ghost-tab-tui select-project subcommand

# Interactive project selection using ghost-tab-tui
# Returns 0 if selected, 1 if cancelled/quit
# Sets: _selected_project_name, _selected_project_path, _selected_project_action
select_project_interactive() {
  local projects_file="$1"

  if ! command -v ghost-tab-tui &>/dev/null; then
    error "ghost-tab-tui binary not found. Please reinstall."
    return 1
  fi

  local result
  if ! result=$(ghost-tab-tui select-project --projects-file "$projects_file" 2>/dev/null); then
    return 1
  fi

  local selected
  if ! selected=$(echo "$result" | jq -r '.selected' 2>/dev/null); then
    error "Failed to parse project selection response"
    return 1
  fi

  # Validate null/empty
  if [[ -z "$selected" || "$selected" == "null" ]]; then
    error "TUI returned invalid selection status"
    return 1
  fi

  if [[ "$selected" != "true" ]]; then
    return 1
  fi

  local name path action
  if ! name=$(echo "$result" | jq -r '.name' 2>/dev/null); then
    error "Failed to parse project name"
    return 1
  fi

  if ! path=$(echo "$result" | jq -r '.path' 2>/dev/null); then
    error "Failed to parse project path"
    return 1
  fi

  if ! action=$(echo "$result" | jq -r '.action // ""' 2>/dev/null); then
    error "Failed to parse project action"
    return 1
  fi

  # Validate null/empty for required fields
  if [[ -z "$name" || "$name" == "null" ]]; then
    error "TUI returned invalid project name"
    return 1
  fi

  if [[ -z "$path" || "$path" == "null" ]]; then
    error "TUI returned invalid project path"
    return 1
  fi

  # Action can be empty for regular project selection
  if [[ "$action" == "null" ]]; then
    action=""
  fi

  _selected_project_name="$name"
  _selected_project_path="$path"
  _selected_project_action="$action"

  return 0
}
