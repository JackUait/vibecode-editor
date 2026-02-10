# Sleeping Ghost Feature Design

**Date:** 2026-02-10
**Feature:** Idle state for ghost mascot after 2 minutes of inactivity

## Overview

The sleeping ghost feature adds an idle state to the ghost mascot that appears after 2 minutes of user inactivity in the project chooser menu. This enhances the personality of the interface while providing subtle feedback that the system is waiting for input.

## User Experience

### Behavior
- After 2 minutes of no keyboard input, the ghost gradually transitions to a sleeping state
- The transition takes 2-3 seconds (gradual)
- Any key press immediately wakes the ghost and resumes normal behavior
- No warning is shown before the ghost goes to sleep

### Visual Appearance

**Sleeping State:**
- Eyes closed (using ─ or ▬ characters)
- Slightly dimmed/darker colors (reduced brightness on all color codes)
- "z Z Z" indicator floating above the ghost's head
- Ghost is completely still (no bobbing animation)

**Awake State:**
- Normal bright colors
- Open eyes with pupils
- Bobbing animation active
- No Zzz indicator

### Interactions

**Timer Reset Triggers:**
- Any key press resets the 2-minute inactivity timer
- This includes: arrow keys, numbers, letters, Enter, settings key, all inputs

**Wake Behavior:**
- Instant transition from sleeping to awake on any key press
- Eyes open immediately
- Zzz disappears
- Bobbing animation resumes right away
- Colors return to full brightness

## Technical Design

### Architecture

**Core Components:**

1. **Timer tracking** (`ghostty/claude-wrapper.sh`)
   - Track last interaction time using bash's `SECONDS` variable
   - Check elapsed time during main menu loop

2. **Sleep state management** (`ghostty/claude-wrapper.sh`)
   - Boolean flag `_ghost_sleeping` to track current state
   - State transitions: awake ↔ sleeping

3. **Sleeping ghost art** (`lib/logo-animation.sh`)
   - New variants: `logo_art_claude_sleeping()`, `logo_art_codex_sleeping()`, etc.
   - Dimmed colors and closed eyes

4. **Zzz renderer** (`lib/logo-animation.sh`)
   - Function to draw/clear "z Z Z" above the ghost

5. **Transition logic** (`ghostty/claude-wrapper.sh`)
   - Gradual fade to sleeping state
   - Instant wake-up

### State Management & Timer

**Variables:**
```bash
_last_interaction=$SECONDS  # Timestamp of last key press
_sleep_timeout=120          # 2 minutes in seconds
_ghost_sleeping=0           # 0=awake, 1=sleeping
```

**State Transitions:**

- **Awake → Sleeping**:
  - Trigger: `(SECONDS - _last_interaction) >= _sleep_timeout`
  - Action: Call `initiate_sleep_transition()`
  - Duration: 2-3 seconds

- **Sleeping → Awake**:
  - Trigger: Any key press when `_ghost_sleeping=1`
  - Action: Call `wake_ghost()`
  - Duration: Instant

**Timer Check Implementation:**

Since `read -rsn1` blocks, we use `read -rsn1 -t 0.5 key` with a 0.5-second timeout. This allows checking the timer every half-second even without input.

```bash
while true; do
  # Check for sleep timeout
  if [ "$_ghost_sleeping" -eq 0 ] && [ "$_LOGO_LAYOUT" != "hidden" ]; then
    if [ $((SECONDS - _last_interaction)) -ge "$_sleep_timeout" ]; then
      initiate_sleep_transition
    fi
  fi

  # Non-blocking read with timeout
  read -rsn1 -t 0.5 key || continue

  # Wake ghost if sleeping
  if [ "$_ghost_sleeping" -eq 1 ]; then
    wake_ghost
  fi
  _last_interaction=$SECONDS

  # ... existing key handling ...
done
```

### Visual Implementation

**Sleeping Ghost Art:**

Create new functions for each AI tool in `lib/logo-animation.sh`:

```bash
logo_art_claude_sleeping() {
  # Same structure as logo_art_claude() but with:
  # 1. Darker color codes (e.g., 209→166, 255→250)
  # 2. Closed eyes on lines 5-6
}

logo_art_codex_sleeping() { ... }
logo_art_copilot_sleeping() { ... }
logo_art_opencode_sleeping() { ... }
```

**Closed Eyes Design:**

Replace eye rows (lines 5-6 in the logo array):

```bash
# Awake eyes (white with black pupils):
"${O}████${W}███${K}██${O}██████${W}███${K}██${O}████"
"${O}████${W}███${K}██${O}██████${W}███${K}██${O}████"

# Sleeping eyes (closed with horizontal lines):
"${D}████${K}▬▬▬▬${D}██████${K}▬▬▬▬${D}████"
"${D}████████████████████████"
```

**Zzz Indicator:**

```bash
draw_zzz() {
  local row=$1 col=$2
  moveto "$row" "$((col + _LOGO_WIDTH - 8))"
  printf "${_DIM}z${_NC}"
  moveto "$((row + 1))" "$((col + _LOGO_WIDTH - 6))"
  printf "${_DIM}Z${_NC}"
  moveto "$((row + 2))" "$((col + _LOGO_WIDTH - 4))"
  printf "Z"
}

clear_zzz() {
  local row=$1 col=$2
  # Clear 3 rows where Zzz was drawn
  for i in 0 1 2; do
    moveto "$((row + i))" "$((col + _LOGO_WIDTH - 8))"
    printf "          "  # Clear enough space
  done
}
```

Position: 2-3 rows above ghost's head, offset to the right.

### Animation & Transitions

**Sleep Transition Function:**

```bash
initiate_sleep_transition() {
  # Stop bobbing immediately
  stop_logo_animation

  # Pause for gradual effect (2.5 seconds)
  sleep 2.5

  # Draw sleeping ghost with closed eyes and dimmed colors
  draw_logo_sleeping "$_logo_row" "$_logo_col" "$SELECTED_AI_TOOL"

  # Draw Zzz indicator
  draw_zzz "$((\_logo_row - 3))" "$_logo_col"

  _ghost_sleeping=1
}
```

**Wake Function:**

```bash
wake_ghost() {
  # Clear Zzz
  clear_zzz "$((\_logo_row - 3))" "$_logo_col"

  # Resume bobbing animation (draws awake ghost)
  start_logo_animation "$_logo_row" "$_logo_col" "$SELECTED_AI_TOOL"

  _ghost_sleeping=0
  _last_interaction=$SECONDS
}
```

**Helper Function:**

```bash
draw_logo_sleeping() {
  local row=$1 col=$2 tool=$3
  "logo_art_${tool}_sleeping"
  for line in "${_LOGO_LINES[@]}"; do
    moveto "$row" "$col"
    printf '%b' "$line"
    row=$((row + 1))
  done
}
```

### Integration Points

**Main Changes:**

1. `ghostty/claude-wrapper.sh` - Main menu loop (around line 273)
   - Initialize timer variables
   - Change `read -rsn1` to `read -rsn1 -t 0.5`
   - Add sleep timeout check
   - Add wake-on-keypress logic

2. `lib/logo-animation.sh` - Animation functions
   - Add `logo_art_*_sleeping()` functions (4 variants)
   - Add `draw_logo_sleeping()` function
   - Add `draw_zzz()` and `clear_zzz()` functions

**Preserved Functionality:**

- Ghost display setting respected (`$_LOGO_LAYOUT != "hidden"`)
- All existing key bindings unchanged
- Settings menu interaction resets timer
- Delete mode interaction resets timer
- AI tool switching resets timer
- Window resize handling preserved
- Mouse click handling preserved

## Testing Strategy

### Automated Tests (`test/logo-sleep.bats`)

Following test-first development:

1. **Timer initialization**
   - Verify `_last_interaction` set on loop start
   - Verify `_ghost_sleeping=0` initially

2. **Sleep trigger**
   - Mock time passing 120 seconds
   - Verify `initiate_sleep_transition` called
   - Verify sleeping art functions exist

3. **Wake on key press**
   - Set ghost to sleeping state
   - Simulate key press
   - Verify `wake_ghost` called
   - Verify timer reset

4. **Timer reset**
   - Verify all key types reset timer
   - Test: arrows, numbers, letters, Enter, etc.

5. **Sleeping ghost art**
   - Verify all 4 AI tools have sleeping variants
   - Verify closed eyes in art
   - Verify dimmed colors

6. **Zzz drawing**
   - Test `draw_zzz()` renders correctly
   - Test `clear_zzz()` clears area
   - Test positioning above ghost

7. **Ghost display settings**
   - Verify sleep feature disabled when `_LOGO_LAYOUT=hidden`

### Manual Testing

- Test with reduced timeout (10 seconds) during development
- Verify smooth visual transition
- Test all AI tool ghost variants
- Test different terminal sizes
- Test interaction types (arrows, numbers, settings, etc.)

## Implementation Notes

### Color Dimming Reference

For each AI tool, reduce brightness of all colors:

**Claude (orange theme):**
- 209 (orange) → 166 (darker orange)
- 208 (deep orange) → 166
- 223 (peach) → 180
- 220 (gold) → 178
- 255 (white) → 250

**Codex (green theme):**
- 114 (green) → 71 (darker green)
- 113 (yellow-green) → 71
- 157 (light green) → 114
- 78 (teal) → 71

**Copilot (purple theme):**
- 141 (purple) → 98 (darker purple)
- 140 (light purple) → 98
- 183 (lavender) → 140
- 134 (magenta) → 98

**OpenCode (monochrome):**
- 255 → 244
- 252 → 242
- 250 → 240
- 246 → 238
- 244 → 236

### Edge Cases

1. **Ghost hidden**: Don't start timer if `_LOGO_LAYOUT=hidden`
2. **Settings menu**: Timer continues in background, ghost stays in current state
3. **Delete mode**: Timer continues, ghost stays in current state
4. **Window resize**: If sleeping, stay sleeping; if awake, stay awake
5. **Very small terminals**: Ghost already hidden, no sleep feature active

### Performance Considerations

- `read -rsn1 -t 0.5` adds 0.5s latency to input check loop (acceptable)
- No background processes needed for sleep state (unlike bob animation)
- Minimal CPU usage during sleep state

## Success Criteria

- [ ] Ghost transitions to sleep after exactly 2 minutes of inactivity
- [ ] Transition is gradual (takes 2-3 seconds)
- [ ] Any key press immediately wakes the ghost
- [ ] Sleeping ghost shows closed eyes, dimmed colors, and Zzz
- [ ] Sleeping ghost is completely still (no bobbing)
- [ ] Awake ghost resumes bobbing immediately on wake
- [ ] All 4 AI tool ghosts have sleeping variants
- [ ] Timer resets on any key press
- [ ] Feature respects ghost display settings
- [ ] All tests pass
- [ ] shellcheck passes on modified files
