# CLAUDE.md

Project guidance for Claude Code (claude.ai/code) working with this repository.

---

## IMMEDIATE COMPLETION CHECKLIST

**STOP! Before saying "done" or "complete", verify ALL of the following:**

### For ANY Code Change (No Exceptions)

```
[ ] 1. Did I write tests FIRST, watch them FAIL, THEN write code? (IRON RULE)
[ ] 2. Did I run shellcheck on modified scripts? (MANDATORY)
[ ] 3. Did I run final verification with full test suite? (MANDATORY)
[ ] 4. Did I `git push` successfully? (Work NOT complete until push succeeds)
```

**If ANY box is unchecked:** Work is NOT complete. Do it NOW.

**No rationalizations:**
- "Chat is too long, instructions are far down" → INVALID. You're reading them right now.
- "User is in a hurry" → INVALID. Half-done work wastes MORE time later.
- "It's just a small change" → INVALID. Small changes break things too.
- "I'll do it in next session" → INVALID. That leaves work stranded.
- "Tests already cover it" → INVALID. Write test FIRST, watch it FAIL.

### For Session End

```
[ ] 1. All code tested (test first → fail → code → pass)
[ ] 2. shellcheck run on modified scripts
[ ] 3. Full test suite run and passing
[ ] 4. `git push` succeeded
[ ] 5. Issues updated/closed
[ ] 6. `git status` shows "up to date with origin"
```

**Work is DEFINITELY NOT complete if:**
- Changes exist only locally (not pushed)
- shellcheck was never run
- Tests were skipped
- No test was written before code

### Bug Fix IRON RULE

```
[ ] 1. Write regression test FIRST
[ ] 2. Run test → watch it FAIL (proves bug exists)
[ ] 3. Fix bug
[ ] 4. Run test → watch it PASS
[ ] 5. Re-run full test suite
```

**Write code before test?** Delete it. Start over.

### Session End Commands

Run full verification:

```bash
# Run shellcheck on all modified scripts
find lib bin ghostty -name '*.sh' -exec shellcheck {} +

# Run full test suite
./run-tests.sh

# Push changes
git pull --rebase
git push
git status  # MUST show "up to date with origin"
```

**This checklist is ALWAYS executed. NO MATTER how long the chat is.**

### Red Flags - You're About to Violate The Rules

If you catch yourself thinking ANY of these, STOP and DO THE CHECKLIST:

- "Chat is too long, I can't find the instructions"
- "User is in a hurry, I'll skip verification this time"
- "It's just a small change, doesn't need full process"
- "I'll do shellcheck in the next session"
- "Tests already exist, I don't need to write one first"
- "I already manually verified it works"
- "The push can wait, user can do it"
- "Full test suite takes too long"

**ALL of these mean: You're rationalizing. Run the checklist NOW.**

---

## Landing the Plane (Session Completion)

**⚠️ CRITICAL: The completion checklist at the TOP of this file MUST be followed.**

Scroll up to "IMMEDIATE COMPLETION CHECKLIST" and verify ALL items before declaring work done.

**If you're reading this section instead of the checklist:** Go to the TOP of the file.

**Summary (detail is at top):**
1. File issues for remaining work
2. Run quality gates (shellcheck, tests)
3. Full test suite verification
4. Update issue status
5. **PUSH TO REMOTE** (MANDATORY)
6. Clean up
7. Verify `git status` shows "up to date with origin"
8. Hand off context

**Remember:** Every code change needs shellcheck → tests → push. No exceptions.

## Project Overview

Ghost Tab is a Ghostty + tmux wrapper that launches a four-pane dev session with AI coding tools (Claude Code, Codex CLI, Copilot CLI, OpenCode), lazygit, broot, and a spare terminal. It handles complete process cleanup when windows close (no zombie processes).

**Key Features:**
- Interactive project selector with TUI
- Multi-AI tool support (Claude Code, Codex CLI, Copilot CLI, OpenCode)
- Custom status lines showing git info and context usage
- Sound notifications on AI idle
- Auto-cleanup of entire process trees
- Homebrew distribution

## Commands

```bash
./run-tests.sh              # Run full BATS test suite
./run-tests.sh test/foo.bats  # Run specific test file
shellcheck lib/*.sh bin/ghost-tab ghostty/*.sh  # Lint all scripts
./bin/ghost-tab             # Run main installer/setup
```

## Architecture

**Entry Points:**
- `bin/ghost-tab` - Main installer script, sources all modules
- `ghostty/claude-wrapper.sh` - Runtime wrapper launched by Ghostty

**Module System:**
All reusable functionality lives in `lib/` as sourced shell scripts:

- **tui.sh**: Terminal UI helpers (header, success, error, info, warn)
- **install.sh**: Package installation (Homebrew, casks, commands)
- **ai-tools.sh**: AI tool detection and management
- **ai-select.sh**: Interactive AI tool selection menu
- **projects.sh**: Project file parsing and validation
- **project-actions.sh**: Add/delete project operations
- **menu.sh**: Main interactive project selector
- **settings-menu.sh**: Settings configuration menu
- **tmux-session.sh**: tmux session creation and pane setup
- **process.sh**: Process tree management and cleanup
- **ghostty-config.sh**: Ghostty config file operations
- **statusline.sh**: Status line generation logic
- **statusline-setup.sh**: Claude Code status line installation
- **notification-setup.sh**: Sound notification setup
- **settings-json.sh**: JSON manipulation for settings files
- **input.sh**: User input helpers with validation
- **autocomplete.sh**: Path completion for project entry
- **logo-animation.sh**: Animated ASCII logo display
- **update.sh**: Self-update functionality

**Data Files:**
- `~/.config/ghost-tab/projects` - Project list (name:path format)
- `~/.config/ghost-tab/ai-tool` - Default AI tool preference
- `~/.config/ghost-tab/*-features.json` - AI tool feature flags
- `~/.claude/settings.json` - Claude Code settings
- `~/.codex/config.toml` - Codex CLI config
- `~/.config/ghostty/config` - Ghostty terminal config

**Process Hierarchy:**
```
Ghostty window
└─ claude-wrapper.sh (shell command)
   └─ tmux session
      ├─ AI tool (Claude Code/Codex/etc)
      ├─ lazygit
      ├─ broot
      └─ spare shell
```

On window close, wrapper recursively kills entire process tree.

## Code Conventions

### Avoid Over-Engineering
- Don't add features beyond what's asked
- Don't create helpers for one-time operations
- Three similar lines > premature abstraction
- Only comment where logic isn't self-evident

### Shell Scripting Best Practices

**Strict Mode:**
```bash
set -e  # Exit on error (use in scripts)
set -u  # Exit on undefined variable (optional, use carefully)
set -o pipefail  # Pipe failures propagate (optional)
```

**Quoting:**
```bash
# ✅ CORRECT - Always quote variables
"$var"
"${array[@]}"
mkdir -p "$dir/subdir"

# ❌ WRONG - Unquoted (word splitting, glob expansion)
$var
${array[@]}
mkdir -p $dir/subdir
```

**Command Substitution:**
```bash
# ✅ CORRECT - Use $() for nesting and readability
result="$(command)"
outer="$(inner "$(innermost)")"

# ❌ WRONG - Backticks are legacy
result=`command`
```

**Conditionals:**
```bash
# ✅ CORRECT - Use [[ ]] for advanced features
if [[ "$var" == "value" ]]; then
  # Supports &&, ||, =~, <, >
  # No word splitting inside [[ ]]
fi

# ✅ CORRECT - Use [ ] for POSIX compatibility
if [ "$var" = "value" ]; then
  # More portable
fi

# ❌ WRONG - Don't use `test` command directly
if test "$var" = "value"; then
  # Verbose, no benefit
fi
```

**Error Handling:**
```bash
# ✅ CORRECT - Check command success
if command_that_might_fail; then
  success "Operation completed"
else
  error "Operation failed"
  return 1
fi

# ✅ CORRECT - Use || for fallback
result="$(brew --prefix 2>/dev/null || echo "/usr/local")"

# ❌ WRONG - Ignoring errors
command_that_might_fail  # What if it fails?
```

**shellcheck Compliance:**
- **ALWAYS** run `shellcheck` before committing
- Fix ALL warnings (SC1091 source directive is OK if verified)
- Use `# shellcheck disable=SCXXXX` ONLY when necessary with comment explaining why

**File Operations:**
```bash
# ✅ CORRECT - Check file existence
if [ -f "$file" ]; then
  # File exists and is regular file
fi

if [ -d "$dir" ]; then
  # Directory exists
fi

# ✅ CORRECT - Safe file reading
while IFS=: read -r name path; do
  echo "$name -> $path"
done < "$projects_file"

# ❌ WRONG - Cat abuse (useless use of cat)
cat file | grep pattern  # Use: grep pattern file
```

**Functions:**
```bash
# ✅ CORRECT - Clear function declarations
function_name() {
  local var1="$1"
  local var2="$2"

  # Always use local for function variables
  # Return 0 for success, non-zero for failure
  return 0
}

# ❌ WRONG - Global variables in functions
bad_function() {
  result="$1"  # Pollutes global scope
}
```

**Array Handling:**
```bash
# ✅ CORRECT - Proper array operations
array=("item1" "item2" "item3")
echo "${array[0]}"  # First element
echo "${array[@]}"  # All elements
echo "${#array[@]}"  # Length

# Iterate over array
for item in "${array[@]}"; do
  echo "$item"
done

# ❌ WRONG - Word splitting
for item in ${array[@]}; do  # Missing quotes
  echo "$item"
done
```

### Project-Specific Patterns

**TUI Output:**
```bash
# Use standardized TUI functions from tui.sh
header "Section Title"
success "Operation succeeded"
error "Something failed"
info "FYI message"
warn "Warning message"
```

**Configuration Files:**
```bash
# Read project file (name:path format)
while IFS=: read -r name path; do
  [[ "$name" =~ ^#.*$ ]] && continue  # Skip comments
  [[ -z "$name" ]] && continue  # Skip empty
  # Process $name and $path
done < "$PROJECTS_FILE"
```

**AI Tool Integration:**
```bash
# Check if command exists
if command -v claude &>/dev/null; then
  # claude is available
fi

# Install with verification
ensure_command "claude" \
  "curl -fsSL https://claude.ai/install.sh | bash" \
  "Run 'claude' to authenticate" \
  "Claude Code"
```

**Process Management:**
```bash
# Get process tree recursively
get_process_tree() {
  local pid="$1"
  local children
  children=$(pgrep -P "$pid" 2>/dev/null || true)

  echo "$pid"
  for child in $children; do
    get_process_tree "$child"
  done
}

# Kill with grace period then force
kill -TERM "$pid" 2>/dev/null || true
sleep 0.5
kill -KILL "$pid" 2>/dev/null || true
```

## Testing

### IRON RULE: No Code Without Tests

**⚠️ This is also in the completion checklist at the TOP of this file.**

**ALL code changes require behavior tests.**

**Bug fixes MUST follow this exact order:**
1. Write regression test
2. Run it → watch it FAIL (proves bug exists)
3. Fix the bug
4. Run it → watch it PASS
5. Only THEN is the fix complete

**No exceptions. No "I'll test later". No "it's obvious".**

Write test first. If you write code before test, delete it and start over.

**See "IMMEDIATE COMPLETION CHECKLIST" at TOP of file for the full workflow.**

### Commands
```bash
./run-tests.sh                    # Full suite
./run-tests.sh test/foo.bats      # Single file
./run-tests.sh -f "test name"     # Filter by name
```

### BATS Test Structure

**Test Files:**
- Located in `test/`
- Named `*.bats` (e.g., `menu.bats`, `projects.bats`)
- Each file tests one module from `lib/`

**Basic Test:**
```bash
#!/usr/bin/env bats

load test_helper/bats-support/load
load test_helper/bats-assert/load

setup() {
  # Runs before each test
  source "$BATS_TEST_DIRNAME/../lib/module.sh"
  TEMP_DIR="$(mktemp -d)"
}

teardown() {
  # Runs after each test
  rm -rf "$TEMP_DIR"
}

@test "descriptive test name" {
  # Arrange
  local input="test"

  # Act
  run function_to_test "$input"

  # Assert
  assert_success
  assert_output "expected output"
}
```

**Critical Rules:**

**Setup/Teardown:**
- ALWAYS clean up temp files in teardown
- NEVER leave test artifacts (files, processes)
- Source the module being tested in setup()
- Create isolated temp directories for each test

**Assertions:**
```bash
# BATS run command captures output and exit code
run command args

# Success/failure
assert_success        # Exit code 0
assert_failure        # Exit code non-zero
assert_failure 127    # Specific exit code

# Output
assert_output "exact"           # Exact match
assert_output --partial "sub"   # Contains substring
assert_output --regexp "^pat"   # Regex match
refute_output "should not see"  # Should NOT appear

# Line-based
assert_line "exact line"
assert_line --index 0 "first"
assert_line --partial "contains"
refute_line "should not exist"
```

**Mocking:**
```bash
# Override functions
function_name() {
  echo "mocked output"
  return 0
}

# Mock external commands
command() {
  if [[ "$1" == "specific" ]]; then
    return 0
  fi
  return 1
}

# Capture calls to mocked function
mock_calls=()
mock_fn() {
  mock_calls+=("$*")
}
```

**File System Tests:**
```bash
@test "creates config file" {
  local config="$TEMP_DIR/config"

  run create_config "$config"

  assert_success
  assert [ -f "$config" ]  # File exists
  assert_line --partial "expected content"
}
```

**Input Simulation:**
```bash
@test "reads user input" {
  # Simulate user typing "yes"
  run bash -c 'echo "yes" | function_that_reads'

  assert_success
  assert_output --partial "Confirmed"
}
```

### What to Test vs Not

**DO Test:**
- Public function contracts
- User-facing behavior
- File operations (create, modify, delete)
- Error conditions (bad input, missing files)
- Integration between modules
- Edge cases (empty input, special chars)

**DO NOT Test:**
- Private helper functions (unless complex)
- Third-party commands (brew, tmux, etc)
- Obvious shell behavior
- Visual formatting (unless critical)

### Common Pitfalls

| Pitfall | Solution |
|---------|----------|
| Temp files leak between tests | Always use teardown() to clean up |
| Tests depend on order | Each test should be independent |
| Missing `run` wrapper | Use `run` to capture output/exit code |
| Unquoted variables in tests | Quote everything: `"$var"` |
| Assuming clean environment | Set up everything in setup() |
| Not testing error paths | Test both success and failure |
| Forgetting assertions | Every test needs assert_* |

### Red Flags - You're About to Violate The Rules

**⚠️ More red flags are in the completion checklist at the TOP of this file.**

If you catch yourself thinking ANY of these, STOP:
- "This is too simple to test"
- "I'll test it after"
- "Tests would just duplicate the code"
- "It's about the spirit, not the letter"
- "This case is different"
- "I already verified it manually"

**These thoughts mean you're rationalizing. Write the test first.**

## Configuration

**DO NOT modify** without explicit request: `run-tests.sh`, `.gitignore`, `.gitmodules`, `VERSION`, Homebrew formula

## Important Patterns

1. **Modularity**: Each `lib/*.sh` file is independently sourceable
2. **Error Propagation**: Use `set -e` and proper exit codes
3. **User Feedback**: Consistent TUI output (header/success/error/info/warn)
4. **Graceful Degradation**: Detect and adapt to missing optional features
5. **Process Cleanup**: Recursive tree killing with grace period
6. **Config Management**: Support both merge and replace for existing configs
7. **Cross-Shell Compatibility**: Source user's profile (bash/zsh) for environment
8. **Symlink Management**: Use `ln -sf` for idempotent linking
9. **Path Expansion**: Always expand `~` to `$HOME` for validation
10. **Sound Notification**: Pluggable hook system for AI idle events
