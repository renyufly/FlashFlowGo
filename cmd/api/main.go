// Command api runs the FlashFlow modular-monolith HTTP API.
package main

import (
	"context"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/renyufly/FlashFlowGo/internal/platform/config"
	"github.com/renyufly/FlashFlowGo/internal/platform/httpserver"
)

var (
	version   = "dev"
	commit    = "none"
	buildTime = "unknown"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cfg, err := config.Load(os.LookupEnv)
	if err != nil {
		logger.Error("invalid configuration", "error", err)
		os.Exit(1)
	}
	logger.Info("configuration loaded", "config", cfg.SafeSummary())
	handler := httpserver.NewHandler(httpserver.BuildInfo{Version: version, Commit: commit, BuildTime: buildTime}, cfg.HTTP.HandlerTimeout)
	server := httpserver.New(cfg.HTTP, handler)
	listener, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", cfg.HTTP.Address)
	if err != nil {
		logger.Error("listen failed", "error", err)
		os.Exit(1)
	}
	logger.Info("HTTP server listening", "address", listener.Addr().String(), "version", version)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := httpserver.Serve(ctx, server, listener, cfg.HTTP.ShutdownTimeout, logger); err != nil {
		logger.Error("HTTP server stopped with error", "error", err)
		os.Exit(1)
	}
	logger.Info("HTTP server stopped")
}
