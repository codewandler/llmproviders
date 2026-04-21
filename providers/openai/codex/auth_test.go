package codex

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func testJWT(exp time.Time) string {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none","typ":"JWT"}`))
	payload := base64.RawURLEncoding.EncodeToString([]byte(fmt.Sprintf(`{"exp":%d}`, exp.Unix())))
	return header + "." + payload + ".sig"
}

func TestLoadAuthFrom(t *testing.T) {
	t.Run("loads valid auth and parses jwt expiry", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "auth.json")
		token := testJWT(time.Now().Add(time.Hour))
		content := fmt.Sprintf(`{"auth_mode":"%s","tokens":{"access_token":%q,"refresh_token":"refresh","account_id":"acct-1"}}`, ChatGPTAuthMode, token)
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatalf("WriteFile() error = %v", err)
		}

		a, err := loadAuthFrom(path)
		if err != nil {
			t.Fatalf("loadAuthFrom() error = %v", err)
		}
		if a.auth.Tokens.AccountID != "acct-1" {
			t.Fatalf("account id = %q, want acct-1", a.auth.Tokens.AccountID)
		}
		if a.expiry.IsZero() {
			t.Fatal("expiry = zero, want parsed JWT expiry")
		}
	})

	t.Run("rejects unsupported auth mode", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "auth.json")
		if err := os.WriteFile(path, []byte(`{"auth_mode":"api_key","tokens":{"access_token":"x"}}`), 0o600); err != nil {
			t.Fatalf("WriteFile() error = %v", err)
		}

		_, err := loadAuthFrom(path)
		if err == nil || !strings.Contains(err.Error(), "unsupported auth mode") {
			t.Fatalf("loadAuthFrom() error = %v, want unsupported auth mode", err)
		}
	})

	t.Run("rejects missing tokens", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "auth.json")
		if err := os.WriteFile(path, []byte(`{"auth_mode":"chatgpt","tokens":{}}`), 0o600); err != nil {
			t.Fatalf("WriteFile() error = %v", err)
		}

		_, err := loadAuthFrom(path)
		if err == nil || !strings.Contains(err.Error(), "no tokens") {
			t.Fatalf("loadAuthFrom() error = %v, want no tokens", err)
		}
	})
}

func TestLocalAvailable(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "auth.json")
	if err := os.WriteFile(path, []byte(`{"auth_mode":"chatgpt","tokens":{"refresh_token":"refresh"}}`), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	t.Setenv(EnvCodexAuthPath, path)
	if !LocalAvailable() {
		t.Fatal("LocalAvailable() = false, want true")
	}
}

func TestAuthToken(t *testing.T) {
	t.Run("returns unexpired access token without refresh", func(t *testing.T) {
		want := testJWT(time.Now().Add(10 * time.Minute))
		a := &Auth{
			auth:       authFile{Tokens: tokenStore{AccessToken: want}},
			expiry:     time.Now().Add(10 * time.Minute),
			httpClient: http.DefaultClient,
		}

		got, err := a.Token(context.Background())
		if err != nil {
			t.Fatalf("Token() error = %v", err)
		}
		if got != want {
			t.Fatalf("Token() = %q, want %q", got, want)
		}
	})

	t.Run("refreshes expired token and persists auth file", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "auth.json")
		oldToken := testJWT(time.Now().Add(-time.Hour))
		if err := os.WriteFile(path, []byte(fmt.Sprintf(`{"auth_mode":"chatgpt","tokens":{"access_token":%q,"refresh_token":"old-refresh"}}`, oldToken)), 0o600); err != nil {
			t.Fatalf("WriteFile() error = %v", err)
		}
		newToken := testJWT(time.Now().Add(time.Hour))
		a := &Auth{
			auth:   authFile{Tokens: tokenStore{AccessToken: oldToken, RefreshToken: "old-refresh"}},
			path:   path,
			expiry: time.Now().Add(-time.Minute),
			httpClient: &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				if req.URL.String() != TokenEndpoint {
					t.Fatalf("refresh URL = %q, want %q", req.URL.String(), TokenEndpoint)
				}
				body, _ := io.ReadAll(req.Body)
				if !strings.Contains(string(body), "refresh_token=old-refresh") {
					t.Fatalf("refresh body = %q, want refresh token", string(body))
				}
				respBody := fmt.Sprintf(`{"access_token":%q,"refresh_token":"new-refresh","expires_in":3600}`, newToken)
				return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(respBody))}, nil
			})},
		}

		got, err := a.Token(context.Background())
		if err != nil {
			t.Fatalf("Token() error = %v", err)
		}
		if got != newToken {
			t.Fatalf("Token() = %q, want %q", got, newToken)
		}
		if a.auth.Tokens.RefreshToken != "new-refresh" {
			t.Fatalf("refresh token = %q, want new-refresh", a.auth.Tokens.RefreshToken)
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("ReadFile() error = %v", err)
		}
		if !strings.Contains(string(raw), "new-refresh") || !strings.Contains(string(raw), newToken) {
			t.Fatalf("saved auth file = %s, want refreshed tokens", string(raw))
		}
	})
}

func TestAuthSetHeaders(t *testing.T) {
	token := testJWT(time.Now().Add(time.Hour))
	a := &Auth{
		auth:       authFile{Tokens: tokenStore{AccessToken: token, AccountID: "acct-42"}},
		expiry:     time.Now().Add(time.Hour),
		httpClient: http.DefaultClient,
	}
	h := make(http.Header)
	if err := a.SetHeaders(context.Background(), h); err != nil {
		t.Fatalf("SetHeaders() error = %v", err)
	}
	if got := h.Get("Authorization"); got != "Bearer "+token {
		t.Fatalf("Authorization = %q, want Bearer token", got)
	}
	if got := h.Get(HeaderChatGPTAccountID); got != "acct-42" {
		t.Fatalf("%s = %q, want acct-42", HeaderChatGPTAccountID, got)
	}
}

func TestJWTExpiry(t *testing.T) {
	exp := time.Now().Add(2 * time.Hour).Truncate(time.Second)
	got, err := jwtExpiry(testJWT(exp))
	if err != nil {
		t.Fatalf("jwtExpiry() error = %v", err)
	}
	if !got.Equal(exp) {
		t.Fatalf("jwtExpiry() = %v, want %v", got, exp)
	}
}

func TestFetchRawModelsAndFetchModels(t *testing.T) {
	a := &Auth{
		auth:       authFile{Tokens: tokenStore{AccessToken: testJWT(time.Now().Add(time.Hour)), AccountID: "acct-9"}},
		expiry:     time.Now().Add(time.Hour),
		httpClient: http.DefaultClient,
	}
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.String() != ModelsEndpoint {
			t.Fatalf("request URL = %q, want %q", req.URL.String(), ModelsEndpoint)
		}
		if got := req.Header.Get("Authorization"); !strings.HasPrefix(got, "Bearer ") {
			t.Fatalf("Authorization = %q, want Bearer token", got)
		}
		if got := req.Header.Get(HeaderChatGPTAccountID); got != "acct-9" {
			t.Fatalf("%s = %q, want acct-9", HeaderChatGPTAccountID, got)
		}
		body := `{"models":[{"slug":"o3","display_name":"o3"},{"slug":"gpt-5.4","display_name":"GPT-5.4"},{"slug":"gpt-5.4-mini","display_name":""}]}`
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}, nil
	})}

	raw, err := FetchRawModels(context.Background(), a, client)
	if err != nil {
		t.Fatalf("FetchRawModels() error = %v", err)
	}
	if !strings.Contains(string(raw), `"models"`) {
		t.Fatalf("FetchRawModels() = %s, want models payload", string(raw))
	}

	models, err := FetchModels(context.Background(), a, client)
	if err != nil {
		t.Fatalf("FetchModels() error = %v", err)
	}
	wantIDs := []string{"gpt-5.4", "gpt-5.4-mini", "o3"}
	if len(models) != len(wantIDs) {
		t.Fatalf("len(models) = %d, want %d", len(models), len(wantIDs))
	}
	for i, want := range wantIDs {
		if models[i].ID != want {
			t.Fatalf("models[%d].ID = %q, want %q; models = %+v", i, models[i].ID, want, models)
		}
	}
	if models[1].Name != "gpt-5.4-mini" {
		t.Fatalf("models[1].Name = %q, want fallback slug", models[1].Name)
	}
}

func TestModelsResolve(t *testing.T) {
	models := Models{{ID: ModelGPT54, Name: "GPT-5.4"}, {ID: ModelGPT54Mini, Name: "GPT-5.4 Mini"}}

	if got, ok := models.Resolve(AliasCodex); !ok || got.ID != ModelGPT54 {
		t.Fatalf("Resolve(%q) = %+v, %v; want %q, true", AliasCodex, got, ok, ModelGPT54)
	}
	if got, ok := models.Resolve("missing-model"); ok || got.ID != "missing-model" || got.Name != "missing-model" {
		t.Fatalf("Resolve(missing-model) = %+v, %v; want fallback missing-model, false", got, ok)
	}
}

func TestLoadModelsHasFallbacks(t *testing.T) {
	models := LoadModels()
	if len(models) == 0 {
		t.Fatal("LoadModels() returned no models")
	}
	if _, ok := models.Resolve(ModelGPT54); !ok {
		t.Fatalf("LoadModels().Resolve(%q) did not find fallback model", ModelGPT54)
	}
}

func TestLoadAuthUsesOverridePath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "auth.json")
	content := map[string]any{
		"auth_mode": ChatGPTAuthMode,
		"tokens":    map[string]any{"refresh_token": "refresh"},
	}
	raw, _ := json.Marshal(content)
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	t.Setenv(EnvCodexAuthPath, path)
	if _, err := LoadAuth(); err != nil {
		t.Fatalf("LoadAuth() error = %v", err)
	}
}
