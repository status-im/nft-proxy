package main

import (
	"fmt"
	"os"

	"go.uber.org/zap"

	"nft-proxy/internal/alchemy"
	"nft-proxy/internal/config"
	"nft-proxy/internal/handlers"
)

// CompositionRoot holds all application dependencies and provides a centralized
// place for dependency injection and service initialization.
type CompositionRoot struct {
	// Configuration
	Config  *config.Config
	APIKeys []string
	Logger  *zap.Logger

	// Services
	AlchemyClient *alchemy.Client
	HTTPServer    *handlers.Server
	MetricsServer *handlers.MetricsServer
}

// NewCompositionRoot creates and initializes all application dependencies
func NewCompositionRoot() (*CompositionRoot, error) {
	root := &CompositionRoot{}

	if err := root.initLogger(); err != nil {
		return nil, fmt.Errorf("failed to initialize logger: %w", err)
	}

	if err := root.loadConfig(); err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	if err := root.loadAPIKeys(); err != nil {
		return nil, fmt.Errorf("failed to load API keys: %w", err)
	}

	if err := root.initServices(); err != nil {
		return nil, fmt.Errorf("failed to initialize services: %w", err)
	}

	if err := root.initHTTPServer(); err != nil {
		return nil, fmt.Errorf("failed to initialize HTTP server: %w", err)
	}

	if err := root.initMetricsServer(); err != nil {
		return nil, fmt.Errorf("failed to initialize metrics server: %w", err)
	}

	return root, nil
}

// initLogger initializes the application logger
func (r *CompositionRoot) initLogger() error {
	logger, err := zap.NewProduction()
	if err != nil {
		return err
	}
	r.Logger = logger
	return nil
}

// loadConfig loads the application configuration
func (r *CompositionRoot) loadConfig() error {
	configPath := os.Getenv("CACHE_CONFIG_FILE")
	if configPath == "" {
		configPath = "/app/cache_config.yaml"
	}

	cfg, err := config.LoadConfig(configPath, r.Logger)
	if err != nil {
		return err
	}

	r.Config = cfg
	return nil
}

// loadAPIKeys loads API keys from JSON file
func (r *CompositionRoot) loadAPIKeys() error {
	keysPath := os.Getenv("API_KEYS_FILE")
	if keysPath == "" {
		keysPath = "/app/secrets/alchemy_api_keys.json"
	}

	apiKeys, err := config.LoadAPIKeys(keysPath, r.Logger)
	if err != nil {
		return err
	}

	r.APIKeys = apiKeys
	return nil
}

// initServices initializes application services
func (r *CompositionRoot) initServices() error {
	// In the future, this could be enhanced with rotation logic
	apiKey := r.APIKeys[0]

	r.AlchemyClient = alchemy.NewClient(
		apiKey,
		r.Config.Alchemy.BaseURLs,
		r.Config.Alchemy.Retry,
	)

	return nil
}

// initHTTPServer initializes the HTTP server
func (r *CompositionRoot) initHTTPServer() error {
	r.HTTPServer = handlers.NewServer(
		r.AlchemyClient,
		r.Logger,
	)

	return nil
}

// Cleanup performs cleanup of all resources
func (r *CompositionRoot) Cleanup() error {
	if r.Logger != nil {
		if err := r.Logger.Sync(); err != nil {
			return fmt.Errorf("failed to sync logger: %w", err)
		}
	}

	return nil
}

// initMetricsServer initializes the metrics HTTP server
func (r *CompositionRoot) initMetricsServer() error {
	r.MetricsServer = handlers.NewMetricsServer(r.Logger)
	return nil
}

// GetSocketPath returns the Unix socket path for the server
func (r *CompositionRoot) GetSocketPath() string {
	return os.Getenv("CACHE_SOCKET_PATH")
}

// GetMetricsPort returns the port for the metrics HTTP server
func (r *CompositionRoot) GetMetricsPort() string {
	port := os.Getenv("METRICS_PORT")
	if port == "" {
		port = "9090"
	}
	return port
}
