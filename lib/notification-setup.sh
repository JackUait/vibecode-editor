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

# Check if sound notifications are enabled for the given AI tool.
# Usage: is_sound_enabled <tool> <config_dir>
# Outputs "true" or "false".
is_sound_enabled() {
  local tool="$1" config_dir="$2"
  local features_file="$config_dir/${tool}-features.json"
  if [ -f "$features_file" ]; then
    local val
    val="$(python3 -c "
import json, sys
try:
    d = json.load(open(sys.argv[1]))
    print('true' if d.get('sound') else 'false')
except Exception:
    print('false')
" "$features_file" 2>/dev/null)"
    echo "${val:-false}"
  else
    echo "false"
  fi
}

# Set sound feature flag for the given AI tool.
# Usage: set_sound_feature_flag <tool> <config_dir> <true|false>
set_sound_feature_flag() {
  local tool="$1" config_dir="$2" enabled="$3"
  local features_file="$config_dir/${tool}-features.json"
  mkdir -p "$config_dir"
  python3 -c "
import json, sys, os
path = sys.argv[1]
enabled = sys.argv[2] == 'true'
try:
    d = json.load(open(path))
except Exception:
    d = {}
d['sound'] = enabled
with open(path, 'w') as f:
    json.dump(d, f)
    f.write('\n')
" "$features_file" "$enabled"
}

# Remove sound notification hook from Claude settings.
# Usage: remove_sound_notification <settings_path> <sound_command>
remove_sound_notification() {
  local settings_path="$1" sound_command="$2"
  local result
  result="$(remove_sound_notification_hook "$settings_path" "$sound_command")"
  echo "$result"
}
