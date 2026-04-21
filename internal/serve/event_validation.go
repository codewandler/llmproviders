package serve

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/getkin/kin-openapi/openapi3"
)

type OpenAPIEventValidator struct {
	doc     *openapi3.T
	schemas map[string]*openapi3.Schema
}

func NewOpenAPIEventValidator() (*OpenAPIEventValidator, error) {
	loader := openapi3.NewLoader()
	doc, err := loader.LoadFromData(openAIResponsesAPISpec)
	if err != nil {
		return nil, fmt.Errorf("load embedded OpenAPI spec: %w", err)
	}

	v := &OpenAPIEventValidator{
		doc:     doc,
		schemas: make(map[string]*openapi3.Schema),
	}
	return v, nil
}

func (v *OpenAPIEventValidator) ValidateSSEEvent(ev SSEEvent) error {
	if v == nil {
		return nil
	}
	var payload any
	if err := json.Unmarshal(ev.Data, &payload); err != nil {
		return fmt.Errorf("unmarshal SSE payload for validation: %w", err)
	}

	eventType := ev.Name
	if eventType == "" {
		if m, ok := payload.(map[string]any); ok {
			if s, ok := m["type"].(string); ok {
				eventType = s
			}
		}
	}
	if eventType == "" {
		return fmt.Errorf("SSE event missing event name and type")
	}

	schema, err := v.schemaForEventType(eventType)
	if err != nil {
		return err
	}
	return v.doc.ValidateSchemaJSON(schema, payload, openapi3.MultiErrors())
}

func (v *OpenAPIEventValidator) schemaForEventType(eventType string) (*openapi3.Schema, error) {
	if schema, ok := v.schemas[eventType]; ok {
		return schema, nil
	}

	streamSchemaRef, ok := v.doc.Components.Schemas["ResponseStreamEvent"]
	if !ok || streamSchemaRef == nil || streamSchemaRef.Value == nil {
		return nil, fmt.Errorf("ResponseStreamEvent schema not found")
	}

	for _, candidate := range streamSchemaRef.Value.AnyOf {
		if candidate == nil || candidate.Value == nil {
			continue
		}
		schema := candidate.Value
		prop, ok := schema.Properties["type"]
		if !ok || prop == nil || prop.Value == nil || len(prop.Value.Enum) == 0 {
			continue
		}
		if enumValue, ok := prop.Value.Enum[0].(string); ok && enumValue == eventType {
			v.schemas[eventType] = schema
			return schema, nil
		}
	}

	return nil, fmt.Errorf("no OpenAPI schema found for SSE event type %q", eventType)
}

var (
	defaultEventValidatorOnce sync.Once
	defaultEventValidator     *OpenAPIEventValidator
	defaultEventValidatorErr  error
)

func defaultOpenAPIEventValidator() (*OpenAPIEventValidator, error) {
	defaultEventValidatorOnce.Do(func() {
		defaultEventValidator, defaultEventValidatorErr = NewOpenAPIEventValidator()
	})
	return defaultEventValidator, defaultEventValidatorErr
}
