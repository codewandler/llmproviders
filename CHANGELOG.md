# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.5.2] - 2026-04-21

### Added

- Testing strategy documentation in `docs/testing-strategy.md`
- Unit tests for previously uncovered packages and paths:
  - `cli/cli_test.go`
  - `providers/dockermr/probe_test.go`
  - `providers/ollama/probe_test.go`
  - `providers/openai/codex/auth_test.go`
  - `providers/openai/codex/provider_test.go`
  - `registry/auto/auto_test.go`
  - `integration/helpers_test.go`

### Changed

- Expanded `service_test.go` coverage for model resolution, parsing, and priority behavior
- Updated `docs/review-20260421-115912.md` to reflect completed testing improvements, current validation status, and remaining gaps

### Fixed

- Default test workflow now aligns with deterministic execution expectations by keeping live integration coverage opt-in via environment gating
- CLI tests were updated to match current provider and catalog APIs

## [0.5.1] - 2026-04-21

### Fixed

- **Claude OAuth authentication** now works correctly
  - Fixed header conflict: `Anthropic-Beta` was being overwritten, losing the `oauth-2025-04-20` flag
  - Added `?beta=true` query parameter for Claude OAuth compatibility
  - Claude provider (priority 15) now properly takes precedence over API key provider (priority 20)

- **Provider alias resolution** correctly tracks instance names
  - `AliasTarget` now includes `InstanceName` field
  - `ProviderFor()` uses the registering instance directly, not just the service
  - `resolve` and `infer --verbose` show correct provider instance (e.g., "claude" not "anthropic")

### Added

- **Claude OAuth smoke tests** (`integration/claude_smoke_test.go`)
  - Basic streaming test
  - Thinking mode test

## [0.5.0] - 2026-04-21

### Added

- **`infer` command** for LLM inference
  - Stream responses to stdout with real-time output
  - `--model` flag with intent/alias/model support
  - `--system` for system prompts
  - `--thinking` mode (auto, on, off)
  - `--effort` level (low, medium, high, max)
  - `--temperature` and `--max-tokens` controls
  - `--verbose` for resolution details and usage stats

- **`claude` provider** (`providers/anthropic/claude_registration.go`)
  - Uses Claude CLI OAuth credentials from `~/.claude/.credentials.json`
  - Detected via `LocalTokenStoreAvailable()`
  - Priority 15 (higher than anthropic API key)

- **`skill` command** for AI assistant integration
  - `llmcli skill show` - print skill content
  - `llmcli skill install` - install to `~/.claude/skills/llmcli/`
  - Embedded SKILL.md with llmcli usage instructions

- **Shell completions** for all commands
  - Model completions (intents, aliases, service models)
  - Service completions for `--service` flag
  - Thinking mode completions (auto, on, off)
  - Effort level completions (low, medium, high, max)

- **`NewLLMCommand()` factory** (`cli/llm.go`)
  - Reusable CLI builder for embedding in other tools
  - Accepts `Use` parameter for flexible command naming
  - Used by `llmcli` and importable by other CLIs

- **URL probing for local providers**
  - Ollama: probes actual URL with 500ms timeout
  - Docker Model Runner: probes actual URL with 500ms timeout
  - Environment variables: `OLLAMA_URL`, `DOCKER_MODEL_RUNNER_URL`

### Changed

- **Provider priorities reordered**
  - codex: 80 → 10 (highest priority)
  - claude: 15 (new)
  - anthropic: 20 (unchanged)
  - openai: 40 → 30
  - openrouter: 50 (unchanged)
  - minimax: 60 (unchanged)
  - ollama: 70 (unchanged)
  - dockermr: 90 (unchanged)

- **MiniMax no longer registers `fast` intent**
  - Highspeed tier is premium, not cheap
  - Removes misleading cost association

- **Local provider detection improved**
  - Previously always returned true
  - Now probes actual URLs to verify availability

## [0.4.0] - 2026-04-21

### Added

- **CLI infrastructure** (`cli/`, `cmd/llmcli`)
  - Importable CLI commands for embedding in other tools
  - `llmcli` standalone binary
  - Commands: `intents`, `providers`, `aliases`, `models`, `resolve`, `catalog`
  - IO abstraction for testability and composition
  - JSON output support for programmatic use

- **Registry and Service layer** (`registry/`, `service.go`)
  - `Registry` for provider registration with priority ordering
  - `Service` as the main entry point for model resolution
  - Auto-detection registry (`registry/auto`) that discovers available providers
  - Provider priority: anthropic(20) < openai(40) < openrouter(50) < minimax(60) < ollama(70) < codex(80) < dockermr(90)

- **Intent aliases** (`intent.go`)
  - Semantic model shortcuts: `fast`, `default`, `powerful`
  - Resolution based on highest-priority detected provider
  - Example: `fast` → `claude-haiku-4-5-20251001` (anthropic)

- **Provider aliases**
  - Short names registered per provider: `sonnet`, `opus`, `haiku`, `mini`, `nano`, etc.
  - First detected provider wins for duplicate aliases
  - Viewable via `llmcli aliases`

- **Model resolution** with documented priority order:
  1. Intent aliases (fast, default, powerful)
  2. Provider aliases (sonnet, opus, mini, etc.)
  3. Catalog wire model lookup
  4. Parse as [instance/]service/model
  5. Bare model search across all services

- **Provider type alias** at root package (`provider.go`)
  - `llmproviders.Provider` as type alias to `registry.Provider`
  - Cleaner imports for consumers

- **Provider registration files**
  - Each provider now has `registration.go` with `Registration()` function
  - Declares: InstanceName, ServiceID, Order, Aliases, IntentAliases, Detect, Build

### Fixed

- Model resolution for wire model IDs like `anthropic/claude-3-5-haiku`
  - Now correctly routes to OpenRouter (which has this in its catalog)
  - Previously failed by parsing as service=anthropic, model=claude-3-5-haiku

- Default models for local providers (ollama, dockermr) use larger models for reliable chat

### Changed

- Providers now implement `registry.Provider` interface with `Name()` method
- Integration tests updated to use new Service layer

## [0.3.0] - 2026-04-20

### Added

- **Ollama provider** (`providers/ollama`)
  - Native Ollama API support via agentapis `api/ollama`
  - Local LLM server integration (default: `localhost:11434`)
  - Default model: `qwen2.5:0.5b` (small, fast, suitable for testing)
  - `FetchModels()` to list locally installed models
  - `Download()` to pull models from Ollama registry
  - Curated model list with common Ollama models

- **Docker Model Runner provider** (`providers/dockermr`)
  - OpenAI Chat Completions API support via llama.cpp engine
  - Local LLM via Docker Model Runner (default: `localhost:12434`)
  - Default model: `ai/smollm2` (360M params, low-memory friendly)
  - Engine selection via `WithEngine()` option
  - `FetchModels()` to list available models
  - Curated model list from Docker Hub's `ai/` namespace

- **Local integration tests**
  - Tests guarded by `TEST_INTEGRATION_LOCAL` environment variable
  - Basic streaming, fetch models, and multi-turn conversation tests
  - Run with: `TEST_INTEGRATION_LOCAL=1 go test ./integration/... -run "Ollama|DockerMR"`

## [0.2.0] - 2026-04-20

### Added

- **Codex provider** (`providers/openai/codex`)
  - OpenAI Codex CLI backend support (`chatgpt.com/backend-api/codex`)
  - ChatGPT OAuth authentication via `~/.codex/auth.json`
  - Automatic token refresh
  - Responses API streaming over HTTP/SSE
  - Replay-based conversations (full history per turn)
  - Model resolution with alias support (`codex` → `gpt-5.4`)
  - Tool calling support
  - Comprehensive headers documentation with Codex CLI source references
  - FetchModels for live model discovery

- **Prompt caching support**
  - Session ID auto-generated by conversation layer
  - Flows to `prompt_cache_key` in request body
  - Used for `session_id` and `x-codex-window-id` headers
  - Enables server-side prompt caching across turns

### Changed

- Updated to agentapis v0.9.0
  - Adds `PromptCacheKey` to Responses API request
  - Adds automatic session ID generation in conversation layer

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

[0.5.1]: https://github.com/codewandler/llmproviders/releases/tag/v0.5.1
[0.5.0]: https://github.com/codewandler/llmproviders/releases/tag/v0.5.0
[0.4.0]: https://github.com/codewandler/llmproviders/releases/tag/v0.4.0
[0.3.0]: https://github.com/codewandler/llmproviders/releases/tag/v0.3.0
[0.2.0]: https://github.com/codewandler/llmproviders/releases/tag/v0.2.0
[0.1.0]: https://github.com/codewandler/llmproviders/releases/tag/v0.1.0
