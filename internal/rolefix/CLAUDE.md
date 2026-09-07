# rolefix — gotchas

Gotchas for the Featherless request/response repair proxy. Loaded when Claude opens a file in this package.

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
