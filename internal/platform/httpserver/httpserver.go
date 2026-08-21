// Package httpserver provides the FlashFlow HTTP process baseline.
package httpserver

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/renyufly/FlashFlowGo/internal/platform/config"
)

const requestIDHeader = "X-Request-ID"

type contextKey string

const requestIDKey contextKey = "request_id"

// BuildInfo identifies the running artifact.
type BuildInfo struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	BuildTime string `json:"build_time"`
}

// ErrorResponse is the stable public error envelope.
type ErrorResponse struct {
	Code      string         `json:"code"`
	Message   string         `json:"message"`
	Details   map[string]any `json:"details,omitempty"`
	RequestID string         `json:"request_id"`
}

// NewHandler builds the Phase 0 router.
func NewHandler(build BuildInfo, handlerTimeout time.Duration) http.Handler {
	router := chi.NewRouter()
	router.Use(requestID)
	router.Use(middleware.Recoverer)
	router.Use(middleware.Timeout(handlerTimeout))
	router.Get("/healthz", func(writer http.ResponseWriter, request *http.Request) {
		writeJSON(writer, http.StatusOK, map[string]any{
			"status": "ok", "version": build.Version, "commit": build.Commit,
			"build_time": build.BuildTime, "request_id": RequestID(request.Context()),
		})
	})
	router.NotFound(func(writer http.ResponseWriter, request *http.Request) {
		WriteError(writer, request, http.StatusNotFound, "NOT_FOUND", "resource not found", nil)
	})
	return router
}

// New builds a configured server without starting network I/O.
func New(cfg config.HTTPConfig, handler http.Handler) *http.Server {
	return &http.Server{
		Addr: cfg.Address, Handler: handler,
		ReadHeaderTimeout: cfg.ReadHeaderTimeout, ReadTimeout: cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout, IdleTimeout: cfg.IdleTimeout,
	}
}

// Serve runs until cancellation and drains in-flight requests.
func Serve(ctx context.Context, server *http.Server, listener net.Listener, shutdownTimeout time.Duration, logger *slog.Logger) error {
	serveResult := make(chan error, 1)
	go func() { serveResult <- server.Serve(listener) }()
	select {
	case err := <-serveResult:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		logger.Info("shutdown requested")
	}
	shutdownContext, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := server.Shutdown(shutdownContext); err != nil {
		_ = server.Close()
		return fmt.Errorf("shutdown HTTP server: %w", err)
	}
	err := <-serveResult
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

// RequestID returns the correlation ID attached by middleware.
func RequestID(ctx context.Context) string {
	value, _ := ctx.Value(requestIDKey).(string)
	return value
}

// WriteError writes a stable JSON error response.
func WriteError(writer http.ResponseWriter, request *http.Request, status int, code, message string, details map[string]any) {
	writeJSON(writer, status, ErrorResponse{Code: code, Message: message, Details: details, RequestID: RequestID(request.Context())})
}

func requestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		id := strings.TrimSpace(request.Header.Get(requestIDHeader))
		if id == "" || len(id) > 128 {
			id = newRequestID()
		}
		writer.Header().Set(requestIDHeader, id)
		ctx := context.WithValue(request.Context(), requestIDKey, id)
		next.ServeHTTP(writer, request.WithContext(ctx))
	})
}

func newRequestID() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return fmt.Sprintf("fallback-%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(bytes)
}

func writeJSON(writer http.ResponseWriter, status int, value any) {
	writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(value)
}
