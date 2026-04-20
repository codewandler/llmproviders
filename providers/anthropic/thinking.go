// Package messagesutil contains shared helpers for providers that use the
// Anthropic Messages API (anthropic, claude).
package anthropic

import messagesapi "github.com/codewandler/agentapis/api/messages"

// CoerceThinkingTemperature enforces the Anthropic constraint that when extended
// thinking is enabled, temperature must be exactly 0 or 1. If it is neither,
// it is forced to 1.
//
// Shared between the anthropic and claude providers.
func CoerceThinkingTemperature(req *messagesapi.Request) {
	if req == nil || req.Thinking == nil || req.Thinking.Type == "disabled" {
		return
	}
	if req.Temperature != 0 && req.Temperature != 1 {
		req.Temperature = 1
	}
}
