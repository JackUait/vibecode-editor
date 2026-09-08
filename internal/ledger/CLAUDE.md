# ledger — gotchas

### An untracked ledger row is not always a file, so discard needs `clean -ffd`

`git ls-files --others` — the query BOTH renderers build the "new" group from —
does not descend into a nested repository. A subagent's
`.claude/worktrees/<name>` checkout therefore arrives as **one row whose path is
`"<dir>/"`**, not as its files. Both renderers already knew that at *display*
time (`inspectWorktreeFile`, `untracked_numstat`) and neither knew it at
*discard* time.

`git clean -f` removes no directory at all, and refuses a nested repository even
with `-d` — **exiting 0 and printing nothing either way**. So discarding those
rows reported success, cleared the selection, refreshed, and left every one of
them exactly where it was. It shipped as "discard doesn't work" on a tab full of
agent worktrees.

- **The flags are `-ffdq`, unconditionally.** `-d` reaches a directory and the
  second `-f` reaches a nested repository; both are no-ops on a plain file, so a
  trailing-slash gate would only add a branch. This does delete a nested
  repository's own history — which is what discarding a row the ledger listed
  means, and strictly better than the silent no-op it replaced.
- **Removing the checkout is half of it.** The registration in `.git/worktrees/`
  survives, and the project menu polls `git worktree list` every ~2s
  (`DetectWorktreesFor`) without filtering `prunable` — so a discarded worktree
  came back as a menu row launching into a path that no longer exists.
  `git worktree prune` follows the removal; it only drops registrations whose
  directory is already gone, so it can never unregister a live worktree.
- **Both renderers, as always.** A pane picks between the Go ledger and
  `lib/compact-view.sh` by binary capability, so a fix in one is a fix for half
  the users.

Guarded by `TestDiscardRemovesAnUntrackedNestedWorktree` and
`TestDiscardUnregistersTheWorktreeItRemoved`
(`internal/ledger/actions_test.go`, which takes the path from a real
`Source.Load` so the test exercises the `"<dir>/"` shape production produces),
plus `TestDiscardWorktreeFile_deletes_an_untracked_nested_worktree`
(`test/bash/compact_view_test.go`).
