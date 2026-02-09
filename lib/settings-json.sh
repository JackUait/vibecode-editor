#!/bin/bash
# Claude settings.json manipulation helpers.

# Merge statusLine into Claude settings.json (create if missing).
merge_claude_settings() {
  local path="$1"
  mkdir -p "$(dirname "$path")"
  if [ -f "$path" ]; then
    if grep -q '"statusLine"' "$path"; then
      success "Claude status line already configured"
    else
      sed -i '' '$ s/}$/,\n  "statusLine": {\n    "type": "command",\n    "command": "bash ~\/.claude\/statusline-wrapper.sh"\n  }\n}/' "$path"
      success "Added status line to Claude settings"
    fi
  else
    cat > "$path" << 'CSEOF'
{
  "statusLine": {
    "type": "command",
    "command": "bash ~/.claude/statusline-wrapper.sh"
  }
}
CSEOF
    success "Created Claude settings with status line"
  fi
}

# Add a sound notification hook (idle_prompt) to settings.json.
add_sound_notification_hook() {
  local path="$1" command="$2"
  mkdir -p "$(dirname "$path")"
  _GT_HOOK_CMD="$command" python3 - "$path" << 'PYEOF'
import json, sys, os

settings_path = sys.argv[1]
hook_cmd = os.environ["_GT_HOOK_CMD"]

# Load existing settings or start fresh
if os.path.exists(settings_path):
    try:
        with open(settings_path, "r") as f:
            settings = json.load(f)
    except (json.JSONDecodeError, ValueError):
        settings = {}
else:
    settings = {}

# Build the hook entry we want
new_hook_entry = {
    "matcher": "idle_prompt",
    "hooks": [
        {
            "type": "command",
            "command": hook_cmd
        }
    ]
}

# Ensure hooks.Notification exists
hooks = settings.setdefault("hooks", {})
notification_list = hooks.setdefault("Notification", [])

# Check if idle_prompt hook already exists
already_exists = any(
    entry.get("matcher") == "idle_prompt"
    for entry in notification_list
)

if not already_exists:
    notification_list.append(new_hook_entry)
    with open(settings_path, "w") as f:
        json.dump(settings, f, indent=2)
        f.write("\n")
    print("added")
else:
    print("exists")
PYEOF
}



