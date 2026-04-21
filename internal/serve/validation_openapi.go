package serve

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"io"
	"net/http"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers"
	"github.com/getkin/kin-openapi/routers/gorillamux"
)

//go:embed openai_responses_api.yaml
var openAIResponsesAPISpec []byte

type OpenAPIValidator struct {
	router      routers.Router
	requestPath string
}

func NewOpenAPIValidator(requestPath string) (*OpenAPIValidator, error) {
	loader := openapi3.NewLoader()
	doc, err := loader.LoadFromData(openAIResponsesAPISpec)
	if err != nil {
		return nil, fmt.Errorf("load embedded OpenAPI spec: %w", err)
	}

	router, err := gorillamux.NewRouter(doc)
	if err != nil {
		return nil, fmt.Errorf("build OpenAPI router: %w", err)
	}

	return &OpenAPIValidator{
		router:      router,
		requestPath: requestPath,
	}, nil
}

func (v *OpenAPIValidator) Middleware(next http.Handler) http.Handler {
	if v == nil {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != v.requestPath {
			next.ServeHTTP(w, r)
			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			WriteJSONError(w, http.StatusBadRequest, "read_error", fmt.Sprintf("Failed to read body: %v", err))
			return
		}
		_ = r.Body.Close()

		restore := func() io.ReadCloser {
			return io.NopCloser(bytes.NewReader(body))
		}
		r.Body = restore()
		r.GetBody = func() (io.ReadCloser, error) {
			return restore(), nil
		}

		validationReq := cloneRequestForValidation(r)
				validationReq.URL.Scheme = "https"
				validationReq.URL.Host = "api.openai.com"
				validationReq.Host = "api.openai.com"
				validationReq.URL.Path = "/v1/responses"
				validationReq.RequestURI = "/v1/responses"
		route, pathParams, err := v.router.FindRoute(validationReq)
		if err != nil {
			WriteJSONError(w, http.StatusInternalServerError, "validation_route_error", fmt.Sprintf("Failed to match OpenAPI route: %v", err))
			return
		}

		if err := openapi3filter.ValidateRequest(context.Background(), &openapi3filter.RequestValidationInput{
			Request:    validationReq,
			PathParams: pathParams,
			Route:      route,
			Options: &openapi3filter.Options{
				AuthenticationFunc: func(context.Context, *openapi3filter.AuthenticationInput) error {
					return nil
				},
			},
		}); err != nil {
			WriteJSONError(w, http.StatusBadRequest, "invalid_request", formatOpenAPIValidationError(err))
			return
		}

		// Restore the original request body for downstream consumers.
		r.Body = restore()
		r.GetBody = func() (io.ReadCloser, error) {
			return restore(), nil
		}

		next.ServeHTTP(w, r)
	})
}

func cloneRequestForValidation(r *http.Request) *http.Request {
	clone := r.Clone(r.Context())
	clone.URL.Path = r.URL.Path
	clone.RequestURI = r.URL.RequestURI()
	if r.GetBody != nil {
		body, err := r.GetBody()
		if err == nil {
			clone.Body = body
			clone.GetBody = r.GetBody
		}
	}
	return clone
}

func formatOpenAPIValidationError(err error) string {
	if err == nil {
		return "request validation failed"
	}
	return err.Error()
}
