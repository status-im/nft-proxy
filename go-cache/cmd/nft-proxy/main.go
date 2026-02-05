package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"
)

func main() {
	root, err := NewCompositionRoot()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize application: %v\n", err)
		os.Exit(1)
	}
	defer func() {
		if err := root.Cleanup(); err != nil {
			root.Logger.Error("Cleanup error", zap.Error(err))
		}
	}()

	socketPath := root.GetSocketPath()

	root.Logger.Info("Starting NFT proxy server on Unix socket", zap.String("socket_path", socketPath))
	go func() {
		if err := root.HTTPServer.StartUnixSocket(socketPath); err != nil {
			root.Logger.Error("Server failed to start on Unix socket", zap.Error(err))
		}
	}()

	metricsPort := root.GetMetricsPort()
	root.Logger.Info("Starting metrics server", zap.String("port", metricsPort))
	go func() {
		if err := root.MetricsServer.Start(metricsPort); err != nil {
			root.Logger.Error("Metrics server failed to start", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	root.Logger.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := root.HTTPServer.Stop(ctx); err != nil {
		root.Logger.Error("HTTP server forced to shutdown", zap.Error(err))
	}
	if err := root.MetricsServer.Stop(ctx); err != nil {
		root.Logger.Error("Metrics server forced to shutdown", zap.Error(err))
	}

	root.Logger.Info("Server exited")
}
