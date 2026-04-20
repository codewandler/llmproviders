package anthropic

import messagesapi "github.com/codewandler/agentapis/api/messages"

// APIError represents a structured error response from the Anthropic API.
// Re-exported from agentapis/api/messages for convenience.
type APIError = messagesapi.APIError

// Error type constants.
// Re-exported from agentapis/api/messages for convenience.
const (
	ErrTypeInvalidRequest    = messagesapi.ErrTypeInvalidRequest
	ErrTypeAuthentication    = messagesapi.ErrTypeAuthentication
	ErrTypePermission        = messagesapi.ErrTypePermission
	ErrTypeNotFound          = messagesapi.ErrTypeNotFound
	ErrTypeRateLimit         = messagesapi.ErrTypeRateLimit
	ErrTypeAPI               = messagesapi.ErrTypeAPI
	ErrTypeOverloaded        = messagesapi.ErrTypeOverloaded
	ErrTypeInsufficientQuota = messagesapi.ErrTypeInsufficientQuota
)

// Sentinel errors for type checking with errors.Is.
// Re-exported from agentapis/api/messages for convenience.
var (
	ErrRateLimit  = messagesapi.ErrRateLimit
	ErrAuth       = messagesapi.ErrAuth
	ErrOverloaded = messagesapi.ErrOverloaded
	ErrInvalidReq = messagesapi.ErrInvalidReq
	ErrNoQuota    = messagesapi.ErrNoQuota
)

// AsAPIError attempts to extract an *APIError from err.
// Re-exported from agentapis/api/messages for convenience.
var AsAPIError = messagesapi.AsAPIError
