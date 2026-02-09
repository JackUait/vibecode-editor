# AI Tool Toggle Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Let users switch between Claude Code, Codex CLI, and OpenCode from the project selector menu, with the choice persisted across sessions.

**Architecture:** Add AI tool installation to the setup script (`bin/ghost-tab`), then modify the wrapper script (`ghostty/claude-wrapper.sh`) to show a toggle row in the project selector and dispatch the selected tool when launching a tmux session. Preference stored in a plain text file.

**Tech Stack:** Bash, tmux

---

### Task 1: Add AI tool preference file reading to wrapper script

**Files:**
- Modify: `ghostty/claude-wrapper.sh:1-8`

**Step 1: Add preference file path and tool detection variables after existing PATH/command setup**

At line 8 (after the `CLAUDE_CMD` line), add:

```bash
CODEX_CMD="$(command -v codex)"
OPENCODE_CMD="$(command -v opencode)"

# AI tool preference
AI_TOOL_PREF_FILE="${XDG_CONFIG_HOME:-$HOME/.config}/ghost-tab/ai-tool"
AI_TOOLS_AVAILABLE=()
[ -n "$CLAUDE_CMD" ] && AI_TOOLS_AVAILABLE+=("claude")
[ -n "$CODEX_CMD" ] && AI_TOOLS_AVAILABLE+=("codex")
[ -n "$OPENCODE_CMD" ] && AI_TOOLS_AVAILABLE+=("opencode")

# Read saved preference, default to first available
SELECTED_AI_TOOL=""
if [ -f "$AI_TOOL_PREF_FILE" ]; then
  SELECTED_AI_TOOL="$(cat "$AI_TOOL_PREF_FILE" 2>/dev/null | tr -d '[:space:]')"
fi
# Validate saved preference is still installed
_valid=0
for _t in "${AI_TOOLS_AVAILABLE[@]}"; do
  [ "$_t" == "$SELECTED_AI_TOOL" ] && _valid=1
done
if [ "$_valid" -eq 0 ] && [ ${#AI_TOOLS_AVAILABLE[@]} -gt 0 ]; then
  SELECTED_AI_TOOL="${AI_TOOLS_AVAILABLE[0]}"
fi
```

**Step 2: Verify the file saves correctly**

Run: `bash -n ghostty/claude-wrapper.sh`
Expected: no output (no syntax errors)

**Step 3: Commit**

```bash
git add ghostty/claude-wrapper.sh
git commit -m "feat: add AI tool detection and preference reading to wrapper"
```

---

### Task 2: Add AI tool toggle row to the project selector menu

**Files:**
- Modify: `ghostty/claude-wrapper.sh` — `draw_menu()` function (lines 319-397)

**Step 1: Add an `ai_tool_display_name` helper function before `draw_menu`**

Insert before the `draw_menu()` function (around line 319):

```bash
ai_tool_display_name() {
  case "$1" in
    claude)   echo "Claude Code" ;;
    codex)    echo "Codex CLI" ;;
    opencode) echo "OpenCode" ;;
    *)        echo "$1" ;;
  esac
}
```

**Step 2: Add the toggle row rendering inside `draw_menu()`, after the project/action items loop ends and before the bottom separator**

Find the lines (currently ~395-397):

```bash
      moveto "$r" "$c"; printf "${_DIM}──────────────────────────────────────${_NC}\033[K"; r=$((r+1))
      moveto "$r" "$c"; printf "${_DIM}  ↑↓${_NC} navigate  ${_DIM}⏎${_NC} select\033[K"
```

Replace with:

```bash
      # AI tool toggle row
      if [ ${#AI_TOOLS_AVAILABLE[@]} -gt 0 ]; then
        moveto "$r" "$c"; printf "\033[K"; r=$((r+1))
        _ai_row=$r
        local _ai_name
        _ai_name="$(ai_tool_display_name "$SELECTED_AI_TOOL")"
        moveto "$r" "$c"
        if [ ${#AI_TOOLS_AVAILABLE[@]} -gt 1 ]; then
          printf "  ${_DIM}◀${_NC}  AI: ${_BOLD}${_CYAN}%s${_NC}  ${_DIM}▶${_NC}\033[K" "$_ai_name"
        else
          printf "     AI: ${_BOLD}${_CYAN}%s${_NC}\033[K" "$_ai_name"
        fi
        r=$((r+1))
      fi

      moveto "$r" "$c"; printf "${_DIM}──────────────────────────────────────${_NC}\033[K"; r=$((r+1))
      if [ ${#AI_TOOLS_AVAILABLE[@]} -gt 1 ]; then
        moveto "$r" "$c"; printf "${_DIM}  ↑↓${_NC} navigate  ${_DIM}◀▶${_NC} AI tool  ${_DIM}⏎${_NC} select\033[K"
      else
        moveto "$r" "$c"; printf "${_DIM}  ↑↓${_NC} navigate  ${_DIM}⏎${_NC} select\033[K"
      fi
```

**Step 3: Update the `_menu_h` calculation in `draw_menu()` to account for the toggle row**

Find (around line 333):

```bash
      _menu_h=$(( 3 + _update_line + total * 2 + _sep_count + 2 ))
```

Replace with:

```bash
      _ai_toggle_h=0
      [ ${#AI_TOOLS_AVAILABLE[@]} -gt 0 ] && _ai_toggle_h=2
      _menu_h=$(( 3 + _update_line + total * 2 + _sep_count + _ai_toggle_h + 2 ))
```

**Step 4: Verify syntax**

Run: `bash -n ghostty/claude-wrapper.sh`
Expected: no output

**Step 5: Commit**

```bash
git add ghostty/claude-wrapper.sh
git commit -m "feat: render AI tool toggle row in project selector"
```

---

### Task 3: Add left/right arrow key handling for the AI toggle

**Files:**
- Modify: `ghostty/claude-wrapper.sh` — input loop (around lines 554-574, the escape key handling block)

**Step 1: Add a `cycle_ai_tool` helper function before the input loop**

Insert before `_add_mode=0` (around line 399):

```bash
    cycle_ai_tool() {
      local direction="$1" i
      [ ${#AI_TOOLS_AVAILABLE[@]} -le 1 ] && return
      for i in "${!AI_TOOLS_AVAILABLE[@]}"; do
        if [ "${AI_TOOLS_AVAILABLE[$i]}" == "$SELECTED_AI_TOOL" ]; then
          if [ "$direction" == "next" ]; then
            SELECTED_AI_TOOL="${AI_TOOLS_AVAILABLE[$(( (i + 1) % ${#AI_TOOLS_AVAILABLE[@]} ))]}"
          else
            SELECTED_AI_TOOL="${AI_TOOLS_AVAILABLE[$(( (i - 1 + ${#AI_TOOLS_AVAILABLE[@]}) % ${#AI_TOOLS_AVAILABLE[@]} ))]}"
          fi
          break
        fi
      done
      # Save preference
      mkdir -p "$(dirname "$AI_TOOL_PREF_FILE")"
      echo "$SELECTED_AI_TOOL" > "$AI_TOOL_PREF_FILE"
    }
```

**Step 2: Add left/right arrow handling in the escape sequence case block**

Find (around lines 569-572):

```bash
          case "$_esc_seq" in
            "A") selected=$(( (selected - 1 + total) % total )); draw_menu ;;
            "B") selected=$(( (selected + 1) % total )); draw_menu ;;
          esac
```

Replace with:

```bash
          case "$_esc_seq" in
            "A") selected=$(( (selected - 1 + total) % total )); draw_menu ;;
            "B") selected=$(( (selected + 1) % total )); draw_menu ;;
            "C") cycle_ai_tool "next"; draw_menu ;;
            "D") cycle_ai_tool "prev"; draw_menu ;;
          esac
```

**Step 3: Verify syntax**

Run: `bash -n ghostty/claude-wrapper.sh`
Expected: no output

**Step 4: Commit**

```bash
git add ghostty/claude-wrapper.sh
git commit -m "feat: add left/right arrow key handling for AI tool toggle"
```

---

### Task 4: Modify tmux session launch to use the selected AI tool

**Files:**
- Modify: `ghostty/claude-wrapper.sh` — tmux session creation (lines 691-704)

**Step 1: Add a `launch_cmd_for_tool` helper function before the tmux session block**

Insert before the `"$TMUX_CMD" new-session` line (around line 691):

```bash
# Build the AI tool launch command
case "$SELECTED_AI_TOOL" in
  codex)
    AI_LAUNCH_CMD="$CODEX_CMD --cd \"$PROJECT_DIR\""
    ;;
  opencode)
    AI_LAUNCH_CMD="$OPENCODE_CMD \"$PROJECT_DIR\""
    ;;
  *)
    AI_LAUNCH_CMD="$CLAUDE_CMD $*"
    ;;
esac
```

**Step 2: Replace the hardcoded `$CLAUDE_CMD` in the tmux session creation**

Find (lines 698-699):

```bash
  split-window -h -p 50 -c "$PROJECT_DIR" \
  "$CLAUDE_CMD $*; exec bash" \; \
```

Replace with:

```bash
  split-window -h -p 50 -c "$PROJECT_DIR" \
  "$AI_LAUNCH_CMD; exec bash" \; \
```

**Step 3: Update the background watcher to detect the correct AI tool's ready prompt**

Find (lines 650-659):

```bash
(
  while true; do
    sleep 0.5
    content=$("$TMUX_CMD" capture-pane -t "$SESSION_NAME:0.1" -p 2>/dev/null)
    if echo "$content" | grep -q '>'; then
      "$TMUX_CMD" select-pane -t "$SESSION_NAME:0.1"
      break
    fi
  done
) &
```

Replace with:

```bash
(
  while true; do
    sleep 0.5
    content=$("$TMUX_CMD" capture-pane -t "$SESSION_NAME:0.1" -p 2>/dev/null)
    # All three tools show a prompt character when ready
    if echo "$content" | grep -qE '[>$❯]'; then
      "$TMUX_CMD" select-pane -t "$SESSION_NAME:0.1"
      break
    fi
  done
) &
```

**Step 4: Verify syntax**

Run: `bash -n ghostty/claude-wrapper.sh`
Expected: no output

**Step 5: Commit**

```bash
git add ghostty/claude-wrapper.sh
git commit -m "feat: launch selected AI tool in tmux session"
```

---

### Task 5: Add AI tool installation step to setup script

**Files:**
- Modify: `bin/ghost-tab` — after the Claude Code section (lines 70-113) and before the Ghostty section (lines 115-128)

**Step 1: Replace the Claude Code section with a multi-tool installation step**

Find the entire Claude Code section (lines 70-113, from `# ---------- Claude Code (native) ----------` through the closing `fi`).

Replace it with:

```bash
# ---------- AI Coding Tools ----------
header "Setting up AI coding tools..."
echo ""
echo -e "  Ghost Tab supports multiple AI coding assistants."
echo -e "  Select which ones to install:"
echo ""

# Detect already installed tools
_cc_installed=0; command -v claude &>/dev/null && _cc_installed=1
_codex_installed=0; command -v codex &>/dev/null && _codex_installed=1
_oc_installed=0; command -v opencode &>/dev/null && _oc_installed=1

# Default selections: pre-check installed tools, always pre-check Claude Code
_sel_claude=1
_sel_codex=$_codex_installed
_sel_opencode=$_oc_installed

# Display multi-select menu
_selecting=1
_cursor=0
while [ "$_selecting" -eq 1 ]; do
  # Move cursor up to redraw (skip first draw)
  [ -n "$_drawn" ] && printf '\033[4A'
  _drawn=1

  for _i in 0 1 2; do
    case $_i in
      0) _name="Claude Code"; _sel=$_sel_claude; _tag="" ;;
      1) _name="Codex CLI (OpenAI)"; _sel=$_sel_codex; _tag="" ;;
      2) _name="OpenCode (anomalyco)"; _sel=$_sel_opencode; _tag="" ;;
    esac
    case $_i in
      0) [ "$_cc_installed" -eq 1 ] && _tag=" ${YELLOW}(installed)${NC}" ;;
      1) [ "$_codex_installed" -eq 1 ] && _tag=" ${YELLOW}(installed)${NC}" ;;
      2) [ "$_oc_installed" -eq 1 ] && _tag=" ${YELLOW}(installed)${NC}" ;;
    esac

    if [ "$_i" -eq "$_cursor" ]; then
      if [ "$_sel" -eq 1 ]; then
        echo -e "  ${BOLD}❯ [x] ${_name}${NC}${_tag}"
      else
        echo -e "  ${BOLD}❯ [ ] ${_name}${NC}${_tag}"
      fi
    else
      if [ "$_sel" -eq 1 ]; then
        echo -e "    [x] ${_name}${_tag}"
      else
        echo -e "    [ ] ${_name}${_tag}"
      fi
    fi
  done
  echo -e "  ${BLUE}↑↓${NC} navigate  ${BLUE}Space${NC} toggle  ${BLUE}Enter${NC} confirm"

  read -rsn1 _key </dev/tty
  if [[ "$_key" == $'\x1b' ]]; then
    read -rsn1 _s1 </dev/tty
    if [[ "$_s1" == "[" ]]; then
      read -rsn1 _s2 </dev/tty
      case "$_s2" in
        A) _cursor=$(( (_cursor - 1 + 3) % 3 )) ;;
        B) _cursor=$(( (_cursor + 1) % 3 )) ;;
      esac
    fi
  elif [[ "$_key" == " " ]]; then
    case $_cursor in
      0) _sel_claude=$(( 1 - _sel_claude )) ;;
      1) _sel_codex=$(( 1 - _sel_codex )) ;;
      2) _sel_opencode=$(( 1 - _sel_opencode )) ;;
    esac
  elif [[ "$_key" == "" ]]; then
    # Require at least one selection
    if [ $(( _sel_claude + _sel_codex + _sel_opencode )) -eq 0 ]; then
      echo -e "  ${RED}✗${NC} Select at least one AI tool"
      sleep 0.8
      printf '\033[1A\033[K'
    else
      _selecting=0
    fi
  fi
done

echo ""

# Install Claude Code
if [ "$_sel_claude" -eq 1 ] && [ "$_cc_installed" -eq 0 ]; then
  CLAUDE_NATIVE="$HOME/.local/bin/claude"
  info "Installing Claude Code..."
  if curl -fsSL https://claude.ai/install.sh | bash; then
    success "Claude Code installed"
    info "Run 'claude' to authenticate before opening Ghostty."
  else
    warn "Claude Code installation failed — install manually: curl -fsSL https://claude.ai/install.sh | bash"
  fi
elif [ "$_sel_claude" -eq 1 ]; then
  success "Claude Code already installed"
fi

# Install Codex CLI
if [ "$_sel_codex" -eq 1 ] && [ "$_codex_installed" -eq 0 ]; then
  info "Installing Codex CLI..."
  if brew install --cask codex; then
    success "Codex CLI installed"
  else
    warn "Codex CLI installation failed — install manually: brew install --cask codex"
  fi
elif [ "$_sel_codex" -eq 1 ]; then
  success "Codex CLI already installed"
fi

# Install OpenCode
if [ "$_sel_opencode" -eq 1 ] && [ "$_oc_installed" -eq 0 ]; then
  info "Installing OpenCode..."
  if brew install anomalyco/tap/opencode; then
    success "OpenCode installed"
  else
    warn "OpenCode installation failed — install manually: brew install anomalyco/tap/opencode"
  fi
elif [ "$_sel_opencode" -eq 1 ]; then
  success "OpenCode already installed"
fi

# Save default AI tool preference (first selected tool)
AI_TOOL_PREF_DIR="${XDG_CONFIG_HOME:-$HOME/.config}/ghost-tab"
mkdir -p "$AI_TOOL_PREF_DIR"
if [ "$_sel_claude" -eq 1 ]; then
  echo "claude" > "$AI_TOOL_PREF_DIR/ai-tool"
elif [ "$_sel_codex" -eq 1 ]; then
  echo "codex" > "$AI_TOOL_PREF_DIR/ai-tool"
elif [ "$_sel_opencode" -eq 1 ]; then
  echo "opencode" > "$AI_TOOL_PREF_DIR/ai-tool"
fi
success "Default AI tool set to $(cat "$AI_TOOL_PREF_DIR/ai-tool")"
```

**Step 2: Verify syntax**

Run: `bash -n bin/ghost-tab`
Expected: no output

**Step 3: Commit**

```bash
git add bin/ghost-tab
git commit -m "feat: add multi-tool AI installation step to setup"
```

---

### Task 6: Conditionally set up status line based on selected AI tool

**Files:**
- Modify: `bin/ghost-tab` — status line section (lines 235-383)

**Step 1: Wrap the entire Claude Code status line section in a conditional**

Find (line 236):

```bash
header "Setting up Claude Code status line..."
```

Replace the header and wrap the section:

```bash
if [ "$_sel_claude" -eq 1 ]; then
header "Setting up Claude Code status line..."
```

Find the closing `fi` of the status line section (around line 383 — the line that reads just `fi` after the `npm list -g ccstatusline` block). After that `fi`, add another `fi` plus an else branch:

```bash
else
  header "Skipping Claude Code status line..."
  info "Status line features are only available with Claude Code"
fi
```

**Step 2: Verify syntax**

Run: `bash -n bin/ghost-tab`
Expected: no output

**Step 3: Commit**

```bash
git add bin/ghost-tab
git commit -m "feat: conditionally set up status line for Claude Code only"
```

---

### Task 7: Update setup summary to show installed AI tools

**Files:**
- Modify: `bin/ghost-tab` — summary section (lines 604-621)

**Step 1: Add AI tools to the summary output**

Find (around lines 607-608):

```bash
success "Wrapper script:  ~/.config/ghostty/claude-wrapper.sh"
success "Ghostty config:  ~/.config/ghostty/config"
```

After the Ghostty config line, add:

```bash
_ai_default="$(cat "${XDG_CONFIG_HOME:-$HOME/.config}/ghost-tab/ai-tool" 2>/dev/null)"
success "AI tool:         $(echo "$_ai_default" | sed 's/claude/Claude Code/;s/codex/Codex CLI/;s/opencode/OpenCode/')"
```

**Step 2: Verify syntax**

Run: `bash -n bin/ghost-tab`
Expected: no output

**Step 3: Commit**

```bash
git add bin/ghost-tab
git commit -m "feat: show selected AI tool in setup summary"
```

---

### Task 8: End-to-end manual testing

**Step 1: Verify wrapper script syntax**

Run: `bash -n ghostty/claude-wrapper.sh`
Expected: no output

**Step 2: Verify setup script syntax**

Run: `bash -n bin/ghost-tab`
Expected: no output

**Step 3: Test preference file round-trip**

Run:
```bash
mkdir -p ~/.config/ghost-tab
echo "codex" > ~/.config/ghost-tab/ai-tool
cat ~/.config/ghost-tab/ai-tool
```
Expected: `codex`

**Step 4: Restore preference to claude**

Run: `echo "claude" > ~/.config/ghost-tab/ai-tool`

**Step 5: Bump version**

Update `VERSION` file to `1.6.0`.

**Step 6: Final commit**

```bash
git add VERSION
git commit -m "chore: bump version to 1.6.0"
```
