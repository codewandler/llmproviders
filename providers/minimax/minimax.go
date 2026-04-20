// Package minimax provides an LLM provider implementation for MiniMax.
//
// MiniMax provides an Anthropic Messages API-compatible endpoint at
// https://api.minimax.io/anthropic. This provider uses the same wire
// protocol as the Anthropic provider but with MiniMax-specific authentication.
//
// # Authentication
//
// MiniMax uses API key authentication via the MINIMAX_API_KEY environment
// variable. The API key is sent in both the Authorization: Bearer header
// and the x-api-key header.
//
// # Models
//
// Available models include:
//   - MiniMax-M2.7 (default, aliases: "minimax", "default", "fast")
//   - MiniMax-M2.7-highspeed
//   - MiniMax-M2.5
//   - MiniMax-M2.5-highspeed
//   - MiniMax-M2.1
//   - MiniMax-M2.1-highspeed
//   - MiniMax-M2
//
// # Example
//
//	p, err := minimax.New(minimax.WithAPIKey("your-api-key"))
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	session := p.Session(conversation.WithModel("MiniMax-M2.7"))
//	events, err := session.Request(ctx, conversation.Request{
//	    Inputs: []conversation.Input{{
//	        Role: unified.RoleUser,
//	        Text: "Hello, world!",
//	    }},
//	})
package minimax
