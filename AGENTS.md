# Agent Instructions

## Overview

This repository contains LLM provider implementations that wrap [agentapis](https://github.com/codewandler/agentapis) protocols into ready-to-use providers.

## Architecture Principles

1. **Composition over inheritance** - Providers compose protocol clients, not extend them
2. **Explicit authentication** - No implicit env var fallbacks; users must explicitly choose auth method
3. **No magic values** - Use exported constants (`EnvAPIKey`) and helpers (`EnvAPIKeyValue()`)
4. **One package per provider** - Each provider is self-contained in `providers/<name>/`

## Adding a New Provider

1. Create `providers/<name>/` directory
2. Implement required files:
   - `config.go` - Constants, default URLs, env var names
   - `auth.go` - `EnvAPIKeyValue()` helper and auth option
   - `provider.go` - Main provider struct implementing `conversation.Provider`
   - `options.go` - Functional options for configuration
3. Add integration test in `integration/<name>_smoke_test.go`
4. Update README.md with provider details

## Testing

```bash
# Unit tests
go test ./...

# Integration tests (requires API keys)
ANTHROPIC_API_KEY=... go test -tags=integration ./integration/...
```

## Plans

Working plans and design documents are in `.agents/plans/`.
