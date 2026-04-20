// Package openrouter provides an OpenRouter provider for the llmproviders framework.
//
// OpenRouter acts as a unified gateway to multiple LLM providers. This provider
// automatically routes requests to the appropriate backend based on the model:
//   - Models prefixed with "anthropic/" use the Anthropic Messages API
//   - All other models use the OpenAI Responses API
//
// # Authentication
//
// The provider reads the API key from the OPENROUTER_API_KEY environment variable,
// or you can pass an explicit key:
//
//	p, err := openrouter.NewWithAPIKey("sk-or-...")
//
// # Basic Usage
//
//	p, err := openrouter.New()
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
// OpenRouter supports hundreds of models. The default is "openrouter/auto" which
// automatically selects the best model. Common aliases:
//   - "default", "auto", "openrouter" -> openrouter/auto
//   - "fast" -> openrouter/auto
//
// For specific models, use the full model ID:
//
//	session := p.Session(conversation.WithModel("anthropic/claude-sonnet-4-6"))
//	session := p.Session(conversation.WithModel("openai/gpt-5.4"))
package openrouter
