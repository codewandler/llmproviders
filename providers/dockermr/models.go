package dockermr

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// Model represents a Docker Model Runner model.
type Model struct {
	// ID is the model identifier (e.g., "ai/smollm2").
	ID string

	// Name is the display name.
	Name string
}

// Models is a list of models.
type Models []Model

// Resolve finds a model by ID.
func (m Models) Resolve(id string) (Model, bool) {
	id = strings.ToLower(id)
	for _, model := range m {
		if strings.ToLower(model.ID) == id {
			return model, true
		}
	}
	return Model{}, false
}

// FetchModels fetches the list of locally available models from Docker Model Runner.
func (p *Provider) FetchModels(ctx context.Context) ([]Model, error) {
	engineURL := EngineURL(p.cfg.BaseURL, p.cfg.Engine)
	endpoint := strings.TrimRight(engineURL, "/") + "/v1/models"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("dockermr list models: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("dockermr list models: status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Data []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode models response: %w", err)
	}

	models := make([]Model, len(result.Data))
	for i, m := range result.Data {
		name := m.Name
		if name == "" {
			name = m.ID
		}
		models[i] = Model{
			ID:   m.ID,
			Name: name,
		}
	}
	return models, nil
}

// GetModels returns the curated list of commonly used models.
func (p *Provider) GetModels() Models {
	return curatedModels
}

// curatedModels is the static list of commonly used Docker Model Runner models.
// These models are from Docker Hub's ai/ namespace.
var curatedModels = Models{
	// Tiny models (< 500M params) - very fast, minimal memory
	{ID: ModelSmoLLM2, Name: "SmolLM2 360M"},
	{ID: ModelSmoLLM2Tiny, Name: "SmolLM2 135M"},
	{ID: ModelQwen25Small, Name: "Qwen2.5 0.5B"},

	// Medium models (3-8B params)
	{ID: ModelQwen25, Name: "Qwen2.5 7B"},
	{ID: ModelQwen3, Name: "Qwen3"},
	{ID: ModelQwen3Coder, Name: "Qwen3 Coder"},
	{ID: ModelLlama32, Name: "Llama 3.2"},
	{ID: ModelLlama33, Name: "Llama 3.3"},
	{ID: ModelPhi4Mini, Name: "Phi-4 Mini"},
	{ID: ModelPhi4, Name: "Phi-4"},
	{ID: ModelGemma3, Name: "Gemma 3"},
	{ID: ModelGemma4, Name: "Gemma 4"},
	{ID: ModelMistralSmall, Name: "Mistral Small 3.2"},
	{ID: ModelGLM47Flash, Name: "GLM-4.7 Flash"},
	{ID: ModelGranite4Nano, Name: "Granite 4.0 Nano"},
	{ID: ModelFunctionGemma, Name: "FunctionGemma"},

	// Large models
	{ID: ModelDeepSeekR1, Name: "DeepSeek R1"},
}
