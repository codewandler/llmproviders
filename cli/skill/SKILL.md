---
name: llmcli
description: LLM provider CLI for model resolution, inference, and provider management
user-invocable: true
---

# llmcli - LLM Provider CLI

Use `llmcli` for LLM inference, model discovery, and provider management. Run commands via Bash tool.

## Quick Start

```bash
llmcli infer "Hello, how are you?"          # Quick inference with default model
llmcli infer -m powerful "Write a poem"     # Use most capable model
llmcli resolve fast                         # See what 'fast' resolves to
llmcli providers                            # List detected providers
```

## Inference

```bash
llmcli infer "message"                      # Send message, stream response
llmcli infer -m sonnet "message"            # Use specific model/alias
llmcli infer -m anthropic/claude-sonnet-4-6 "message"  # Use full model path
llmcli infer -s "You are helpful" "Hi"      # With system prompt
llmcli infer --thinking on "Explain X"      # Enable thinking/reasoning
llmcli infer --effort high "Complex task"   # High effort mode
llmcli infer -v "message"                   # Verbose (show resolution, usage)
llmcli infer --max-tokens 512 "Short answer"  # Limit output length
llmcli infer --temperature 0.7 "Creative"   # Higher randomness
```

### Inference Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--model` | `-m` | `default` | Model alias or full path |
| `--system` | `-s` | - | System prompt |
| `--max-tokens` | - | 8000 | Maximum output tokens |
| `--temperature` | - | 0 | Sampling temperature (0 = provider default) |
| `--thinking` | - | - | Thinking mode: auto, on, off |
| `--effort` | - | - | Effort level: low, medium, high, max |
| `--verbose` | `-v` | false | Show model resolution and usage |

## Model Discovery

```bash
llmcli models                               # List all available models
llmcli models -s anthropic                  # Filter by service
llmcli models --show-intents                # Show intent markers [fast], [default], [powerful]
llmcli models -q sonnet                     # Search models by name
```

## Intent Aliases

Semantic shortcuts for model selection based on use case:

| Intent | Description | Example Resolution |
|--------|-------------|-------------------|
| `fast` | Fastest model | claude-haiku, gpt-mini |
| `default` | Balanced model | claude-sonnet, gpt-4o |
| `powerful` | Most capable | claude-opus, o3 |

```bash
llmcli intents                              # Show current intent resolution
llmcli intents --all                        # Show all provider intent mappings
llmcli intents --json                       # JSON output
```

## Provider Aliases

Short names registered by each provider:

```bash
llmcli aliases                              # Show merged aliases
llmcli aliases --by-provider                # Group by provider
llmcli aliases --json                       # JSON output
```

Common aliases: `sonnet`, `opus`, `haiku`, `mini`, `nano`, `gpt`, `o3`, `codex`

## Resolution

Explain how a model reference resolves:

```bash
llmcli resolve fast                         # Explain intent alias
llmcli resolve sonnet                       # Explain provider alias
llmcli resolve anthropic/claude-3-5-haiku   # Explain wire model routing
llmcli resolve --json fast                  # JSON output
```

Resolution order:
1. Intent aliases (fast, default, powerful)
2. Provider aliases (sonnet, opus, mini, etc.)
3. Catalog wire model lookup
4. Parse as service/model
5. Bare model search

## Providers

```bash
llmcli providers                            # List detected providers
llmcli providers --json                     # JSON output
```

## Provider Priority

Resolution prefers providers in this order (lower = higher priority):

| Priority | Provider | Auth Method |
|----------|----------|-------------|
| 10 | codex | ChatGPT OAuth (~/.codex/auth.json) |
| 15 | claude | Claude.ai OAuth (~/.claude/.credentials.json) |
| 20 | anthropic | API key (ANTHROPIC_API_KEY) |
| 30 | openai | API key (OPENAI_API_KEY) |
| 50 | openrouter | API key (OPENROUTER_API_KEY) |
| 60 | minimax | API key (MINIMAX_API_KEY) |
| 70 | ollama | Local (OLLAMA_URL or localhost:11434) |
| 90 | dockermr | Local (DOCKER_MODEL_RUNNER_URL or localhost:12434) |

## Skill Management

```bash
llmcli skill                                # Show this skill content
llmcli skill show                           # Same as above
llmcli skill install                        # Install to ~/.claude/skills/llmcli/
llmcli skill install --path /custom/path    # Install to custom location
```
