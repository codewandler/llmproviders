# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.0] - 2026-04-20

### Added

- **Anthropic provider** (`providers/anthropic`)
  - Messages API support with streaming
  - API key authentication via `ANTHROPIC_API_KEY`
  - OAuth support via `WithLocalOAuth()` for Claude.ai credentials
  - Extended thinking with signature verification
  - Auto system cache control option

- **OpenAI provider** (`providers/openai`)
  - Responses API support with streaming
  - API key authentication via `OPENAI_API_KEY` (with `OPENAI_KEY` fallback)
  - Reasoning effort mapping to model categories
  - Tool calling support

- **MiniMax provider** (`providers/minimax`)
  - Anthropic Messages API-compatible endpoint
  - Dual header authentication (both `Authorization` and `x-api-key`)
  - Extended thinking support

- **OpenRouter provider** (`providers/openrouter`)
  - Automatic backend routing based on model prefix
  - Anthropic models (`anthropic/` prefix) use Messages API
  - All other models use Responses API
  - Single unified interface for multiple model providers

- **Integration test suite** for all providers
- **README** with quick start and usage examples

### Architecture

- Composition-based design with pluggable authentication
- Explicit auth configuration (no magic fallbacks)
- Environment variable constants exported by each provider
- Built on [agentapis](https://github.com/codewandler/agentapis) for protocol adapters

[0.1.0]: https://github.com/codewandler/llmproviders/releases/tag/v0.1.0
