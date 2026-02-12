#!/bin/bash
# TUI wrapper for main menu
# Uses ghost-tab-tui main-menu subcommand

# Interactive project selection using ghost-tab-tui main-menu
# Returns 0 if an actionable item was selected, 1 if quit/cancelled
# Sets: _selected_project_name, _selected_project_path, _selected_project_action, _selected_ai_tool
select_project_interactive() {
  local projects_file="$1"

  if ! command -v ghost-tab-tui &>/dev/null; then
    error "ghost-tab-tui binary not found. Please reinstall."
    return 1
  fi

  # Read preferences from settings file
  local ghost_display="animated"
  local tab_title="full"
  local settings_file="${XDG_CONFIG_HOME:-$HOME/.config}/ghost-tab/settings"
  if [ -f "$settings_file" ]; then
    local saved_display
    saved_display=$(grep '^ghost_display=' "$settings_file" 2>/dev/null | cut -d= -f2)
    if [ -n "$saved_display" ]; then
      ghost_display="$saved_display"
    fi
    local saved_tab_title
    saved_tab_title=$(grep '^tab_title=' "$settings_file" 2>/dev/null | cut -d= -f2)
    if [ -n "$saved_tab_title" ]; then
      tab_title="$saved_tab_title"
    fi
  fi

  # Read sound notification state
  local sound_enabled="false"
  local gt_config_dir="${XDG_CONFIG_HOME:-$HOME/.config}/ghost-tab"
  if type is_sound_enabled &>/dev/null; then
    sound_enabled="$(is_sound_enabled "${SELECTED_AI_TOOL:-claude}" "$gt_config_dir")"
  fi

  # Build AI tools comma-separated list
  local ai_tools_csv
  ai_tools_csv=$(IFS=,; echo "${AI_TOOLS_AVAILABLE[*]}")

  # Build command args
  local cmd_args=("main-menu" "--projects-file" "$projects_file")
  cmd_args+=("--ai-tool" "${SELECTED_AI_TOOL:-claude}")
  cmd_args+=("--ai-tools" "$ai_tools_csv")
  cmd_args+=("--ghost-display" "$ghost_display")
  cmd_args+=("--tab-title" "$tab_title")
  if [[ "$sound_enabled" == "true" ]]; then
    cmd_args+=("--sound-enabled")
  fi
  if [ -n "${_update_version:-}" ]; then
    cmd_args+=("--update-version" "$_update_version")
  fi

  local result
  if ! result=$(ghost-tab-tui "${cmd_args[@]}" 2>/dev/null); then
    return 1
  fi

  local action
  if ! action=$(echo "$result" | jq -r '.action' 2>/dev/null); then
    error "Failed to parse menu response"
    return 1
  fi

  if [[ -z "$action" || "$action" == "null" ]]; then
    return 1
  fi

  # Update AI tool if changed
  local ai_tool
  ai_tool=$(echo "$result" | jq -r '.ai_tool // ""' 2>/dev/null)
  if [[ -n "$ai_tool" && "$ai_tool" != "null" ]]; then
    _selected_ai_tool="$ai_tool"
    # Persist for next session if tool changed
    if [[ "$ai_tool" != "${SELECTED_AI_TOOL:-}" ]]; then
      local ai_tool_file="${XDG_CONFIG_HOME:-$HOME/.config}/ghost-tab/ai-tool"
      mkdir -p "$(dirname "$ai_tool_file")"
      echo "$ai_tool" > "$ai_tool_file"
    fi
  fi

  # Persist ghost display if changed
  local ghost_display_new
  ghost_display_new=$(echo "$result" | jq -r '.ghost_display // ""' 2>/dev/null)
  if [[ -n "$ghost_display_new" && "$ghost_display_new" != "null" ]]; then
    mkdir -p "$(dirname "$settings_file")"
    if [ -f "$settings_file" ] && grep -q '^ghost_display=' "$settings_file" 2>/dev/null; then
      sed -i '' "s/^ghost_display=.*/ghost_display=$ghost_display_new/" "$settings_file"
    else
      echo "ghost_display=$ghost_display_new" >> "$settings_file"
    fi
  fi

  # Persist tab title if changed
  local tab_title_new
  tab_title_new=$(echo "$result" | jq -r '.tab_title // ""' 2>/dev/null)
  if [[ -n "$tab_title_new" && "$tab_title_new" != "null" ]]; then
    mkdir -p "$(dirname "$settings_file")"
    if [ -f "$settings_file" ] && grep -q '^tab_title=' "$settings_file" 2>/dev/null; then
      sed -i '' "s/^tab_title=.*/tab_title=$tab_title_new/" "$settings_file"
    else
      echo "tab_title=$tab_title_new" >> "$settings_file"
    fi
  fi

  # Handle sound toggle
  local sound_enabled_new
  sound_enabled_new=$(echo "$result" | jq -r '.sound_enabled // ""' 2>/dev/null)
  if [[ -n "$sound_enabled_new" && "$sound_enabled_new" != "null" ]]; then
    if type toggle_sound_notification &>/dev/null; then
      local claude_settings="$HOME/.claude/settings.json"
      toggle_sound_notification "${SELECTED_AI_TOOL:-claude}" "$gt_config_dir" "$claude_settings"
    fi
  fi

  _selected_project_action="$action"

  case "$action" in
    select-project|open-once)
      local name path
      name=$(echo "$result" | jq -r '.name' 2>/dev/null)
      path=$(echo "$result" | jq -r '.path' 2>/dev/null)

      if [[ -z "$name" || "$name" == "null" ]]; then
        error "TUI returned invalid project name"
        return 1
      fi
      if [[ -z "$path" || "$path" == "null" ]]; then
        error "TUI returned invalid project path"
        return 1
      fi

      _selected_project_name="$name"
      _selected_project_path="$path"
      return 0
      ;;
    quit)
      return 1
      ;;
    *)
      # Other actions (plain-terminal, settings)
      return 0
      ;;
  esac
}
