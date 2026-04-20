package auth

import "net/http"

func SetBearerToken(h http.Header, token string) {
	if token == "" {
		return
	}
	h.Set("Authorization", "Bearer "+token)
}
