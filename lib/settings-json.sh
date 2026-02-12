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

# Add a sound notification hook (Stop event) to settings.json.
# Migrates old Notification.idle_prompt hooks to Stop.
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

hooks = settings.setdefault("hooks", {})

# Migrate: remove old Notification.idle_prompt entries
if "Notification" in hooks:
    hooks["Notification"] = [
        entry for entry in hooks["Notification"]
        if entry.get("matcher") != "idle_prompt"
    ]
    if not hooks["Notification"]:
        del hooks["Notification"]

# Build the Stop hook entry
new_hook_entry = {
    "hooks": [
        {
            "type": "command",
            "command": hook_cmd
        }
    ]
}

# Ensure hooks.Stop exists
stop_list = hooks.setdefault("Stop", [])

# Check if a sound hook already exists in Stop
already_exists = any(
    hook_cmd in h.get("command", "")
    for entry in stop_list
    for h in entry.get("hooks", [])
)

if not already_exists:
    stop_list.append(new_hook_entry)
    with open(settings_path, "w") as f:
        json.dump(settings, f, indent=2)
        f.write("\n")
    print("added")
else:
    print("exists")
PYEOF
}

# Remove a sound notification hook (Stop event) from settings.json.
# Outputs "removed" or "not_found".
remove_sound_notification_hook() {
  local path="$1" command="$2"
  if [ ! -f "$path" ]; then
    echo "not_found"
    return 0
  fi
  _GT_HOOK_CMD="$command" python3 - "$path" << 'PYEOF'
import json, sys, os

settings_path = sys.argv[1]
hook_cmd = os.environ["_GT_HOOK_CMD"]

try:
    with open(settings_path, "r") as f:
        settings = json.load(f)
except (json.JSONDecodeError, ValueError, FileNotFoundError):
    print("not_found")
    sys.exit(0)

hooks = settings.get("hooks", {})
stop_list = hooks.get("Stop", [])

if not stop_list:
    print("not_found")
    sys.exit(0)

# Filter out entries containing the matching command
new_stop = [
    entry for entry in stop_list
    if not any(hook_cmd in h.get("command", "") for h in entry.get("hooks", []))
]

if len(new_stop) == len(stop_list):
    print("not_found")
    sys.exit(0)

if new_stop:
    hooks["Stop"] = new_stop
else:
    del hooks["Stop"]

if not hooks:
    del settings["hooks"]

with open(settings_path, "w") as f:
    json.dump(settings, f, indent=2)
    f.write("\n")

print("removed")
PYEOF
}
