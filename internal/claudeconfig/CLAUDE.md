# claudeconfig — gotchas

### A self-hosted subscription supplies what the catalog cannot

The `custom` provider is a subscription whose endpoint and model belong to the
user, not to a vendor. Everything the catalog answers for a gateway — base URL,
model ids, context window, price — has no answer here, so the Subscription modal
offers three text fields (Endpoint, Model, Context) where a gateway gets the
alias cycler. The cycler is not merely unhelpful there: `cycleSubscriptionMapping`
returns early on an empty model list, so those rows are inert.

- **It is LAST in `Providers`, and must stay there.** `Providers[0]` is the
  fallback for every profile name matching no alias, so a user-configured
  provider in that slot would claim every stray config on the machine.
- **The save path must skip `WriteModelMappings`.** That function writes the four
  aliases from the draft's model list, which is empty for this provider — so
  running it deletes the model the user just typed, on every save. The regression
  guard is `TestSubscriptionModal_savingACustomProfileKeepsItsModelMapping`.
- **The window is entered, never inferred.** `ContextBudget` cannot size a model
  the catalog has never heard of, and `stampContextBudget` deliberately keeps
  whatever the profile already had in that case — which is exactly what lets a
  hand-typed figure survive `WriteModelMappings` and the `ensure-budget` sweep
  `bin/wisp-deck` runs on every install. It must equal the endpoint's real limit;
  overshooting it is the unrecoverable wedge described below.
- **Every alias names the one model.** `/model` and subagents move freely across
  all four, so a partially mapped profile launches some tiers with no model at
  all. An absent default is written as no key rather than an empty string: a
  blank `ANTHROPIC_DEFAULT_OPUS_MODEL` reaches Claude Code as a real, broken id.
- **A key is still required.** `ConfigReady` gates on it, and these endpoints
  routinely sit on public URLs (a `proxy.runpod.net` host is reachable by anyone
  who guesses it).
- **`MirrorOpenCode` stays off.** OpenCode's catalog cannot size a model nobody
  has published, and `Sync` skips non-mirrored providers outright.

The endpoint has to speak the **Anthropic Messages API** — `/v1/messages`,
streaming, `tool_use`. An OpenAI-compatible server (vLLM, SGLang, Ollama) needs a
translating proxy in front of it; without tool calls Claude Code cannot work at
all.

Guarded by `internal/claudeconfig/custom_provider_test.go` and
`internal/tui/subscription_modal_custom_test.go`.

### A self-hosted pane disarms the byte stall watchdog, because nothing pings it

Claude Code arms a raw-byte stall watchdog on every `text/event-stream` body it
receives. It measures bytes on the wire, not events: 20s of silence paints
`Waiting for API response · will retry in <N> · check your network`, and at the
end of the budget it aborts the stream and replays the whole turn.

Its premise is that a healthy stream is never silent, which holds for Anthropic
and for the vendor gateways, because they forward Anthropic's own `event: ping`.
An endpoint the user supplies promises nothing of the sort — a self-hosted model
prefilling a large prompt sends no bytes at all until its first output token. So
on a `custom` profile the watchdog reports a working model as a broken network,
then kills the turn it was working on.

Decoded from 2.1.247 and then measured against a live pane, because the decode
alone hides the second half:

- **The 20s trigger is a hardcoded interval**, not a budget. No env var moves it,
  and `CLAUDE_ENABLE_STREAM_WATCHDOG` does not cover it — that flag gates the
  abort timers only, while the banner timer is armed either way. The single
  lever is `CLAUDE_ENABLE_BYTE_WATCHDOG`: falsy means the instrumented body is
  never installed, so the banner has nothing to read and cannot fire.
- **A custom base URL is still `firstParty`.** `getAPIProvider()` answers from
  the `CLAUDE_CODE_USE_*` flags alone and never looks at `ANTHROPIC_BASE_URL`,
  so a self-hosted pane draws the **180s** first-party abort budget rather than
  the 300s one — measured, not inferred: the live banner counted down from
  `2m 37s` at the 20s tick.
- **It is not a cosmetic banner.** A mock endpoint byte-silent for 200s was
  aborted at 180s and re-dispatched; the replay is the same prompt, so it takes
  the same time and is killed again — a self-hosted endpoint slower than 180s to
  first token can never complete a turn. The same profile carrying the key ran
  that turn to completion in one attempt, with no banner.
- **Only a user-configured profile is disarmed.** A gateway heartbeats, so
  disarming it there trades a real dead-connection signal for nothing.
- **The provider comes from the `WISP_DECK_SUBSCRIPTION_PROVIDER` marker**, never
  from the filename: an unmatched name resolves to `Providers[0]`, so
  "qwen.json" would read as Zhipu.
- **A declared value is never overwritten,** exactly like `stampContextBudget` —
  the user may have armed it on an endpoint that does keep its stream warm, and
  every launch path may run the sweep.
- **The installer copies a default profile only when the file is absent**, so a
  profile written before this was declared is reachable only by a sweep:
  `claude-config ensure-watchdog`, which `bin/wisp-deck` runs beside
  `ensure-budget`. The modal's own custom-field save runs it too, so a profile
  self-heals the moment the user edits it.

This is the same failure the GPT bridge answers with `event: ping` every 10s —
there wisp-deck owns the server and can keep the socket warm, here it owns only
the profile. What still stands for a self-hosted pane is the separate *event*
watchdog, which aborts a turn after `CLAUDE_STREAM_IDLE_TIMEOUT_MS` (minimum and
default 300s) with no SSE events.

Guarded by `internal/claudeconfig/bytewatchdog_test.go`,
`TestSubscriptionModal_savingACustomProfileDisarmsTheByteWatchdog`, and
`test/bash/byte_watchdog_sweep_test.go`.

### A keepalive buys 30 pings, so every pane disarms the EVENT watchdog too

The byte watchdog above is one of **two** stall watchdogs, and answering it with
a ping does not answer the other. Decoded from 2.1.263 and then measured live:

```js
var Fis=1e4, Bis=30;                       // 10s probe, 30-ping cap
async function*Uis(e,t,r=Fis){ …           // t = the byte tracker
  if(I===E){ if(t.lastAt>d){ d=…; yield {type:"ping"} } continue } … }
…
if(VV(Ma)){ if(Vy++, Vy<=Bis) nW(); else if(!XI||nS) continue; … }
Vy=0, nW();                                // a REAL event resets the counter
```

`nW()` arms `zj=setTimeout(… "Streaming idle timeout: no chunks received", VR)`
with `VR = Math.max(CLAUDE_STREAM_IDLE_TIMEOUT_MS||0, 300000)`. So the client
**synthesizes** a `{type:"ping"}` event every 10s while raw bytes keep arriving,
re-arms the idle timer for at most **30 consecutive** synthetic pings, and then
lets it run out — aborting the stream and replaying the whole turn. A turn that
emits no **real** Anthropic stream event for `30 x 10s + 300s` is killed.

Measured against a live 2.1.263 client with a mock endpoint that sends only
`event: ping`: **400s of pings completed in one attempt; 700s of pings was
aborted and re-POSTed at 610s**; the same 700s stream with
`CLAUDE_ENABLE_STREAM_WATCHDOG=0` completed in **one** attempt.

The premise — a healthy stream emits a real event within 10 minutes — holds for
Anthropic's API and for nothing wisp-deck points Claude Code at. The GPT bridge
forwards **only** keepalives while Codex reasons (`display:"omitted"` maps to
`Effort==""`, so `stream.go` drops every reasoning delta), a gateway routes to
models that think for minutes, and a self-hosted endpoint sends nothing at all
until its first token. Worse, the replay repeats the same work, so a turn slow
enough to exceed the ceiling *every* time can never complete — which is what
"compaction never finishes" looks like.

- **Every provider, not just the user-configured ones**, unlike
  `stampByteWatchdog`. A gateway forwards Anthropic's own `event: ping`, which
  the SDK drops *before* the stream loop — so a gateway keepalive feeds the byte
  tier and the 30-ping cap exactly like a self-hosted silence does.
- **The byte tier stays armed wherever it is today.** It measures raw bytes,
  which is what a keepalive honestly reports, so it still catches an endpoint
  that has actually gone away. Disarming both would leave a wedged bridge with
  no abort at all.
- **Two delivery points, because there are two launch shapes.**
  `BuildClaudeEnvironment` stamps the bridge's child env; every subscription
  profile declares the key in `env`, which the launch overlay copies verbatim.
- **A declared value is the user's own and is kept**, exactly like the byte
  watchdog's key, and the `ensure-watchdog` sweep repairs profiles written
  before it existed.
- **Raising `CLAUDE_STREAM_IDLE_TIMEOUT_MS` is not the fix.** It also raises the
  byte timeout (`dno`'s `o=t` when the key is set), it is clamped at 1800000,
  and it only moves the ceiling — 30 pings plus 30 minutes is still a ceiling.

Guarded by `internal/claudeconfig/streamwatchdog_test.go`,
`internal/gptbridge/adapter_streamwatchdog_test.go` (including
`_LeavesTheByteWatchdogAlone`), and
`test/bash/stream_watchdog_launch_test.go` (the overlay end of the chain, which
goes red if the overlay is ever narrowed).

### Claude Code's floor is ~20,000 tokens, so a 32K model is not a small model

Everything Claude Code sends before the conversation starts was measured on the
wire on 2026-09-02, by pointing a bare headless pane (`--strict-mcp-config`, an
empty `CLAUDE_CONFIG_DIR`, a two-file project) at a logging proxy and posting the
captured request to Featherless's `/v1/chat/completions`, which reports the
prompt cost its `/v1/messages` route does not:

| part | bytes |
|---|---|
| 26 tool schemas | 65,526 |
| system prompt | 7,037 |
| agent + skill rosters (a `role: "system"` message) | 7,390 |
| **charged by the endpoint** | **19,838 tokens** |

A profile also reserves a quarter of the window for the reply, so a
32,768-token model has `32768 - 8192 - 19838` = **4,738 tokens** for the whole
conversation. One file read spends that. The model is not "small for long
sessions" — it cannot finish one task.

That is **88% of Featherless's tool-calling catalog**: of 15,573 models
declaring `tool_use` and a context length, 13,712 are exactly 32768, another
1,835 are 4096 or 8192, and **26** are 65536 or wider. So `featherless.Parse`
drops anything under `MinContext`, for the same reason it already drops a model
without tool calling: it produces a pane that cannot do the work.

- **`MinContext` is derived, not chosen.** The room left for the conversation
  (`window - window/4 - floor`) has to be at least as large as the floor itself,
  which puts the bar at 53,334; 65536 is the next power of two, and the catalog
  holds nothing between 32768 and 131072 anyway.
- **`ClaudeCodeFloorTokens` and the proxy's estimator are pinned to one
  recording.** `test/internal/rolefix/testdata/claude-code-first-turn.json` is that
  captured request, and `TestEstimateInputTokens_matches_what_the_endpoint_charged`
  holds the estimate to what Featherless billed for it. It lives under `test/`
  because the host-effect audit reads every tracked text file under `internal/`
  as production source, and this one is a verbatim capture of Claude Code's own
  tool descriptions. Re-capture it when
  Claude Code's tool set changes shape; both numbers move together.

### A text-only model is declared by the profile, because Claude Code has no flag for it

Claude Code sends images to whatever `ANTHROPIC_BASE_URL` points at. Decoding
2.1.247 turns up **no** lever to stop it: there is no `supportsVision` on a
model, no vision capability consulted before a block is built, and no env var —
`imageLimits` (from the catalog's `image_limits`) only resizes, and
`CLAUDE_CODE_DISABLE_ATTACHMENTS` governs system-prompt sections, not content.
`appendSystemPrompt` is a managed-settings/SDK key that a user settings file
ignores (verified live: a codeword placed there never reached the model).

A text-only endpoint answers an image with a hard failure — vLLM's
`At most 0 image(s) may be provided in one prompt (parameter=image)`, surfaced
as a 500 that kills the turn, and a subagent with it. So the Subscription
modal's **Images** row (user-configured providers only) writes the one thing a
settings profile can enforce: `permissions.deny` rules on the Reads that produce
a non-text block.

- **`Read(//**/*.png)` — the `//` is load-bearing.** A Read rule's path is
  gitignore-style and relative to the project without it, and the images that
  reach a model live outside the project: a screenshot directory, the Desktop, a
  temp path. Verified against a live pane, not the decode — a rooted rule denies
  `/tmp`, an absolute scratch path and an uppercase `.PNG` alike, **in
  `bypassPermissions` mode**, and a Task subagent inherits the denial.
- **A PDF belongs in the list.** Read returns it as a `document` block (and, on
  the per-page fallback, as image blocks) — just as unsendable. An SVG does
  not: git tracks it as text and Claude Code reads it as text, so denying it
  removes a working capability instead of preventing a failure.
- **This is never stamped for the user**, unlike `stampByteWatchdog` and
  `stampContextBudget`. A self-hosted endpoint may serve a model that sees
  images perfectly well, so there is no sweep and no `EnsureAll` — only the
  toggle, default off.
- **The state is the rules themselves,** with no `WISP_DECK_*` marker to drift.
  *Any* owned rule reads as on, so a profile stamped by an older, shorter list
  still shows the toggle set and is filled in on the next write; turning it off
  removes only the owned strings, never a deny rule the user wrote by hand.
- **The launch overlay is what delivers it.** `write_claude_launch_settings`
  copies the whole settings object, so `permissions` travels untouched — and
  `_apply_subscription_switch` regenerates the overlay, so a mid-session switch
  picks the toggle up too.

Known limits, by construction: this stops the model reading an image. It cannot
stop an image the *user* puts in the prompt — a paste, a drag, or an `@`-mentioned
path Claude Code inlines — nor an MCP tool that returns one. `.ipynb` is
deliberately absent from the list for the same reason as SVG in reverse: only a
cell whose *output* is an image would break the turn, and denying every notebook
read costs far more than that. Wisp's own screenshot-inject binding could
consult the toggle; it does not yet.

Guarded by `internal/claudeconfig/imagereads_test.go`,
`internal/tui/subscription_modal_images_test.go`, and
`test/bash/images_blocked_launch_test.go` (the overlay end of the chain, which
goes red if the overlay is ever narrowed to an allowlist of keys).

### A subscription pane declares its provider's context window, or gets stranded

Claude Code budgets auto-compaction from **its own model catalog**, which knows
nothing about the server `ANTHROPIC_BASE_URL` points at. Decoded from the 2.1.x
bundle, the window is picked in this order:

```js
function sae(){ return Q.CLAUDE_CODE_DISABLE_1M_CONTEXT }        // unset → false
function Ov(e){ if(sae()) return false; return /\[1m\]/i.test(e) }
                                             // ^ a regex on the model STRING
if (Ov(model))                       return 1e6;
if (betas.includes(1m) && EW(model)) return 1e6;
if (L2(model))                       return 1e6;
let n = Q.CLAUDE_CODE_MAX_CONTEXT_TOKENS;
if (n > 0 && !Bo(ls(model)).startsWith("claude-")) return n;
return 200000;                                                  // Xbr
```

Two consequences, both shipped:

- An unrecognized subscription model lands on the flat **200000**, which is
  *wrong in both directions*: it strands `glm-4.5-air` (real window 131072) and
  it silently costs a Kimi user a quarter of the 262144 they pay for.
- A session model still carrying Anthropic's `[1m]` marker gets **1000000**
  regardless of provider — `Ov()` never looks past the string.

Overshooting a provider's cap is **unrecoverable**: `/compact` must itself send
the oversized transcript, so it fails with the same 400 as every other turn (and
is *larger* than the turn that already failed — it appends the summarization
prompt). A real session sat at 253,954 tokens under `claude-fable-5[1m]`, was
switched to Kimi (`k3`, cap 262144), and the next tool result killed it for good.

Every subscription profile therefore declares its real window with
`CLAUDE_CODE_MAX_CONTEXT_TOKENS`, taken from the catalog that already knows
each model's limit (`claudeconfig.ContextBudget`) or entered by the user for a
custom endpoint. A window below 1M also coordinates two safeguards:
`CLAUDE_CODE_AUTO_COMPACT_WINDOW` directly caps current Claude versions, and
`CLAUDE_CODE_DISABLE_1M_CONTEXT=1` keeps the inherited marker from winning in
older ones. Rules that fell out of building it:

- **The budget is the MINIMUM across all four `ANTHROPIC_DEFAULT_*_MODEL`
  mappings.** One env var governs the whole session — `/model` and subagents
  move freely between the aliases — so anything larger lets the session grow
  past whichever mapped model has the tightest cap.
- **A provider-native mapping does not remove a global `[1m]` selection.**
  With global `model: "opus[1m]"`, a custom Qwen mapping rendered as
  `Qwen-3.8-Uncensored[1m]`; Claude ignored the max-context key, never
  compacted, then sent 230145 input plus 32000 output tokens to a 262144-token
  endpoint. The auto-compact override is the direct guard, while disabling the
  marker preserves the same result on older Claude versions.
- **The installer copies a default profile only when the file is absent**, so a
  profile created before these keys were declared can never be repaired by
  shipping a new default. `claude-config ensure-budget` sweeps existing ones,
  including a custom profile whose user-entered max window is the only size
  available, and `bin/wisp-deck` runs it.
- **A custom window is always user-owned.** Never infer one for a custom
  profile or replace its declared value just because its model id also appears
  in the catalog; the self-hosted endpoint may enforce a different limit. The
  profile stays unavailable until its endpoint, model, positive context window,
  and API key are all present.
- **A switch re-checks that the conversation still fits.** Retargeting replays
  the WHOLE transcript to the new provider, so `_guard_subscription_context`
  measures the live conversation against the target's declared window and
  refuses while the roomier backend can still run `/compact`. Unknown is not
  too big: an absent transcript, no `jq`, or a target declaring no window all
  allow the switch — blocking on missing data is a worse trade.

Verify a change against a live pane, not the decode: launch `claude --settings
<profile>` on an isolated tmux server and run `/context`, which prints
`Auto-compact window: <N> tokens` outright. Guarded by
`internal/claudeconfig/contextbudget_test.go` (including a check that every
shipped default declares its provider's window) and
`test/bash/subscription_context_guard_test.go`.

### The window holds the reply too, so the reply's room comes out of it

Declaring the window is only half of fitting inside it. Claude Code also picks
`max_tokens` from its own catalog, which has never heard of a subscription
model, and settles on **32000** — and an inference server enforces
`input + max_tokens <= context`, not `input <= context`. On a 32768-token model
that leaves ~768 tokens for the system prompt, the tool schemas and the
conversation, so *every* real turn is rejected before the model reads a word:
`API Error: 400 The request was rejected as invalid. Please check your request
parameters.`

Measured against api.featherless.ai on 2026-09-02 by replaying a request
captured from a live pane through a logging proxy: `max_tokens` 32000 and 30000
both answered 400, 28000 and below answered 200, and adding ~4000 tokens of
input moved that boundary down by the same amount. The identical profile
carrying an 8192 reserve ran the turn to completion.

Claude Code's own accounting says the same thing outright. `/context` on a live
pane carrying the pre-fix profile reported `Autocompact buffer: 33k tokens
(100.7%)` and **no free space at all** — the room it holds back for the reply is
larger than the entire window. The same pane with the reserve declared reported
`Autocompact buffer: 21.2k (64.7%)` and `Free space: 9.3k (28.3%)`.

So a declared window also declares `CLAUDE_CODE_MAX_OUTPUT_TOKENS`. That key is
the whole of the fix.

- **Do not also take the reserve out of `CLAUDE_CODE_AUTO_COMPACT_WINDOW`.**
  Claude Code sizes its own auto-compact buffer from the reserve, so once the
  reserve is right the reply's room is already carved out — the buffer figures
  above are that happening. Shrinking the compact key on top of it was measured
  and changes nothing: 32768, 24576, 20000 and 10000 in a launch overlay all
  produced byte-identical `/context` accounting. It is also actively risky,
  because Claude Code's own parser documents the accepted range as
  `'auto' or 100k-1M tokens`, so a window minus its reserve can land below 100k
  and be rejected — `131072 - 32000 = 99072` does.
- **Both window keys keep naming the endpoint's real limit.**
  `_guard_subscription_context`, the statusline and the modal all read
  `CLAUDE_CODE_MAX_CONTEXT_TOKENS` as the truth about the endpoint.
- **The reserve never rises above 32000.** That is what Claude Code would have
  asked for unprompted; this exists to fit a small window, not to ask a provider
  for more than it was already going to be asked for. A quarter of the window,
  capped there — no cataloged model's real max output falls below that, so
  consulting `Model.Output` would never lower it.
- **A declared reserve is the user's own figure and is kept**, exactly like
  `stampByteWatchdog`'s key: they may know their endpoint's real output cap.
  Only `WriteCustomContextWindow` re-derives one, because there the user is
  changing the size of the thing being divided.
- **`EnsureContextBudget`'s change-check must compare the reserve.** It
  enumerates its keys explicitly, and every profile that has this bug already
  declares all three window keys correctly — a check that omits the fourth
  reports "unchanged" and never writes the file, silently skipping exactly the
  broken profiles the sweep exists to repair.
- **A window of 1M or more is left alone**, the same branch that already ships
  no compaction cap there. The overflow exists at 1M too, just far rarer.

Known limit, by construction: a single huge tool result can still carry one turn
past the threshold in one step and be rejected once. That is inherent to a 32K
window, not something a profile can prevent.

Guarded by `internal/claudeconfig/outputreserve_test.go` (including the sweep's
backfill of a window-current profile, whose change-check is the trap above, and
a check that every shipped sub-1M default declares the reserve its window
implies) and
`TestApplyPendingSubscriptionModel_reserves_output_room_for_a_small_window`.
