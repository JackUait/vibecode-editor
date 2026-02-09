#!/bin/bash
# Notification setup — sound hooks.
# Depends on: tui.sh (success, warn), settings-json.sh (add_sound_notification_hook)

# Add sound notification hook to Claude settings.
# Usage: setup_sound_notification <settings_path> <sound_command>
setup_sound_notification() {
  local settings_path="$1" sound_command="$2"
  local result
  result="$(add_sound_notification_hook "$settings_path" "$sound_command")"
  if [ "$result" = "added" ]; then
    success "Sound notification configured"
  elif [ "$result" = "exists" ]; then
    success "Sound notification already configured"
  else
    warn "Failed to configure sound notification"
    return 1
  fi
}


