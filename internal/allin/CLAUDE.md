# allin — gotchas

Gotchas for the All-In model router: one Claude settings profile whose `/model`
picker lists every configured subscription, routed through a loopback proxy
with no pane relaunch. Loaded when Claude opens a file in this package. Spec
and live measurements: `docs/all-in-model-router.md`.

### The id grammar splits on the first two segments only

A picker row's id is `wisp/<source>/<model>`, and `Route` (`route.go`) cuts it
on the first two `/`s, not all of them. A Featherless model id contains its own
slash (`TurboVadim/Qwen3.8-27B-OBLITERATED`), so cutting on every `/` would
misparse the model as another path segment. `<source>` is `acct.<dir>` (a
Claude login) or `cfg.<file-without-.json>` (a subscription profile). An id
this build cannot place — wrong prefix, empty source, empty model — returns
`KindSession` rather than an error: the turn must still run on the session's
own credential. Guarded by `TestRoute_keeps_slashes_inside_the_model_id` and
`TestRoute_treats_an_unparseable_prefix_as_the_session`.

### `[1m]` is the only way a row gets a 1M window

Measured live through `/context`: a `wisp/…` row carrying `behavesAs:
claude-opus-5` gets the flat 200k window — `behavesAs` does not carry a window
across the router. Only the literal `[1m]` suffix on the raw model string does,
because Claude Code reads that marker off the string itself. `strip1M`
(`route.go`) removes it before the id reaches `Resolve` or the upstream, and
`Want1M` carries the fact forward so `proxy.go` can add the
`context-1m-2025-08-07` beta header — the suffix and the beta header are the
only two places this bit is represented; dropping either one silently serves a
200k session to a row that promised 1M.

### A stale credential must surface as HTTP 400, never 401 or 5xx

Claude Code retries a 401, a 429, or a 5xx about eleven times before giving up.
Every routing failure in this package is deterministic — the same request
would fail again the same way — so `writeRoutingError` (`proxy.go`) always
answers 400, in Anthropic's own error envelope. A malformed upstream base URL
shipped as a 502 in an earlier draft (the raw `url.Parse` failure reached
`httputil.ReverseProxy`, which reports a dial failure as retryable) and was
caught in review; `validUpstream` now rejects a URL with no scheme or host
before the proxy ever runs. Guarded by
`TestHandler_reports_a_stale_account_as_400_not_401` and
`TestHandler_reports_a_malformed_upstream_as_400_not_502`.

### The body read goes one byte past the cap

`maxRouteBytes` bounds how large a request this handler will re-address.
`io.LimitReader(r.Body, maxRouteBytes)` alone can return an exactly-full,
silently truncated body with a nil error — there is no way to tell "the body
was exactly this size" from "the body was cut off" without seeing one more
byte arrive. The handler reads `maxRouteBytes+1` and rejects anything that
actually filled it. This matters more here than in an ordinary proxy: the
All-In profile sets `replaceBuiltInOptions: true` on its picker, so there is no
built-in row to fall back to — a truncated body's `model` field would parse to
some other row's id, or to none, and forward silently instead of failing
loudly. Same lesson `internal/rolefix` already records for its own repair
budget. Guarded by `TestHandler_rejects_a_body_over_the_routing_cap` and
`TestHandler_routes_correctly_when_content_length_is_unknown` (a chunked
request reports `ContentLength: -1`, so the cap can only be enforced by reading).

### Both credential headers are deleted before the swap, unconditionally

`r.Header.Del("Authorization")` and `r.Header.Del("X-Api-Key")` both run before
`r.Header.Set(credential.Header, credential.Value)`, even though `Credential`
only ever names one of the two today. A conditional delete — clearing only the
header the swap is about to set — would let the session's own OAuth bearer (or
API key) survive on the other header and travel to the swapped-to endpoint
alongside the new credential, the day a provider using `x-api-key` is added.
Guarded by `TestHandler_clears_both_credential_headers_before_swapping_in_one`.

### `ErrorLog` is a discarding logger

`httputil.ReverseProxy`'s `ErrorLog` field defaults to nil, which falls back to
the standard `log` package writing to stderr. In this repo the wrapped
process's stderr is the terminal Claude Code paints on (see
`wrapper-stderr-is-the-ai-pane` in project memory), so an unset `ErrorLog` here
would print raw proxy errors into the middle of the agent's own screen.
`newReverseProxy` (`proxy.go`) sets it to a `log.New(io.Discard, ...)` logger.

### `FlushInterval: -1` is load-bearing

Buffering the response would swallow the `: keep-alive` bytes a slow upstream
sends before its first token. Claude Code arms a byte-stall watchdog on every
stream and replays the whole turn after 20s of silence on the wire; a
buffering proxy would sit on those bytes and manufacture exactly that silence,
even though the upstream was sending the whole time. `newReverseProxy`
forwards each write as it arrives instead.

### The launch gate is the settings file's own rows, never the profile's display name

`gt_claude_launch_wrapper` (`lib/tmux-session.sh`) decides whether to wrap a
Claude launch in this router by grepping the settings file itself for
`wisp/acct.` or `wisp/cfg.` — the exact prefixes `roster.go` writes — never by
checking the profile's display name (`allin.ProfileName`, `"All-In"`).
Renaming a profile is reachable from the UI in more than one place, and a
renamed-but-still-routed profile would silently stop being wrapped while its
picker kept offering rows nothing was resolving.

The match must stay anchored to those two literal prefixes. A bare `wisp/`
substring also matches a statusline path or a model id under an org literally
named `wisp`, and a false positive here silently steals the launch from the
**other** branch of the same function: a Featherless profile's launch is
gated on the same `if`/`elif` chain, so a false-positive All-In match would
strip the role-repair proxy that is the only reason a Featherless pane can
call a tool at all. This was mutation-proven in review. The two proxies never
need to stack — the router already forwards to whatever endpoint a row names,
so a routed row never also needs Featherless's role-repair.

### The config root must match `wrapper.sh` and `root.go` exactly

`Env`'s four paths (`AccountsList`, `AccountsDir`, `ConfigsList`, `ConfigsDir`)
are built in `lib/tmux-session.sh` from
`${XDG_CONFIG_HOME:-$HOME/.config}/wisp-deck`, the same root `wrapper.sh` and
`cmd/wisp-deck-tui/root.go` use. A divergence here does not error — it makes
`Roster` and `Resolve` read from a directory that simply does not exist, and
`readLines`/`os.ReadFile` failures are treated as "nothing configured" rather
than surfaced, so the picker would quietly offer zero extra rows instead of
naming the wrong path.

### Models narrower than 200000 tokens are not offered

`minRosterContext` in `roster.go` is Claude Code's own floor: the tool
schemas, system prompt, and agent/skill rosters it sends before a conversation
starts already cost roughly this many tokens (see the "Claude Code's floor" and
"window holds the reply too" sections in the root `CLAUDE.md`), so a narrower
model cannot complete a single turn. `configRows`'s check is
`model.Context != 0 && model.Context < minRosterContext` — a `Context` of
exactly zero skips the floor entirely, because zero also means "the catalog
never populated this field." A `SuppliesOwnModel` provider (Featherless,
custom/self-hosted) ships no static `Models` list at all, so `providerModels`
must build the one synthetic `Model` for it by reading the profile's own
declared window (`claudeconfig.ReadContextWindow`) and rejecting it outright
when that value is missing or non-positive — returning a `Model` with
`Context` left at zero would read as "no floor" and offer a model narrower
than Claude Code can run a turn on. Guarded by
`TestRoster_omits_a_model_too_narrow_for_claude_code` and
`TestRoster_omits_a_self_hosted_model_too_narrow_for_claude_code`.

### A ChatGPT profile is skipped, on purpose

`configRows` skips any provider whose `Auth` is not `claudeconfig.AuthAPIKey`.
A ChatGPT profile authenticates through `codex login` and is served by a
bridge process, not an endpoint holding an API key — `Resolve`'s `KindConfig`
branch has no credential to hand it, so a row for it would resolve to nothing.
This is v1 scope, not an oversight: see "Решения по v1" in the spec. Guarded by
`TestRoster_omits_a_provider_that_is_not_served_by_an_api_key`.

### Known limitation: there is no token refresh

An account whose Keychain access token has expired resolves as
`ErrStaleAccount`, and the 400 response names the login by directory so the
user can fix it — but the fix is manual: open a wisp-deck tab on that login
once, which makes Claude Code itself refresh the token. Nothing in this
package attempts a refresh. This is a stated v1 decision (see "Открытые
риски" in the spec), not a bug to fix reflexively.
