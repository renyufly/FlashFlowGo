package httpserver

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHealthIncludesVersionAndRequestID(t *testing.T) {
	t.Parallel()
	handler := NewHandler(BuildInfo{Version: "test-version", Commit: "abc123", BuildTime: "now"}, time.Second)
	request := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/healthz", nil)
	request.Header.Set(requestIDHeader, "request-from-client")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d", recorder.Code)
	}
	var response map[string]any
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}
	if response["version"] != "test-version" || response["request_id"] != "request-from-client" {
		t.Fatalf("unexpected response: %#v", response)
	}
}

func TestNotFoundUsesStableErrorEnvelope(t *testing.T) {
	t.Parallel()
	recorder := httptest.NewRecorder()
	NewHandler(BuildInfo{}, time.Second).ServeHTTP(recorder, httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/missing", nil))
	var response ErrorResponse
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}
	if recorder.Code != http.StatusNotFound || response.Code != "NOT_FOUND" || response.RequestID == "" {
		t.Fatalf("unexpected response: status=%d body=%#v", recorder.Code, response)
	}
}

func TestServeDrainsInflightRequest(t *testing.T) {
	listener, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	started := make(chan struct{})
	release := make(chan struct{})
	handler := http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		close(started)
		<-release
		writer.WriteHeader(http.StatusNoContent)
	})
	server := &http.Server{Handler: handler, ReadHeaderTimeout: time.Second}
	ctx, cancel := context.WithCancel(context.Background())
	serveDone := make(chan error, 1)
	go func() {
		serveDone <- Serve(ctx, server, listener, time.Second, slog.New(slog.NewTextHandler(io.Discard, nil)))
	}()
	requestDone := make(chan error, 1)
	go func() {
		request, requestErr := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://"+listener.Addr().String(), nil)
		if requestErr != nil {
			requestDone <- requestErr
			return
		}
		response, requestErr := http.DefaultClient.Do(request)
		if requestErr == nil {
			_ = response.Body.Close()
		}
		requestDone <- requestErr
	}()
	<-started
	cancel()
	close(release)
	if err := <-requestDone; err != nil {
		t.Fatalf("request: %v", err)
	}
	select {
	case err := <-serveDone:
		if err != nil {
			t.Fatalf("Serve() error = %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Serve() leaked its run goroutine after shutdown")
	}
}
