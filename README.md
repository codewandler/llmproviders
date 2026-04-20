# llmproviders

A unified Go library for interacting with multiple LLM providers through a consistent interface.

## Features

- **Unified Interface**: Single `conversation.Session` API across all providers
- **Multiple Providers**: Anthropic, OpenAI, MiniMax, OpenRouter (with more coming)
- **Streaming Support**: Full streaming with events for text, tool calls, thinking, and usage
- **Tool Calling**: Consistent tool definition and execution across providers
- **Multi-turn Conversations**: Automatic history management and replay
- **Thinking/Reasoning**: Support for extended thinking modes where available

## Installation

```bash
go get github.com/codewandler/llmproviders
```

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/codewandler/agentapis/api/unified"
    "github.com/codewandler/agentapis/conversation"
    "github.com/codewandler/llmproviders/providers/anthropic"
)

func main() {
    // Create provider (uses ANTHROPIC_API_KEY env var)
    p, err := anthropic.New()
    if err != nil {
        log.Fatal(err)
    }

    // Create a session
    session := p.Session()

    // Send a request
    ctx := context.Background()
    events, err := session.Request(ctx, conversation.Request{
        Inputs: []conversation.Input{{
            Role: unified.RoleUser,
            Text: "Hello!",
        }},
    })
    if err != nil {
        log.Fatal(err)
    }

    // Process events
    for ev := range events {
        switch e := ev.(type) {
        case conversation.TextDeltaEvent:
            fmt.Print(e.Text)
        case conversation.ErrorEvent:
            log.Fatal(e.Err)
        }
    }
}
```

## Supported Providers

| Provider | Package | Env Variable | API |
|----------|---------|--------------|-----|
| Anthropic | `providers/anthropic` | `ANTHROPIC_API_KEY` | Messages API |
| OpenAI | `providers/openai` | `OPENAI_API_KEY` | Responses API |
| MiniMax | `providers/minimax` | `MINIMAX_API_KEY` | Messages API (Anthropic-compatible) |
| OpenRouter | `providers/openrouter` | `OPENROUTER_API_KEY` | Auto-routes based on model |

## Provider-Specific Features

### Anthropic

```go
// With OAuth (Claude.ai credentials)
p, _ := anthropic.New(anthropic.WithLocalOAuth())

// With explicit API key
p, _ := anthropic.NewWithAPIKey("sk-ant-...")

// With auto system cache control
p, _ := anthropic.New(anthropic.WithAutoSystemCacheControl())
```

### OpenAI

```go
// Supports both OPENAI_API_KEY and OPENAI_KEY
p, _ := openai.New()

// With specific model
session := p.Session(conversation.WithModel("gpt-5.4"))
```

### OpenRouter

```go
// Auto-routes to correct backend based on model
p, _ := openrouter.New()

// Anthropic models use Messages API
session := p.Session(conversation.WithModel("anthropic/claude-sonnet-4-6"))

// Other models use Responses API
session := p.Session(conversation.WithModel("openai/gpt-5.4"))
```

## Tool Calling

```go
tool := unified.Tool{
    Name:        "get_weather",
    Description: "Get weather for a location",
    Parameters: map[string]any{
        "type": "object",
        "properties": map[string]any{
            "location": map[string]any{"type": "string"},
        },
        "required": []string{"location"},
    },
}

session := p.Session(conversation.WithTools([]unified.Tool{tool}))

// Handle tool calls in event loop
for ev := range events {
    switch e := ev.(type) {
    case conversation.ToolCallEvent:
        // Execute tool and continue
        result := executeMyTool(e.ToolCall)
        events, _ = session.Request(ctx, conversation.Request{
            Inputs: []conversation.Input{{
                Role: unified.RoleTool,
                ToolResult: &conversation.ToolResult{
                    ToolCallID: e.ToolCall.ID,
                    Output:     result,
                },
            }},
        })
    }
}
```

## Architecture

```
llmproviders/
├── providers/
│   ├── anthropic/    # Anthropic Claude models
│   ├── openai/       # OpenAI GPT models
│   ├── minimax/      # MiniMax models
│   └── openrouter/   # OpenRouter gateway
└── integration/      # Integration tests
```

This library builds on [agentapis](https://github.com/codewandler/agentapis) which provides:
- Unified request/response types
- Protocol adapters (Messages API, Responses API)
- Conversation session management
- Event streaming infrastructure

## License

MIT
