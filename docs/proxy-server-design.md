# Design: OpenAI-Compatible Proxy Server and OpenCode Integration

## Status

Implemented after the fact to document the current change set.

## Summary

This change set adds an HTTP proxy server that exposes an OpenAI-compatible Responses API on top of `llmproviders.Service`, plus CLI support for integrating that proxy with OpenCode. It also makes streaming a first-class capability of registered providers and enables Anthropic prompt caching in the Claude Code configuration.

The resulting system allows:

- `llmcli serve` to expose `POST /v1/responses`
- model aliases such as `sonnet`, `fast`, and `powerful` to work through the proxy
- both streaming (`stream: true`) and non-streaming (`stream: false`) response modes
- passthrough of native Responses API events when upstream providers already emit them
- synthetic Responses API event generation for providers that do not
- `llmcli opencode configure` to register the proxy as an OpenCode provider
- Claude Code-style Anthropic usage to automatically benefit from prompt caching

---

## Motivation

Before this change, `llmproviders` already normalized model and provider selection for library consumers, but it did not expose that capability as a network service. That created two gaps:

1. External tools expecting an OpenAI-compatible HTTP API could not use `llmproviders` directly.
2. Local tools such as OpenCode needed manual configuration and could not automatically see alias-based models exposed by the service.

This design addresses both gaps by putting a thin HTTP layer on top of the existing service and by generating compatible OpenCode configuration from the detected provider catalog.

---

## Goals

### Primary goals

- Expose an OpenAI-compatible Responses API endpoint over HTTP.
- Reuse existing service-level model resolution and alias handling.
- Support SSE streaming for interactive clients and SDKs.
- Support non-streaming JSON responses for simpler clients.
- Preserve provider-specific advantages where possible.
- Keep the server implementation internal and relatively small.

### Secondary goals

- Make OpenCode integration a one-command setup.
- Surface model metadata such as context and output limits into generated config.
- Add test coverage around the new transport layer.
- Improve Claude Code compatibility by enabling prompt caching.

### Non-goals

- Implement the full OpenAI platform surface beyond Responses API.
- Add persistent server-side conversation state.
- Normalize every provider-specific feature into the HTTP contract.
- Replace native provider SDKs when direct use is more appropriate.

---

## Scope of the implemented change

### Included

- `serve` CLI command
- `internal/serve` package
- OpenAI-compatible `/v1/responses` endpoint
- health endpoint
- optional CORS support
- request logging and configurable log output
- OpenCode configuration command
- provider interface update to require streaming
- service catalog accessor for config generation
- Claude Code option enabling Anthropic auto cache control

### Also included in the broader current diff

- README and changelog updates
- integration test for Anthropic prompt caching
- CLI tests for command registration and address formatting

---

## High-level architecture

```text
Client / SDK / OpenCode
          |
          v
     llmcli serve
          |
          v
   internal/serve.Handler
          |
          v
  llmproviders.Service
   - resolve alias/model
   - select provider
          |
          v
 registered provider.Stream(...)
          |
          v
 unified stream events
          |
   +------+------+
   |             |
   v             v
Emitter      ResponseCollector
(streaming)  (non-streaming)
   |             |
   v             v
 SSE          JSON payload
```

The architecture deliberately reuses the existing `Service` for provider lookup and model alias resolution, then builds only the transport-specific logic needed to expose HTTP-compatible behavior.

---

## Provider contract change

## Decision

`registry.Provider` now embeds `conversation.Streamer`.

## Rationale

The HTTP proxy needs raw unified stream events to:

- forward native Responses API wire events when available
- synthesize Responses API SSE events when not
- build non-streaming payloads from the same event stream

If streaming were optional, the proxy would need per-provider fallbacks or session-specific behavior. By making streaming part of the provider contract, the server can treat all providers uniformly.

## Impact

- All registered providers must support `Stream(ctx, unified.Request)`.
- Tests and mocks must implement streaming.
- The transport layer can directly consume provider output without conversation session state.

---

## HTTP API design

## Endpoint

- `POST /v1/responses` by default
- configurable with `--path`

## Health check

- `GET /health`
- returns a simple JSON status payload

## Request handling

The handler accepts a Responses-style request body and performs these steps:

1. Validate method is `POST`.
2. Read and decode the request body.
3. Parse request content using a flexible request format that tolerates either:
   - string content
   - array-based content blocks
4. Validate `model` is present.
5. Resolve the requested model through `llmproviders.Service`.
6. Replace the client-facing model alias with the resolved wire model ID.
7. Convert the request into a unified request using `adapt.RequestFromResponses`.
8. Call `provider.Stream(...)`.
9. Return either:
   - SSE stream for `stream: true`
   - collected JSON payload for `stream: false`

## Error handling

Errors are returned in OpenAI-style JSON error format.

Handled cases include:

- method not allowed
- invalid JSON
- missing model
- model not found
- ambiguous model
- upstream request failure
- request cancellation / timeout

---

## Streaming design

## Why SSE

OpenAI Responses streaming is event-based. The proxy uses standard server-sent events so existing OpenAI SDKs and compatible tools can consume the stream without custom client logic.

## Components

### `SSEWriter`

Responsible for:

- setting `text/event-stream` headers
- writing `event:` and `data:` lines
- flushing after each event
- failing early when the response writer does not support flushing

### `Emitter`

Converts unified stream events into Responses API SSE events.

It supports two modes:

#### 1. Raw passthrough

If an upstream provider already emits Responses wire events via `Extras.RawEventName` and `Extras.RawJSON`, the emitter forwards those events as-is.

This is intended for Responses-native providers such as:

- OpenAI
n- OpenRouter
- Codex

Benefits:

- preserves upstream fidelity
- avoids lossy remapping
- minimizes proxy logic for native providers

#### 2. Synthetic event construction

For providers that emit normalized unified events but not Responses wire events, the emitter constructs compatible SSE events.

This is intended for providers such as:

- Anthropic
- Ollama
- other non-Responses backends

Synthetic mappings cover:

- response lifecycle
- text deltas and completion
- reasoning deltas and summaries
- tool call argument deltas and completion
- item and segment lifecycle events
- final usage on completion
- API-style error event emission

## Sequence numbers

Synthetic SSE events receive incrementing sequence numbers so downstream clients can process a stable ordered stream.

## Completion behavior

On completion, the emitter emits:

- content part done
- output item done
- response completed

If usage was seen earlier in the unified stream, it is attached to the final completed response event.

---

## Non-streaming design

For `stream: false`, the proxy still uses the unified event stream from the provider but collects it into a single `responses.ResponsePayload`.

### `ResponseCollector`

The collector accumulates:

- response metadata
- text parts
- tool calls and tool arguments
- usage
- stop reason
- last error

The final response status is derived from the terminal stream state:

- `completed` for normal completion
- `incomplete` for max-token termination
- `failed` when an upstream error is observed

This keeps streaming and non-streaming behavior aligned while avoiding separate provider code paths.

---

## Flexible request parsing

The server introduces a `FlexibleRequest` / `FlexibleInput` layer before conversion to the standard Responses request type.

## Problem

Clients may encode input content differently:

- plain strings
- arrays of structured content blocks

## Decision

Flatten input content into a string for the current implementation.

## Rationale

- keeps the proxy tolerant of common client encodings
- is sufficient for current provider adaptation paths
- avoids rejecting valid client requests just because the wire representation differs

## Limitation

Flattening is intentionally simple and may lose block-level structure. That is acceptable for the current compatibility goal but may need refinement if multimodal or richer structured content becomes a first-class requirement.

---

## OpenCode integration design

## Command

`llmcli opencode configure`

## Purpose

Write or update OpenCode's global `opencode.json` so the `llmcli serve` proxy appears as an OpenAI-compatible provider.

## Behavior

The command:

1. Locates the OpenCode config path.
2. Loads existing JSON if present.
3. Builds a provider entry named `llmproviders`.
4. Uses the configured server address to compute `baseURL`.
5. Enumerates detected aliases from `Service.ProviderAliases()`.
6. Pulls model limits from `Service.Catalog()` when available.
7. Writes the provider entry back while preserving unrelated config keys.
8. Optionally sets the default model.
9. Supports removal via `--remove`.

## Generated provider shape

The provider uses OpenAI-compatible transport settings:

- npm package: `@ai-sdk/openai`
- base URL: `http://<host>:<port>/v1`
- api key: placeholder `unused`

Each detected alias is exposed as an OpenCode-visible model, with a descriptive name and optional context/output limits.

## Design choice: config preservation

The command manipulates only the parts of `opencode.json` it owns and preserves unknown fields. This reduces risk of destructive edits to user-managed configuration.

---

## Service catalog exposure

A new `Service.Catalog()` accessor exposes the underlying `modeldb.Catalog`.

## Why this was needed

The OpenCode configuration generator needs model metadata, especially:

- context window
- max output tokens

Without catalog access, generated config could only expose alias names, not useful model limits.

## Trade-off

This modestly broadens the public surface area of `Service`, but it avoids duplicating catalog plumbing in the CLI layer.

---

## Claude Code prompt caching

## Change

`providers/anthropic.WithClaudeCode()` now enables auto system cache control.

## Motivation

Claude Code-style usage often involves a large and stable system prompt or prefix. Anthropic prompt caching can materially reduce latency and token cost when repeated prefixes are reused across turns.

## Result

Claude Code configuration now implies:

- local OAuth auth
- Claude-compatible headers
- `claude` instance naming
- automatic prompt caching with default retention

## Validation

The new integration test exercises multiple turns and asserts:

- first request writes cacheable input
- later requests read from cache
- cache reads remain significant across turns

---

## CLI design

## `llmcli serve`

Flags:

- `--addr` listen address
- `--path` endpoint path
- `--cors` enable CORS headers
- `--log-level` logging verbosity
- `--log-file` optional file sink

Runtime behavior:

- detects providers on startup
- logs detected instances
- serves the proxy and health endpoint
- shuts down gracefully when context is cancelled

## `llmcli opencode`

Currently includes:

- `configure`

This establishes a command group for future OpenCode-related operations while keeping current functionality focused on provider injection/removal.

---

## Logging and operability

The server uses `slog` and supports optional dual logging to stderr and a file.

Current logging covers:

- provider detection at startup
- high-level request routing details
- debug request/response tracing
- raw request bodies at debug level

The explicit logging design helps when using the proxy as glue between third-party clients and multiple upstream providers.

---

## Testing strategy for this feature set

### Unit tests added around the new server layer

- handler request validation and happy path behavior
- SSE emitter passthrough and synthetic mappings
- response collector behavior for non-streaming mode
- CLI command registration and helper behavior
- OpenCode config generation and removal

### Integration coverage

- Anthropic prompt caching smoke test validates the Claude Code caching change end to end

### Why this mix

The HTTP proxy logic is mostly transport and mapping code, which is best validated through deterministic unit tests. Live provider behavior is only used where the feature depends on real upstream semantics, such as Anthropic caching.

---

## Key trade-offs

## Chosen trade-offs

### Use internal package for server implementation

Pros:

- keeps transport internals out of the public library API
- allows iteration on event mapping without long-term compatibility promises

Cons:

- external reuse of the exact handler components is intentionally limited

### Prefer passthrough when upstream is already Responses-native

Pros:

- higher fidelity
- less maintenance
- fewer semantic mismatches

Cons:

- mixed implementation model between providers

### Synthesize responses for non-native providers

Pros:

- one HTTP contract for all providers
- consistent client integration surface

Cons:

- some provider-specific semantics are necessarily approximated

### Flatten structured input content

Pros:

- practical compatibility with minimal complexity

Cons:

- richer structure is not preserved yet

---

## Limitations and follow-up opportunities

### Current limitations

- compatibility is focused on the Responses API, not broader OpenAI APIs
- request content flattening is text-centric
- non-streaming reconstruction may not preserve every nuanced provider-specific event shape
- synthetic completion events assume a mostly message-first output model
- there is no auth layer on the proxy itself yet

### Plausible next steps

- add explicit auth or local-only protection for the proxy server
- support richer structured content and multimodal request translation
- improve lifecycle synthesis for more complex multi-item outputs
- document the exact supported subset of Responses API fields
- add end-to-end tests against OpenAI SDK clients where practical
- expose server metrics or richer health details

---

## Resulting user flows

### Flow 1: SDK-compatible proxy use

1. User starts `llmcli serve`.
2. Client sends `POST /v1/responses` with model alias like `sonnet`.
3. Service resolves alias to provider and wire model ID.
4. Provider streams unified events.
5. Proxy returns either SSE or a final JSON payload.

### Flow 2: OpenCode setup

1. User runs `llmcli opencode configure`.
2. Command discovers aliases and limits from the service.
3. Command injects `llmproviders` into OpenCode config.
4. OpenCode talks to the local `llmcli serve` proxy.

### Flow 3: Claude Code cached follow-up requests

1. User configures Anthropic via `WithClaudeCode()`.
2. Large system prompt is sent on turn 1.
3. Anthropic caches the reusable prefix.
4. Later turns read cached tokens, reducing repeated prompt cost.

---

## Files introduced or materially involved

### New

- `cli/serve.go`
- `cli/opencode.go`
- `cli/opencode_test.go`
- `internal/serve/doc.go`
- `internal/serve/handler.go`
- `internal/serve/emitter.go`
- `internal/serve/sse.go`
- `internal/serve/request.go`
- `internal/serve/errors.go`
- `internal/serve/collector.go`
- `internal/serve/*_test.go`

### Updated

- `cli/llm.go`
- `cli/cli_test.go`
- `registry/registry.go`
- `service.go`
- `providers/anthropic/options.go`
- `integration/claude_smoke_test.go`
- `README.md`
- `CHANGELOG.md`

---

## Conclusion

This change set turns `llmproviders` from a library-only provider abstraction into a library plus local HTTP compatibility layer. The design intentionally keeps the existing service as the source of truth for resolution and provider selection, while adding a thin server package for protocol adaptation.

The most important design decisions are:

- require streaming support from all registered providers
- proxy a single OpenAI-compatible Responses endpoint
- passthrough native Responses streams when possible
- synthesize compatible SSE and JSON payloads otherwise
- generate OpenCode configuration from live service aliases and catalog metadata
- enable Anthropic prompt caching in Claude Code mode by default

Together, these changes make the project easier to integrate with external tools without abandoning the provider-neutral architecture already present in the codebase.
