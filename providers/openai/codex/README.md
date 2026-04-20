# Codex Provider

Go provider for the OpenAI Codex CLI backend (`chatgpt.com/backend-api/codex`).

## Overview

- **Authentication**: ChatGPT OAuth via `~/.codex/auth.json` (created by `codex login`)
- **Transport**: HTTP/SSE streaming (Responses API)
- **Conversation Strategy**: Replay-based (resends full history each turn)
- **Default Model**: `gpt-5.4`

## Quick Start

```go
import "github.com/codewandler/llmproviders/providers/openai/codex"

// Create provider (uses ~/.codex/auth.json)
provider, err := codex.New()
if err != nil {
    log.Fatal(err)
}

// Create a conversation session
session := provider.Session()

// Make requests
events, _ := session.Request(ctx, conversation.Request{
    Inputs: []conversation.Input{{
        Role: unified.RoleUser,
        Text: "Hello!",
    }},
})
```

## Authentication

The provider reads OAuth credentials from `~/.codex/auth.json`, which is created by running `codex login` in the Codex CLI.

```json
{
  "auth_mode": "chatgpt",
  "chatgpt_account_id": "...",
  "chatgpt_access_token": "...",
  "chatgpt_refresh_token": "...",
  "chatgpt_expires_at": "..."
}
```

Tokens are automatically refreshed when expired.

## Headers Reference

All headers are documented with references to the [Codex CLI source code](https://github.com/openai/codex).

### Authentication Headers

| Header | Description | Source |
|--------|-------------|--------|
| `Authorization` | Bearer token for OAuth | [bearer_auth_provider.rs](https://github.com/openai/codex/blob/main/codex-rs/model-provider/src/bearer_auth_provider.rs) |
| `ChatGPT-Account-ID` | Workspace/account identifier | [bearer_auth_provider.rs#L33](https://github.com/openai/codex/blob/main/codex-rs/model-provider/src/bearer_auth_provider.rs#L33) |
| `X-OpenAI-Fedramp` | Set to "true" for FedRAMP accounts | [bearer_auth_provider.rs#L36](https://github.com/openai/codex/blob/main/codex-rs/model-provider/src/bearer_auth_provider.rs#L36) |

### Session & Routing Headers

| Header | Description | Source |
|--------|-------------|--------|
| `session_id` | Conversation/thread identifier | [headers.rs#L8](https://github.com/openai/codex/blob/main/codex-rs/codex-api/src/requests/headers.rs#L8) |
| `x-codex-window-id` | Format: `{session_id}:{window_generation}` for cache key derivation | [client.rs#L133](https://github.com/openai/codex/blob/main/codex-rs/core/src/client.rs#L133) |
| `x-codex-turn-state` | Sticky routing token for KV cache hits within a turn | [client.rs#L130](https://github.com/openai/codex/blob/main/codex-rs/core/src/client.rs#L130) |
| `x-codex-installation-id` | Unique installation identifier for analytics | [client.rs#L129](https://github.com/openai/codex/blob/main/codex-rs/core/src/client.rs#L129) |
| `x-codex-beta-features` | Comma-separated list of enabled beta features | [client.rs#L1600](https://github.com/openai/codex/blob/main/codex-rs/core/src/client.rs#L1600) |

### Observability Headers

| Header | Description | Source |
|--------|-------------|--------|
| `x-codex-turn-metadata` | Per-turn metadata for observability | [client.rs#L131](https://github.com/openai/codex/blob/main/codex-rs/core/src/client.rs#L131) |
| `x-codex-parent-thread-id` | Parent thread ID for spawned sub-agents | [client.rs#L132](https://github.com/openai/codex/blob/main/codex-rs/core/src/client.rs#L132) |

### Sub-agent Headers

| Header | Description | Source |
|--------|-------------|--------|
| `x-openai-subagent` | Sub-agent type: "review", "compact", "memory_consolidation", "collab_spawn" | [client.rs#L135](https://github.com/openai/codex/blob/main/codex-rs/core/src/client.rs#L135) |
| `x-openai-memgen-request` | Set to "true" for memory consolidation requests | [client.rs#L134](https://github.com/openai/codex/blob/main/codex-rs/core/src/client.rs#L134) |
| `x-responsesapi-include-timing-metrics` | Request timing breakdown in response | [client.rs#L136](https://github.com/openai/codex/blob/main/codex-rs/core/src/client.rs#L136) |

## Prompt Caching

The Codex backend supports server-side prompt caching via multiple mechanisms:

### Request Body: `prompt_cache_key`

Set to the conversation session ID. Ensures the same prompt prefix is cached across turns within a conversation.

**Flow:**
```
conversation.Session (auto-generates sessionID)
    ↓
unified.Request.Extras.Responses.PromptCacheKey
    ↓
responses.Request.prompt_cache_key (wire format)
    ↓
Also used for session_id and x-codex-window-id headers
```

### Header: `x-codex-turn-state`

A sticky routing token returned by the server. When replayed on subsequent requests within the same turn, it routes to the same backend instance for better KV cache hits.

### Header: `x-codex-window-id`

Format: `{session_id}:{window_generation}`. Used for prompt cache key derivation on the server. The window generation increments on context compaction (not currently implemented in this provider).

## WebSocket Migration Roadmap

### Current Status: HTTP/SSE

This provider uses HTTP/SSE streaming, which requires **replay-based conversations** (resending full message history each turn).

### Why WebSocket?

The Codex CLI uses WebSocket transport primarily, which enables:

1. **`previous_response_id` support** - Efficient conversation continuation without resending history
2. **Incremental input delta** - Send only new messages, not full history
3. **WebSocket prewarm** - Pre-establish connection for lower latency on first request
4. **Connection reuse** - Maintain persistent connection across turns

### What Changes

| Feature | HTTP/SSE (current) | WebSocket (future) |
|---------|-------------------|-------------------|
| Conversation | Replay (full history) | Native continuation via `previous_response_id` |
| Request size | Grows with history | Constant (delta only) |
| Latency | New connection per request | Persistent connection |
| Token usage | Re-processes history | Cached via `previous_response_id` |

### Implementation Notes

The Codex CLI WebSocket implementation is in:
- [`codex-api/src/common.rs`](https://github.com/openai/codex/blob/main/codex-rs/codex-api/src/common.rs) - `ResponseCreateWsRequest` (has `previous_response_id`)
- [`core/src/client.rs`](https://github.com/openai/codex/blob/main/codex-rs/core/src/client.rs) - WebSocket connection management, prewarm logic

Key difference: `ResponsesApiRequest` (HTTP) does NOT have `previous_response_id`; only `ResponseCreateWsRequest` (WebSocket) does.

## Provider Options

```go
codex.New(
    codex.WithModel("gpt-5.4"),           // Default model
    codex.WithInstallationID("..."),       // Analytics identifier
    codex.WithBetaFeatures("feature1,feature2"), // Enable beta features
    codex.WithHTTPClient(client),          // Custom HTTP client
)
```

## Limitations

- **No `previous_response_id`**: HTTP/SSE transport doesn't support it; use replay strategy
- **No streaming tool execution**: Tools are executed client-side
- **OAuth only**: Requires ChatGPT account, no API key support
