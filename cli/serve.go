package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/codewandler/llmproviders/internal/serve"
	"github.com/spf13/cobra"
)

// ServeCommandOptions configures the serve command.
type ServeCommandOptions struct {
	IO          IO
	LoadService ServiceLoader
}

// NewServeCommand creates the "serve" command that starts an OpenAI-compatible
// Responses API proxy server.
func NewServeCommand(opts ServeCommandOptions) *cobra.Command {
	ioCfg := opts.IO.WithDefaults()

	var (
		addr     string
		path     string
		cors     bool
		logLevel string
		logFile  string
	)

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start an OpenAI-compatible Responses API proxy server",
		Long: `Start an HTTP server that exposes an OpenAI Responses API endpoint.

Incoming requests are routed through the llmproviders Service to any
detected upstream provider. Model aliases (sonnet, fast, powerful, etc.)
work transparently.

Examples:
  llmcli serve                                     # Listen on :8080
  llmcli serve --addr :3000                        # Custom port
  llmcli serve --cors                              # Enable CORS

Test with curl:
  curl -N http://localhost:8080/v1/responses \
    -H "Content-Type: application/json" \
    -d '{"model":"sonnet","input":[{"role":"user","content":"Hello"}],"stream":true}'

Test with OpenAI Python SDK:
  from openai import OpenAI
  client = OpenAI(base_url="http://localhost:8080/v1", api_key="unused")
  for event in client.responses.create(model="sonnet", input="Hello", stream=True):
      print(event)`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runServe(cmd.Context(), opts, ioCfg, serveParams{
				addr:     addr,
				path:     path,
				cors:     cors,
				logLevel: logLevel,
				logFile:  logFile,
			})
		},
	}

	cmd.SetOut(ioCfg.Out)
	cmd.SetErr(ioCfg.Err)

	f := cmd.Flags()
	f.StringVarP(&addr, "addr", "a", ":8080", "Listen address (host:port)")
	f.StringVar(&path, "path", "/v1/responses", "Endpoint path")
	f.BoolVar(&cors, "cors", false, "Enable CORS headers")
	f.StringVar(&logLevel, "log-level", "info", "Log level (debug, info, warn, error)")
	f.StringVar(&logFile, "log-file", "", "Log file path (optional, e.g. /tmp/llmcli.log)")

	return cmd
}

type serveParams struct {
	addr     string
	path     string
	cors     bool
	logLevel string
	logFile  string
}

func runServe(ctx context.Context, opts ServeCommandOptions, ioCfg IO, params serveParams) error {
	errOut := ioCfg.Err

	// Open log file if specified.
	var logFileHandle *os.File
	var logWriter io.Writer = errOut
	if params.logFile != "" {
		f, err := os.OpenFile(params.logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return fmt.Errorf("failed to open log file %q: %w", params.logFile, err)
		}
		logFileHandle = f
		// Write to both stderr and file
		logWriter = io.MultiWriter(errOut, f)
		fmt.Fprintf(errOut, "Logging to file: %s\n", params.logFile)
	}
	// Ensure log file is closed on shutdown.
	defer func() {
		if logFileHandle != nil {
			_ = logFileHandle.Close()
		}
	}()

	// Configure logger.
	var level slog.Level
	switch params.logLevel {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}
	logger := slog.New(slog.NewJSONHandler(&indentedJSONWriter{w: logWriter}, &slog.HandlerOptions{Level: level}))

	// Load the provider service.
	fmt.Fprintf(errOut, "Detecting providers...\n")
	svc, err := opts.LoadService(ctx)
	if err != nil {
		return fmt.Errorf("failed to load service: %w", err)
	}

	// Report detected providers.
	for _, inst := range svc.RegisteredInstances() {
		svcID, _ := svc.ServiceIDForInstance(inst)
		fmt.Fprintf(errOut, "  ✓ %s (%s)\n", inst, svcID)
	}
	fmt.Fprintln(errOut)

	// Build handler.
	handler := serve.NewHandler(svc, logger)
	validator, err := serve.NewOpenAPIValidator(params.path)
	if err != nil {
		return fmt.Errorf("failed to initialize OpenAPI validator: %w", err)
	}

	// Set up mux.
	mux := http.NewServeMux()

	var h http.Handler = handler
	h = validator.Middleware(h)
	if params.cors {
		h = corsMiddleware(h)
	}
	// Add request logging middleware
	h = loggingMiddleware(h, logger)
	mux.Handle(params.path, h)

	// Health endpoint.
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}` + "\n"))
	})

	// Start server.
	server := &http.Server{
		Addr:    params.addr,
		Handler: mux,
		BaseContext: func(_ net.Listener) context.Context {
			return ctx
		},
	}

	// Graceful shutdown goroutine.
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()

	fmt.Fprintf(errOut, "Listening on %s — POST %s\n", params.addr, params.path)
	fmt.Fprintf(errOut, "Health check: GET http://localhost%s/health\n\n", resolveAddr(params.addr))

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("server error: %w", err)
	}
	return nil
}

// corsMiddleware adds CORS headers to responses.
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// loggingMiddleware logs HTTP requests and responses.
func loggingMiddleware(next http.Handler, logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Wrap response writer to capture status code
		wrapped := &statusCaptureWriter{ResponseWriter: w, statusCode: http.StatusOK}

		logger.Debug("request received",
			"method", r.Method,
			"path", r.URL.Path,
			"remote_addr", r.RemoteAddr,
		)

		next.ServeHTTP(wrapped, r)

		logger.Debug("response sent",
			"method", r.Method,
			"path", r.URL.Path,
			"status", wrapped.statusCode,
		)
	})
}

// statusCaptureWriter wraps http.ResponseWriter to capture the status code.
// It also implements http.Flusher to support SSE streaming.
type statusCaptureWriter struct {
	http.ResponseWriter
	statusCode int
}

func (w *statusCaptureWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *statusCaptureWriter) Flush() {
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// resolveAddr ensures the address has a host part for display.
func resolveAddr(addr string) string {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return addr
	}
	if host == "" {
		return "localhost:" + port
	}
	return addr
}

// indentedJSONWriter reformats one-line slog JSON records into pretty-printed JSON.
type indentedJSONWriter struct {
	w io.Writer
}

func (w *indentedJSONWriter) Write(p []byte) (int, error) {
	trimmed := trimASCIISpace(p)
	if len(trimmed) == 0 {
		return w.w.Write(p)
	}

	var v any
	if err := json.Unmarshal(trimmed, &v); err != nil {
		return w.w.Write(p)
	}

	pretty, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return w.w.Write(p)
	}
	pretty = append(pretty, '\n')

	if _, err := w.w.Write(pretty); err != nil {
		return 0, err
	}
	return len(p), nil
}

func trimASCIISpace(b []byte) []byte {
	start := 0
	for start < len(b) {
		switch b[start] {
		case ' ', '\t', '\n', '\r':
			start++
		default:
			goto endStart
		}
	}

endStart:
	end := len(b)
	for end > start {
		switch b[end-1] {
		case ' ', '\t', '\n', '\r':
			end--
		default:
			return b[start:end]
		}
	}
	return b[start:end]
}
