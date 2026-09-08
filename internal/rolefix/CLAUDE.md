# rolefix — gotchas

Gotchas for the Featherless request/response repair proxy. Loaded when Claude opens a file in this package.

### A Featherless pane runs behind a request repair proxy

Featherless serves the Anthropic Messages API, but it validates the **published
schema**, where a message role is only `user` or `assistant`. Claude Code puts
its capability listings — the agent-type roster and the skills roster — into
`messages[]` as entries with `role: "system"`. Anthropic's own API accepts them;
Featherless answers the whole request with
`400 messages.1.role: Invalid enum value ... received 'system'`, which kills the
turn before the model ever sees it.

Measured, not decoded: a request captured from a live pane was replayed as sent
(**400**) and with that one role rewritten to `"user"` (**200**, normal
completion).

The proxy repairs a second thing, and it is the reason a Featherless pane could
look alive and still do nothing. **A request that declares `thinking` turns
Featherless's tool-call parser off.** Extended thinking is on by default, so
Claude Code puts `thinking: {"type":"adaptive","display":"omitted"}` on every
request; the model still emits a tool call, but the endpoint stops converting
it, so the raw Qwen `<tool_call><function=…><parameter=…>` XML arrives as
assistant **text**. The pane renders the markup, no tool runs, and the turn ends
`end_turn` — so nothing errors and the session simply spins.

Measured 2026-09-02, same prompt and model, both arms, on
`TurboVadim/Qwen3.8-27B-OBLITERATED` and
`huihui-ai/Huihui-Qwen3.8-27B-abliterated`: without the field `stop_reason` is
`tool_use` and a `tool_use` block arrives; with it `stop_reason` is `end_turn`
and the XML sits in a text block. Confirmed end to end through the launch chain
— the pre-fix binary printed the bare XML and called nothing, the fixed one ran
`Read` and answered.

- **Dropping `thinking` costs no reasoning.** Featherless returns a `thinking`
  block whether or not the request asks for one, so the field buys the turn
  nothing and breaks its tool calling. It is the *presence* of the key that does
  it — `{"type":"enabled","budget_tokens":N}` fails the same way, and
  `output_config.effort` is innocent. So is the header: both arms sent the same
  `Anthropic-Beta: …,interleaved-thinking-2025-05-14,…`, and the repaired one
  called tools with it still there. The body field is the whole lever.
- **The strip is safe for a model the bug never touched.** Not every class
  mis-parses: `zai-org/GLM-5.3-Flash` and `GLM-4.7-Flash` answer `tool_use` with
  a `thinking` block in BOTH arms, identically. So stripping unconditionally for
  every Featherless pane repairs the broken classes and takes nothing from the
  working ones — which is why this needs no per-model probe.
- **The repair is one pass, and `changed` gates the rewrite.** `Rewrite` returns
  the body untouched when nothing moved, so a deletion recorded after that guard
  would be silently thrown away.
- **There is no settings-level escape.** `--disallowedTools Task` removes the
  agent roster and the skills roster takes its place; both are Claude Code's own
  emissions. Disabling enough tools to silence them costs more than the proxy.
- **The settings file beats the process environment.** Verified live: launching
  with `ANTHROPIC_BASE_URL` exported and a profile declaring its own, the profile
  won. So the proxy cannot be delivered by env override the way the GPT bridge
  does it — the session's **settings overlay** is what gets pointed at the proxy.
- **The overlay is the session's own copy.** `write_claude_launch_settings` never
  modifies the stored profile, so `PointSettingsAt` rewrites the overlay in place
  and every other key in it — the API key the proxy forwards but never holds, the
  picked model, the declared window, the image deny rules — travels untouched.
  The stored profile keeps naming the real endpoint, so `ConfigReady`, the
  budget sweep and the modal all keep working on the truth.
- **`FlushInterval: -1` is load-bearing.** Buffering the response would swallow
  the `: keep-alive` comments Featherless sends while awaiting its first token,
  which is the whole reason the byte watchdog stays armed for this provider.
- **Nothing here may cost a session.** An overlay that cannot be read, declares
  no endpoint, or already points at loopback runs the child exactly as it would
  have run anyway.
- **A model that refuses to call tools is the model, not the proxy.** Verified
  end to end: `zai-org/GLM-5.3-Flash` through this proxy called Read and quoted
  the file back, while `moonshotai/Kimi-K3` on the same setup insisted "tool use
  has been temporarily disabled for this turn" with all 29 tools present in the
  request.

Guarded by `internal/rolefix/*_test.go`,
`cmd/wisp-deck-tui/claude_rolefix_test.go`, and
`test/bash/claude_rolefix_launch_test.go`.

### That proxy also repairs the response, because a JSON schema is only advice here

`/goal`, a prompt hook, memory selection and auto-mode setup all ask the model
for a JSON object and then run a plain `JSON.parse` over the reply — Claude Code
strips a markdown fence (`$x`) and nothing else. What makes that safe on
Anthropic's API is `output_config.format`: the server constrains the decode, so
the reply cannot be anything but the object.

Featherless accepts that field and ignores it. Measured 2026-09-02 on
`TurboVadim/Qwen3.8-27B-OBLITERATED`: under a schema requiring
`{capital_city, population_millions}`, a system prompt asking for one sentence of
prose answered `Paris`. Every other lever an OpenAI-compatible server usually
offers — `response_format`, `guided_json`, `extra_body.guided_json` — is likewise
accepted and unhonoured, and there is no client-side off switch either:
`s3o`/`hEt` add `format` for any model whose name is not an old Claude, so a
Featherless pane sends the schema and the `structured-outputs-2025-12-15` beta on
every one of these side queries (confirmed on the wire).

So the contract degrades to a suggestion in the prompt, and the model breaks it
whenever it feels like explaining itself first. A `/goal` Stop hook came back as
a paragraph followed by a perfectly good verdict object; Claude Code reported
**`Stop hook error: JSON validation failed`** and the goal never held. The proxy
therefore delivers what the endpoint dropped: a text block that does not parse is
replaced by the JSON object inside it.

- **Only a request that declared a schema is touched.** `output_config` carries
  `effort` on every ordinary turn, so the trigger is `format`, never the object
  holding it. Extracting an object out of a normal reply would replace the answer
  with a fragment of itself.
- **A conforming block is replayed verbatim, delta for delta.** Repairing what is
  not broken is how a proxy invents bugs, and a byte-identical passthrough is
  what the test asserts.
- **Nothing is invented.** Text holding no complete object is forwarded as it
  arrived: the client's own error beats a fabricated verdict.
- **The outermost object wins, and the schema breaks the tie.** A member object
  decodes at its own opening brace too, so the scan skips past each match; among
  the outermost ones the last carrying every `required` key is the verdict, which
  is what stops a trailing `the schema is {…}` aside from being read as one.
- **Holding a block back is silence, so the proxy fills it.** It writes
  `: keep-alive` comments while buffering — bytes are all the byte-stall watchdog
  counts, the same mechanism Featherless uses before a first token.
- **A body past the repair budget is delivered, not truncated.** An
  `io.LimitReader` alone would silently cut it; the read goes one byte past the
  budget and hands the rest on unread.
- **The request asks for `identity`.** A compressed body is bytes the repair
  cannot read, and `ReverseProxy` does not decode one.

Verified live before and after on the real endpoint: the same request direct to
Featherless returned prose then JSON (a hook failure), and through the proxy
returned the bare verdict.

Guarded by `internal/rolefix/structured_test.go`.

### Featherless reports no usage at all, so nothing ever compacts

`/v1/messages` answers **every** turn with `usage: {input_tokens: 0,
output_tokens: 0}` — streamed and not, on every model tried. The count exists:
the same conversation through `/v1/chat/completions` reports 19,838 prompt
tokens. Only the Anthropic adapter drops it.

Claude Code sizes auto-compaction from that figure, so a permanent zero is not a
cosmetic statusline bug. The transcript never compacts, `/context` reads empty,
the cost line reads 0.0%, and the conversation grows until the endpoint starts
rejecting every turn — at which point there is no way back, because `/compact`
must itself send the oversized transcript and is larger than the turn that
already failed. On a narrow model that arrives within a couple of turns.

So `internal/rolefix/usage.go` supplies what the endpoint dropped:
`message_start` gets the request's own estimate, `message_delta` gets the reply's
streamed bytes, and a body that was not streamed gets both.

- **A figure the endpoint reported is never replaced.** A gateway that counts
  knows better than an estimate.
- **The estimate counts the JSON whole, not the strings inside it.** A tool
  schema reaches the model as the schema — braces, keys and all — and counting
  only its string values read 17,023 for a request charged 19,838.
- **It leans high, deliberately.** `bytesPerTokenDenominator` is 3.8 against a
  measured 3.98, because reading low ends a session and reading high only
  compacts a little early — the same direction `cellWidth` rounds.
- **An image is priced flat** (`imageTokens`, the GPT bridge's own figure). Its
  base64 is orders of magnitude larger than its token cost, so counting those
  bytes would report one screenshot as larger than the window.

### The endpoint passes through a tool name the model mis-spelled

Anthropic's API validates a tool call's name against the tools the request
declared, so a client never receives one it did not supply. Featherless passes
the model's own spelling through: `TurboVadim/Qwen3.8-27B-OBLITERATED` answered
a tool declared as `Read` with `read`, on both routes, and not every time —
which is what makes it read as a flaky model rather than a missing check. Claude
Code answers that with `No such tool available`, spending the turn on an error
for a call that was right in every way that matters.

`internal/rolefix/toolnames.go` restores the declared spelling. A name matching
no declared tool is left exactly as it arrived — that is the model inventing a
tool (`read_file` was observed), and the client's own error beats running
something nobody asked for. Two tools whose names differ only by case resolve to
neither.

### `NewHandler`'s own dial failures must not print into the pane

`httputil.ReverseProxy`'s `ErrorLog` defaults to nil, which falls back to
package `log` writing straight to stderr — and the wrapped process's stderr is
the terminal Claude Code paints on (see `wrapper-stderr-is-the-ai-pane` in
project memory). A Featherless dial failure — the endpoint down, DNS broken —
used to print a raw `http: proxy error: dial tcp ...` into the agent's screen.
`NewHandler` now sets `ErrorLog` to a `log.New(io.Discard, "", 0)` logger,
exactly like `internal/allin/proxy.go`'s own `discardLog` does for its plain
reverse-proxy path. Fixing it here rather than in each caller covers both: a
dedicated `claude-rolefix`-wrapped pane, and `internal/allin`'s router
delegating a `NeedsRepair` (Featherless) target to this same handler. Guarded
by `TestNewHandler_does_not_log_a_dial_failure_to_stderr`.
