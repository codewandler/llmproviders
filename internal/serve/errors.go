package serve

import (
	"encoding/json"
	"net/http"
)

// apiError matches the OpenAI error response format.
type apiError struct {
	Error apiErrorBody `json:"error"`
}

type apiErrorBody struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    string `json:"code"`
}

// WriteJSONError writes an OpenAI-format JSON error response.
func WriteJSONError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(apiError{
		Error: apiErrorBody{
			Message: message,
			Type:    errorTypeForStatus(status),
			Code:    code,
		},
	})
}

func errorTypeForStatus(status int) string {
	switch {
	case status == http.StatusNotFound:
		return "not_found_error"
	case status >= 400 && status < 500:
		return "invalid_request_error"
	default:
		return "api_error"
	}
}
