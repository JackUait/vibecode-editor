# usage — gotchas

Gotchas for usage accounting. Loaded when Claude opens a file in this package.

### A Codex conversation is spread over many rollout files, and they replay each other

Claude keeps a conversation in one transcript, so per-file dedup is enough for it.
Codex does not. Forking a subagent, resuming a thread, and compacting each open a
**fresh** rollout file that begins by replaying the ancestor's entire
`token_count` history, re-stamped with the load instant. One real month held 1,684
rollout files that were only **43 session chains** — a single chain spanned 448
files — and summing every `last_token_usage` reported **466B tokens / $309K**
against **14.9B / ~$10.5K** actually spent. A **31x** over-count, shipped and
visible in the Stats tab.

A replayed record is byte-identical to a live one, so nothing about a single line
identifies it. What separates them is **density**: a replay dumps hundreds of
events inside one second, while a real request round-trip takes seconds. So
`ParseCodexRollout` drops every event in a second holding more than
`codexReplayBurstPerSecond` events, and collapses a verbatim re-emission of the
request it just counted (Codex repeats one). Measured against the full corpus,
those two per-file rules reproduce the cross-file **deduplicated** truth to three
decimal places while keeping **100%** of distinct requests — which is what lets
dedup stay per-file and keeps the incremental `Aggregate` cache intact.

Consequences to respect:

- **Never restore a plain "sum every `last_token_usage`" reading**, and never
  reach for `total_token_usage`: it is cumulative *including* the replays, so a
  forked rollout's final value hit 2.42B for one thread.
- **Changing this parser requires bumping `cacheVersion`** (`internal/usage/cache.go`).
  The per-file cache *and* the append-only journal both store parsed months
  tagged with a parser version, and only a higher version supersedes them —
  without the bump the old inflated numbers are simply reloaded.
- Guarded by the `TestParseCodexRollout_dropsReplayedHistoryBurst`,
  `_collapsesConsecutiveDuplicateTokenCounts`, and `_keepsBurstFreeRapidRequests`
  tests in `internal/usage/codex_test.go` — the last one is the counterweight, so
  a future tightening cannot start eating genuinely busy seconds.
