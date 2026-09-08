# featherless — gotchas

### Featherless speaks Anthropic natively, and keeps its own socket warm

Featherless is documented as an OpenAI-compatible provider, which by the rule in
`internal/claudeconfig/CLAUDE.md` would put it behind a translating proxy. It
does not need one:
`POST /v1/messages` is a real Anthropic Messages route, undocumented and verified
live — it answers unauthenticated with Anthropic's error envelope where
`/v1/chat/completions` answers with OpenAI's and an unknown path answers with a
fastify 404, and with a key it streams `content_block_start{tool_use}` →
`input_json_delta` → `stop_reason:"tool_use"`. So the provider is an ordinary
API-key gateway at `https://api.featherless.ai`, and ~15,500 models are reachable
from the Subscription modal's picker.

- **The credential is `ANTHROPIC_AUTH_TOKEN`, never `ANTHROPIC_API_KEY`.**
  Featherless answers `x-api-key` with a 401 and `Authorization: Bearer` with a
  200.
- **The byte watchdog stays ARMED here**, unlike a self-hosted profile.
  Featherless fills the wait before the first token with
  `: keep-alive (awaiting first token)` SSE **comments** — the watchdog counts
  bytes, not events, so a comment is as good as a token. Measured on a cold 14B
  with a 22k-token prompt: comments every ~1.2s across a 12s model load, worst
  byte silence 4.8s against the 20s trigger. Disarming would trade a real
  dead-connection signal for nothing.
- **It sends no `event: ping` at all**, so those comments are the whole
  mechanism, and they appear **only when there is a wait to fill** — a small
  prompt to a hot model emits none. That is why the live guard sends a large
  prompt and asserts the measured byte silence rather than the comments'
  presence: the property is "never silent long enough to trip the watchdog", and
  a keep-alive count is only how Featherless currently achieves it.
- **Only tool-calling models are offered.** 15,571 of the 21,908 report
  `features.tool_use`; the rest produce a pane that cannot read or edit a single
  file, so `Parse` drops them — along with any model declaring no
  `context_length`, because an undeclared window falls back to the flat 200000
  that strands a 32768 model permanently.
- **`available_on_current_plan` is absent on an unauthenticated listing**, and
  absent must read as available, or the picker is empty until a key is typed.
- **`is_gated` is about HuggingFace, not Featherless.**
  `meta-llama/Llama-3.3-70B-Instruct` is gated and serves normally, so it is
  never shown and never blocks a pick.
- **`RemoteCatalog` shares `UserConfigured`'s save path** via
  `Provider.SuppliesOwnModel()`: both must skip `WriteModelMappings`, which
  writes the four aliases from an empty model list and so deletes the picked
  model on every save. What it does **not** share is the watchdog disarm, which
  stays keyed to `UserConfigured` alone.
- **The launch wraps a Featherless pane in a repair proxy** — see
  `internal/rolefix/CLAUDE.md`. That makes `featherless` the second key `get_claude_config_provider`
  must report (it allowlists only the gateways whose marker changes a launch
  decision; `custom` still resolves to the empty string because nothing branches
  on it). Pinned by `TestGetClaudeConfigProviderReportsFeatherless` and
  `TestGetClaudeConfigProvider_never_reports_featherless_as_the_gpt_bridge`.
- **Alias resolution takes the LONGEST matching alias, not the first in slice
  order.** Profiles are named after the picked model, so "Featherless GLM-5.2"
  contains zhipu's `glm` and "Featherless Kimi-K3" contains moonshot's `kimi` —
  and zhipu is `Providers[0]`, so no placement could fix it. Ties keep slice
  order, so "kimi for coding" still beats "kimi".

Guarded by `internal/claudeconfig/featherless_provider_test.go`,
`internal/featherless/*_test.go`, `internal/tui/subscription_model_picker_test.go`,
and `internal/tui/subscription_modal_featherless_test.go`.

### Featherless's Qwen tool parser destroys the tool call it strips

This is what "I can't use Qwen from Featherless" actually is. On the captured
first request (`internal/claudeconfig/CLAUDE.md`), `Qwen/Qwen3-VL-30B-A3B-Instruct` billed **181 completion
tokens** and returned **22 tokens of prose and no tool call** — 159 generated
tokens discarded. `Qwen/Qwen3.5-397B-A17B` billed the same and returned an
**empty** reply. The parser lifts the model's `<tool_call>` markup out of the
text and then emits nothing in its place; `finish_reason: null` marks it on the
chat-completions route, and the Anthropic route normalizes that to
`stop_reason: "end_turn"`, so the pane sees a model that announces work and
stops. Forever — a `/goal` Stop hook loops on it until the block cap fires.

It is Qwen-specific, not endpoint-wide. Replaying that same request across all
26 models wide enough to run Claude Code: **21 call tools normally** (every GLM,
every Kimi, DeepSeek V3.1/V3.2/V4, MiniMax M2/M2.5/M2.7/M3, Step-3.5-Flash,
Laguna-S-2.1), most of them with a prose preamble in the same message. The
failures are `Qwen/Qwen3.5-397B-A17B`, `Qwen/Qwen3-VL-30B-A3B-Instruct`, the two
oldest DeepSeeks, and one MiniMax that 400s. **There is currently no Qwen on
Featherless that can run a pane**: the ones that call tools are all 32768.

- **The discarded bytes cannot be recovered**, so the proxy does the one thing
  left: it refuses to pass the silence on. A reply that declares tools and
  arrives with no content block at all — a shape Anthropic's API cannot produce
  — is given a text block naming what happened. A reply carrying anything at all
  is the model's, and commenting on it would put words in its mouth.
- **`thinking` is not a way out.** It disables the parser for the 27B class (the
  raw markup then arrives as text, recoverable in principle), and does nothing
  for Qwen3-VL, whose call is destroyed either way. Stripping `thinking` stays
  right.
- **Neither is prompting.** A rule telling the model to emit the call with no
  preamble changed nothing across 12 runs on both models.
