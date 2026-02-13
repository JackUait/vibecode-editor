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
    print('false' if d.get('sound') is False else 'true')
except Exception:
    print('true')
" "$features_file" 2>/dev/null)"
    echo "${val:-true}"
  else
    echo "true"
  fi
}

# Get the sound name for the given AI tool.
# Returns the sound name (e.g. "Bottle") or empty string if sound is disabled.
# Usage: get_sound_name <tool> <config_dir>
get_sound_name() {
  local tool="$1" config_dir="$2"
  local features_file="$config_dir/${tool}-features.json"
  if [ -f "$features_file" ]; then
    python3 -c "
import json, sys
try:
    d = json.load(open(sys.argv[1]))
    if d.get('sound') is False:
        print('')
    else:
        print(d.get('sound_name', 'Bottle'))
except Exception:
    print('Bottle')
" "$features_file" 2>/dev/null
  else
    echo "Bottle"
  fi
}

# Set the sound name for the given AI tool.
# Usage: set_sound_name <tool> <config_dir> <name>
set_sound_name() {
  local tool="$1" config_dir="$2" name="$3"
  local features_file="$config_dir/${tool}-features.json"
  mkdir -p "$config_dir"
  python3 -c "
import json, sys
path = sys.argv[1]
name = sys.argv[2]
try:
    d = json.load(open(path))
except Exception:
    d = {}
d['sound_name'] = name
with open(path, 'w') as f:
    json.dump(d, f)
    f.write('\n')
" "$features_file" "$name"
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

# Toggle sound notification for the given AI tool.
# Usage: toggle_sound_notification <tool> <config_dir> <settings_path>
# Reads current state, flips it, applies the change.
toggle_sound_notification() {
  local tool="$1" config_dir="$2" settings_path="$3"
  local current
  current="$(is_sound_enabled "$tool" "$config_dir")"
  local sound_command="afplay /System/Library/Sounds/Bottle.aiff &"

  if [[ "$current" == "true" ]]; then
    # Disable
    set_sound_feature_flag "$tool" "$config_dir" false
    case "$tool" in
      claude)
        remove_sound_notification "$settings_path" "$sound_command"
        ;;
    esac
    success "Sound notifications disabled"
  else
    # Enable
    set_sound_feature_flag "$tool" "$config_dir" true
    case "$tool" in
      claude)
        setup_sound_notification "$settings_path" "$sound_command"
        ;;
    esac
    success "Sound notifications enabled"
  fi
}
