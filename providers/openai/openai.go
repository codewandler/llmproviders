// Package openai provides an OpenAI provider for the llmproviders framework.
//
// The provider uses the OpenAI Responses API (/v1/responses) for all models,
// and handles automatic effort/reasoning mapping based on model capabilities.
//
// # Authentication
//
// The provider reads the API key from environment variables:
//   - OPENAI_API_KEY (primary)
//   - OPENAI_KEY (fallback)
//
// Or you can pass an explicit key:
//
//	p, err := openai.NewWithAPIKey("sk-...")
//
// # Basic Usage
//
//	p, err := openai.New()
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	session := p.Session()
//	events, err := session.Request(ctx, conversation.Request{
//	    Inputs: []conversation.Input{{
//	        Role: unified.RoleUser,
//	        Text: "Hello!",
//	    }},
//	})
//
// # Model Selection
//
// The provider supports various model aliases:
//   - "openai", "default" -> gpt-5.4-mini
//   - "flagship" -> gpt-5.4
//   - "mini" -> gpt-5.4-mini
//   - "nano", "fast" -> gpt-5.4-nano
//   - "pro" -> gpt-5.4-pro
//   - "codex" -> gpt-5.3-codex
//   - "o3", "o4" -> respective reasoning models
//
// Or use explicit model IDs:
//
//	session := p.Session(conversation.WithModel("gpt-5.4"))
package openai
