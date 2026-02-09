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

# Add spinner start/stop hooks to settings.json.
add_spinner_hooks() {
  local path="$1" start_cmd="$2" stop_cmd="$3"
  _GT_START_CMD="$start_cmd" _GT_STOP_CMD="$stop_cmd" python3 - "$path" << 'PYEOF'
import json, sys, os

settings_path = sys.argv[1]
start_cmd = os.environ["_GT_START_CMD"]
stop_cmd = os.environ["_GT_STOP_CMD"]

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

# Add UserPromptSubmit hook for start script (spinner while Claude works)
submit_list = hooks.setdefault("UserPromptSubmit", [])

# Find or create the matcher group (use empty string as matcher)
submit_group = None
for group in submit_list:
    if group.get("matcher") == "":
        submit_group = group
        break

if submit_group is None:
    submit_group = {"matcher": "", "hooks": []}
    submit_list.append(submit_group)

if not any(h.get("command") == start_cmd for h in submit_group["hooks"]):
    submit_group["hooks"].append({"type": "command", "command": start_cmd})

# Add idle_prompt notification for stop script (spinner stops when Claude is done)
notification_list = hooks.setdefault("Notification", [])

# Find or create idle_prompt matcher group
idle_group = None
for group in notification_list:
    if group.get("matcher") == "idle_prompt":
        idle_group = group
        break

if idle_group is None:
    idle_group = {"matcher": "idle_prompt", "hooks": []}
    notification_list.append(idle_group)

# Add stop command if not already present
if not any(h.get("command") == stop_cmd for h in idle_group["hooks"]):
    idle_group["hooks"].append({"type": "command", "command": stop_cmd})

with open(settings_path, "w") as f:
    json.dump(settings, f, indent=2)
    f.write("\n")
PYEOF
}
