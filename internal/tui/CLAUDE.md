# tui — gotchas

Gotchas for the Go TUI. Loaded when Claude opens a file in this package.

### The diff pager measures text in cells, never in runes

Everything the file preview lays out — `fitColumn`, `wrapColumns`, `tintColumn`,
`truncatePath`, the popup frame — measures through `cellWidth`/`forEachCell` in
`internal/tui/diffview.go`. **Never reach for `len([]rune(s))`,
`runewidth.RuneWidth` per rune, `runewidth.StringWidth` over a whole string, or
`lipgloss.Width` on text that came out of a file.** Each of those has already
shipped a broken preview:

- Per-rune `RuneWidth` counts a Bengali vowel sign as its own cell. A locale
  file read ~50% wider than the terminal drew it, so every line truncated and
  wrapped early and the side-by-side divider walked off its column.
- Whole-string `runewidth.StringWidth` disagrees the other way: this repo's
  Unicode tables are newer than tmux's, so an Indic conjunct reads one cell
  narrower than tmux paints it.
- `lipgloss.Width` (and therefore `lipgloss`'s `Width()` padding, `Border()` and
  `Place()`) uses a third table. Letting lipgloss frame the popup re-padded rows
  that were already exact and produced 122- and 124-cell rows inside a 120-cell
  box — hence `framePopup` and `placeBox` draw the chrome by hand.
- Segmenting each colored run separately splits a letter from its accent,
  because the highlighter wraps every rune in its own SGR pair. Escapes are
  transparent to the terminal's composition, so `forEachCell` walks the
  escape-free projection of the line.

tmux owns the cell grid the popup is painted into, so tmux is the authority —
not Unicode, not a library. The model is pinned to it by
`internal/tui/testdata/tmux_widths.json`, a recording of what a real tmux
painted for 555 strings (real shipped-locale lines across ~60 languages, the
constructs that segment strangely, and every codepoint class where the tables
were ever found to disagree). `TestCellWidth_matches_a_live_tmux` re-derives it
from a live tmux on demand — run it after bumping tmux, go-runewidth or uniseg.

Where the model cannot be exact it **rounds up, never down**: over-counting
leaves a blank cell, while under-counting writes past the column edge and shoves
everything after it sideways. `TestCellWidth_never_under_counts` enforces that
direction, and it holds for all 154,996 assigned codepoints.

### The project menu polls for worktrees, because nothing tells it

`git worktree list` used to run exactly once, when the menu was built, plus on
the four changes the menu made itself (created a worktree, removed one, added a
project, deleted one). Nothing watched the filesystem and there was no refresh
key, so a worktree created in a terminal while the menu sat open simply never
appeared — the only cure was quitting and relaunching.

So `worktreeRefreshCmd` re-detects every project's worktrees on a ~2s loop and
`initCmds` arms it **unconditionally** — the ghost tickers beside it are gated
on `ghostDisplay == "animated"`, and copying that gate would leave a static-ghost
session with the original bug. Rules that fell out of building it:

- **`AppModel` delegates to `a.top()`, so the loop needs its own route.** The
  poll reschedules itself from the menu's own `Update`; delivered to the topmost
  screen it would be swallowed by the branch picker and the chain would be dead
  for the rest of the session. `worktreesRefreshedMsg` is routed to `stack[0]`.
- **Detection never touches `m.projects`.** `PopulateWorktrees` writes through
  the slice `View` is reading — a data race the moment it runs off the Update
  loop. `models.DetectWorktreesFor` returns fresh data keyed by path instead,
  and it spawns one git process per project, so it must stay in a `tea.Cmd`.
- **Results are applied by path, never by index.** A project can be added or
  deleted between the spawn and the delivery, and an index would write one
  project's worktrees onto another. A path the round did not report keeps what
  it had: absent means "not measured", not "none".
- **The cursor is anchored to the worktree's path, not its row.** Enter launches
  whatever the cursor is on; a worktree appearing above it shifts every row
  below, and re-anchoring by index would silently move the cursor onto a
  different worktree.
- **A refresh is held back mid-flow** (`inputMode`, `deleteMode`, `cloning`, a
  pending branch pick). `deleteSelected` is a flat index, so a row moving under
  an open delete confirm is how someone removes the wrong worktree.
- **An emptied project stays expanded.** `ToggleWorktrees` deliberately expands
  a worktree-less project to its lone add-worktree row, so collapsing on the
  poll would close a row the user opened on purpose —
  `reloadAfterWorktreeRemoval` prunes because it ends a removal flow, not
  because expansion implies worktrees.

The poll surfaces ephemeral worktrees too (subagent `.claude/worktrees/*`,
this suite's own temp checkouts) for as long as they exist; that is the list
being true, not a defect.

Guarded by `internal/tui/mainmenu_worktree_refresh_test.go` — including
`_picksUpAWorktreeGitCreatedAfterTheMenuOpened`, which drives the real loop
against a real repo, and `TestAppModel_deliversAWorktreeRefreshToTheMenuUnderAPushedScreen` —
plus `TestDetectWorktreesFor_*` in `test/internal/models/worktree_refresh_test.go`.

### The popup backdrop is built for a user, not for a timer

`refreshBackdrop` ran on every 2s ledger refresh, spawning `tmux
display-message`, `tmux list-panes` and one `tmux capture-pane` **per pane**,
then writing and renaming a temp file — 220-247ms, forever, in every session,
maintaining the dimmed screen behind a popup that only ever opens on a click.
It is now armed by interaction, which also makes it *fresher* than a 2s timer.

- **Throttled to one rebuild per 750ms**: a mouse crossing the pane emits a
  motion event per cell, and un-throttled that costs more than the timer did.
- **An interaction that OPENED a popup is excluded**, preserving the existing
  contract that a click never waits on or starts a refresh
  (`TestLedgerOpenClickStartsPopupOffInputLoopOnCacheMiss`). The hover that
  necessarily preceded the click is what armed it.
