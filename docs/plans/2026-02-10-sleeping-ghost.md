# Sleeping Ghost Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add idle state to ghost mascot that activates after 2 minutes of inactivity, showing closed eyes, dimmed colors, and "Zzz" indicator.

**Architecture:** Extend existing logo animation system in `lib/logo-animation.sh` with sleeping art variants and Zzz rendering. Add timer tracking and state management to main menu loop in `ghostty/claude-wrapper.sh` using timeout-based reads.

**Tech Stack:** Bash, BATS testing framework, ANSI terminal escape codes

---

## Task 1: Add Sleeping Ghost Art - Claude

**Files:**
- Modify: `lib/logo-animation.sh` (add after line 49)
- Test: `test/logo-sleep.bats` (create new)

**Step 1: Write the failing test**

Create `test/logo-sleep.bats`:

```bash
#!/usr/bin/env bats

load test_helper/bats-support/load
load test_helper/bats-assert/load

setup() {
  source "$BATS_TEST_DIRNAME/../lib/logo-animation.sh"
}

@test "logo_art_claude_sleeping function exists" {
  run type logo_art_claude_sleeping
  assert_success
}

@test "logo_art_claude_sleeping sets required variables" {
  logo_art_claude_sleeping

  assert [ "${#_LOGO_LINES[@]}" -eq 15 ]
  assert [ "$_LOGO_HEIGHT" -eq 15 ]
  assert [ "$_LOGO_WIDTH" -eq 28 ]
}

@test "logo_art_claude_sleeping has closed eyes" {
  logo_art_claude_sleeping

  # Lines 5 and 6 should have closed eyes (no white 255 color in eyes area)
  # Check that line 5 contains the closed eye pattern
  [[ "${_LOGO_LINES[5]}" =~ "▬▬▬▬" ]]
}
```

**Step 2: Run test to verify it fails**

```bash
./run-tests.sh test/logo-sleep.bats
```

Expected: FAIL - "logo_art_claude_sleeping: command not found"

**Step 3: Write minimal implementation**

Add to `lib/logo-animation.sh` after the `logo_art_claude()` function (after line 49):

```bash
# ─────────────────────────────────────────────────────────────────
# Claude Ghost — SLEEPING variant (dimmed, closed eyes)
# ─────────────────────────────────────────────────────────────────
logo_art_claude_sleeping() {
  local O D B L W K Y R
  O=$(_c 166) D=$(_c 166) B=$(_c 94) L=$(_c 180)
  W=$(_c 250) K=$(_c 232) Y=$(_c 178) R=$(_r)

  _LOGO_LINES=(
    "${R}       ${L}▄▄▄▄▄▄▄▄▄▄▄▄▄▄${R}       "
    "${R}     ${L}▄${O}████████████████${L}▄${R}     "
    "${R}    ${L}▄${O}██████████████████${L}▄${R}    "
    "${R}   ${O}██████████████████████${R}   "
    "${R}  ${O}████████████████████████${R}  "
    "${R}  ${D}████${K}▬▬▬▬${D}██████${K}▬▬▬▬${D}████${R}  "
    "${R}  ${D}████████████████████████${R}  "
    "${R}  ${D}████████████████████████${R}  "
    "${R}  ${D}█████████${Y}██${D}█████████████${R}  "
    "${R}  ${D}████████${Y}█▀▀█${D}████████████${R}  "
    "${R}  ${D}████████${Y}█▄▄█${D}████████████${R}  "
    "${R}  ${D}█████████${Y}██${D}█████████████${R}  "
    "${R}  ${B}████████████████████████${R}  "
    "${R}  ${B}██ █████ ██████ █████ ██${R}  "
    "${R}  ${B}█${R}  ${B}▀████▀${R} ${B}████${R} ${B}▀████▀${R}  ${B}█${R}  "
  )

  _LOGO_HEIGHT=${#_LOGO_LINES[@]}
  _LOGO_WIDTH=28
}
```

**Step 4: Run test to verify it passes**

```bash
./run-tests.sh test/logo-sleep.bats
```

Expected: PASS (all 3 tests)

**Step 5: Run shellcheck**

```bash
shellcheck lib/logo-animation.sh
```

Expected: No errors

**Step 6: Commit**

```bash
git add lib/logo-animation.sh test/logo-sleep.bats
git commit -m "Add sleeping variant for Claude ghost

- Dimmed colors (166 instead of 209/208)
- Closed eyes using ▬ characters
- Test coverage for sleeping art function

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Task 2: Add Sleeping Ghost Art - Codex, Copilot, OpenCode

**Files:**
- Modify: `lib/logo-animation.sh` (add after each logo function)
- Modify: `test/logo-sleep.bats` (add tests)

**Step 1: Write the failing tests**

Add to `test/logo-sleep.bats`:

```bash
@test "logo_art_codex_sleeping function exists and sets variables" {
  logo_art_codex_sleeping
  assert [ "${#_LOGO_LINES[@]}" -eq 15 ]
  assert [ "$_LOGO_HEIGHT" -eq 15 ]
  assert [ "$_LOGO_WIDTH" -eq 28 ]
}

@test "logo_art_copilot_sleeping function exists and sets variables" {
  logo_art_copilot_sleeping
  assert [ "${#_LOGO_LINES[@]}" -eq 15 ]
  assert [ "$_LOGO_HEIGHT" -eq 15 ]
  assert [ "$_LOGO_WIDTH" -eq 28 ]
}

@test "logo_art_opencode_sleeping function exists and sets variables" {
  logo_art_opencode_sleeping
  assert [ "${#_LOGO_LINES[@]}" -eq 15 ]
  assert [ "$_LOGO_HEIGHT" -eq 15 ]
  assert [ "$_LOGO_WIDTH" -eq 28 ]
}
```

**Step 2: Run tests to verify they fail**

```bash
./run-tests.sh test/logo-sleep.bats -f "sleeping"
```

Expected: FAIL - functions not found

**Step 3: Implement Codex sleeping variant**

Add to `lib/logo-animation.sh` after `logo_art_codex()` (after line 80):

```bash
# ─────────────────────────────────────────────────────────────────
# Codex Ghost — SLEEPING variant (dimmed, closed eyes)
# ─────────────────────────────────────────────────────────────────
logo_art_codex_sleeping() {
  local G Y D L W K H R
  G=$(_c 71) Y=$(_c 71) D=$(_c 58) L=$(_c 114)
  W=$(_c 250) K=$(_c 232) H=$(_c 65) R=$(_r)

  _LOGO_LINES=(
    "${R}       ${L}▄▄▄▄▄▄▄▄▄▄▄▄▄▄${R}       "
    "${R}     ${L}▄${G}████████████████${L}▄${R}     "
    "${R}    ${L}▄${G}██████████████████${L}▄${R}    "
    "${R}   ${G}██████████████████████${R}   "
    "${R}  ${G}████████████████████████${R}  "
    "${R}  ${Y}████${K}▬▬▬▬${Y}██████${K}▬▬▬▬${Y}████${R}  "
    "${R}  ${Y}████████████████████████${R}  "
    "${R}  ${Y}████████████████████████${R}  "
    "${R}  ${Y}████${H}▄▀▀▀▄${Y}██████${H}▄▀▀▀▄${Y}████${R}  "
    "${R}  ${Y}████${H}█${Y}████${H}█${Y}████${H}█${Y}████${H}█${Y}████${R}  "
    "${R}  ${Y}████${H}▀▄▄▄▀${Y}██████${H}▀▄▄▄▀${Y}████${R}  "
    "${R}  ${Y}████████████████████████${R}  "
    "${R}  ${D}████████████████████████${R}  "
    "${R}  ${D}██ █████ ██████ █████ ██${R}  "
    "${R}  ${D}█${R}  ${D}▀████▀${R} ${D}████${R} ${D}▀████▀${R}  ${D}█${R}  "
  )

  _LOGO_HEIGHT=${#_LOGO_LINES[@]}
  _LOGO_WIDTH=28
}
```

**Step 4: Implement Copilot sleeping variant**

Add to `lib/logo-animation.sh` after `logo_art_copilot()` (after line 111):

```bash
# ─────────────────────────────────────────────────────────────────
# Copilot Ghost — SLEEPING variant (dimmed, closed eyes)
# ─────────────────────────────────────────────────────────────────
logo_art_copilot_sleeping() {
  local P LP DP LV W K M R
  P=$(_c 98) LP=$(_c 98) DP=$(_c 60) LV=$(_c 140)
  W=$(_c 250) K=$(_c 232) M=$(_c 96) R=$(_r)

  _LOGO_LINES=(
    "${R}       ${LV}▄▄▄▄▄▄▄▄▄▄▄▄▄▄${R}       "
    "${R}     ${LV}▄${P}████████████████${LV}▄${R}     "
    "${R}    ${LV}▄${P}██████████████████${LV}▄${R}    "
    "${R}   ${P}██████████████████████${R}   "
    "${R}  ${P}████████████████████████${R}  "
    "${R}  ${P}██${M}▄▄▄▄▄▄${P}████████${M}▄▄▄▄▄▄${P}██${R}  "
    "${R}  ${P}██${M}▌${K}▬▬▬▬${M}▐${P}██████${M}▌${K}▬▬▬▬${M}▐${P}██${R}  "
    "${R}  ${P}██${M}▀▀▀▀▀▀${P}████████${M}▀▀▀▀▀▀${P}██${R}  "
    "${R}  ${LP}████████████████████████${R}  "
    "${R}  ${LP}████████████████████████${R}  "
    "${R}  ${LP}████████████████████████${R}  "
    "${R}  ${LP}████████████████████████${R}  "
    "${R}  ${DP}████████████████████████${R}  "
    "${R}  ${DP}██ █████ ██████ █████ ██${R}  "
    "${R}  ${DP}█${R}  ${DP}▀████▀${R} ${DP}████${R} ${DP}▀████▀${R}  ${DP}█${R}  "
  )

  _LOGO_HEIGHT=${#_LOGO_LINES[@]}
  _LOGO_WIDTH=28
}
```

**Step 5: Implement OpenCode sleeping variant**

Add to `lib/logo-animation.sh` after `logo_art_opencode()` (after line 144):

```bash
# ─────────────────────────────────────────────────────────────────
# OpenCode Ghost — SLEEPING variant (dimmed, closed eyes)
# ─────────────────────────────────────────────────────────────────
logo_art_opencode_sleeping() {
  local W VL L ML M MD D K SM R
  W=$(_c 244) VL=$(_c 242) L=$(_c 240) ML=$(_c 238)
  M=$(_c 236) MD=$(_c 234) D=$(_c 232) K=$(_c 232)
  SM=$(_c 234) R=$(_r)

  _LOGO_LINES=(
    "${R}       ${VL}▄▄▄▄▄▄▄▄▄▄▄▄▄▄${R}       "
    "${R}     ${VL}▄${W}████████████████${VL}▄${R}     "
    "${R}    ${VL}▄${W}██████████████████${VL}▄${R}    "
    "${R}   ${L}██████████████████████${R}   "
    "${R}  ${L}████████████████████████${R}  "
    "${R}  ${ML}████${K}▬▬▬▬${ML}██████${K}▬▬▬▬${ML}████${R}  "
    "${R}  ${ML}████████████████████████${R}  "
    "${R}  ${M}████████████████████████${R}  "
    "${R}  ${M}████████████████████████${R}  "
    "${R}  ${M}████████${SM}█▀▀█${M}████████████${R}  "
    "${R}  ${M}████████████████████████${R}  "
    "${R}  ${MD}████████████████████████${R}  "
    "${R}  ${MD}████████████████████████${R}  "
    "${R}  ${D}██ █████ ██████ █████ ██${R}  "
    "${R}  ${D}█${R}  ${D}▀████▀${R} ${D}████${R} ${D}▀████▀${R}  ${D}█${R}  "
  )

  _LOGO_HEIGHT=${#_LOGO_LINES[@]}
  _LOGO_WIDTH=28
}
```

**Step 6: Run tests to verify they pass**

```bash
./run-tests.sh test/logo-sleep.bats
```

Expected: PASS (all tests)

**Step 7: Run shellcheck**

```bash
shellcheck lib/logo-animation.sh
```

Expected: No errors

**Step 8: Commit**

```bash
git add lib/logo-animation.sh test/logo-sleep.bats
git commit -m "Add sleeping variants for Codex, Copilot, OpenCode ghosts

- All variants use dimmed color codes
- Closed eyes with ▬ characters or integrated into design
- Test coverage for all sleeping art functions

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Task 3: Add Zzz Drawing Functions

**Files:**
- Modify: `lib/logo-animation.sh` (add after line 175)
- Modify: `test/logo-sleep.bats` (add tests)

**Step 1: Write the failing tests**

Add to `test/logo-sleep.bats`:

```bash
@test "draw_zzz function exists" {
  run type draw_zzz
  assert_success
}

@test "clear_zzz function exists" {
  run type clear_zzz
  assert_success
}

@test "draw_logo_sleeping function exists" {
  run type draw_logo_sleeping
  assert_success
}
```

**Step 2: Run tests to verify they fail**

```bash
./run-tests.sh test/logo-sleep.bats -f "zzz\|draw_logo_sleeping"
```

Expected: FAIL - functions not found

**Step 3: Implement the functions**

Add to `lib/logo-animation.sh` after `clear_logo_area()` function (after line 174):

```bash
# draw_zzz row col
#   Render "z Z Z" indicator above the ghost at the given position.
#   Used to show the ghost is sleeping.
draw_zzz() {
  local row=$1 col=$2
  moveto "$row" "$((col + _LOGO_WIDTH - 8))"
  printf "${_DIM}z${_NC}"
  moveto "$((row + 1))" "$((col + _LOGO_WIDTH - 6))"
  printf "${_DIM}Z${_NC}"
  moveto "$((row + 2))" "$((col + _LOGO_WIDTH - 4))"
  printf "Z"
}

# clear_zzz row col
#   Clear the "z Z Z" indicator area by overwriting with spaces.
clear_zzz() {
  local row=$1 col=$2
  local i
  for i in 0 1 2; do
    moveto "$((row + i))" "$((col + _LOGO_WIDTH - 8))"
    printf "          "
  done
}

# draw_logo_sleeping row col tool_name
#   Draw the sleeping variant of the ghost (closed eyes, dimmed).
draw_logo_sleeping() {
  local row=$1 col=$2 tool=$3 line
  "logo_art_${tool}_sleeping"
  for line in "${_LOGO_LINES[@]}"; do
    moveto "$row" "$col"
    printf '%b' "$line"
    row=$((row + 1))
  done
}
```

**Step 4: Run tests to verify they pass**

```bash
./run-tests.sh test/logo-sleep.bats
```

Expected: PASS (all tests)

**Step 5: Run shellcheck**

```bash
shellcheck lib/logo-animation.sh
```

Expected: No errors

**Step 6: Commit**

```bash
git add lib/logo-animation.sh test/logo-sleep.bats
git commit -m "Add Zzz indicator and sleeping logo drawing functions

- draw_zzz(): Renders z Z Z above ghost
- clear_zzz(): Clears Zzz indicator area
- draw_logo_sleeping(): Draws sleeping ghost variant

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Task 4: Add Sleep Transition Functions

**Files:**
- Modify: `lib/logo-animation.sh` (add after draw_logo_sleeping)
- Modify: `test/logo-sleep.bats` (add tests)

**Step 1: Write the failing tests**

Add to `test/logo-sleep.bats`:

```bash
@test "initiate_sleep_transition function exists" {
  run type initiate_sleep_transition
  assert_success
}

@test "wake_ghost function exists" {
  run type wake_ghost
  assert_success
}
```

**Step 2: Run tests to verify they fail**

```bash
./run-tests.sh test/logo-sleep.bats -f "transition\|wake"
```

Expected: FAIL - functions not found

**Step 3: Implement the functions**

Add to `lib/logo-animation.sh` after `draw_logo_sleeping()`:

```bash
# initiate_sleep_transition
#   Transition the ghost from awake to sleeping state.
#   Stops bobbing animation, pauses briefly, then draws sleeping ghost with Zzz.
#   Uses global variables: _logo_row, _logo_col, SELECTED_AI_TOOL, _ghost_sleeping
initiate_sleep_transition() {
  # Stop bobbing animation
  stop_logo_animation

  # Pause for gradual transition effect (2.5 seconds)
  sleep 2.5

  # Draw sleeping ghost with closed eyes and dimmed colors
  draw_logo_sleeping "$_logo_row" "$_logo_col" "$SELECTED_AI_TOOL"

  # Draw Zzz indicator above ghost
  draw_zzz "$((\_logo_row - 3))" "$_logo_col"

  _ghost_sleeping=1
}

# wake_ghost
#   Wake the ghost from sleeping to awake state.
#   Clears Zzz indicator and resumes bobbing animation.
#   Uses global variables: _logo_row, _logo_col, SELECTED_AI_TOOL, _ghost_sleeping, _last_interaction
wake_ghost() {
  # Clear Zzz indicator
  clear_zzz "$((\_logo_row - 3))" "$_logo_col"

  # Resume bobbing animation (this redraws awake ghost)
  start_logo_animation "$_logo_row" "$_logo_col" "$SELECTED_AI_TOOL"

  # Update state
  _ghost_sleeping=0
  _last_interaction=$SECONDS
}
```

**Step 4: Run tests to verify they pass**

```bash
./run-tests.sh test/logo-sleep.bats
```

Expected: PASS (all tests)

**Step 5: Run shellcheck**

```bash
shellcheck lib/logo-animation.sh
```

Expected: No errors

**Step 6: Commit**

```bash
git add lib/logo-animation.sh test/logo-sleep.bats
git commit -m "Add sleep transition functions

- initiate_sleep_transition(): Stops bobbing, draws sleeping ghost
- wake_ghost(): Clears Zzz, resumes bobbing animation
- Both functions manage _ghost_sleeping state flag

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Task 5: Integrate Timer and State Management

**Files:**
- Modify: `ghostty/claude-wrapper.sh` (main menu loop around line 150-300)
- Create: `test/integration-sleep.bats` (manual test helper)

**Step 1: Write manual test helper**

Create `test/integration-sleep.bats` (for manual testing with short timeout):

```bash
#!/usr/bin/env bats
# Manual integration test - demonstrates sleep feature with 10-second timeout
# Run: ./run-tests.sh test/integration-sleep.bats
# Then wait 10 seconds to see ghost sleep, press any key to wake

@test "sleep feature integration test (manual)" {
  skip "Manual test only - requires visual inspection"
  # This test documents the expected behavior
  # Actual integration testing requires running ghost-tab
}
```

**Step 2: Find the menu loop initialization**

Read `ghostty/claude-wrapper.sh` around line 150-200 to find where the menu loop starts and the logo animation begins.

**Step 3: Add timer initialization**

After the logo animation starts (after `start_logo_animation` call), add:

```bash
# Initialize sleep timer
_last_interaction=$SECONDS
_ghost_sleeping=0
_sleep_timeout=120  # 2 minutes
```

**Step 4: Modify the read command**

Find the main `read -rsn1 key` line (around line 273). Change it to:

```bash
# Non-blocking read with 0.5s timeout to allow sleep checking
read -rsn1 -t 0.5 key || {
  # Check for sleep timeout when no input
  if [ "$_ghost_sleeping" -eq 0 ] && [ "$_LOGO_LAYOUT" != "hidden" ]; then
    if [ $((SECONDS - _last_interaction)) -ge "$_sleep_timeout" ]; then
      initiate_sleep_transition
    fi
  fi
  continue
}
```

**Step 5: Add wake-on-keypress logic**

Right after the read command (before existing key handling), add:

```bash
# Wake ghost if sleeping and any key pressed
if [ "$_ghost_sleeping" -eq 1 ]; then
  wake_ghost
fi

# Reset interaction timer on any keypress
_last_interaction=$SECONDS
```

**Step 6: Test manually**

For quick testing, temporarily change `_sleep_timeout=10` and run:

```bash
./bin/ghost-tab
```

Wait 10 seconds to see ghost sleep, press any key to wake. Then change back to 120.

**Step 7: Run shellcheck**

```bash
shellcheck ghostty/claude-wrapper.sh
```

Expected: No errors

**Step 8: Commit**

```bash
git add ghostty/claude-wrapper.sh test/integration-sleep.bats
git commit -m "Integrate sleep timer and state management into menu loop

- Initialize timer variables after logo animation starts
- Change read to timeout-based (0.5s) for sleep checking
- Check for timeout and trigger sleep transition
- Wake ghost immediately on any keypress
- Reset timer on all interactions

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Task 6: Handle Edge Cases

**Files:**
- Modify: `ghostty/claude-wrapper.sh` (delete mode, settings menu, AI tool switching)

**Step 1: Check delete mode input loop**

Find the delete mode `read -rsn1 key` (around line 233). Add same timeout and wake logic:

```bash
# In delete mode, also use timeout-based read
read -rsn1 -t 0.5 key || {
  if [ "$_ghost_sleeping" -eq 0 ] && [ "$_LOGO_LAYOUT" != "hidden" ]; then
    if [ $((SECONDS - _last_interaction)) -ge "$_sleep_timeout" ]; then
      initiate_sleep_transition
    fi
  fi
  continue
}

# Wake ghost if sleeping
if [ "$_ghost_sleeping" -eq 1 ]; then
  wake_ghost
fi
_last_interaction=$SECONDS
```

**Step 2: Preserve timer during AI tool switching**

Verify that AI tool switching (left/right arrows) already resets `_last_interaction` via the main input loop. No additional changes needed.

**Step 3: Preserve timer during settings menu**

Settings menu is a separate function. The timer will naturally pause while in settings (since main loop is blocked), and resume when returning. This is acceptable behavior.

**Step 4: Test edge cases manually**

```bash
./bin/ghost-tab
```

Test:
- Wait for sleep, then delete a project (should wake)
- Wait for sleep, switch AI tools (should wake)
- Open settings while sleeping (ghost stays sleeping)
- Close settings (timer resumes from before settings)

**Step 5: Run shellcheck**

```bash
shellcheck ghostty/claude-wrapper.sh
```

Expected: No errors

**Step 6: Commit**

```bash
git add ghostty/claude-wrapper.sh
git commit -m "Handle sleep feature in delete mode

- Add timeout-based read in delete mode
- Wake ghost on any input in delete mode
- Timer continues during settings (acceptable behavior)

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Task 7: Final Testing and Verification

**Files:**
- All modified files

**Step 1: Run full test suite**

```bash
./run-tests.sh
```

Expected: All tests PASS

**Step 2: Run shellcheck on all modified scripts**

```bash
shellcheck lib/logo-animation.sh ghostty/claude-wrapper.sh
```

Expected: No errors

**Step 3: Manual end-to-end testing**

```bash
./bin/ghost-tab
```

Test checklist:
- [ ] Ghost bobs normally when menu appears
- [ ] After 2 minutes of no input, ghost gradually sleeps (2-3 sec transition)
- [ ] Sleeping ghost shows closed eyes
- [ ] Sleeping ghost shows dimmed colors
- [ ] Sleeping ghost shows "z Z Z" above head
- [ ] Sleeping ghost is completely still (no bobbing)
- [ ] Any key press immediately wakes ghost
- [ ] Waking removes Zzz instantly
- [ ] Waking resumes bobbing immediately
- [ ] Waking restores bright colors
- [ ] Timer resets on arrow keys, numbers, Enter, all inputs
- [ ] All 4 AI tool ghosts have sleeping variants
- [ ] Works with all AI tools (claude, codex, copilot, opencode)
- [ ] Respects ghost display setting (none/static/animated)

**Step 4: Test with reduced timeout for faster iteration**

Temporarily set `_sleep_timeout=10` in `ghostty/claude-wrapper.sh` for faster testing, then restore to 120.

**Step 5: Verify no regressions**

Test that existing features still work:
- [ ] Project selection
- [ ] Adding projects
- [ ] Deleting projects
- [ ] Settings menu
- [ ] AI tool switching
- [ ] Number key selection
- [ ] Mouse clicks (if enabled)

**Step 6: Final commit if any fixes needed**

If any bugs found, fix them following TDD:
1. Write regression test
2. Run to verify failure
3. Fix bug
4. Run to verify pass
5. Commit

---

## Task 8: Update Documentation (Optional)

**Files:**
- Modify: `README.md` (if feature should be documented)

**Step 1: Consider if user documentation needed**

The sleeping ghost is a subtle UX enhancement. It may not need explicit documentation in README. Skip this task unless you want to add it to a "Features" section.

**Step 2: Commit if documentation added**

```bash
git add README.md
git commit -m "Document sleeping ghost feature in README

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Success Criteria

- [ ] All 4 AI tools have sleeping ghost variants
- [ ] Sleeping ghosts show closed eyes and dimmed colors
- [ ] Zzz indicator appears above sleeping ghost
- [ ] Ghost sleeps after exactly 2 minutes of inactivity
- [ ] Transition to sleep takes 2-3 seconds
- [ ] Any key press immediately wakes the ghost
- [ ] Timer resets on all keyboard input
- [ ] No bobbing while sleeping
- [ ] Bobbing resumes immediately on wake
- [ ] Feature respects ghost display settings
- [ ] All tests pass
- [ ] shellcheck passes on all modified files
- [ ] No regressions in existing features

## Notes

- **Testing timeout**: For development, temporarily set `_sleep_timeout=10` for faster manual testing
- **Color dimming**: Each ghost uses tool-specific dimmed color codes (see design doc)
- **Performance**: 0.5s read timeout adds minimal latency to input loop
- **State preservation**: Timer naturally pauses during settings menu (acceptable UX)
- **YAGNI**: No animated Zzz rising/fading in initial version (can add later if desired)
