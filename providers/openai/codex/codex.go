// Package codex provides an LLM provider for OpenAI Codex using ChatGPT OAuth authentication.
//
// Codex is OpenAI's coding-focused offering accessed through the ChatGPT backend API.
// Unlike the standard OpenAI API, Codex requires OAuth authentication obtained via
// the `codex login` CLI command, which stores credentials in ~/.codex/auth.json.
//
// # Authentication
//
// The provider reads OAuth tokens from ~/.codex/auth.json and automatically refreshes
// them when expired. The auth file is created by running `codex login` from the
// official Codex CLI.
//
// # Usage
//
//	// Check if Codex auth is available
//	if !codex.LocalAvailable() {
//	    log.Fatal("Run 'codex login' to authenticate")
//	}
//
//	// Create provider
//	p, err := codex.New()
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Create session with native conversation handling
//	session := p.Session()
//
//	// Make requests
//	events, err := session.Request(ctx, conversation.Request{
//	    Inputs: []conversation.Input{{Role: unified.RoleUser, Text: "Hello!"}},
//	})
//
// # Headers
//
// The provider sets Codex-specific headers for routing and caching optimization:
//
//   - ChatGPT-Account-ID: Workspace/account identifier
//   - x-codex-installation-id: Unique installation identifier
//   - x-codex-window-id: Prompt caching key derivation
//   - x-codex-turn-state: Sticky routing for KV cache optimization
//   - x-codex-beta-features: Comma-separated beta feature flags
//   - session_id: Conversation/thread tracking
//
// # Request Mutations
//
// The Codex backend has specific requirements:
//
//   - store: always set to false
//   - Stripped parameters: prompt_cache_retention, max_tokens, temperature, top_p, top_k, response_format
//   - Effort mapping: "max" is mapped to "xhigh"
//   - Reasoning summary: defaults to "auto" if not specified
//
// # Native Conversation
//
// Like the standard OpenAI provider, Codex supports native conversation continuation
// via previous_response_id, which is more efficient than replaying the full history.
//
// # Models
//
// Use FetchModels to retrieve the current list of available models, or use the
// embedded fallback list via LoadModels. Common models include:
//
//   - codex: Default Codex model
//   - codex-mini: Faster, lighter variant
//   - o3, o4-mini: Reasoning models
//
// # References
//
// Source code references for header constants and protocol details:
//   - https://github.com/openai/codex/blob/main/codex-rs/core/src/client.rs
//   - https://github.com/openai/codex/blob/main/codex-rs/model-provider/src/bearer_auth_provider.rs
//   - https://github.com/openai/codex/blob/main/codex-rs/features/src/lib.rs
package codex
