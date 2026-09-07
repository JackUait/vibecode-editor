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

### Every row is 200k, and the roster emits no `[1m]` — on purpose

Measured live through `/context`: a `wisp/…` row carrying `behavesAs:
claude-opus-5` gets the flat 200k window, so `behavesAs` does not carry a window
across the router. The literal `[1m]` suffix on the raw model string does,
because Claude Code reads that marker off the string itself.

**`roster.go` no longer writes it.** A 1M window granted off the model string is
granted to the whole *session*, and nothing narrows it again when the user picks
a 200k row later in the same conversation: by then the transcript is already
past the new endpoint's cap, and `/compact` cannot escape it — it sends that
same oversized transcript plus a summarization prompt, so it is larger than the
turn that already failed. This is the unrecoverable class the root `CLAUDE.md`
documents, and `_guard_subscription_context` — which catches it on a
subscription *switch* — does not run on a `/model` pick. A uniform 200k across
every row is the one shape no pick can wedge, so the capability was traded away
rather than shipped as a trap.

The machinery it needed is intact and still tested: `strip1M` (`route.go`)
removes the marker before an id reaches `Resolve` or the upstream, `Want1M`
carries the fact forward, and `proxy.go` adds the `context-1m-2025-08-07` beta
header. 1M returns by putting a size guard in front of that, never by
re-emitting the suffix from the roster. Guarded by
`TestRoster_never_offers_a_1m_row`.

The generated profile therefore declares its own window, because the rows are
not the only model string in play: the session's *starting* model comes from the
user's global settings, and a global `opus[1m]` would grant 1M before any row is
picked. Every other sub-1M profile gets that from `stampContextBudget`, which
returns early here — All-In has no model mappings for it to size a window from —
so this is the one profile that must declare the set itself.

It is a set of **four** keys, not one. `CLAUDE_CODE_DISABLE_1M_CONTEXT` alone is
not the guard: the decoded `sae()` is read by `Ov()` only, so it gates the
string-marker branch of the window choice while
`betas.includes(1m) && EW(model)` and `L2(model)` reach 1e6 ungated —
`CLAUDE_CODE_AUTO_COMPACT_WINDOW` is the direct cap on current versions and
`DISABLE_1M_CONTEXT` covers older ones. `routerEnv` writes
`CLAUDE_CODE_MAX_CONTEXT_TOKENS=200000`,
`CLAUDE_CODE_AUTO_COMPACT_WINDOW=200000`, `CLAUDE_CODE_DISABLE_1M_CONTEXT=1` and
`CLAUDE_CODE_MAX_OUTPUT_TOKENS=32000` — exactly what `contextWindowEnv` computes
for a 200000 window, because `bin/wisp-deck` runs `ensure-budget` over this file
on every install and a set that disagrees is rewritten every time.
`TestEnsureProfile_survives_the_context_budget_sweep` is what holds the two
together, and it pins the two window values in both directions; the reserve is
pinned by `TestEnsureProfile_declares_every_key_a_200k_window_implies` instead,
because the sweep deliberately keeps any declared reserve below the window as
the user's own figure.

### `Row` declares no `behavesAs`, and re-adding one costs the pane its effort control

Measured on a live pane: an unknown model id **without** `behavesAs` is given
`thinking: {"type":"adaptive"}` and keeps effort available; the same id **with**
`behavesAs: claude-sonnet-4-5` is given `thinking: {"type":"enabled", budget}`
and the pane reports "Effort not supported". Absent is the more capable default,
and (per the measurement above) it carries no context window across the router
either — so the field bought nothing and cost the pane a control. It was
declared on `Row` and never set by anything; it is now gone. Guarded by
`TestRoster_never_declares_behaves_as`, which reads the serialized rows so it
fails on a field that is re-declared *and* populated.

### The roster's exclusions are advice until `Resolve` enforces them too

`configRows` decides what the picker OFFERS. `Resolve` decides what the router
will actually address, and a model id arrives off the wire — hand-typed into
`/model`, or saved as a picker default by an older roster — so a row the roster
would never have written still reaches it. `routableProfile` (`credential.go`)
therefore re-applies the same two rules: not `AuthAPIKey` is refused (a ChatGPT
profile has no key to swap in, and so does the All-In profile itself, which
sits in the very same configs list), and `RemoteCatalog` is refused (Featherless
needs `internal/rolefix`'s repairs to call a tool at all).

It must key on `RemoteCatalog`, never on `SuppliesOwnModel()` — that is true for
a self-hosted profile too, and a self-hosted endpoint speaks the Anthropic API
directly and needs no repair. Guarded by
`TestResolve_refuses_a_provider_the_roster_would_not_offer`,
`TestResolve_refuses_the_router_profile_itself`, and the two
`TestResolve_still_serves_*` counterweights that stop the refusal widening.

### Every error carries the `allin:` prefix, never `wisp-deck:`

`writeRoutingError` renders `fmt.Sprintf("wisp-deck: %v", err)` itself, so an
error that already begins with that string reaches the user as
`wisp-deck: wisp-deck: …`. Every error raised in this package uses the package
prefix instead. Guarded by `TestHandler_never_doubles_the_wisp_deck_prefix`.

### The rewrite fails closed, because failing open sends a credential with a routing id

`rewriteModel` (`proxy.go`) refuses a nil payload and a body that will not
re-encode, and the handler answers 400 rather than forwarding. Both are
unreachable today, and only because a non-local invariant holds: `Route` answers
`KindSession` for the empty id a nil payload yields, and a payload decoded from
JSON always re-encodes. Failing open there is worse than it looks — the
credential is swapped **before** the body is rewritten, so a third-party
endpoint would receive an unresolvable `wisp/…` id while holding someone's real
token. Guarded by `TestRewriteModel_refuses_a_body_it_cannot_re_address`.

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
actually filled it, and it never consults `r.ContentLength`: a chunked request
reports `-1` there, so a cap enforced on that field passes the very body it
exists to stop. This matters more here than in an ordinary proxy: the
All-In profile sets `replaceBuiltInOptions: true` on its picker, so there is no
built-in row to fall back to — a truncated body's `model` field would parse to
some other row's id, or to none, and forward silently instead of failing
loudly. Same lesson `internal/rolefix` already records for its own repair
budget. Guarded by `TestHandler_rejects_a_body_over_the_routing_cap` and
`TestHandler_routes_correctly_when_content_length_is_unknown` (a chunked
request reports `ContentLength: -1`, so the cap can only be enforced by reading),
and by `TestHandler_rejects_an_over_cap_body_that_declares_no_length`, which is
the one that actually kills the `ContentLength`-plus-plain-`LimitReader` mutant:
the two tests before it both pass while a truncated body is forwarded with a
200.

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
call a tool at all. This was mutation-proven in review.

### Featherless is excluded from the roster, because the router does no repair

`configRows` (`roster.go`) skips any provider with `RemoteCatalog` set —
currently only Featherless — even though it is `AuthAPIKey` and would
otherwise pass the ChatGPT-exclusion check above. `internal/rolefix` exists
because Featherless 400s on Claude Code's `role:"system"` capability
listings and silently stops parsing tool calls once a request carries a
`thinking` field; both are things Claude Code sends on every turn. This
router does neither repair, so an unfiltered Featherless row would fail on
its first turn — either a hard 400, or the failure mode the root `CLAUDE.md`
already documents for an unrepaired Featherless pane: it can "look alive and
still do nothing," rendering a model's raw tool-call markup as plain text
while no tool ever runs.

`UserConfigured` (the self-hosted `custom` provider) is deliberately NOT
skipped by the same check: it speaks the Anthropic API directly and needs no
repair. The two flags look similar (`SuppliesOwnModel()` is true for both)
but only `RemoteCatalog` names the one needing a proxy this router doesn't
have — checking `SuppliesOwnModel()` instead of `RemoteCatalog` would wrongly
exclude every self-hosted profile too. Re-admitting Featherless requires
composing `rolefix`'s request/response repairs into this router; `rolefix`'s
own repair function is unexported and not reusable as-is. Guarded by
`TestRoster_omits_featherless_because_the_router_has_no_role_repair`.

### `Env` is built twice, from two different files, and both must agree

`Roster` and `Resolve` are never called from the same `Env`. `Roster` (via
`EnsureProfile`, through the shared `EnsureProfileIfEligible` gate) is reached
from the CLI's `ensure-allin`/`add`/`delete` (`cmd/wisp-deck-tui/claude_config.go`)
and from the TUI's own login and subscription add/delete
(`internal/tui/subscription_modal*.go`, via `(*MainMenuModel).ensureAllIn`) —
every one of those call sites builds its `allin.Env` from
`${XDG_CONFIG_HOME:-$HOME/.config}/wisp-deck`, whether through `bin/wisp-deck`'s
`CONFIGS_DIR` (`bin/wisp-deck:194`) or the TUI's own `gt_config_dir`
(`lib/menu-tui.sh`). `Resolve` is reached only from the launch wrapper:
`gt_claude_launch_wrapper` (`lib/tmux-session.sh`) builds its own
`config_root` and passes it to `claude-allin`
(`cmd/wisp-deck-tui/claude_allin.go`), which never runs `Roster` or
`EnsureProfile`.

The two are **textually independent recomputations** of
`${XDG_CONFIG_HOME:-$HOME/.config}/wisp-deck` — `CONFIGS_DIR` is not exported,
and `lib/tmux-session.sh` never reads it. Editing one without the other does
not error: it makes the roster get built from one directory while the router
resolves credentials from another. A row's picker id would then name a
profile the router's `ConfigsDir` cannot find, and `Resolve` would return
`allin: profile "…" is not ready` for a profile that plainly is. Change the
config root in `bin/wisp-deck` and `lib/tmux-session.sh` together, or not at
all.

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

### Known exposure: the loopback port mints turns on any credential, unauthenticated

The router will mint a turn on **any** login's OAuth token or **any** provider's
API key, chosen by a string in the request body, for anything that can reach its
loopback port. `claude-rolefix` does not do this — it forwards the credential the
client already had — so this is a real escalation over the proxy it is modelled
on, and it is recorded here rather than fixed.

It is **not** an escalation over what a local process on this machine can already
do. `keychainToken` shells out to `security find-generic-password -w`, which
returns without a prompt for the user's own login keychain, and every provider key
sits in a 0600 file the same process can read. Anything that can open the loopback
port can already read both sources directly.

The right end state is a shared secret: mint a random id at launch, stamp it into
the settings overlay as a header the client sends, and refuse a request without
it — the same shape `gptbridge` already uses for `randomBridgeID`. It was
deliberately **not** bundled with the profile-shape redesign this wave: an auth
scheme added alongside a Critical fix, with one review pass left, is how a worse
bug ships. Add it as its own change, with its own tests.

### The stream watchdog is disarmed inside `EnsureProfileIfEligible` itself, not left to a sweep

A profile can now be born on a keypress — the moment a second subscription
connects, from the TUI or the CLI's `add`/`delete` — never touching
`bin/wisp-deck`'s `ensure-watchdog` sweep at all. `EnsureProfileIfEligible`
therefore calls `claudeconfig.EnsureStreamWatchdog` itself, right after
`EnsureProfile` writes the file, rather than relying on that install-time
sweep to catch up on the next run. Without it, an All-In pane routed to a
gateway sits with the event-tier watchdog armed for its whole life: root
`CLAUDE.md`'s "keepalive buys 30 pings" section measures the consequence — a
stream carrying only keepalives is aborted and replayed at 610s, and the
replay repeats the same work.

**Never move this into `routerEnv`.** That block rewrites its keys on every
call, while a declared watchdog value is documented as the user's own and is
kept untouched by every other path (`stampStreamWatchdog` already no-ops on a
key that is already declared) — writing it there would blow away a value the
user set deliberately. `EnsureStreamWatchdog` needs the settings object to
already have an `env` key, which `routerEnv` guarantees the file always has by
the time this runs. Guarded by
`TestEnsureProfileIfEligible_disarms_the_stream_watchdog_on_a_freshly_created_profile`.

### `ensure-allin` and `claude-allin` name the same two files the same way

Both take `--configs-list` and `--configs-dir` (plus `--accounts-list` and
`--accounts-dir`). `ensure-allin`'s siblings under `claude-config` use bare
`--list`/`--dir`, but this command already takes an explicit `--accounts-*`
pair, so the bare names identified the configs pair only by omission — and the
two commands read the same files from the same roots. `bin/wisp-deck` passes the
long names; see also the `Env` section above, which is what those roots must
agree with.

### Known limitation: there is no token refresh

An account whose Keychain access token has expired resolves as
`ErrStaleAccount`, and the 400 response names the login by directory so the
user can fix it — but the fix is manual: open a wisp-deck tab on that login
once, which makes Claude Code itself refresh the token. Nothing in this
package attempts a refresh. This is a stated v1 decision (see "Открытые
риски" in the spec), not a bug to fix reflexively.

### `allin` is a third key in `get_claude_config_provider`'s allowlist, and it is admitted for a different reason than the other two

`ConfigReady` (`internal/claudeconfig/claudeconfig.go`) treats `AuthWispRouter`
as always ready — the profile's credentials are resolved per request from the
Keychain, so there is nothing local to check. That marks the switcher's
All-In row selectable on the Go side, but the shell side that actually
performs the switch (`lib/account-switch.sh`) never learned the same fact:

- `get_claude_config_provider` (`lib/claude-configs.sh`) only returns a marker
  it allowlists; `allin` was absent, so it read as no marker at all.
- With no marker, `_subscription_choice_ready`'s name-fallback `case`
  (`lib/account-switch.sh`) matches nothing and lands on `provider=zhipu`.
- Zhipu is an `AuthAPIKey` provider in spirit, so readiness fell through to
  `jq -er '.env.ANTHROPIC_AUTH_TOKEN | select(...)'` — a check the generated
  All-In profile can never pass, because it deliberately carries no
  `ANTHROPIC_AUTH_TOKEN` (`routerEnv` in `profile.go`).

So clicking All-In in the pill's menu did nothing: the row was offered
(`ConfigReady` said yes) and then silently refused (the shell said no).

`allin` joins the allowlist for a **different reason** than `featherless` (the
gateway needing the repair proxy) or `openai-chatgpt` (the GPT bridge): no
shell launch decision is keyed to it — `wrapper.sh`, `lib/tmux-session.sh` and
`lib/account-switch.sh` all branch only on `openai-chatgpt` or `featherless`
literal strings, never on `allin`. It exists purely so
`_subscription_choice_ready` can name the router profile and give it its own
branch — `return 0` before the token check — mirroring the honest parallel
already in that function: the ChatGPT branch, which likewise cannot be judged
by a token.

Guarded by `TestApplyAccountSwitchChoice_allin_relaunches_without_a_token`
(`test/bash/account_switch_subscription_test.go`), which fails two different
ways depending on which half of the fix is missing: no allowlist entry means
`get_claude_config_provider` reports no marker at all, while no `allin` branch
in `_subscription_choice_ready` means the marker resolves correctly but the
switch is still refused on the missing token.
