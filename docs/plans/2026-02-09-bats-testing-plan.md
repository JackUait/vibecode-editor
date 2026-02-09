# Bats Testing Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add comprehensive bats-core testing by extracting pure functions from the two main shell scripts into testable library files.

**Architecture:** Extract ~15 pure/near-pure functions from `ghostty/claude-wrapper.sh` and `bin/ghost-tab` into 6 library files under `lib/`. Each lib file is sourceable with no side effects. Tests go in `test/*.bats` using bats-core with bats-assert/bats-support submodules. Main scripts source these libs and behave identically.

**Tech Stack:** Bash, bats-core, bats-assert, bats-support (git submodules)

---

### Task 1: Set Up Bats Test Infrastructure

**Files:**
- Create: `test/test_helper/common.bash`
- Create: `run-tests.sh`

**Step 1: Add bats-core git submodules**

Run:
```bash
git submodule add https://github.com/bats-core/bats-core.git test/test_helper/bats-core
git submodule add https://github.com/bats-core/bats-support.git test/test_helper/bats-support
git submodule add https://github.com/bats-core/bats-assert.git test/test_helper/bats-assert
```

Expected: Three submodule directories created, `.gitmodules` file created.

**Step 2: Create the shared test helper**

Create `test/test_helper/common.bash`:
```bash
_common_setup() {
  load 'bats-support/load'
  load 'bats-assert/load'

  PROJECT_ROOT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)"
}
```

**Step 3: Create the test runner**

Create `run-tests.sh`:
```bash
#!/bin/bash
set -e
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
"$SCRIPT_DIR/test/test_helper/bats-core/bin/bats" "$SCRIPT_DIR/test/"*.bats "$@"
```

Then: `chmod +x run-tests.sh`

**Step 4: Create a smoke test to verify infrastructure**

Create `test/smoke.bats`:
```bash
setup() {
  load 'test_helper/common'
  _common_setup
}

@test "bats infrastructure works" {
  run echo "hello"
  assert_success
  assert_output "hello"
}
```

**Step 5: Run smoke test**

Run: `./run-tests.sh`
Expected: `1 test, 0 failures`

**Step 6: Commit**

```bash
git add .gitmodules test/ run-tests.sh
git commit -m "Add bats-core test infrastructure"
```

---

### Task 2: Create lib/ai-tools.sh + Tests

**Files:**
- Create: `lib/ai-tools.sh`
- Create: `test/ai-tools.bats`
- Reference: `ghostty/claude-wrapper.sh:350-368` (display_name, color)
- Reference: `ghostty/claude-wrapper.sh:477-493` (cycle)
- Reference: `ghostty/claude-wrapper.sh:26-32` (validate)

**Step 1: Write the failing tests**

Create `test/ai-tools.bats`:
```bash
setup() {
  load 'test_helper/common'
  _common_setup
  source "$PROJECT_ROOT/lib/ai-tools.sh"
}

# --- ai_tool_display_name ---

@test "ai_tool_display_name: claude -> Claude Code" {
  run ai_tool_display_name "claude"
  assert_output "Claude Code"
}

@test "ai_tool_display_name: codex -> Codex CLI" {
  run ai_tool_display_name "codex"
  assert_output "Codex CLI"
}

@test "ai_tool_display_name: copilot -> Copilot CLI" {
  run ai_tool_display_name "copilot"
  assert_output "Copilot CLI"
}

@test "ai_tool_display_name: opencode -> OpenCode" {
  run ai_tool_display_name "opencode"
  assert_output "OpenCode"
}

@test "ai_tool_display_name: unknown passes through" {
  run ai_tool_display_name "vim"
  assert_output "vim"
}

# --- ai_tool_color ---

@test "ai_tool_color: claude returns orange ANSI" {
  result="$(ai_tool_color "claude")"
  [[ "$result" == $'\033[38;5;209m' ]]
}

@test "ai_tool_color: codex returns green ANSI" {
  result="$(ai_tool_color "codex")"
  [[ "$result" == $'\033[38;5;114m' ]]
}

@test "ai_tool_color: copilot returns purple ANSI" {
  result="$(ai_tool_color "copilot")"
  [[ "$result" == $'\033[38;5;141m' ]]
}

@test "ai_tool_color: opencode returns blue ANSI" {
  result="$(ai_tool_color "opencode")"
  [[ "$result" == $'\033[38;5;75m' ]]
}

@test "ai_tool_color: unknown returns default cyan" {
  result="$(ai_tool_color "vim")"
  [[ "$result" == $'\033[0;36m' ]]
}

# --- cycle_ai_tool ---

@test "cycle_ai_tool: next wraps from last to first" {
  AI_TOOLS_AVAILABLE=("claude" "codex" "opencode")
  SELECTED_AI_TOOL="opencode"
  cycle_ai_tool "next"
  [[ "$SELECTED_AI_TOOL" == "claude" ]]
}

@test "cycle_ai_tool: next advances by one" {
  AI_TOOLS_AVAILABLE=("claude" "codex" "opencode")
  SELECTED_AI_TOOL="claude"
  cycle_ai_tool "next"
  [[ "$SELECTED_AI_TOOL" == "codex" ]]
}

@test "cycle_ai_tool: prev wraps from first to last" {
  AI_TOOLS_AVAILABLE=("claude" "codex" "opencode")
  SELECTED_AI_TOOL="claude"
  cycle_ai_tool "prev"
  [[ "$SELECTED_AI_TOOL" == "opencode" ]]
}

@test "cycle_ai_tool: no-op with single tool" {
  AI_TOOLS_AVAILABLE=("claude")
  SELECTED_AI_TOOL="claude"
  cycle_ai_tool "next"
  [[ "$SELECTED_AI_TOOL" == "claude" ]]
}

# --- validate_ai_tool ---

@test "validate_ai_tool: keeps valid preference" {
  AI_TOOLS_AVAILABLE=("claude" "codex")
  SELECTED_AI_TOOL="codex"
  validate_ai_tool
  [[ "$SELECTED_AI_TOOL" == "codex" ]]
}

@test "validate_ai_tool: falls back to first when pref is invalid" {
  AI_TOOLS_AVAILABLE=("claude" "codex")
  SELECTED_AI_TOOL="vim"
  validate_ai_tool
  [[ "$SELECTED_AI_TOOL" == "claude" ]]
}

@test "validate_ai_tool: falls back to first when pref is empty" {
  AI_TOOLS_AVAILABLE=("codex" "opencode")
  SELECTED_AI_TOOL=""
  validate_ai_tool
  [[ "$SELECTED_AI_TOOL" == "codex" ]]
}
```

**Step 2: Run tests to verify they fail**

Run: `./run-tests.sh test/ai-tools.bats`
Expected: FAIL — `lib/ai-tools.sh: No such file or directory`

**Step 3: Write the implementation**

Create `lib/ai-tools.sh`:
```bash
#!/bin/bash
# AI tool helper functions — pure, no side effects on source.

ai_tool_display_name() {
  case "$1" in
    claude)   echo "Claude Code" ;;
    codex)    echo "Codex CLI" ;;
    copilot)  echo "Copilot CLI" ;;
    opencode) echo "OpenCode" ;;
    *)        echo "$1" ;;
  esac
}

ai_tool_color() {
  case "$1" in
    claude)   printf '\033[38;5;209m' ;;
    codex)    printf '\033[38;5;114m' ;;
    copilot)  printf '\033[38;5;141m' ;;
    opencode) printf '\033[38;5;75m' ;;
    *)        printf '\033[0;36m' ;;
  esac
}

# Cycles SELECTED_AI_TOOL through AI_TOOLS_AVAILABLE array.
# Expects both globals to be set. Does NOT write to disk.
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
}

# Validates SELECTED_AI_TOOL against AI_TOOLS_AVAILABLE.
# Falls back to first available if current selection is invalid.
validate_ai_tool() {
  local _valid=0 _t
  for _t in "${AI_TOOLS_AVAILABLE[@]}"; do
    [ "$_t" == "$SELECTED_AI_TOOL" ] && _valid=1
  done
  if [ "$_valid" -eq 0 ] && [ ${#AI_TOOLS_AVAILABLE[@]} -gt 0 ]; then
    SELECTED_AI_TOOL="${AI_TOOLS_AVAILABLE[0]}"
  fi
}
```

**Step 4: Run tests to verify they pass**

Run: `./run-tests.sh test/ai-tools.bats`
Expected: `8 tests, 0 failures` (all pass — the 5 display_name + 5 color + 4 cycle + 3 validate = 17 tests but the color tests use `[[ ]]` not `run`, so bats counts those correctly)

**Step 5: Commit**

```bash
git add lib/ai-tools.sh test/ai-tools.bats
git commit -m "Add lib/ai-tools.sh with tests"
```

---

### Task 3: Create lib/projects.sh + Tests

**Files:**
- Create: `lib/projects.sh`
- Create: `test/projects.bats`
- Reference: `ghostty/claude-wrapper.sh:120-126` (load projects)
- Reference: `ghostty/claude-wrapper.sh:133-136` (parse name/path)
- Reference: `ghostty/claude-wrapper.sh:161` (path expand)
- Reference: `ghostty/claude-wrapper.sh:452-456` (path truncate)

**Step 1: Write the failing tests**

Create `test/projects.bats`:
```bash
setup() {
  load 'test_helper/common'
  _common_setup
  source "$PROJECT_ROOT/lib/projects.sh"
  TEST_DIR="$(mktemp -d)"
}

teardown() {
  rm -rf "$TEST_DIR"
}

# --- parse_project_name ---

@test "parse_project_name: extracts name before colon" {
  run parse_project_name "myapp:/Users/me/myapp"
  assert_output "myapp"
}

@test "parse_project_name: handles name with spaces" {
  run parse_project_name "my app:/Users/me/my app"
  assert_output "my app"
}

# --- parse_project_path ---

@test "parse_project_path: extracts path after first colon" {
  run parse_project_path "myapp:/Users/me/myapp"
  assert_output "/Users/me/myapp"
}

@test "parse_project_path: handles paths with colons" {
  run parse_project_path "myapp:/Users/me/path:with:colons"
  assert_output "/Users/me/path:with:colons"
}

# --- load_projects ---

@test "load_projects: reads name:path lines" {
  cat > "$TEST_DIR/projects" << 'EOF'
app1:/path/to/app1
app2:/path/to/app2
EOF
  run load_projects "$TEST_DIR/projects"
  assert_line --index 0 "app1:/path/to/app1"
  assert_line --index 1 "app2:/path/to/app2"
}

@test "load_projects: skips blank lines" {
  cat > "$TEST_DIR/projects" << 'EOF'
app1:/path/to/app1

app2:/path/to/app2
EOF
  run load_projects "$TEST_DIR/projects"
  assert_output --partial "app1:/path/to/app1"
  assert_output --partial "app2:/path/to/app2"
  [ "$(echo "$output" | wc -l | tr -d ' ')" -eq 2 ]
}

@test "load_projects: skips comment lines" {
  cat > "$TEST_DIR/projects" << 'EOF'
# This is a comment
app1:/path/to/app1
EOF
  run load_projects "$TEST_DIR/projects"
  assert_output "app1:/path/to/app1"
}

@test "load_projects: returns empty for missing file" {
  run load_projects "$TEST_DIR/nonexistent"
  assert_output ""
}

# --- path_expand ---

@test "path_expand: converts ~ to HOME" {
  run path_expand "~/projects/app"
  assert_output "$HOME/projects/app"
}

@test "path_expand: leaves absolute paths unchanged" {
  run path_expand "/usr/local/bin"
  assert_output "/usr/local/bin"
}

@test "path_expand: leaves relative paths unchanged" {
  run path_expand "relative/path"
  assert_output "relative/path"
}

# --- path_truncate ---

@test "path_truncate: returns short paths unchanged" {
  run path_truncate "~/short" 38
  assert_output "~/short"
}

@test "path_truncate: truncates long paths with ellipsis" {
  long_path="~/very/long/deeply/nested/project/directory/name"
  result="$(path_truncate "$long_path" 30)"
  # Should contain ...
  [[ "$result" == *"..."* ]]
  # Should be at most 30 chars
  [ "${#result}" -le 30 ]
}

@test "path_truncate: preserves start and end of path" {
  long_path="~/very/long/deeply/nested/project/directory/name"
  result="$(path_truncate "$long_path" 30)"
  # Starts with ~/
  [[ "$result" == "~/"* ]]
  # Ends with part of /name
  [[ "$result" == *"name" ]]
}

@test "path_truncate: respects max width parameter" {
  long_path="~/this/is/a/really/long/path/that/goes/on/forever"
  result20="$(path_truncate "$long_path" 20)"
  result40="$(path_truncate "$long_path" 40)"
  [ "${#result20}" -le 20 ]
  [ "${#result40}" -le 40 ]
}
```

**Step 2: Run tests to verify they fail**

Run: `./run-tests.sh test/projects.bats`
Expected: FAIL — `lib/projects.sh: No such file or directory`

**Step 3: Write the implementation**

Create `lib/projects.sh`:
```bash
#!/bin/bash
# Project file helpers — pure, no side effects on source.

# Extracts project name from a "name:path" line.
parse_project_name() {
  echo "${1%%:*}"
}

# Extracts project path from a "name:path" line.
# Uses non-greedy match so paths with colons work.
parse_project_path() {
  echo "${1#*:}"
}

# Reads a projects file and outputs valid lines (skips blanks and comments).
# Usage: mapfile -t projects < <(load_projects "$file")
load_projects() {
  local file="$1" line
  [ ! -f "$file" ] && return
  while IFS= read -r line; do
    [[ -z "$line" || "$line" == \#* ]] && continue
    echo "$line"
  done < "$file"
}

# Expands ~ to $HOME at the start of a path.
path_expand() {
  echo "${1/#\~/$HOME}"
}

# Truncates a path to max_width chars with ... in the middle.
path_truncate() {
  local path="$1" max_width="$2"
  if [ "${#path}" -le "$max_width" ]; then
    echo "$path"
  else
    local half=$(( (max_width - 3) / 2 ))
    echo "${path:0:$half}...${path: -$half}"
  fi
}
```

**Step 4: Run tests to verify they pass**

Run: `./run-tests.sh test/projects.bats`
Expected: All tests pass.

**Step 5: Commit**

```bash
git add lib/projects.sh test/projects.bats
git commit -m "Add lib/projects.sh with tests"
```

---

### Task 4: Create lib/process.sh + Tests

**Files:**
- Create: `lib/process.sh`
- Create: `test/process.bats`
- Reference: `ghostty/claude-wrapper.sh:762-770`

**Step 1: Write the failing tests**

Create `test/process.bats`:
```bash
setup() {
  load 'test_helper/common'
  _common_setup
  source "$PROJECT_ROOT/lib/process.sh"
}

@test "kill_tree: kills a parent and its children" {
  # Spawn a parent that spawns a child
  bash -c 'sleep 300 & sleep 300 & wait' &
  parent_pid=$!
  sleep 0.3  # Let children start

  # Verify parent is running
  kill -0 "$parent_pid" 2>/dev/null

  kill_tree "$parent_pid" TERM
  sleep 0.3

  # Parent should be dead
  run kill -0 "$parent_pid"
  assert_failure
}

@test "kill_tree: handles nonexistent PID gracefully" {
  # Should not error out
  run kill_tree 999999 TERM
  assert_success
}

@test "kill_tree: defaults to TERM signal" {
  sleep 300 &
  pid=$!
  kill_tree "$pid"
  sleep 0.2
  run kill -0 "$pid"
  assert_failure
}
```

**Step 2: Run tests to verify they fail**

Run: `./run-tests.sh test/process.bats`
Expected: FAIL — `lib/process.sh: No such file or directory`

**Step 3: Write the implementation**

Create `lib/process.sh`:
```bash
#!/bin/bash
# Process management helpers — no side effects on source.

# Recursively kills a process tree (depth-first: children first, then parent).
kill_tree() {
  local pid=$1
  local sig=${2:-TERM}
  for child in $(pgrep -P "$pid" 2>/dev/null); do
    kill_tree "$child" "$sig"
  done
  kill -"$sig" "$pid" 2>/dev/null
  return 0
}
```

**Step 4: Run tests to verify they pass**

Run: `./run-tests.sh test/process.bats`
Expected: All tests pass.

**Step 5: Commit**

```bash
git add lib/process.sh test/process.bats
git commit -m "Add lib/process.sh with tests"
```

---

### Task 5: Create lib/input.sh + Tests

**Files:**
- Create: `lib/input.sh`
- Create: `test/input.bats`
- Reference: `ghostty/claude-wrapper.sh:317-348`

Note: The original `read_esc` function reads interactively byte-by-byte from stdin. For testability, we refactor it into `parse_esc_sequence` which reads from stdin and outputs the result to stdout (instead of setting a global variable).

**Step 1: Write the failing tests**

Create `test/input.bats`:
```bash
setup() {
  load 'test_helper/common'
  _common_setup
  source "$PROJECT_ROOT/lib/input.sh"
}

@test "parse_esc_sequence: up arrow" {
  result="$(printf '[A' | parse_esc_sequence)"
  [[ "$result" == "A" ]]
}

@test "parse_esc_sequence: down arrow" {
  result="$(printf '[B' | parse_esc_sequence)"
  [[ "$result" == "B" ]]
}

@test "parse_esc_sequence: left arrow" {
  result="$(printf '[D' | parse_esc_sequence)"
  [[ "$result" == "D" ]]
}

@test "parse_esc_sequence: right arrow" {
  result="$(printf '[C' | parse_esc_sequence)"
  [[ "$result" == "C" ]]
}

@test "parse_esc_sequence: SGR mouse left click" {
  # SGR format: \e[<button;col;rowM — we feed after the initial \e
  # button=0 (left click), col=15, row=3, M=press
  result="$(printf '[<0;15;3M' | parse_esc_sequence)"
  [[ "$result" == "click:3" ]]
}

@test "parse_esc_sequence: SGR mouse left click different row" {
  result="$(printf '[<0;22;10M' | parse_esc_sequence)"
  [[ "$result" == "click:10" ]]
}

@test "parse_esc_sequence: ignores mouse release" {
  # m = release (lowercase)
  result="$(printf '[<0;15;3m' | parse_esc_sequence)"
  [[ "$result" == "" ]]
}

@test "parse_esc_sequence: ignores right click" {
  # button=2 (right click)
  result="$(printf '[<2;15;3M' | parse_esc_sequence)"
  [[ "$result" == "" ]]
}

@test "parse_esc_sequence: ignores middle click" {
  # button=1 (middle click)
  result="$(printf '[<1;15;3M' | parse_esc_sequence)"
  [[ "$result" == "" ]]
}
```

**Step 2: Run tests to verify they fail**

Run: `./run-tests.sh test/input.bats`
Expected: FAIL — `lib/input.sh: No such file or directory`

**Step 3: Write the implementation**

Create `lib/input.sh`:
```bash
#!/bin/bash
# Input parsing helpers — no side effects on source.
# Reads escape sequences from stdin and outputs the parsed result.

# Parses an escape sequence from stdin (bytes AFTER the initial \x1b).
# Outputs to stdout:
#   "A"/"B"/"C"/"D" for arrow keys
#   "click:ROW" for SGR mouse left-click press
#   "" (empty) for ignored events (release, non-left-click)
parse_esc_sequence() {
  local _b1 _b2 _mc _mouse_data _mouse_btn _mouse_rest _mouse_col _mouse_row

  read -rsn1 _b1
  if [[ "$_b1" == "[" ]]; then
    read -rsn1 _b2
    if [[ "$_b2" == "<" ]]; then
      # SGR mouse: read until M (press) or m (release)
      _mouse_data=""
      while true; do
        read -rsn1 _mc
        if [[ "$_mc" == "M" || "$_mc" == "m" ]]; then
          break
        fi
        _mouse_data="${_mouse_data}${_mc}"
      done
      # Only handle press (M), ignore release (m)
      if [[ "$_mc" == "M" ]]; then
        _mouse_btn="${_mouse_data%%;*}"
        _mouse_rest="${_mouse_data#*;}"
        _mouse_col="${_mouse_rest%%;*}"
        _mouse_row="${_mouse_rest##*;}"
        if [[ "$_mouse_btn" == "0" ]]; then
          echo "click:${_mouse_row}"
        fi
      fi
    else
      echo "$_b2"
    fi
  fi
}
```

**Step 4: Run tests to verify they pass**

Run: `./run-tests.sh test/input.bats`
Expected: All tests pass.

**Step 5: Commit**

```bash
git add lib/input.sh test/input.bats
git commit -m "Add lib/input.sh with tests"
```

---

### Task 6: Create lib/statusline.sh + Tests

**Files:**
- Create: `lib/statusline.sh`
- Create: `test/statusline.bats`
- Reference: `bin/ghost-tab:439-447` (memory formatting in statusline-wrapper.sh template)
- Reference: `bin/ghost-tab:411` (cwd parsing in statusline-command.sh template)

**Step 1: Write the failing tests**

Create `test/statusline.bats`:
```bash
setup() {
  load 'test_helper/common'
  _common_setup
  source "$PROJECT_ROOT/lib/statusline.sh"
}

# --- format_memory ---

@test "format_memory: converts KB to MB" {
  run format_memory 512000
  assert_output "500M"
}

@test "format_memory: small MB value" {
  run format_memory 102400
  assert_output "100M"
}

@test "format_memory: converts to GB with decimal" {
  # 1572864 KB = 1536 MB = 1.5 GB
  run format_memory 1572864
  assert_output "1.5G"
}

@test "format_memory: exactly 1 GB" {
  # 1048576 KB = 1024 MB = 1.0G
  run format_memory 1048576
  assert_output "1.0G"
}

@test "format_memory: zero returns 0M" {
  run format_memory 0
  assert_output "0M"
}

# --- parse_cwd_from_json ---

@test "parse_cwd_from_json: extracts current_dir" {
  run parse_cwd_from_json '{"current_dir":"/Users/me/project"}'
  assert_output "/Users/me/project"
}

@test "parse_cwd_from_json: handles nested JSON" {
  run parse_cwd_from_json '{"foo":"bar","current_dir":"/tmp/test","baz":1}'
  assert_output "/tmp/test"
}

@test "parse_cwd_from_json: returns empty for missing key" {
  run parse_cwd_from_json '{"foo":"bar"}'
  assert_output ""
}
```

**Step 2: Run tests to verify they fail**

Run: `./run-tests.sh test/statusline.bats`
Expected: FAIL — `lib/statusline.sh: No such file or directory`

**Step 3: Write the implementation**

Create `lib/statusline.sh`:
```bash
#!/bin/bash
# Statusline helper functions — pure, no side effects on source.

# Converts kilobytes to human-readable memory string.
# Usage: format_memory 512000  =>  "500M"
# Usage: format_memory 1572864 =>  "1.5G"
format_memory() {
  local mem_kb="$1"
  local mem_mb=$((mem_kb / 1024))
  if [ "$mem_mb" -ge 1024 ]; then
    local mem_gb
    mem_gb=$(echo "scale=1; $mem_mb / 1024" | bc)
    echo "${mem_gb}G"
  else
    echo "${mem_mb}M"
  fi
}

# Extracts the current_dir value from a JSON string.
# Uses sed — no jq dependency.
parse_cwd_from_json() {
  echo "$1" | sed -n 's/.*"current_dir":"\([^"]*\)".*/\1/p'
}
```

**Step 4: Run tests to verify they pass**

Run: `./run-tests.sh test/statusline.bats`
Expected: All tests pass.

**Step 5: Commit**

```bash
git add lib/statusline.sh test/statusline.bats
git commit -m "Add lib/statusline.sh with tests"
```

---

### Task 7: Create lib/setup.sh + Tests

**Files:**
- Create: `lib/setup.sh`
- Create: `test/setup.bats`
- Reference: `bin/ghost-tab:28-34`

**Step 1: Write the failing tests**

Create `test/setup.bats`:
```bash
setup() {
  load 'test_helper/common'
  _common_setup
  source "$PROJECT_ROOT/lib/setup.sh"
  TEST_DIR="$(mktemp -d)"
}

teardown() {
  rm -rf "$TEST_DIR"
}

@test "resolve_share_dir: returns brew share when in brew prefix" {
  # Simulate: script_dir=/opt/homebrew/bin, brew_prefix=/opt/homebrew
  run resolve_share_dir "/opt/homebrew/bin" "/opt/homebrew"
  assert_output "/opt/homebrew/share/ghost-tab"
}

@test "resolve_share_dir: returns parent dir when not in brew prefix" {
  mkdir -p "$TEST_DIR/ghost-tab/bin"
  run resolve_share_dir "$TEST_DIR/ghost-tab/bin" ""
  assert_output "$TEST_DIR/ghost-tab"
}

@test "resolve_share_dir: returns parent dir when brew prefix is empty" {
  mkdir -p "$TEST_DIR/ghost-tab/bin"
  run resolve_share_dir "$TEST_DIR/ghost-tab/bin" ""
  assert_output "$TEST_DIR/ghost-tab"
}
```

**Step 2: Run tests to verify they fail**

Run: `./run-tests.sh test/setup.bats`
Expected: FAIL — `lib/setup.sh: No such file or directory`

**Step 3: Write the implementation**

Create `lib/setup.sh`:
```bash
#!/bin/bash
# Setup helper functions — pure, no side effects on source.

# Determines the SHARE_DIR (where supporting files live).
# Usage: resolve_share_dir "$SCRIPT_DIR" "$BREW_PREFIX"
# When script is in $BREW_PREFIX/bin, returns $BREW_PREFIX/share/ghost-tab.
# Otherwise, returns the parent of the script directory.
resolve_share_dir() {
  local script_dir="$1"
  local brew_prefix="$2"
  if [[ -n "$brew_prefix" && "$script_dir" == "$brew_prefix/bin" ]]; then
    echo "$brew_prefix/share/ghost-tab"
  else
    (cd "$script_dir/.." && pwd)
  fi
}
```

**Step 4: Run tests to verify they pass**

Run: `./run-tests.sh test/setup.bats`
Expected: All tests pass.

**Step 5: Commit**

```bash
git add lib/setup.sh test/setup.bats
git commit -m "Add lib/setup.sh with tests"
```

---

### Task 8: Update Main Scripts to Source lib/

**Files:**
- Modify: `ghostty/claude-wrapper.sh:1-32` (add source + use validate_ai_tool)
- Modify: `ghostty/claude-wrapper.sh:350-368` (remove inlined ai_tool_display_name/color)
- Modify: `ghostty/claude-wrapper.sh:477-493` (replace cycle_ai_tool, keep file-save)
- Modify: `ghostty/claude-wrapper.sh:762-770` (remove inlined kill_tree)
- Modify: `bin/ghost-tab:250-255` (copy lib/ during setup)

**Step 1: Add source lines to claude-wrapper.sh**

At the top of `ghostty/claude-wrapper.sh`, after `export PATH=...` (line 2), add:
```bash
# Load shared library functions
_WRAPPER_DIR="$(cd "$(dirname "$0")" && pwd)"
source "$_WRAPPER_DIR/lib/ai-tools.sh"
source "$_WRAPPER_DIR/lib/projects.sh"
source "$_WRAPPER_DIR/lib/process.sh"
source "$_WRAPPER_DIR/lib/input.sh"
```

**Step 2: Replace inlined functions in claude-wrapper.sh**

Remove the `ai_tool_display_name()` function (lines 350-358) — already in lib/ai-tools.sh.

Remove the `ai_tool_color()` function (lines 360-368) — already in lib/ai-tools.sh.

Replace the `cycle_ai_tool()` function (lines 477-493) with a wrapper that calls the lib function then saves to disk:
```bash
    cycle_ai_tool() {
      source "$_WRAPPER_DIR/lib/ai-tools.sh"  # reload not needed, just using the function
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

**Actually — keep `cycle_ai_tool` in-place in the wrapper** since it has the file-save side effect and is defined inside a nested scope. The lib version is the pure logic for testing. The wrapper's version calls the same logic plus saves. This avoids breaking the nested function scope. **Do NOT remove `cycle_ai_tool` from the wrapper.**

Similarly for `read_esc` (lines 317-348) — keep it in the wrapper since it sets `_esc_seq` as a global used by the input loop. The lib version (`parse_esc_sequence`) is the testable equivalent with stdout output.

So the changes to `claude-wrapper.sh` are:
1. Add source lines at top
2. Replace the validation block (lines 26-32) with `validate_ai_tool`
3. Remove `ai_tool_display_name` and `ai_tool_color` (already sourced from lib)
4. Remove `kill_tree` (line 762-770) — already sourced from lib

**Step 3: Update bin/ghost-tab to copy lib/ during setup**

In `bin/ghost-tab`, after the wrapper script copy (around line 253-255), add:
```bash
# Copy shared libraries
if [ -d "$SHARE_DIR/lib" ]; then
  cp -R "$SHARE_DIR/lib" ~/.config/ghostty/lib
  success "Copied shared libraries to ~/.config/ghostty/lib/"
fi
```

**Step 4: Run all tests**

Run: `./run-tests.sh`
Expected: All tests pass.

**Step 5: Manual smoke test**

Verify the scripts still work:
1. Run `ghostty/claude-wrapper.sh /tmp` — should launch tmux session in /tmp
2. Ctrl-C to exit

**Step 6: Commit**

```bash
git add ghostty/claude-wrapper.sh bin/ghost-tab
git commit -m "Wire up lib/ sources in main scripts"
```

---

### Task 9: Clean Up and Final Verification

**Step 1: Remove smoke test**

Delete `test/smoke.bats` (was only for infrastructure verification).

**Step 2: Run full test suite**

Run: `./run-tests.sh`
Expected: All tests pass across all `.bats` files.

**Step 3: Commit**

```bash
git rm test/smoke.bats
git commit -m "Remove smoke test, all lib tests passing"
```
