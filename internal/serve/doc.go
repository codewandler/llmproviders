// Package serve implements the HTTP server for the llmcli serve command.
// It exposes an OpenAI-compatible Responses API endpoint that proxies
// requests through llmproviders.Service to any detected upstream provider.
package serve
