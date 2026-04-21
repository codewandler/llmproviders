package ollama

import (
	"context"
	"net/http"
	"time"
)

// probeTimeout is the timeout for probing the Ollama API.
const probeTimeout = 500 * time.Millisecond

// ProbeAvailable checks if Ollama is running and accessible.
// It makes a quick HTTP GET request to the /api/tags endpoint.
func ProbeAvailable(ctx context.Context) bool {
	url := ResolveBaseURL() + "/api/tags"

	ctx, cancel := context.WithTimeout(ctx, probeTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return false
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}
