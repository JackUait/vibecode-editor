# Sound Notification Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add a setup prompt that optionally configures a Claude Code hook to play a sound when Claude finishes generating.

**Architecture:** A new section in `bin/ghost-tab` after the status line setup (line 323) asks y/n. If yes, uses Python's `json` module (ships with macOS) to safely merge the hook into `~/.claude/settings.json` without clobbering existing keys.

**Tech Stack:** Bash, Python 3 `json` module (macOS built-in)

---

### Task 1: Add sound notification prompt and hook logic to setup script

**Files:**
- Modify: `bin/ghost-tab:323` (insert new section before the summary block)

**Step 1: Add the sound notification section**

Insert the following block in `bin/ghost-tab` between the end of the status line section (after the last `fi` on line 323) and the summary header on line 325. This new section:

1. Asks the user y/n
2. If yes, uses inline Python to merge the hook into settings.json (handles all edge cases: no file, no hooks key, hooks key exists, idle_prompt already present)
3. If no, skips

```bash
# ---------- Sound Notification ----------
header "Sound notification..."
echo ""
echo -e "  Claude Code can play a sound when it finishes generating"
echo -e "  and is waiting for your input."
echo ""
read -rn1 -p "$(echo -e "${BLUE}Enable sound notification? (y/n):${NC} ")" enable_sound </dev/tty
echo ""

if [[ "$enable_sound" =~ ^[yY]$ ]]; then
  CLAUDE_SETTINGS="$HOME/.claude/settings.json"
  mkdir -p ~/.claude

  # Use Python (ships with macOS) to safely merge hook into settings.json
  python3 - "$CLAUDE_SETTINGS" << 'PYEOF'
import json, sys, os

settings_path = sys.argv[1]

# Load existing settings or start fresh
if os.path.exists(settings_path):
    with open(settings_path, "r") as f:
        settings = json.load(f)
else:
    settings = {}

# Build the hook entry we want
new_hook_entry = {
    "matcher": "idle_prompt",
    "hooks": [
        {
            "type": "command",
            "command": "afplay /System/Library/Sounds/Bottle.aiff &"
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

  result=$?
  if [ $result -eq 0 ]; then
    success "Sound notification configured"
  else
    warn "Failed to configure sound notification"
  fi
else
  info "Skipping sound notification"
fi
```

**Step 2: Update the summary section to show sound notification status**

After the existing status line summary check (line 331-333), add a line showing sound notification status:

```bash
if grep -q '"idle_prompt"' ~/.claude/settings.json 2>/dev/null; then
  success "Sound:           Notification on idle"
fi
```

**Step 3: Test the changes manually**

Run: `bash bin/ghost-tab`

Expected behavior:
- After status line setup, see the sound notification prompt
- Press `y` → hook added to `~/.claude/settings.json`, see "✓ Sound notification configured"
- Press `n` → see "→ Skipping sound notification"
- Run again, press `y` → see "✓ Sound notification configured" (idempotent, doesn't duplicate)
- Summary section shows "✓ Sound: Notification on idle" if configured

Verify `~/.claude/settings.json` contains the hook and all other keys are preserved.

**Step 4: Commit**

```bash
git add bin/ghost-tab
git commit -m "Add sound notification prompt during setup"
```
