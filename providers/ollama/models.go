package ollama

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	ollamaapi "github.com/codewandler/agentapis/api/ollama"
)

// Model represents an Ollama model.
type Model struct {
	// ID is the model identifier (e.g., "qwen2.5:0.5b").
	ID string

	// Name is the display name.
	Name string

	// Size is the model size in bytes (if known).
	Size int64
}

// Models is a list of models.
type Models []Model

// Resolve finds a model by ID or alias.
func (m Models) Resolve(id string) (Model, bool) {
	id = strings.ToLower(id)
	for _, model := range m {
		if strings.ToLower(model.ID) == id {
			return model, true
		}
	}
	return Model{}, false
}

// FetchModels fetches the list of locally installed models from Ollama.
func (p *Provider) FetchModels(ctx context.Context) ([]Model, error) {
	resp, err := p.ollamaClient.ListModels(ctx)
	if err != nil {
		return nil, fmt.Errorf("ollama list models: %w", err)
	}

	models := make([]Model, len(resp.Models))
	for i, m := range resp.Models {
		name := m.Name
		if name == "" {
			name = m.Model
		}
		models[i] = Model{
			ID:   m.Name,
			Name: name,
			Size: m.Size,
		}
	}
	return models, nil
}

// Download pulls one or more models from the Ollama registry.
// It skips models that are already installed.
// Progress is reported via streaming; this method blocks until complete.
func (p *Provider) Download(ctx context.Context, modelIDs ...string) error {
	if len(modelIDs) == 0 {
		return nil
	}

	// Get list of installed models
	installed, err := p.FetchModels(ctx)
	if err != nil {
		return fmt.Errorf("fetch installed models: %w", err)
	}

	installedMap := make(map[string]bool)
	for _, m := range installed {
		installedMap[m.ID] = true
	}

	// Download each model that isn't already installed
	for _, modelID := range modelIDs {
		if installedMap[modelID] {
			continue
		}
		if err := p.downloadModel(ctx, modelID); err != nil {
			return fmt.Errorf("download %s: %w", modelID, err)
		}
	}
	return nil
}

// downloadModel pulls a single model from the Ollama registry.
func (p *Provider) downloadModel(ctx context.Context, modelID string) error {
	reqBody := map[string]any{
		"name":   modelID,
		"stream": true,
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}

	endpoint := strings.TrimRight(p.cfg.BaseURL, "/") + "/api/pull"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("pull request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		errBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("pull failed with status %d: %s", resp.StatusCode, string(errBody))
	}

	// Stream the pull progress
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		var status struct {
			Status string `json:"status"`
			Error  string `json:"error"`
		}
		if err := json.Unmarshal(scanner.Bytes(), &status); err != nil {
			continue
		}
		if status.Error != "" {
			return fmt.Errorf("pull error: %s", status.Error)
		}
		if status.Status == "success" || strings.Contains(strings.ToLower(status.Status), "successfully") {
			return nil
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read pull response: %w", err)
	}

	return nil
}

// GetModels returns the curated list of commonly used models.
func (p *Provider) GetModels() Models {
	return curatedModels
}

// curatedModels is the static list of commonly used Ollama models.
var curatedModels = Models{
	// Small models (< 2B params) - fast, low memory
	{ID: ModelQwen2505B, Name: "Qwen 2.5 0.5B"},
	{ID: ModelQwen306B, Name: "Qwen 3 0.6B"},
	{ID: ModelLlama321B, Name: "Llama 3.2 1B"},
	{ID: ModelQwen317B, Name: "Qwen 3 1.7B"},
	{ID: ModelGranite31MoE1B, Name: "Granite 3.1 MoE 1B"},

	// Medium models (3-8B params)
	{ID: ModelLlama32, Name: "Llama 3.2"},
	{ID: ModelLlama31, Name: "Llama 3.1"},
	{ID: ModelQwen25, Name: "Qwen 2.5"},
	{ID: ModelPhi3, Name: "Phi-3"},
	{ID: ModelMistral, Name: "Mistral"},
	{ID: ModelGemma3, Name: "Gemma 3"},
	{ID: ModelGLM47Flash, Name: "GLM-4.7 Flash"},
	{ID: ModelMinistral38B, Name: "Ministral 3 8B"},

	// Specialized models
	{ID: ModelDeepSeekR1, Name: "DeepSeek R1"},
	{ID: ModelFunctionGemma, Name: "FunctionGemma"},
	{ID: ModelDevstralSmall2, Name: "Devstral Small 2"},
}

// Ensure we use the ollamaapi types (compile-time check)
var _ = ollamaapi.TagsResponse{}
